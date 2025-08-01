package api

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/eyzaun/godash/internal/api/handlers"
	"github.com/eyzaun/godash/internal/api/middleware"
	"github.com/eyzaun/godash/internal/config"
	"github.com/eyzaun/godash/internal/repository"
)

// Router wraps gin router with dependencies
type Router struct {
	engine         *gin.Engine
	config         *config.Config
	metricsRepo    repository.MetricsRepository
	metricsHandler *handlers.MetricsHandler
	healthHandler  *handlers.HealthHandler
}

// New creates a new API router
func New(cfg *config.Config, metricsRepo repository.MetricsRepository) *Router {
	// Set gin mode
	gin.SetMode(cfg.Server.Mode)

	// Create gin engine
	engine := gin.New()

	// Create handlers
	metricsHandler := handlers.NewMetricsHandler(metricsRepo)
	healthHandler := handlers.NewHealthHandler(metricsRepo)

	router := &Router{
		engine:         engine,
		config:         cfg,
		metricsRepo:    metricsRepo,
		metricsHandler: metricsHandler,
		healthHandler:  healthHandler,
	}

	// Setup middleware
	router.setupMiddleware()

	// Setup routes
	router.setupRoutes()

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

	// Static file serving (for future web interface)
	r.engine.Static("/static", "./web/static")
	r.engine.LoadHTMLGlob("web/templates/*")

	// Catch-all route for SPA (Single Page Application)
	r.engine.NoRoute(func(c *gin.Context) {
		// If it's an API route, return 404
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "The requested endpoint does not exist",
				"path":    c.Request.URL.Path,
			})
			return
		}

		// For non-API routes, serve index.html (for SPA)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "GoDash System Monitor",
		})
	})
}

// GetEngine returns the gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
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
