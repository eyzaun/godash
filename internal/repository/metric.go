package repository

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/eyzaun/godash/internal/models"
)

// MetricsRepository interface defines methods for metrics data access
type MetricsRepository interface {
	// Create operations
	Create(metric *models.Metric) error

	// Read operations
	GetLatest() (*models.Metric, error)
	GetLatestByHostname(hostname string) (*models.Metric, error)
	GetHistory(from, to time.Time, limit, offset int) ([]*models.Metric, error)
	GetHistoryByHostname(hostname string, from, to time.Time, limit, offset int) ([]*models.Metric, error)
	GetAverageUsage(duration time.Duration) (*models.AverageMetrics, error)
	GetAverageUsageByHostname(hostname string, duration time.Duration) (*models.AverageMetrics, error)
	GetAverageUsageAllRecords() (*models.AverageMetrics, error)

	// Aggregation operations
	GetMetricsSummary(from, to time.Time) (*models.MetricsSummary, error)
	GetTopHostsByUsage(metricType string, limit int) ([]*models.HostUsage, error)
	GetUsageTrends(hostname string, hours int) ([]*models.UsageTrend, error)

	// Maintenance operations
	DeleteOldRecords(olderThan time.Time) (int64, error)
	GetTotalCount() (int64, error)
	GetCountByDateRange(from, to time.Time) (int64, error)

	// Health check operations
	GetSystemStatus() ([]*models.SystemStatus, error)
}

// metricsRepository implements MetricsRepository interface
type metricsRepository struct {
	db *gorm.DB
}

// NewMetricsRepository creates a new metrics repository
func NewMetricsRepository(db *gorm.DB) MetricsRepository {
	return &metricsRepository{
		db: db,
	}
}

// Create inserts a new metric record
func (r *metricsRepository) Create(metric *models.Metric) error {
	if err := r.db.Create(metric).Error; err != nil {
		return fmt.Errorf("failed to create metric: %w", err)
	}
	return nil
}

// GetLatest retrieves the most recent metric record (FIXED VERSION)
func (r *metricsRepository) GetLatest() (*models.Metric, error) {
	var metric models.Metric

	// Use ORDER BY id DESC for maximum performance (ID is auto-incrementing)
	log.Println("🔍 [DEBUG] GetLatest using ORDER BY id DESC query")
	err := r.db.Order("id DESC").First(&metric).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no metrics found")
		}
		return nil, fmt.Errorf("failed to get latest metric: %w", err)
	}

	log.Printf("🔍 [DEBUG] GetLatest returned: CPU=%.1f%%, Memory=%.1f%%, ID=%d, Cores=%d, Freq=%.0f",
		metric.CPUUsage, metric.MemoryPercent, metric.ID, metric.CPUCores, metric.CPUFrequency)
	return &metric, nil
}

// GetLatestByHostname retrieves the most recent metric for a specific hostname
func (r *metricsRepository) GetLatestByHostname(hostname string) (*models.Metric, error) {
	var metric models.Metric
	if err := r.db.Where("hostname = ?", hostname).
		Order("timestamp DESC").
		First(&metric).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no metrics found for hostname %s", hostname)
		}
		return nil, fmt.Errorf("failed to get latest metric for hostname: %w", err)
	}
	return &metric, nil
}

// GetHistory retrieves metrics within a time range
func (r *metricsRepository) GetHistory(from, to time.Time, limit, offset int) ([]*models.Metric, error) {
	var metrics []*models.Metric

	query := r.db.Where("timestamp BETWEEN ? AND ?", from, to).
		Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&metrics).Error; err != nil {
		return nil, fmt.Errorf("failed to get metrics history: %w", err)
	}

	return metrics, nil
}

// GetHistoryByHostname retrieves metrics for a specific hostname within a time range
func (r *metricsRepository) GetHistoryByHostname(hostname string, from, to time.Time, limit, offset int) ([]*models.Metric, error) {
	var metrics []*models.Metric

	query := r.db.Where("hostname = ? AND timestamp BETWEEN ? AND ?", hostname, from, to).
		Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&metrics).Error; err != nil {
		return nil, fmt.Errorf("failed to get metrics history for hostname: %w", err)
	}

	return metrics, nil
}

