package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eyzaun/godash/internal/collector"
	"github.com/eyzaun/godash/internal/models"
	"github.com/eyzaun/godash/internal/repository"
)

// MetricsHandler handles HTTP requests for metrics
type MetricsHandler struct {
	metricsRepo     repository.MetricsRepository
	systemCollector collector.Collector
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(metricsRepo repository.MetricsRepository, systemCollector collector.Collector) *MetricsHandler {
	return &MetricsHandler{
		metricsRepo:     metricsRepo,
		systemCollector: systemCollector,
	}
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
	Error      string      `json:"error,omitempty"`
}

// Pagination represents pagination information
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// DetailedMetricsResponse - REAL-TIME VERİ İÇİN YENİ RESPONSE FORMAT (SPEED FIELDS ADDED)
type DetailedMetricsResponse struct {
	// Basic fields (frontend'in beklediği format)
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	Hostname      string  `json:"hostname"`
	Timestamp     string  `json:"timestamp"`

	// Detailed CPU info
	CPUCores       int       `json:"cpu_cores"`
	CPUFrequency   float64   `json:"cpu_frequency"`
	CPULoadAvg     []float64 `json:"cpu_load_avg"`
	CPUTemperature float64   `json:"cpu_temperature_c"`

	// Detailed Memory info
	MemoryTotal     uint64 `json:"memory_total"`
	MemoryUsed      uint64 `json:"memory_used"`
	MemoryAvailable uint64 `json:"memory_available"`
	MemoryFree      uint64 `json:"memory_free"`

	// Detailed Disk info
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

	// Network info
	NetworkSent     uint64 `json:"network_sent"`
	NetworkReceived uint64 `json:"network_received"`
	// NEW: Network Speed
	NetworkUploadSpeed   float64 `json:"network_upload_speed_mbps"`
	NetworkDownloadSpeed float64 `json:"network_download_speed_mbps"`

	// Process info
	Processes *models.ProcessActivity `json:"processes"`
}

// GetCurrentMetrics gets the latest metrics from system collector (REAL-TIME)
// @Summary Get current metrics
// @Description Retrieve the latest system metrics directly from system collector for real-time data
// @Tags metrics
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=DetailedMetricsResponse}
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics/current [get]
func (h *MetricsHandler) GetCurrentMetrics(c *gin.Context) {
	// FIX: Check if systemCollector is nil
	if h.systemCollector == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "System collector not available",
			Message: "System collector is not initialized",
		})
		return
	}

	// REAL-TIME VERİ: SystemCollector'dan direkt al
	systemMetrics, err := h.systemCollector.GetSystemMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to retrieve current metrics",
			Message: err.Error(),
		})
		return
	}

	// FIX: Check if systemMetrics is nil
	if systemMetrics == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to retrieve current metrics",
			Message: "System metrics returned nil",
		})
		return
	}

	// Frontend'in beklediği format'a dönüştür (SPEED FIELDS ADDED)
	response := DetailedMetricsResponse{
		// Basic metrics (frontend'in chart'ları için)
		CPUUsage:      systemMetrics.CPU.Usage,
		MemoryPercent: systemMetrics.Memory.Percent,
		DiskPercent:   systemMetrics.Disk.Percent,
		Hostname:      systemMetrics.Hostname,
		Timestamp:     systemMetrics.Timestamp.Format(time.RFC3339),

		// Detailed CPU info
		CPUCores:       systemMetrics.CPU.Cores,
		CPUFrequency:   systemMetrics.CPU.Frequency,
		CPULoadAvg:     systemMetrics.CPU.LoadAvg,
		CPUTemperature: systemMetrics.CPU.Temperature,

		// Detailed Memory info
		MemoryTotal:     systemMetrics.Memory.Total,
		MemoryUsed:      systemMetrics.Memory.Used,
		MemoryAvailable: systemMetrics.Memory.Available,
		MemoryFree:      systemMetrics.Memory.Free,

		// Detailed Disk info (SPEED FIELDS ADDED)
		DiskTotal:      systemMetrics.Disk.Total,
		DiskUsed:       systemMetrics.Disk.Used,
		DiskFree:       systemMetrics.Disk.Free,
		DiskReadBytes:  systemMetrics.Disk.IOStats.ReadBytes,
		DiskWriteBytes: systemMetrics.Disk.IOStats.WriteBytes,
		// NEW: Disk I/O Speed
		DiskReadSpeed:  systemMetrics.Disk.ReadSpeed,
		DiskWriteSpeed: systemMetrics.Disk.WriteSpeed,
		// NEW: Individual disk partitions
		DiskPartitions: systemMetrics.Disk.Partitions,

		// Network info (SPEED FIELDS ADDED)
		NetworkSent:     systemMetrics.Network.TotalSent,
		NetworkReceived: systemMetrics.Network.TotalReceived,
		// NEW: Network Speed
		NetworkUploadSpeed:   systemMetrics.Network.UploadSpeed,
		NetworkDownloadSpeed: systemMetrics.Network.DownloadSpeed,

		// Process info
		Processes: &systemMetrics.Processes,
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// GetCurrentMetricsByHostname gets the latest metrics for a specific hostname
// @Summary Get current metrics by hostname
// @Description Retrieve the latest system metrics for a specific host
// @Tags metrics
// @Accept json
// @Produce json
// @Param hostname path string true "Hostname"
// @Success 200 {object} APIResponse{data=models.Metric}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics/current/{hostname} [get]
func (h *MetricsHandler) GetCurrentMetricsByHostname(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	hostname := c.Param("hostname")

	metrics, err := h.metricsRepo.GetLatestByHostname(hostname)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "no metrics found for hostname "+hostname {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   "Failed to retrieve metrics for hostname",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    metrics,
	})
}

