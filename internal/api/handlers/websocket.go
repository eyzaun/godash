package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/eyzaun/godash/internal/collector"
	"github.com/eyzaun/godash/internal/models"
	"github.com/eyzaun/godash/internal/repository"
)

// WebSocketHandler handles WebSocket connections for real-time metrics
type WebSocketHandler struct {
	metricsRepo     repository.MetricsRepository
	systemCollector collector.Collector
	hub             *Hub
}

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

// Client represents a WebSocket client connection
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	clientID  string
	lastPing  time.Time
	userAgent string
}

// WebSocketMessage represents messages sent over WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// DetailedMetricsWebSocketResponse - DETAYLI WEBSOCKET VERİ FORMAT
type DetailedMetricsWebSocketResponse struct {
	// Basic metrics (frontend'in chart'ları için)
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	Hostname      string  `json:"hostname"`
	Timestamp     string  `json:"timestamp"`

	// Detailed CPU info (frontend'in detail alanları için)
	CPUCores     int       `json:"cpu_cores"`
	CPUFrequency float64   `json:"cpu_frequency"`
	CPULoadAvg   []float64 `json:"cpu_load_avg"`

	// Detailed Memory info
	MemoryTotal     uint64 `json:"memory_total"`
	MemoryUsed      uint64 `json:"memory_used"`
	MemoryAvailable uint64 `json:"memory_available"`
	MemoryFree      uint64 `json:"memory_free"`
	MemoryCached    uint64 `json:"memory_cached"`
	MemoryBuffers   uint64 `json:"memory_buffers"`

	// Detailed Disk info
	DiskTotal      uint64 `json:"disk_total"`
	DiskUsed       uint64 `json:"disk_used"`
	DiskFree       uint64 `json:"disk_free"`
	DiskReadBytes  uint64 `json:"disk_read_bytes"`
	DiskWriteBytes uint64 `json:"disk_write_bytes"`

	// Network info
	NetworkSent     uint64 `json:"network_sent"`
	NetworkReceived uint64 `json:"network_received"`
	NetworkErrors   uint64 `json:"network_errors"`
	NetworkDrops    uint64 `json:"network_drops"`

	// System info
	Platform     string        `json:"platform"`
	Uptime       time.Duration `json:"uptime"`
	ProcessCount uint64        `json:"process_count"`
}

// WebSocket upgrader with proper configuration
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
	EnableCompression: true,
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(metricsRepo repository.MetricsRepository, systemCollector collector.Collector) *WebSocketHandler {
	hub := &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	handler := &WebSocketHandler{
		metricsRepo:     metricsRepo,
		systemCollector: systemCollector,
		hub:             hub,
	}

	// Start the hub
	go hub.run()

	return handler
}

// HandleWebSocket handles WebSocket connection requests
// @Summary WebSocket connection for real-time metrics
// @Description Establish WebSocket connection for real-time system metrics streaming
// @Tags websocket
// @Accept json
// @Produce json
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {object} APIResponse
// @Router /ws [get]
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	log.Printf("🌐 WebSocket connection attempt from %s", c.ClientIP())

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Failed to upgrade connection to WebSocket",
			Message: err.Error(),
		})
		return
	}

	log.Printf("✅ WebSocket upgrade successful")

	// Extract client information
	clientID := c.GetHeader("X-Client-ID")
	if clientID == "" {
		clientID = generateClientID()
	}

	userAgent := c.GetHeader("User-Agent")

	// Create new client
	client := &Client{
		hub:       h.hub,
		conn:      conn,
		send:      make(chan []byte, 256),
		clientID:  clientID,
		lastPing:  time.Now(),
		userAgent: userAgent,
	}

	// Register client
	client.hub.register <- client

	// Start goroutines for this client
	go client.writePump()
	go client.readPump()

	log.Printf("WebSocket client connected: %s (%s)", clientID, userAgent)
}