// GetAverageUsage calculates average resource usage over a duration (FIXED)
func (r *metricsRepository) GetAverageUsage(duration time.Duration) (*models.AverageMetrics, error) {
	since := time.Now().Add(-duration)

	var result struct {
		AvgCPUUsage    *float64 `gorm:"column:avg_cpu_usage"`
		AvgMemoryUsage *float64 `gorm:"column:avg_memory_usage"`
		AvgDiskUsage   *float64 `gorm:"column:avg_disk_usage"`
		MaxCPUUsage    *float64 `gorm:"column:max_cpu_usage"`
		MaxMemoryUsage *float64 `gorm:"column:max_memory_usage"`
		MaxDiskUsage   *float64 `gorm:"column:max_disk_usage"`
		MinCPUUsage    *float64 `gorm:"column:min_cpu_usage"`
		MinMemoryUsage *float64 `gorm:"column:min_memory_usage"`
		MinDiskUsage   *float64 `gorm:"column:min_disk_usage"`
		SampleCount    int64    `gorm:"column:sample_count"`
	}

	if err := r.db.Model(&models.Metric{}).
		Select(`
			AVG(cpu_usage) as avg_cpu_usage,
			AVG(memory_percent) as avg_memory_usage,
			AVG(disk_percent) as avg_disk_usage,
			MAX(cpu_usage) as max_cpu_usage,
			MAX(memory_percent) as max_memory_usage,
			MAX(disk_percent) as max_disk_usage,
			MIN(cpu_usage) as min_cpu_usage,
			MIN(memory_percent) as min_memory_usage,
			MIN(disk_percent) as min_disk_usage,
			COUNT(*) as sample_count
		`).
		Where("timestamp >= ?", since).
		Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate average usage: %w", err)
	}

	// Handle NULL values from database
	averageMetrics := &models.AverageMetrics{
		Duration:    duration,
		From:        since,
		To:          time.Now(),
		SampleCount: result.SampleCount,
	}

	if result.AvgCPUUsage != nil {
		averageMetrics.AvgCPUUsage = *result.AvgCPUUsage
	}
	if result.AvgMemoryUsage != nil {
		averageMetrics.AvgMemoryUsage = *result.AvgMemoryUsage
	}
	if result.AvgDiskUsage != nil {
		averageMetrics.AvgDiskUsage = *result.AvgDiskUsage
	}
	if result.MaxCPUUsage != nil {
		averageMetrics.MaxCPUUsage = *result.MaxCPUUsage
	}
	if result.MaxMemoryUsage != nil {
		averageMetrics.MaxMemoryUsage = *result.MaxMemoryUsage
	}
	if result.MaxDiskUsage != nil {
		averageMetrics.MaxDiskUsage = *result.MaxDiskUsage
	}
	if result.MinCPUUsage != nil {
		averageMetrics.MinCPUUsage = *result.MinCPUUsage
	}
	if result.MinMemoryUsage != nil {
		averageMetrics.MinMemoryUsage = *result.MinMemoryUsage
	}
	if result.MinDiskUsage != nil {
		averageMetrics.MinDiskUsage = *result.MinDiskUsage
	}

	return averageMetrics, nil
}

