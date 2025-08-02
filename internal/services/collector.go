package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/eyzaun/godash/internal/collector"
	"github.com/eyzaun/godash/internal/config"
	"github.com/eyzaun/godash/internal/models"
	"github.com/eyzaun/godash/internal/repository"
)

// CollectorService manages metrics collection and storage
type CollectorService struct {
	systemCollector collector.Collector
	metricsRepo     repository.MetricsRepository
	config          *config.Config

	// Collection state
	isRunning   bool
	stopChan    chan bool
	metricsChan <-chan *models.SystemMetrics
	ctx         context.Context
	cancel      context.CancelFunc
	mutex       sync.RWMutex

	// Statistics
	collectionsCount   int64
	lastCollectionTime time.Time
	errors             []error
}

// NewCollectorService creates a new collector service
func NewCollectorService(cfg *config.Config, metricsRepo repository.MetricsRepository) *CollectorService {
	// Create collector configuration
	collectorConfig := &collector.CollectorConfig{
		CollectInterval: cfg.Metrics.CollectionInterval,
		EnableCPU:       cfg.Metrics.EnableCPU,
		EnableMemory:    cfg.Metrics.EnableMemory,
		EnableDisk:      cfg.Metrics.EnableDisk,
		EnableNetwork:   cfg.Metrics.EnableNetwork,
		EnableProcesses: cfg.Metrics.EnableProcesses,
	}

	// Create system collector
	systemCollector := collector.NewSystemCollector(collectorConfig)

	return &CollectorService{
		systemCollector: systemCollector,
		metricsRepo:     metricsRepo,
		config:          cfg,
		stopChan:        make(chan bool, 1),
		errors:          make([]error, 0),
	}
}

// GetSystemCollector returns the system collector (YENİ METOD)
func (cs *CollectorService) GetSystemCollector() collector.Collector {
	return cs.systemCollector
}

// Start starts the metrics collection service
func (cs *CollectorService) Start(ctx context.Context) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if cs.isRunning {
		return fmt.Errorf("collector service is already running")
	}

	log.Printf("🚀 Starting collector service with %v interval", cs.config.Metrics.CollectionInterval)

	// Create context for this service
	cs.ctx, cs.cancel = context.WithCancel(ctx)

	// Start metrics collection from system collector
	cs.metricsChan = cs.systemCollector.StartCollection(cs.ctx, cs.config.Metrics.CollectionInterval)

	// Start the collection processing goroutine
	go cs.processMetrics()

	// Start cleanup routine if retention is configured
	if cs.config.Metrics.RetentionDays > 0 {
		go cs.startCleanupRoutine()
	}

	cs.isRunning = true
	log.Println("✅ Collector service started successfully")

	return nil
}

// Stop stops the metrics collection service
func (cs *CollectorService) Stop() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if !cs.isRunning {
		return fmt.Errorf("collector service is not running")
	}

	log.Println("🛑 Stopping collector service...")

	// Cancel context to stop collection
	if cs.cancel != nil {
		cs.cancel()
	}

	// Send stop signal
	select {
	case cs.stopChan <- true:
	default:
		// Channel might be full or closed
	}

	cs.isRunning = false
	log.Println("✅ Collector service stopped successfully")

	return nil
}

// processMetrics processes incoming metrics and stores them in database
func (cs *CollectorService) processMetrics() {
	log.Println("📊 Starting metrics processing routine...")

	for {
		select {
		case metrics, ok := <-cs.metricsChan:
			if !ok {
				log.Println("📊 Metrics channel closed, stopping processing")
				return
			}

			if err := cs.storeMetrics(metrics); err != nil {
				cs.recordError(fmt.Errorf("failed to store metrics: %w", err))
				log.Printf("❌ Error storing metrics: %v", err)
			} else {
				cs.mutex.Lock()
				cs.collectionsCount++
				cs.lastCollectionTime = time.Now()
				cs.mutex.Unlock()

				log.Printf("✅ Stored metrics: CPU=%.1f%%, Memory=%.1f%%, Disk=%.1f%%",
					metrics.CPU.Usage, metrics.Memory.Percent, metrics.Disk.Percent)
			}

		case <-cs.stopChan:
			log.Println("📊 Received stop signal, stopping metrics processing")
			return

		case <-cs.ctx.Done():
			log.Println("📊 Context cancelled, stopping metrics processing")
			return
		}
	}
}

// storeMetrics stores system metrics in the database
func (cs *CollectorService) storeMetrics(systemMetrics *models.SystemMetrics) error {
	if systemMetrics == nil {
		return fmt.Errorf("received nil system metrics")
	}

	// Convert SystemMetrics to database Metric model
	dbMetric := models.ConvertSystemMetricsToDBMetric(systemMetrics)

	// Store in database
	if err := cs.metricsRepo.Create(dbMetric); err != nil {
		return fmt.Errorf("failed to create metric record: %w", err)
	}

	return nil
}

