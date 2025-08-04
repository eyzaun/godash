package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/eyzaun/godash/internal/api/handlers"
	"github.com/eyzaun/godash/internal/api/middleware"
	"github.com/eyzaun/godash/internal/config"
	"github.com/eyzaun/godash/internal/repository"
	"github.com/eyzaun/godash/internal/services"
)

// Router wraps gin router with dependencies
type Router struct {
	engine           *gin.Engine
	config           *config.Config
	metricsRepo      repository.MetricsRepository
	collectorService *services.CollectorService
	metricsHandler   *handlers.MetricsHandler
	healthHandler    *handlers.HealthHandler
	websocketHandler *handlers.WebSocketHandler
}

// New creates a new API router
func New(cfg *config.Config, metricsRepo repository.MetricsRepository, collectorService *services.CollectorService) *Router {
	// Set gin mode
	gin.SetMode(cfg.Server.Mode)

	// Create gin engine
	engine := gin.New()

	// Create handlers with system collector support
	metricsHandler := handlers.NewMetricsHandler(metricsRepo, collectorService.GetSystemCollector())
	healthHandler := handlers.NewHealthHandler(metricsRepo)
	websocketHandler := handlers.NewWebSocketHandler(metricsRepo, collectorService.GetSystemCollector())

	router := &Router{
		engine:           engine,
		config:           cfg,
		metricsRepo:      metricsRepo,
		collectorService: collectorService,
		metricsHandler:   metricsHandler,
		healthHandler:    healthHandler,
		websocketHandler: websocketHandler,
	}

	// Setup middleware
	router.setupMiddleware()

	// Setup routes
	router.setupRoutes()

	// Start WebSocket metrics broadcasting
	go router.startWebSocketBroadcasting()

	return router
}

// setupMiddleware configures global middleware
func (r *Router) setupMiddleware() {
	// Recovery middleware
	r.engine.Use(gin.Recovery())

	// Custom logger middleware
	r.engine.Use(middleware.Logger())

	// Request ID middleware
	r.engine.Use(middleware.RequestID())

	// CORS middleware
	r.engine.Use(cors.New(cors.Config{
		AllowOrigins:     r.config.Server.CORS.AllowOrigins,
		AllowMethods:     r.config.Server.CORS.AllowMethods,
		AllowHeaders:     r.config.Server.CORS.AllowHeaders,
		ExposeHeaders:    r.config.Server.CORS.ExposeHeaders,
		AllowCredentials: r.config.Server.CORS.AllowCredentials,
		MaxAge:           r.config.Server.CORS.MaxAge,
	}))

	// Rate limiting middleware (if needed)
	if !r.config.IsDevelopment() {
		r.engine.Use(middleware.RateLimit())
	}

	// Security headers middleware
	r.engine.Use(middleware.SecurityHeaders())

	// API versioning middleware
	r.engine.Use(middleware.APIVersion())
}