// GetAverageUsageAllRecords calculates average resource usage for all records (VERY EFFICIENT - SPEED SUPPORT ADDED)
func (r *metricsRepository) GetAverageUsageAllRecords() (*models.AverageMetrics, error) {
	var result struct {
		AvgCPUUsage        *float64 `gorm:"column:avg_cpu_usage"`
		AvgMemoryUsage     *float64 `gorm:"column:avg_memory_usage"`
		AvgDiskUsage       *float64 `gorm:"column:avg_disk_usage"`
		AvgDiskReadSpeed   *float64 `gorm:"column:avg_disk_read_speed"`
		AvgDiskWriteSpeed  *float64 `gorm:"column:avg_disk_write_speed"`
		AvgNetworkUpload   *float64 `gorm:"column:avg_network_upload"`
		AvgNetworkDownload *float64 `gorm:"column:avg_network_download"`
		MaxCPUUsage        *float64 `gorm:"column:max_cpu_usage"`
		MaxMemoryUsage     *float64 `gorm:"column:max_memory_usage"`
		MaxDiskUsage       *float64 `gorm:"column:max_disk_usage"`
		MaxDiskReadSpeed   *float64 `gorm:"column:max_disk_read_speed"`
		MaxDiskWriteSpeed  *float64 `gorm:"column:max_disk_write_speed"`
		MaxNetworkUpload   *float64 `gorm:"column:max_network_upload"`
		MaxNetworkDownload *float64 `gorm:"column:max_network_download"`
		MinCPUUsage        *float64 `gorm:"column:min_cpu_usage"`
		MinMemoryUsage     *float64 `gorm:"column:min_memory_usage"`
		MinDiskUsage       *float64 `gorm:"column:min_disk_usage"`
		MinDiskReadSpeed   *float64 `gorm:"column:min_disk_read_speed"`
		MinDiskWriteSpeed  *float64 `gorm:"column:min_disk_write_speed"`
		MinNetworkUpload   *float64 `gorm:"column:min_network_upload"`
		MinNetworkDownload *float64 `gorm:"column:min_network_download"`
		SampleCount        int64    `gorm:"column:sample_count"`
	}

	if err := r.db.Model(&models.Metric{}).
		Select(`
			AVG(cpu_usage) as avg_cpu_usage,
			AVG(memory_percent) as avg_memory_usage,
			AVG(disk_percent) as avg_disk_usage,
			AVG(disk_read_speed_mbps) as avg_disk_read_speed,
			AVG(disk_write_speed_mbps) as avg_disk_write_speed,
			AVG(network_upload_speed_mbps) as avg_network_upload,
			AVG(network_download_speed_mbps) as avg_network_download,
			MAX(cpu_usage) as max_cpu_usage,
			MAX(memory_percent) as max_memory_usage,
			MAX(disk_percent) as max_disk_usage,
			MAX(disk_read_speed_mbps) as max_disk_read_speed,
			MAX(disk_write_speed_mbps) as max_disk_write_speed,
			MAX(network_upload_speed_mbps) as max_network_upload,
			MAX(network_download_speed_mbps) as max_network_download,
			MIN(cpu_usage) as min_cpu_usage,
			MIN(memory_percent) as min_memory_usage,
			MIN(disk_percent) as min_disk_usage,
			MIN(disk_read_speed_mbps) as min_disk_read_speed,
			MIN(disk_write_speed_mbps) as min_disk_write_speed,
			MIN(network_upload_speed_mbps) as min_network_upload,
			MIN(network_download_speed_mbps) as min_network_download,
			COUNT(*) as sample_count
		`).
		Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate average usage for all records: %w", err)
	}

	// Handle NULL values from database
	averageMetrics := &models.AverageMetrics{
		SampleCount: result.SampleCount,
	}

	if result.AvgCPUUsage != nil {
		averageMetrics.AvgCPUUsage = *result.AvgCPUUsage
	}
	if result.AvgMemoryUsage != nil {
		averageMetrics.AvgMemoryUsage = *result.AvgMemoryUsage
	}
	if result.AvgDiskUsage != nil {
		averageMetrics.AvgDiskUsage = *result.AvgDiskUsage
	}
	if result.AvgDiskReadSpeed != nil {
		averageMetrics.AvgDiskReadSpeed = *result.AvgDiskReadSpeed
	}
	if result.AvgDiskWriteSpeed != nil {
		averageMetrics.AvgDiskWriteSpeed = *result.AvgDiskWriteSpeed
	}
	if result.AvgNetworkUpload != nil {
		averageMetrics.AvgNetworkUpload = *result.AvgNetworkUpload
	}
	if result.AvgNetworkDownload != nil {
		averageMetrics.AvgNetworkDownload = *result.AvgNetworkDownload
	}

	// Max values
	if result.MaxCPUUsage != nil {
		averageMetrics.MaxCPUUsage = *result.MaxCPUUsage
	}
	if result.MaxMemoryUsage != nil {
		averageMetrics.MaxMemoryUsage = *result.MaxMemoryUsage
	}
	if result.MaxDiskUsage != nil {
		averageMetrics.MaxDiskUsage = *result.MaxDiskUsage
	}
	if result.MaxDiskReadSpeed != nil {
		averageMetrics.MaxDiskReadSpeed = *result.MaxDiskReadSpeed
	}
	if result.MaxDiskWriteSpeed != nil {
		averageMetrics.MaxDiskWriteSpeed = *result.MaxDiskWriteSpeed
	}
	if result.MaxNetworkUpload != nil {
		averageMetrics.MaxNetworkUpload = *result.MaxNetworkUpload
	}
	if result.MaxNetworkDownload != nil {
		averageMetrics.MaxNetworkDownload = *result.MaxNetworkDownload
	}

	// Min values
	if result.MinCPUUsage != nil {
		averageMetrics.MinCPUUsage = *result.MinCPUUsage
	}
	if result.MinMemoryUsage != nil {
		averageMetrics.MinMemoryUsage = *result.MinMemoryUsage
	}
	if result.MinDiskUsage != nil {
		averageMetrics.MinDiskUsage = *result.MinDiskUsage
	}

	log.Printf("🔍 [DEBUG] GetAverageUsageAllRecords result: CPU=%.1f%%, Memory=%.1f%%, Disk I/O=%.1f MB/s, Network=%.1f Mbps, Samples=%d",
		averageMetrics.AvgCPUUsage, averageMetrics.AvgMemoryUsage,
		averageMetrics.AvgDiskReadSpeed+averageMetrics.AvgDiskWriteSpeed,
		averageMetrics.AvgNetworkUpload+averageMetrics.AvgNetworkDownload,
		averageMetrics.SampleCount)

	return averageMetrics, nil
}