// GetMetricsHistory gets historical metrics with pagination
// @Summary Get metrics history
// @Description Retrieve historical system metrics with optional filtering and pagination
// @Tags metrics
// @Accept json
// @Produce json
// @Param from query string false "Start time (RFC3339 format)"
// @Param to query string false "End time (RFC3339 format)"
// @Param limit query int false "Number of records to return" default(50)
// @Param page query int false "Page number" default(1)
// @Success 200 {object} PaginatedResponse{data=[]models.Metric}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics/history [get]
func (h *MetricsHandler) GetMetricsHistory(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	// Parse query parameters
	var from, to time.Time
	var err error

	// Parse from time
	if fromStr := c.Query("from"); fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid from time format",
				Message: "Use RFC3339 format (e.g., 2024-01-01T00:00:00Z)",
			})
			return
		}
	} else {
		from = time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	}

	// Parse to time
	if toStr := c.Query("to"); toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid to time format",
				Message: "Use RFC3339 format (e.g., 2024-01-01T00:00:00Z)",
			})
			return
		}
	} else {
		to = time.Now()
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	// Get metrics
	metrics, err := h.metricsRepo.GetHistory(from, to, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to retrieve metrics history",
			Message: err.Error(),
		})
		return
	}

	// Get total count for pagination
	total, err := h.metricsRepo.GetCountByDateRange(from, to)
	if err != nil {
		total = 0 // Continue with response even if count fails
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    metrics,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// YENİ ENDPOINT: GetHistoricalTrends - Frontend'in beklediği trends data
// @Summary Get historical trends for charts
// @Description Get historical trends data formatted for frontend charts
// @Tags metrics
// @Accept json
// @Produce json
// @Param range query string false "Time range (1h, 6h, 24h, 7d)" default("1h")
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics/trends [get]
func (h *MetricsHandler) GetHistoricalTrends(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	timeRange := c.DefaultQuery("range", "1h")

	var duration time.Duration
	var err error

	switch timeRange {
	case "1h":
		duration = 1 * time.Hour
	case "6h":
		duration = 6 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	default:
		duration, err = time.ParseDuration(timeRange)
		if err != nil {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid time range",
				Message: "Use format like '1h', '6h', '24h', '7d' or duration format",
			})
			return
		}
	}

	from := time.Now().Add(-duration)
	to := time.Now()

	// Get historical data with reasonable limit
	limit := 100 // Enough points for smooth chart
	metrics, err := h.metricsRepo.GetHistory(from, to, limit, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to retrieve historical trends",
			Message: err.Error(),
		})
		return
	}

	// Transform data for frontend charts
	var labels []string
	var cpuData []float64
	var memoryData []float64
	var diskData []float64

	for _, metric := range metrics {
		// FIX: Check if metric is nil
		if metric != nil {
			labels = append(labels, metric.Timestamp.Format("15:04:05"))
			cpuData = append(cpuData, metric.CPUUsage)
			memoryData = append(memoryData, metric.MemoryPercent)
			diskData = append(diskData, metric.DiskPercent)
		}
	}

	// Reverse arrays to show chronological order
	for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
		labels[i], labels[j] = labels[j], labels[i]
		cpuData[i], cpuData[j] = cpuData[j], cpuData[i]
		memoryData[i], memoryData[j] = memoryData[j], memoryData[i]
		diskData[i], diskData[j] = diskData[j], diskData[i]
	}

	trendsData := map[string]interface{}{
		"labels": labels,
		"datasets": []map[string]interface{}{
			{
				"label":           "CPU Usage %",
				"data":            cpuData,
				"borderColor":     "#ff6b6b",
				"backgroundColor": "rgba(255, 107, 107, 0.1)",
			},
			{
				"label":           "Memory Usage %",
				"data":            memoryData,
				"borderColor":     "#4ecdc4",
				"backgroundColor": "rgba(78, 205, 196, 0.1)",
			},
			{
				"label":           "Disk Usage %",
				"data":            diskData,
				"borderColor":     "#ffa726",
				"backgroundColor": "rgba(255, 167, 38, 0.1)",
			},
		},
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    trendsData,
	})
}