// setupRoutes configures all API routes
func (r *Router) setupRoutes() {
	// Health check routes (no API version prefix)
	r.engine.GET("/health", r.healthHandler.HealthCheck)
	r.engine.GET("/ready", r.healthHandler.ReadinessCheck)
	r.engine.GET("/metrics", r.healthHandler.PrometheusMetrics) // For monitoring tools

	// WebSocket endpoint for real-time metrics
	r.engine.GET("/ws", r.websocketHandler.HandleWebSocket)

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Metrics routes
		metricsGroup := v1.Group("/metrics")
		{
			metricsGroup.GET("/current", r.metricsHandler.GetCurrentMetrics)
			metricsGroup.GET("/current/:hostname", r.metricsHandler.GetCurrentMetricsByHostname)
			metricsGroup.GET("/history", r.metricsHandler.GetMetricsHistory)
			metricsGroup.GET("/history/:hostname", r.metricsHandler.GetMetricsHistoryByHostname)
			metricsGroup.GET("/average", r.metricsHandler.GetAverageMetrics)
			metricsGroup.GET("/average/:hostname", r.metricsHandler.GetAverageMetricsByHostname)
			metricsGroup.GET("/summary", r.metricsHandler.GetMetricsSummary)
			metricsGroup.GET("/trends/:hostname", r.metricsHandler.GetUsageTrends)
			metricsGroup.GET("/trends", r.metricsHandler.GetHistoricalTrends)
			metricsGroup.GET("/top/:type", r.metricsHandler.GetTopHostsByUsage)
			metricsGroup.POST("", r.metricsHandler.CreateMetric) // For manual metric insertion
		}

		// System routes
		systemGroup := v1.Group("/system")
		{
			systemGroup.GET("/status", r.metricsHandler.GetSystemStatus)
			systemGroup.GET("/hosts", r.metricsHandler.GetHosts)
			systemGroup.GET("/stats", r.metricsHandler.GetStats)
		}

		// WebSocket routes
		wsGroup := v1.Group("/ws")
		{
			wsGroup.GET("/clients", func(c *gin.Context) {
				count := r.websocketHandler.GetConnectedClients()
				c.JSON(http.StatusOK, handlers.APIResponse{
					Success: true,
					Data: map[string]interface{}{
						"connected_clients": count,
					},
				})
			})

			wsGroup.GET("/stats", func(c *gin.Context) {
				stats := r.websocketHandler.GetClientStats()
				c.JSON(http.StatusOK, handlers.APIResponse{
					Success: true,
					Data:    stats,
				})
			})
		}

		// Admin routes (protected)
		adminGroup := v1.Group("/admin")
		{
			// Authentication middleware for admin routes
			if !r.config.IsDevelopment() {
				adminGroup.Use(middleware.BasicAuth())
			}

			adminGroup.DELETE("/metrics/cleanup", r.metricsHandler.CleanupOldMetrics)
			adminGroup.GET("/database/stats", r.healthHandler.DatabaseStats)
		}
	}

	// Static file serving and dashboard routes
	r.engine.Static("/static", "./web/static")
	r.engine.LoadHTMLGlob("web/templates/*")

	// Single dashboard route (simplified)
	dashboardHandler := func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"title":   "GoDash System Monitor",
			"version": "1.0.0",
		})
	}

	r.engine.GET("/", dashboardHandler)
	r.engine.GET("/dashboard", func(c *gin.Context) {
		// Redirect /dashboard to / for consistency
		c.Redirect(http.StatusMovedPermanently, "/")
	})

	// Simplified NoRoute handler for SPA
	r.engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Return 404 for API routes
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "The requested endpoint does not exist",
				"path":    path,
			})
			return
		}

		// Return 404 for WebSocket routes
		if len(path) >= 3 && path[:3] == "/ws" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "WebSocket endpoint not found",
				"path":    path,
			})
			return
		}

		// Serve dashboard for all other routes (SPA)
		dashboardHandler(c)
	})
}

// startWebSocketBroadcasting starts the WebSocket metrics broadcasting goroutine
func (r *Router) startWebSocketBroadcasting() {
	log.Println("🚀 Starting ultra-fast WebSocket broadcasting...")

	// Start metrics broadcasting every 500ms for ultra real-time updates
	ctx := context.Background()
	interval := 500 * time.Millisecond // 500ms for ultra-responsive feel

	// Use a separate goroutine for broadcasting
	go r.websocketHandler.StartMetricsBroadcast(ctx, interval)

	// Start system status broadcasting every 30 seconds
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			r.websocketHandler.BroadcastSystemStatus()
		}
	}()

	log.Printf("✅ WebSocket broadcasting started with %v interval", interval)
}

// GetEngine returns the gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// GetWebSocketHandler returns the WebSocket handler
func (r *Router) GetWebSocketHandler() *handlers.WebSocketHandler {
	return r.websocketHandler
}

// Start starts the HTTP server
func (r *Router) Start() error {
	server := &http.Server{
		Addr:         r.config.GetServerAddress(),
		Handler:      r.engine,
		ReadTimeout:  r.config.Server.ReadTimeout,
		WriteTimeout: r.config.Server.WriteTimeout,
	}

	return server.ListenAndServe()
}

// Routes returns all registered routes
func (r *Router) Routes() gin.RoutesInfo {
	return r.engine.Routes()
}

// GetRoutesList returns a formatted list of all routes
func (r *Router) GetRoutesList() []map[string]string {
	routes := r.engine.Routes()
	routesList := make([]map[string]string, len(routes))

	for i, route := range routes {
		routesList[i] = map[string]string{
			"method": route.Method,
			"path":   route.Path,
		}
	}

	return routesList
}