// GetAverageUsageByHostname calculates average resource usage for a specific hostname
func (r *metricsRepository) GetAverageUsageByHostname(hostname string, duration time.Duration) (*models.AverageMetrics, error) {
	since := time.Now().Add(-duration)

	var result struct {
		AvgCPUUsage    *float64 `gorm:"column:avg_cpu_usage"`
		AvgMemoryUsage *float64 `gorm:"column:avg_memory_usage"`
		AvgDiskUsage   *float64 `gorm:"column:avg_disk_usage"`
		MaxCPUUsage    *float64 `gorm:"column:max_cpu_usage"`
		MaxMemoryUsage *float64 `gorm:"column:max_memory_usage"`
		MaxDiskUsage   *float64 `gorm:"column:max_disk_usage"`
		MinCPUUsage    *float64 `gorm:"column:min_cpu_usage"`
		MinMemoryUsage *float64 `gorm:"column:min_memory_usage"`
		MinDiskUsage   *float64 `gorm:"column:min_disk_usage"`
		SampleCount    int64    `gorm:"column:sample_count"`
	}

	if err := r.db.Model(&models.Metric{}).
		Select(`
			AVG(cpu_usage) as avg_cpu_usage,
			AVG(memory_percent) as avg_memory_usage,
			AVG(disk_percent) as avg_disk_usage,
			MAX(cpu_usage) as max_cpu_usage,
			MAX(memory_percent) as max_memory_usage,
			MAX(disk_percent) as max_disk_usage,
			MIN(cpu_usage) as min_cpu_usage,
			MIN(memory_percent) as min_memory_usage,
			MIN(disk_percent) as min_disk_usage,
			COUNT(*) as sample_count
		`).
		Where("hostname = ? AND timestamp >= ?", hostname, since).
		Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate average usage for hostname: %w", err)
	}

	// Handle NULL values from database
	averageMetrics := &models.AverageMetrics{
		Hostname:    hostname,
		Duration:    duration,
		From:        since,
		To:          time.Now(),
		SampleCount: result.SampleCount,
	}

	if result.AvgCPUUsage != nil {
		averageMetrics.AvgCPUUsage = *result.AvgCPUUsage
	}
	if result.AvgMemoryUsage != nil {
		averageMetrics.AvgMemoryUsage = *result.AvgMemoryUsage
	}
	if result.AvgDiskUsage != nil {
		averageMetrics.AvgDiskUsage = *result.AvgDiskUsage
	}
	if result.MaxCPUUsage != nil {
		averageMetrics.MaxCPUUsage = *result.MaxCPUUsage
	}
	if result.MaxMemoryUsage != nil {
		averageMetrics.MaxMemoryUsage = *result.MaxMemoryUsage
	}
	if result.MaxDiskUsage != nil {
		averageMetrics.MaxDiskUsage = *result.MaxDiskUsage
	}
	if result.MinCPUUsage != nil {
		averageMetrics.MinCPUUsage = *result.MinCPUUsage
	}
	if result.MinMemoryUsage != nil {
		averageMetrics.MinMemoryUsage = *result.MinMemoryUsage
	}
	if result.MinDiskUsage != nil {
		averageMetrics.MinDiskUsage = *result.MinDiskUsage
	}

	return averageMetrics, nil
}

// GetMetricsSummary provides a summary of metrics for a time period
func (r *metricsRepository) GetMetricsSummary(from, to time.Time) (*models.MetricsSummary, error) {
	var summary models.MetricsSummary

	// Get basic statistics
	if err := r.db.Model(&models.Metric{}).
		Select(`
			COUNT(*) as total_records,
			COUNT(DISTINCT hostname) as unique_hosts,
			AVG(cpu_usage) as avg_cpu_usage,
			AVG(memory_percent) as avg_memory_usage,
			AVG(disk_percent) as avg_disk_usage
		`).
		Where("timestamp BETWEEN ? AND ?", from, to).
		Scan(&summary).Error; err != nil {
		return nil, fmt.Errorf("failed to get metrics summary: %w", err)
	}

	summary.From = from
	summary.To = to
	summary.Duration = to.Sub(from)

	return &summary, nil
}