// GetMetricsHistoryByHostname gets historical metrics for a specific hostname
// @Summary Get metrics history by hostname
// @Description Retrieve historical system metrics for a specific host
// @Tags metrics
// @Accept json
// @Produce json
// @Param hostname path string true "Hostname"
// @Param from query string false "Start time (RFC3339 format)"
// @Param to query string false "End time (RFC3339 format)"
// @Param limit query int false "Number of records to return" default(50)
// @Param page query int false "Page number" default(1)
// @Success 200 {object} PaginatedResponse{data=[]models.Metric}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics/history/{hostname} [get]
func (h *MetricsHandler) GetMetricsHistoryByHostname(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	hostname := c.Param("hostname")

	// Parse query parameters (same logic as GetMetricsHistory)
	var from, to time.Time
	var err error

	if fromStr := c.Query("from"); fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid from time format",
				Message: "Use RFC3339 format",
			})
			return
		}
	} else {
		from = time.Now().Add(-24 * time.Hour)
	}

	if toStr := c.Query("to"); toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid to time format",
				Message: "Use RFC3339 format",
			})
			return
		}
	} else {
		to = time.Now()
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	// Get metrics for hostname
	metrics, err := h.metricsRepo.GetHistoryByHostname(hostname, from, to, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to retrieve metrics history for hostname",
			Message: err.Error(),
		})
		return
	}

	// For simplicity, we'll use the length of returned metrics as total
	// In production, you might want a separate count query
	total := int64(len(metrics))
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    metrics,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetAverageMetrics gets average metrics over a time period
// @Summary Get average metrics
// @Description Calculate average resource usage over a specified duration
// @Tags metrics
// @Accept json
// @Produce json
// @Param duration query string false "Duration (e.g., 1h, 24h, 7d)" default("1h")
// @Success 200 {object} APIResponse{data=models.AverageMetrics}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics/average [get]
func (h *MetricsHandler) GetAverageMetrics(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	durationStr := c.DefaultQuery("duration", "1h")

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid duration format",
			Message: "Use format like '1h', '24h', '7d'",
		})
		return
	}

	averageMetrics, err := h.metricsRepo.GetAverageUsage(duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to calculate average metrics",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    averageMetrics,
	})
}

// GetAverageMetricsByHostname gets average metrics for a specific hostname
// @Summary Get average metrics by hostname
// @Description Calculate average resource usage for a specific host over a specified duration
// @Tags metrics
// @Accept json
// @Produce json
// @Param hostname path string true "Hostname"
// @Param duration query string false "Duration (e.g., 1h, 24h, 7d)" default("1h")
// @Success 200 {object} APIResponse{data=models.AverageMetrics}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics/average/{hostname} [get]
func (h *MetricsHandler) GetAverageMetricsByHostname(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	hostname := c.Param("hostname")
	durationStr := c.DefaultQuery("duration", "1h")

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid duration format",
			Message: "Use format like '1h', '24h', '7d'",
		})
		return
	}

	averageMetrics, err := h.metricsRepo.GetAverageUsageByHostname(hostname, duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to calculate average metrics for hostname",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    averageMetrics,
	})
}

