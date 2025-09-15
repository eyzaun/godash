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

// DetailedMetricsWebSocketResponse - DETAYLI WEBSOCKET VERİ FORMAT (SPEED FIELDS ADDED)
type DetailedMetricsWebSocketResponse struct {
	// Basic metrics (frontend'in chart'ları için)
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	Hostname      string  `json:"hostname"`
	Timestamp     string  `json:"timestamp"`

	// Detailed CPU info (frontend'in detail alanları için)
	CPUCores       int       `json:"cpu_cores"`
	CPUFrequency   float64   `json:"cpu_frequency"`
	CPULoadAvg     []float64 `json:"cpu_load_avg"`
	CPUTemperature float64   `json:"cpu_temperature_c"`

	// Detailed Memory info
	MemoryTotal     uint64 `json:"memory_total"`
	MemoryUsed      uint64 `json:"memory_used"`
	MemoryAvailable uint64 `json:"memory_available"`
	MemoryFree      uint64 `json:"memory_free"`
	MemoryCached    uint64 `json:"memory_cached"`
	MemoryBuffers   uint64 `json:"memory_buffers"`

	// Detailed Disk info (SPEED FIELDS ADDED)
	DiskTotal      uint64 `json:"disk_total"`
	DiskUsed       uint64 `json:"disk_used"`
	DiskFree       uint64 `json:"disk_free"`
	DiskReadBytes  uint64 `json:"disk_read_bytes"`
	DiskWriteBytes uint64 `json:"disk_write_bytes"`
	// NEW: Disk I/O Speed
	DiskReadSpeed  float64 `json:"disk_read_speed_mbps"`
	DiskWriteSpeed float64 `json:"disk_write_speed_mbps"`
	// NEW: Individual disk partitions
	DiskPartitions []models.PartitionInfo `json:"disk_partitions"`

	// Network info (SPEED FIELDS ADDED)
	NetworkSent     uint64 `json:"network_sent"`
	NetworkReceived uint64 `json:"network_received"`
	NetworkErrors   uint64 `json:"network_errors"`
	NetworkDrops    uint64 `json:"network_drops"`
	// NEW: Network Speed
	NetworkUploadSpeed   float64 `json:"network_upload_speed_mbps"`
	NetworkDownloadSpeed float64 `json:"network_download_speed_mbps"`

	// System info
	Platform     string        `json:"platform"`
	Uptime       time.Duration `json:"uptime"`
	ProcessCount uint64        `json:"process_count"`

	// Process Activity
	Processes *models.ProcessActivity `json:"processes"`
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

// BroadcastMetrics broadcasts detailed metrics to all connected clients (SPEED FIELDS ADDED)
func (h *WebSocketHandler) BroadcastMetrics(metrics *models.SystemMetrics) {
	if metrics == nil {
		log.Printf("Warning: received nil metrics for broadcast")
		return
	}

	// Get system info for additional details
	systemInfo, err := h.systemCollector.GetSystemInfo()
	if err != nil {
		log.Printf("Warning: failed to get system info for broadcast: %v", err)
	}

	// Convert to detailed structure for frontend (SPEED FIELDS ADDED)
	detailedMetrics := DetailedMetricsWebSocketResponse{
		// Basic metrics (frontend'in chart'ları için)
		CPUUsage:      metrics.CPU.Usage,
		MemoryPercent: metrics.Memory.Percent,
		DiskPercent:   metrics.Disk.Percent,
		Hostname:      metrics.Hostname,
		Timestamp:     metrics.Timestamp.Format(time.RFC3339),

		// Detailed CPU info (frontend'in detail alanları için)
		CPUCores:       metrics.CPU.Cores,
		CPUFrequency:   metrics.CPU.Frequency,
		CPULoadAvg:     metrics.CPU.LoadAvg,
		CPUTemperature: metrics.CPU.Temperature,

		// Detailed Memory info
		MemoryTotal:     metrics.Memory.Total,
		MemoryUsed:      metrics.Memory.Used,
		MemoryAvailable: metrics.Memory.Available,
		MemoryFree:      metrics.Memory.Free,
		MemoryCached:    metrics.Memory.Cached,
		MemoryBuffers:   metrics.Memory.Buffers,

		// Detailed Disk info (SPEED FIELDS ADDED)
		DiskTotal:      metrics.Disk.Total,
		DiskUsed:       metrics.Disk.Used,
		DiskFree:       metrics.Disk.Free,
		DiskReadBytes:  metrics.Disk.IOStats.ReadBytes,
		DiskWriteBytes: metrics.Disk.IOStats.WriteBytes,
		// NEW: Disk I/O Speed
		DiskReadSpeed:  metrics.Disk.ReadSpeed,
		DiskWriteSpeed: metrics.Disk.WriteSpeed,
		// NEW: Individual disk partitions
		DiskPartitions: metrics.Disk.Partitions,

		// Network info (SPEED FIELDS ADDED)
		NetworkSent:     metrics.Network.TotalSent,
		NetworkReceived: metrics.Network.TotalReceived,
		// NEW: Network Speed
		NetworkUploadSpeed:   metrics.Network.UploadSpeed,
		NetworkDownloadSpeed: metrics.Network.DownloadSpeed,

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

	// Add system info if available (FIX: Proper nil check)
	if systemInfo != nil {
		detailedMetrics.Platform = systemInfo.Platform
		detailedMetrics.ProcessCount = systemInfo.Processes
	} else {
		detailedMetrics.Platform = "Unknown"
		detailedMetrics.ProcessCount = 0
	}

	// Add process activity data
	detailedMetrics.Processes = &metrics.Processes

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
		// Debug log removed - too verbose in production
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

	// Add system info if available (FIX: Proper nil check)
	if systemInfo != nil {
		status["platform"] = systemInfo.Platform
		status["platform_version"] = systemInfo.PlatformVersion
		status["kernel_arch"] = systemInfo.KernelArch
		status["process_count"] = systemInfo.Processes
	} else {
		status["platform"] = "Unknown"
		status["platform_version"] = "Unknown"
		status["kernel_arch"] = "Unknown"
		status["process_count"] = uint64(0)
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
		// Debug log removed - too verbose in production
	default:
		log.Println("Broadcast channel full, dropping system status message")
	}
}

// BroadcastAlert broadcasts alert notifications to all connected clients
func (h *WebSocketHandler) BroadcastAlert(alertData map[string]interface{}) {
	message := WebSocketMessage{
		Type:      "alert_triggered",
		Data:      alertData,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal alert for broadcast: %v", err)
		return
	}

	select {
	case h.hub.broadcast <- data:
		log.Printf("🚨 Alert broadcasted to %d clients", h.GetConnectedClients())
	default:
		log.Println("Broadcast channel full, dropping alert message")
	}
}

// GetConnectedClients returns the number of connected WebSocket clients
func (h *WebSocketHandler) GetConnectedClients() int {
	if h.hub == nil {
		return 0
	}
	h.hub.mutex.RLock()
	defer h.hub.mutex.RUnlock()
	return len(h.hub.clients)
}

// GetClientStats returns WebSocket client statistics
func (h *WebSocketHandler) GetClientStats() map[string]interface{} {
	if h.hub == nil {
		return map[string]interface{}{
			"connected_clients": 0,
			"total_connections": 0,
			"broadcast_queue":   0,
		}
	}

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
	if client == nil {
		return
	}

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
		if err := c.conn.Close(); err != nil {
			log.Printf("Error closing WebSocket connection: %v", err)
		}
	}()

	c.conn.SetReadLimit(maxMessageSize)

	// FIX: Handle SetReadDeadline error
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Error setting read deadline: %v", err)
		return
	}

	c.conn.SetPongHandler(func(string) error {
		// FIX: Handle SetReadDeadline error in pong handler
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("Error setting read deadline in pong handler: %v", err)
		}
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
		if err := c.conn.Close(); err != nil {
			log.Printf("Error closing WebSocket connection: %v", err)
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			// FIX: Handle SetWriteDeadline error
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Error setting write deadline: %v", err)
				return
			}

			if !ok {
				// The hub closed the channel
				// FIX: Handle WriteMessage error
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("Error writing close message: %v", err)
				}
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Error getting next writer: %v", err)
				return
			}

			// FIX: Handle w.Write errors
			if _, err := w.Write(message); err != nil {
				log.Printf("Error writing message: %v", err)
				return
			}

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				if _, err := w.Write([]byte{'\n'}); err != nil {
					log.Printf("Error writing newline: %v", err)
					return
				}
				if _, err := w.Write(<-c.send); err != nil {
					log.Printf("Error writing queued message: %v", err)
					return
				}
			}

			if err := w.Close(); err != nil {
				log.Printf("Error closing writer: %v", err)
				return
			}

		case <-ticker.C:
			// FIX: Handle SetWriteDeadline error for ping
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Error setting write deadline for ping: %v", err)
				return
			}

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error writing ping message: %v", err)
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

// StartMetricsBroadcast starts periodic metrics broadcasting (SPEED SUPPORT ADDED)
func (h *WebSocketHandler) StartMetricsBroadcast(ctx context.Context, interval time.Duration) {
	log.Printf("📡 Starting detailed metrics broadcast with speed support - %v interval", interval)
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

			// Get FRESH system metrics from collector (REAL-TIME data with speed)
			systemMetrics, err := h.systemCollector.GetSystemMetrics()
			if err != nil {
				log.Printf("Failed to get system metrics for broadcast: %v", err)
				continue
			}

			// Broadcast detailed metrics with speed to all clients
			h.BroadcastMetrics(systemMetrics)

			// Also broadcast system status periodically (every 30 seconds)
			if time.Now().Unix()%30 == 0 {
				h.BroadcastSystemStatus()
			}

		case <-ctx.Done():
			log.Println("📡 Metrics broadcast with speed support stopped")
			return
		}
	}
}