// GetTopHostsByUsage returns hosts with highest resource usage
func (r *metricsRepository) GetTopHostsByUsage(metricType string, limit int) ([]*models.HostUsage, error) {
	var results []*models.HostUsage

	var orderBy string
	switch metricType {
	case "cpu":
		orderBy = "avg_cpu_usage DESC"
	case "memory":
		orderBy = "avg_memory_usage DESC"
	case "disk":
		orderBy = "avg_disk_usage DESC"
	default:
		return nil, fmt.Errorf("invalid metric type: %s", metricType)
	}

	since := time.Now().Add(-24 * time.Hour) // Last 24 hours

	if err := r.db.Model(&models.Metric{}).
		Select(`
			hostname,
			AVG(cpu_usage) as avg_cpu_usage,
			AVG(memory_percent) as avg_memory_usage,
			AVG(disk_percent) as avg_disk_usage,
			MAX(timestamp) as last_seen
		`).
		Where("timestamp >= ?", since).
		Group("hostname").
		Order(orderBy).
		Limit(limit).
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get top hosts by usage: %w", err)
	}

	return results, nil
}

// GetUsageTrends returns usage trends for a hostname over time (FIXED - PostgreSQL compatible)
func (r *metricsRepository) GetUsageTrends(hostname string, hours int) ([]*models.UsageTrend, error) {
	var trends []*models.UsageTrend

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	// PostgreSQL DATE_TRUNC function for hour grouping
	if err := r.db.Model(&models.Metric{}).
		Select(`
			DATE_TRUNC('hour', timestamp) as hour,
			AVG(cpu_usage) as avg_cpu_usage,
			AVG(memory_percent) as avg_memory_usage,
			AVG(disk_percent) as avg_disk_usage,
			COUNT(*) as sample_count
		`).
		Where("hostname = ? AND timestamp >= ?", hostname, since).
		Group("DATE_TRUNC('hour', timestamp)").
		Order("hour DESC").
		Scan(&trends).Error; err != nil {
		return nil, fmt.Errorf("failed to get usage trends: %w", err)
	}

	return trends, nil
}

// DeleteOldRecords removes metrics older than the specified time
func (r *metricsRepository) DeleteOldRecords(olderThan time.Time) (int64, error) {
	result := r.db.Where("timestamp < ?", olderThan).Delete(&models.Metric{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old records: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// GetTotalCount returns the total number of metric records
func (r *metricsRepository) GetTotalCount() (int64, error) {
	var count int64
	if err := r.db.Model(&models.Metric{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to get total count: %w", err)
	}
	return count, nil
}

// GetCountByDateRange returns the count of metrics within a date range
func (r *metricsRepository) GetCountByDateRange(from, to time.Time) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Metric{}).
		Where("timestamp BETWEEN ? AND ?", from, to).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to get count by date range: %w", err)
	}
	return count, nil
}

// GetSystemStatus returns current system status for all hosts (FIXED PostgreSQL VERSION)
func (r *metricsRepository) GetSystemStatus() ([]*models.SystemStatus, error) {
	var status []*models.SystemStatus

	// PostgreSQL specific query with DISTINCT ON
	if err := r.db.Raw(`
		SELECT DISTINCT ON (hostname)
			hostname,
			cpu_usage,
			memory_percent,
			disk_percent,
			timestamp,
			CASE
				WHEN timestamp > NOW() - INTERVAL '5 minutes' THEN 'online'
				WHEN timestamp > NOW() - INTERVAL '15 minutes' THEN 'warning'
				ELSE 'offline'
			END as status
		FROM metrics
		ORDER BY hostname, timestamp DESC
	`).Scan(&status).Error; err != nil {
		return nil, fmt.Errorf("failed to get system status: %w", err)
	}

	// If no database results, try to provide a default status
	if len(status) == 0 {
		log.Println("🔍 [DEBUG] No system status found in database, providing default status")
		// Return a default status to prevent frontend issues
		status = append(status, &models.SystemStatus{
			Hostname:      "localhost",
			CPUUsage:      0,
			MemoryPercent: 0,
			DiskPercent:   0,
			Timestamp:     time.Now(),
			Status:        "online",
		})
	}

	return status, nil
}