// GetMetricsSummary gets metrics summary for a time period
// @Summary Get metrics summary
// @Description Get summary statistics for metrics over a time period
// @Tags metrics
// @Accept json
// @Produce json
// @Param from query string false "Start time (RFC3339 format)"
// @Param to query string false "End time (RFC3339 format)"
// @Success 200 {object} APIResponse{data=models.MetricsSummary}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics/summary [get]
func (h *MetricsHandler) GetMetricsSummary(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	var from, to time.Time
	var err error

	if fromStr := c.Query("from"); fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid from time format",
			})
			return
		}
	} else {
		from = time.Now().Add(-24 * time.Hour)
	}

	if toStr := c.Query("to"); toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid to time format",
			})
			return
		}
	} else {
		to = time.Now()
	}

	summary, err := h.metricsRepo.GetMetricsSummary(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to get metrics summary",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    summary,
	})
}

// GetUsageTrends gets usage trends for a hostname
// @Summary Get usage trends
// @Description Get hourly usage trends for a specific host
// @Tags metrics
// @Accept json
// @Produce json
// @Param hostname path string true "Hostname"
// @Param hours query int false "Number of hours to analyze" default(24)
// @Success 200 {object} APIResponse{data=[]models.UsageTrend}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics/trends/{hostname} [get]
func (h *MetricsHandler) GetUsageTrends(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	hostname := c.Param("hostname")
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))

	if hours <= 0 || hours > 168 { // Max 1 week
		hours = 24
	}

	trends, err := h.metricsRepo.GetUsageTrends(hostname, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to get usage trends",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    trends,
	})
}

// GetTopHostsByUsage gets top hosts by resource usage
// @Summary Get top hosts by usage
// @Description Get hosts with highest resource usage
// @Tags metrics
// @Accept json
// @Produce json
// @Param type path string true "Metric type (cpu, memory, disk)"
// @Param limit query int false "Number of hosts to return" default(10)
// @Success 200 {object} APIResponse{data=[]models.HostUsage}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics/top/{type} [get]
func (h *MetricsHandler) GetTopHostsByUsage(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	metricType := c.Param("type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if limit <= 0 || limit > 100 {
		limit = 10
	}

	if metricType != "cpu" && metricType != "memory" && metricType != "disk" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid metric type",
			Message: "Type must be 'cpu', 'memory', or 'disk'",
		})
		return
	}

	hosts, err := h.metricsRepo.GetTopHostsByUsage(metricType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to get top hosts by usage",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    hosts,
	})
}

// GetTopProcesses gets top processes by CPU or memory usage
// @Summary Get top processes
// @Description Get top processes by CPU or memory usage
// @Tags metrics
// @Accept json
// @Produce json
// @Param limit query int false "Number of processes to return" default(10)
// @Param sort query string false "Sort by (cpu, memory)" default("cpu")
// @Success 200 {object} APIResponse{data=map[string]interface{}}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics/processes [get]
func (h *MetricsHandler) GetTopProcesses(c *gin.Context) {
	// FIX: Check if systemCollector is nil
	if h.systemCollector == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "System collector not available",
			Message: "System collector is not initialized",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	sortBy := c.DefaultQuery("sort", "cpu")

	if limit <= 0 || limit > 100 {
		limit = 10
	}

	if sortBy != "cpu" && sortBy != "memory" {
		sortBy = "cpu"
	}

	processes, err := h.systemCollector.GetTopProcesses(limit, sortBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to get top processes",
			Message: err.Error(),
		})
		return
	}

	// Format processes as expected by frontend (string array with @{} format)
	topProcesses := make([]string, len(processes))
	for i, process := range processes {
		topProcesses[i] = fmt.Sprintf("@{pid=%d; name=%s; cpu_percent=%.1f; memory_bytes=%d; status=%s}",
			process.PID, process.Name, process.CPUPercent, process.MemoryBytes, process.Status)
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"top_processes": topProcesses,
		},
	})
}

// CreateMetric creates a new metric entry (for manual insertion)
// @Summary Create metric
// @Description Create a new metric entry
// @Tags metrics
// @Accept json
// @Produce json
// @Param metric body models.Metric true "Metric data"
// @Success 201 {object} APIResponse{data=models.Metric}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/metrics [post]
func (h *MetricsHandler) CreateMetric(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	var metric models.Metric

	if err := c.ShouldBindJSON(&metric); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Set timestamp if not provided
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	if err := h.metricsRepo.Create(&metric); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to create metric",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    metric,
		Message: "Metric created successfully",
	})
}

