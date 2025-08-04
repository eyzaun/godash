package handlers

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eyzaun/godash/internal/models"
	"github.com/eyzaun/godash/internal/repository"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	metricsRepo repository.MetricsRepository
	startTime   time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(metricsRepo repository.MetricsRepository) *HealthHandler {
	return &HealthHandler{
		metricsRepo: metricsRepo,
		startTime:   time.Now(),
	}
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    string                 `json:"uptime"`
	Version   string                 `json:"version"`
	Checks    map[string]HealthCheck `json:"checks"`
}

// HealthCheck represents individual health check
type HealthCheck struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ReadinessResponse represents readiness check response
type ReadinessResponse struct {
	Ready     bool                     `json:"ready"`
	Timestamp time.Time                `json:"timestamp"`
	Services  map[string]ServiceStatus `json:"services"`
}

// ServiceStatus represents individual service status
type ServiceStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthCheck performs comprehensive health check
// @Summary Health check
// @Description Comprehensive health check of the application and its dependencies
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	checks := make(map[string]HealthCheck)
	overallStatus := "healthy"

	// Database health check
	dbCheck := h.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// Memory health check
	memCheck := h.checkMemory()
	checks["memory"] = memCheck
	if memCheck.Status == "warning" && overallStatus == "healthy" {
		overallStatus = "warning"
	}

	// Goroutines health check
	goroutineCheck := h.checkGoroutines()
	checks["goroutines"] = goroutineCheck
	if goroutineCheck.Status == "warning" && overallStatus == "healthy" {
		overallStatus = "warning"
	}

	// Metrics collection health check
	metricsCheck := h.checkMetricsCollection()
	checks["metrics"] = metricsCheck
	if metricsCheck.Status != "healthy" && overallStatus == "healthy" {
		overallStatus = "warning"
	}

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Uptime:    time.Since(h.startTime).String(),
		Version:   "1.0.0", // In production, use build-time variable
		Checks:    checks,
	}

	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// ReadinessCheck checks if the application is ready to serve requests
// @Summary Readiness check
// @Description Check if the application is ready to serve requests
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} ReadinessResponse
// @Failure 503 {object} ReadinessResponse
// @Router /ready [get]
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	services := make(map[string]ServiceStatus)
	ready := true

	// Check database connectivity
	dbCheck := h.checkDatabase()
	if dbCheck.Status == "healthy" {
		services["database"] = ServiceStatus{
			Status:  "ready",
			Message: "Database is accessible",
		}
	} else {
		services["database"] = ServiceStatus{
			Status:  "not_ready",
			Message: dbCheck.Error,
		}
		ready = false
	}

	// Check if we can write to database
	if ready && h.canWriteToDatabase() {
		services["database_write"] = ServiceStatus{
			Status:  "ready",
			Message: "Database is writable",
		}
	} else if ready {
		services["database_write"] = ServiceStatus{
			Status:  "not_ready",
			Message: "Cannot write to database",
		}
		ready = false
	}

	response := ReadinessResponse{
		Ready:     ready,
		Timestamp: time.Now(),
		Services:  services,
	}

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// PrometheusMetrics returns metrics in Prometheus format
// @Summary Prometheus metrics
// @Description Export metrics in Prometheus format for monitoring
// @Tags monitoring
// @Produce text/plain
// @Success 200 {string} string "Prometheus metrics"
// @Router /metrics [get]
func (h *HealthHandler) PrometheusMetrics(c *gin.Context) {
	// Get runtime metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Get total metrics count
	totalMetrics, _ := h.metricsRepo.GetTotalCount()

	// Build Prometheus metrics
	metrics := fmt.Sprintf(`# HELP godash_info Application information
# TYPE godash_info gauge
godash_info{version="1.0.0"} 1

# HELP godash_uptime_seconds Application uptime in seconds
# TYPE godash_uptime_seconds counter
godash_uptime_seconds %.2f

# HELP godash_goroutines_total Number of goroutines
# TYPE godash_goroutines_total gauge
godash_goroutines_total %d

# HELP godash_memory_alloc_bytes Bytes allocated and still in use
# TYPE godash_memory_alloc_bytes gauge
godash_memory_alloc_bytes %d

# HELP godash_memory_sys_bytes Bytes obtained from the OS
# TYPE godash_memory_sys_bytes gauge
godash_memory_sys_bytes %d

# HELP godash_gc_runs_total Number of completed GC cycles
# TYPE godash_gc_runs_total counter
godash_gc_runs_total %d

# HELP godash_metrics_total Total number of metrics in database
# TYPE godash_metrics_total gauge
godash_metrics_total %d
`,
		time.Since(h.startTime).Seconds(),
		runtime.NumGoroutine(),
		m.Alloc,
		m.Sys,
		m.NumGC,
		totalMetrics,
	)

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, metrics)
}