// startCleanupRoutine starts the periodic cleanup of old metrics
func (cs *CollectorService) startCleanupRoutine() {
	// Run cleanup every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	log.Printf("🧹 Starting cleanup routine: will delete metrics older than %d days", cs.config.Metrics.RetentionDays)

	for {
		select {
		case <-ticker.C:
			if err := cs.cleanupOldMetrics(); err != nil {
				cs.recordError(fmt.Errorf("cleanup failed: %w", err))
				log.Printf("❌ Error during metrics cleanup: %v", err)
			}

		case <-cs.ctx.Done():
			log.Println("🧹 Context cancelled, stopping cleanup routine")
			return
		}
	}
}

// cleanupOldMetrics removes metrics older than the retention period
func (cs *CollectorService) cleanupOldMetrics() error {
	cutoffTime := time.Now().AddDate(0, 0, -cs.config.Metrics.RetentionDays)

	deletedCount, err := cs.metricsRepo.DeleteOldRecords(cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old records: %w", err)
	}

	if deletedCount > 0 {
		log.Printf("🧹 Cleaned up %d old metric records older than %d days", deletedCount, cs.config.Metrics.RetentionDays)
	} else {
		log.Printf("🧹 No old metrics to clean up (retention: %d days)", cs.config.Metrics.RetentionDays)
	}

	return nil
}

// recordError records an error for statistics
func (cs *CollectorService) recordError(err error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cs.errors = append(cs.errors, err)

	// Keep only last 10 errors
	if len(cs.errors) > 10 {
		cs.errors = cs.errors[len(cs.errors)-10:]
	}
}

// GetStats returns service statistics
func (cs *CollectorService) GetStats() map[string]interface{} {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	return map[string]interface{}{
		"is_running":           cs.isRunning,
		"collections_count":    cs.collectionsCount,
		"last_collection_time": cs.lastCollectionTime,
		"error_count":          len(cs.errors),
		"collection_interval":  cs.config.Metrics.CollectionInterval,
		"retention_days":       cs.config.Metrics.RetentionDays,
		"enabled_metrics": map[string]bool{
			"cpu":       cs.config.Metrics.EnableCPU,
			"memory":    cs.config.Metrics.EnableMemory,
			"disk":      cs.config.Metrics.EnableDisk,
			"network":   cs.config.Metrics.EnableNetwork,
			"processes": cs.config.Metrics.EnableProcesses,
		},
	}
}

// GetLastErrors returns the last recorded errors
func (cs *CollectorService) GetLastErrors() []error {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	// Return a copy of the errors slice
	errors := make([]error, len(cs.errors))
	copy(errors, cs.errors)
	return errors
}

// IsRunning returns whether the service is currently running
func (cs *CollectorService) IsRunning() bool {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	return cs.isRunning
}

// ForceCollection triggers an immediate metrics collection
func (cs *CollectorService) ForceCollection() (*models.SystemMetrics, error) {
	if cs.systemCollector == nil {
		return nil, fmt.Errorf("system collector is not available")
	}

	metrics, err := cs.systemCollector.GetSystemMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get system metrics: %w", err)
	}

	// Store the metrics
	if err := cs.storeMetrics(metrics); err != nil {
		// Log error but still return the metrics
		log.Printf("❌ Warning: failed to store forced collection metrics: %v", err)
	}

	return metrics, nil
}

// GetCurrentMetrics returns current system metrics without storing them
func (cs *CollectorService) GetCurrentMetrics() (*models.SystemMetrics, error) {
	if cs.systemCollector == nil {
		return nil, fmt.Errorf("system collector is not available")
	}

	return cs.systemCollector.GetSystemMetrics()
}

// GetSystemInfo returns current system information
func (cs *CollectorService) GetSystemInfo() (*models.SystemInfo, error) {
	if cs.systemCollector == nil {
		return nil, fmt.Errorf("system collector is not available")
	}

	return cs.systemCollector.GetSystemInfo()
}

// HealthCheck checks if the collector service is healthy
func (cs *CollectorService) HealthCheck() (bool, []string, error) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	issues := []string{}
	healthy := true

	// Check if service is running
	if !cs.isRunning {
		healthy = false
		issues = append(issues, "Collector service is not running")
	}

	// Check if recent collection happened
	if cs.isRunning && !cs.lastCollectionTime.IsZero() {
		timeSinceLastCollection := time.Since(cs.lastCollectionTime)
		maxExpectedInterval := cs.config.Metrics.CollectionInterval * 3

		if timeSinceLastCollection > maxExpectedInterval {
			healthy = false
			issues = append(issues, fmt.Sprintf("No metrics collected for %v", timeSinceLastCollection))
		}
	}

	// Check error count
	if len(cs.errors) > 5 {
		healthy = false
		issues = append(issues, fmt.Sprintf("High error count: %d recent errors", len(cs.errors)))
	}

	// Check system collector health if available
	if cs.systemCollector != nil {
		systemHealthy, systemIssues, err := cs.systemCollector.IsHealthy()
		if err != nil {
			healthy = false
			issues = append(issues, fmt.Sprintf("System collector health check failed: %v", err))
		} else if !systemHealthy {
			healthy = false
			issues = append(issues, systemIssues...)
		}
	}

	return healthy, issues, nil
}