// GetSystemStatus gets current system status
// @Summary Get system status
// @Description Get current status of all monitored systems
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=[]models.SystemStatus}
// @Failure 500 {object} APIResponse
// @Router /api/v1/system/status [get]
func (h *MetricsHandler) GetSystemStatus(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	status, err := h.metricsRepo.GetSystemStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to get system status",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    status,
	})
}

// GetHosts gets list of all monitored hosts
// @Summary Get hosts
// @Description Get list of all monitored hosts
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=[]string}
// @Failure 500 {object} APIResponse
// @Router /api/v1/system/hosts [get]
func (h *MetricsHandler) GetHosts(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	// This is a simplified implementation
	// In practice, you might want a dedicated method in the repository
	status, err := h.metricsRepo.GetSystemStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to get hosts",
			Message: err.Error(),
		})
		return
	}

	hosts := make([]string, len(status))
	for i, s := range status {
		// FIX: Check if s is nil
		if s != nil {
			hosts[i] = s.Hostname
		}
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    hosts,
	})
}

// GetStats gets database statistics (DÜZELTİLMİŞ VERSİYON)
// @Summary Get statistics
// @Description Get database and metrics statistics
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/system/stats [get]
func (h *MetricsHandler) GetStats(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	totalCount, err := h.metricsRepo.GetTotalCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to get statistics",
			Message: err.Error(),
		})
		return
	}

	// Get system status to count hosts
	systemStatus, err := h.metricsRepo.GetSystemStatus()
	totalHosts := 1 // Default to 1 for single host setup
	if err == nil && len(systemStatus) > 0 {
		totalHosts = len(systemStatus)
	}

	// Get average metrics for ALL records - DÜZELTİLMİŞ VERSİYON
	averageMetrics, err := h.metricsRepo.GetAverageUsageAllRecords()
	avgCpu := 0.0
	avgMemory := 0.0
	avgDiskIO := 0.0
	avgNetwork := 0.0

	if err == nil && averageMetrics != nil && averageMetrics.SampleCount > 0 {
		avgCpu = averageMetrics.AvgCPUUsage
		avgMemory = averageMetrics.AvgMemoryUsage
		// Disk I/O: read + write speed
		avgDiskIO = averageMetrics.AvgDiskReadSpeed + averageMetrics.AvgDiskWriteSpeed
		// Network: upload + download speed
		avgNetwork = averageMetrics.AvgNetworkUpload + averageMetrics.AvgNetworkDownload
	} else {
		// Fallback: try to get current metrics if no historical data
		if h.systemCollector != nil {
			currentMetrics, collectorErr := h.systemCollector.GetSystemMetrics()
			// FIX: Check if currentMetrics is nil
			if collectorErr == nil && currentMetrics != nil {
				avgCpu = currentMetrics.CPU.Usage
				avgMemory = currentMetrics.Memory.Percent
				avgDiskIO = currentMetrics.Disk.ReadSpeed + currentMetrics.Disk.WriteSpeed
				avgNetwork = currentMetrics.Network.UploadSpeed + currentMetrics.Network.DownloadSpeed
			}
		}
	}

	stats := map[string]interface{}{
		"total_metrics":    totalCount,
		"total_hosts":      totalHosts,
		"avg_cpu_usage":    avgCpu,
		"avg_memory_usage": avgMemory,
		"avg_disk_io":      avgDiskIO,
		"avg_network":      avgNetwork,
		"timestamp":        time.Now(),
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// CleanupOldMetrics removes old metric records
// @Summary Cleanup old metrics
// @Description Remove metric records older than specified days
// @Tags admin
// @Accept json
// @Produce json
// @Param days query int false "Days to keep" default(30)
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/admin/metrics/cleanup [delete]
func (h *MetricsHandler) CleanupOldMetrics(c *gin.Context) {
	// FIX: Check if metricsRepo is nil
	if h.metricsRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Metrics repository not available",
			Message: "Metrics repository is not initialized",
		})
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	if days <= 0 {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid days parameter",
			Message: "Days must be positive",
		})
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -days)

	deletedCount, err := h.metricsRepo.DeleteOldRecords(cutoffTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to cleanup old metrics",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"deleted_count": deletedCount,
			"cutoff_time":   cutoffTime,
		},
		Message: "Cleanup completed successfully",
	})
}