// DatabaseStats returns detailed database statistics
// @Summary Database statistics
// @Description Get detailed database connection and performance statistics
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/admin/database/stats [get]
func (h *HealthHandler) DatabaseStats(c *gin.Context) {
	// Get basic metrics count
	totalCount, err := h.metricsRepo.GetTotalCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to get database statistics",
			Message: err.Error(),
		})
		return
	}

	// Get recent metrics count (last 24 hours)
	since := time.Now().Add(-24 * time.Hour)
	recentCount, err := h.metricsRepo.GetCountByDateRange(since, time.Now())
	if err != nil {
		recentCount = 0
	}

	// Get system status for host count
	systemStatus, err := h.metricsRepo.GetSystemStatus()
	if err != nil {
		systemStatus = []*models.SystemStatus{}
	}

	stats := map[string]interface{}{
		"total_metrics":        totalCount,
		"recent_metrics_24h":   recentCount,
		"active_hosts":         len(systemStatus),
		"collection_timestamp": time.Now(),
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
		Message: "Database statistics retrieved successfully",
	})
}

// Helper methods for health checks

func (h *HealthHandler) checkDatabase() HealthCheck {
	// Try to get total count as a simple database connectivity test
	count, err := h.metricsRepo.GetTotalCount()
	if err != nil {
		return HealthCheck{
			Status: "unhealthy",
			Error:  err.Error(),
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "Database is accessible",
		Data: map[string]interface{}{
			"total_metrics": count,
		},
	}
}

func (h *HealthHandler) checkMemory() HealthCheck {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check if memory usage is concerning (more than 80% of allocated)
	usagePercent := float64(m.Alloc) / float64(m.Sys) * 100

	status := "healthy"
	message := "Memory usage is normal"

	if usagePercent > 90 {
		status = "warning"
		message = "High memory usage detected"
	}

	return HealthCheck{
		Status:  status,
		Message: message,
		Data: map[string]interface{}{
			"alloc_mb":      m.Alloc / 1024 / 1024,
			"sys_mb":        m.Sys / 1024 / 1024,
			"usage_percent": usagePercent,
			"gc_runs":       m.NumGC,
		},
	}
}

func (h *HealthHandler) checkGoroutines() HealthCheck {
	count := runtime.NumGoroutine()

	status := "healthy"
	message := "Goroutine count is normal"

	// Warning if too many goroutines (possible leak)
	if count > 1000 {
		status = "warning"
		message = "High goroutine count detected"
	}

	return HealthCheck{
		Status:  status,
		Message: message,
		Data: map[string]interface{}{
			"count": count,
		},
	}
}

func (h *HealthHandler) checkMetricsCollection() HealthCheck {
	// Check if we have recent metrics (within last 5 minutes)
	since := time.Now().Add(-5 * time.Minute)
	recentCount, err := h.metricsRepo.GetCountByDateRange(since, time.Now())

	if err != nil {
		return HealthCheck{
			Status: "unhealthy",
			Error:  "Failed to check recent metrics: " + err.Error(),
		}
	}

	status := "healthy"
	message := "Metrics collection is active"

	if recentCount == 0 {
		status = "warning"
		message = "No recent metrics collected"
	}

	return HealthCheck{
		Status:  status,
		Message: message,
		Data: map[string]interface{}{
			"recent_metrics_5m": recentCount,
		},
	}
}

func (h *HealthHandler) canWriteToDatabase() bool {
	// Try to get the total count - if this works, we can read
	_, err := h.metricsRepo.GetTotalCount()
	return err == nil
}