// BroadcastMetrics broadcasts detailed metrics to all connected clients (DÜZELTİLMİŞ VERSİYON)
func (h *WebSocketHandler) BroadcastMetrics(metrics *models.SystemMetrics) {
	// Get system info for additional details
	systemInfo, err := h.systemCollector.GetSystemInfo()
	if err != nil {
		log.Printf("Warning: failed to get system info for broadcast: %v", err)
	}

	// Convert to detailed structure for frontend
	detailedMetrics := DetailedMetricsWebSocketResponse{
		// Basic metrics (frontend'in chart'ları için)
		CPUUsage:      metrics.CPU.Usage,
		MemoryPercent: metrics.Memory.Percent,
		DiskPercent:   metrics.Disk.Percent,
		Hostname:      metrics.Hostname,
		Timestamp:     metrics.Timestamp.Format(time.RFC3339),

		// Detailed CPU info (frontend'in detail alanları için)
		CPUCores:     metrics.CPU.Cores,
		CPUFrequency: metrics.CPU.Frequency,
		CPULoadAvg:   metrics.CPU.LoadAvg,

		// Detailed Memory info
		MemoryTotal:     metrics.Memory.Total,
		MemoryUsed:      metrics.Memory.Used,
		MemoryAvailable: metrics.Memory.Available,
		MemoryFree:      metrics.Memory.Free,
		MemoryCached:    metrics.Memory.Cached,
		MemoryBuffers:   metrics.Memory.Buffers,

		// Detailed Disk info
		DiskTotal:      metrics.Disk.Total,
		DiskUsed:       metrics.Disk.Used,
		DiskFree:       metrics.Disk.Free,
		DiskReadBytes:  metrics.Disk.IOStats.ReadBytes,
		DiskWriteBytes: metrics.Disk.IOStats.WriteBytes,

		// Network info
		NetworkSent:     metrics.Network.TotalSent,
		NetworkReceived: metrics.Network.TotalReceived,

		// System info
		Uptime: metrics.Uptime,
	}

	// Network aggregation
	var totalErrors, totalDrops uint64
	for _, iface := range metrics.Network.Interfaces {
		totalErrors += iface.Errors
		totalDrops += iface.Drops
	}
	detailedMetrics.NetworkErrors = totalErrors
	detailedMetrics.NetworkDrops = totalDrops

	// Add system info if available
	if systemInfo != nil {
		detailedMetrics.Platform = systemInfo.Platform
		detailedMetrics.ProcessCount = systemInfo.Processes
	}

	message := WebSocketMessage{
		Type:      "metrics",
		Data:      detailedMetrics,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal detailed metrics for broadcast: %v", err)
		return
	}

	// Non-blocking broadcast - always send for real-time updates
	select {
	case h.hub.broadcast <- data:
		log.Printf("📡 Broadcasting detailed metrics: CPU=%.1f%%, Memory=%.1f%%, Disk=%.1f%%, Cores=%d, Freq=%.0f MHz",
			detailedMetrics.CPUUsage, detailedMetrics.MemoryPercent, detailedMetrics.DiskPercent,
			detailedMetrics.CPUCores, detailedMetrics.CPUFrequency)
	default:
		log.Println("Broadcast channel full, dropping detailed message")
	}
}

// BroadcastSystemStatus broadcasts system status to all connected clients (DÜZELTİLMİŞ)
func (h *WebSocketHandler) BroadcastSystemStatus() {
	// Get fresh system metrics from collector
	systemMetrics, err := h.systemCollector.GetSystemMetrics()
	if err != nil {
		log.Printf("Failed to get system metrics for status broadcast: %v", err)
		return
	}

	// Get system info
	systemInfo, err := h.systemCollector.GetSystemInfo()
	if err != nil {
		log.Printf("Warning: failed to get system info for status broadcast: %v", err)
	}

	// Create comprehensive system status
	status := map[string]interface{}{
		"status":        "running",
		"cpu_usage":     systemMetrics.CPU.Usage,
		"memory_usage":  systemMetrics.Memory.Percent,
		"disk_usage":    systemMetrics.Disk.Percent,
		"hostname":      systemMetrics.Hostname,
		"uptime":        systemMetrics.Uptime.String(),
		"timestamp":     time.Now(),
		"cpu_cores":     systemMetrics.CPU.Cores,
		"cpu_frequency": systemMetrics.CPU.Frequency,
		"memory_total":  systemMetrics.Memory.Total,
		"disk_total":    systemMetrics.Disk.Total,
		"network_sent":  systemMetrics.Network.TotalSent,
		"network_recv":  systemMetrics.Network.TotalReceived,
	}

	// Add system info if available
	if systemInfo != nil {
		status["platform"] = systemInfo.Platform
		status["platform_version"] = systemInfo.PlatformVersion
		status["kernel_arch"] = systemInfo.KernelArch
		status["process_count"] = systemInfo.Processes
	}

	message := WebSocketMessage{
		Type:      "system_status",
		Data:      status,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal system status for broadcast: %v", err)
		return
	}

	select {
	case h.hub.broadcast <- data:
		log.Printf("📊 Broadcasting system status: CPU=%.1f%%, Memory=%.1f%%, Platform=%s",
			systemMetrics.CPU.Usage, systemMetrics.Memory.Percent,
			func() string {
				if systemInfo != nil {
					return systemInfo.Platform
				}
				return "Unknown"
			}())
	default:
		log.Println("Broadcast channel full, dropping system status message")
	}
}

// GetConnectedClients returns the number of connected WebSocket clients
func (h *WebSocketHandler) GetConnectedClients() int {
	h.hub.mutex.RLock()
	defer h.hub.mutex.RUnlock()
	return len(h.hub.clients)
}

// GetClientStats returns WebSocket client statistics
func (h *WebSocketHandler) GetClientStats() map[string]interface{} {
	h.hub.mutex.RLock()
	defer h.hub.mutex.RUnlock()

	stats := map[string]interface{}{
		"connected_clients": len(h.hub.clients),
		"total_connections": len(h.hub.clients), // Could track lifetime connections
		"broadcast_queue":   len(h.hub.broadcast),
	}

	return stats
}

// Hub methods

// run starts the hub and handles client registration/unregistration
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

			// Send initial data to new client
			go h.sendInitialData(client)

			log.Printf("WebSocket client registered. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mutex.Unlock()

			log.Printf("WebSocket client unregistered. Total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mutex.RLock()
			clients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mutex.RUnlock()

			// Send to all clients
			for _, client := range clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, remove client
					h.mutex.Lock()
					delete(h.clients, client)
					close(client.send)
					h.mutex.Unlock()
				}
			}
		}
	}
}

// sendInitialData sends initial system data to a new client
func (h *Hub) sendInitialData(client *Client) {
	// Send connection confirmation
	initialMessage := WebSocketMessage{
		Type:      "connected",
		Data:      map[string]string{"status": "connected", "client_id": client.clientID},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(initialMessage)
	if err != nil {
		log.Printf("Failed to marshal initial data: %v", err)
		return
	}

	select {
	case client.send <- data:
		log.Printf("✅ Sent initial connection confirmation to client %s", client.clientID)
	default:
		log.Printf("Failed to send initial data to client %s", client.clientID)
	}
}

// Client methods

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.lastPing = time.Now()
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages from client
		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from WebSocket clients
func (c *Client) handleMessage(message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Invalid message from client %s: %v", c.clientID, err)
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		log.Printf("Message without type from client %s", c.clientID)
		return
	}

	switch msgType {
	case "ping":
		// Respond with pong
		pongMsg := WebSocketMessage{
			Type:      "pong",
			Data:      map[string]string{"status": "alive"},
			Timestamp: time.Now(),
		}

		data, _ := json.Marshal(pongMsg)
		select {
		case c.send <- data:
		default:
		}

	case "subscribe":
		// Handle subscription to specific metrics
		log.Printf("Client %s subscribed to metrics", c.clientID)

	case "unsubscribe":
		// Handle unsubscription
		log.Printf("Client %s unsubscribed from metrics", c.clientID)

	default:
		log.Printf("Unknown message type '%s' from client %s", msgType, c.clientID)
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// StartMetricsBroadcast starts periodic metrics broadcasting (DÜZELTİLMİŞ VERSİYON)
func (h *WebSocketHandler) StartMetricsBroadcast(ctx context.Context, interval time.Duration) {
	log.Printf("📡 Starting detailed metrics broadcast with %v interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if we have any clients
			clientCount := h.GetConnectedClients()
			if clientCount == 0 {
				// log.Printf("⏸️ No WebSocket clients connected, skipping broadcast") // Too verbose
				continue
			}

			// Get FRESH system metrics from collector (REAL-TIME data)
			systemMetrics, err := h.systemCollector.GetSystemMetrics()
			if err != nil {
				log.Printf("Failed to get system metrics for broadcast: %v", err)
				continue
			}

			// Broadcast detailed metrics to all clients
			h.BroadcastMetrics(systemMetrics)

			// Also broadcast system status periodically (every 30 seconds)
			if time.Now().Unix()%30 == 0 {
				h.BroadcastSystemStatus()
			}

		case <-ctx.Done():
			log.Println("📡 Metrics broadcast stopped")
			return
		}
	}
}
