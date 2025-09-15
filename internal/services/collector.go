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

	// Alert integration
	alertService *AlertService

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
}

// NewCollectorService creates a new collector service
func NewCollectorService(cfg *config.Config, metricsRepo repository.MetricsRepository) *CollectorService {
	// FIX: Check if cfg is nil
	if cfg == nil {
		cfg = &config.Config{
			Metrics: config.MetricsConfig{
				CollectionInterval: 30 * time.Second,
				EnableCPU:          true,
				EnableMemory:       true,
				EnableDisk:         true,
				EnableNetwork:      true,
				EnableProcesses:    true,
			},
		}
	}

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
	}
}

// SetAlertService sets the alert service for integration
func (cs *CollectorService) SetAlertService(alertService *AlertService) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.alertService = alertService
	log.Println("🔗 Alert service integrated with collector service")
}

// GetSystemCollector returns the system collector
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

	// FIX: Check if config is nil
	if cs.config == nil {
		return fmt.Errorf("collector service configuration is nil")
	}

	log.Printf("🚀 Starting collector service with %v interval", cs.config.Metrics.CollectionInterval)

	// Create context for this service
	cs.ctx, cs.cancel = context.WithCancel(ctx)

	// FIX: Check if systemCollector is nil
	if cs.systemCollector == nil {
		return fmt.Errorf("system collector is not initialized")
	}

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
	log.Println("📊 Starting metrics processing routine with alert integration...")

	for {
		select {
		case metrics, ok := <-cs.metricsChan:
			if !ok {
				log.Println("📊 Metrics channel closed, stopping processing")
				return
			}

			// FIX: Additional nil check for metrics
			if metrics == nil {
				log.Printf("❌ Received nil metrics, skipping")
				continue
			}

			// Store metrics in database
			if err := cs.storeMetrics(metrics); err != nil {
				log.Printf("❌ Error storing metrics: %v", err)
			} else {
				cs.mutex.Lock()
				cs.collectionsCount++
				cs.lastCollectionTime = time.Now()
				cs.mutex.Unlock()

				// Debug log removed - too verbose in production
			}

			// NEW: Check alerts after storing metrics
			cs.checkAlerts(metrics)

		case <-cs.stopChan:
			log.Println("📊 Received stop signal, stopping metrics processing")
			return

		case <-cs.ctx.Done():
			log.Println("📊 Context cancelled, stopping metrics processing")
			return
		}
	}
}

// checkAlerts runs alert checking if alert service is available
func (cs *CollectorService) checkAlerts(metrics *models.SystemMetrics) {
	cs.mutex.RLock()
	alertService := cs.alertService
	cs.mutex.RUnlock()

	if alertService == nil {
		return // No alert service configured
	}

	// Run alert checking in background to avoid blocking metrics collection
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("❌ Panic in alert checking: %v", r)
			}
		}()

		alertService.CheckMetrics(metrics)
	}()
}

// storeMetrics stores system metrics in the database
func (cs *CollectorService) storeMetrics(systemMetrics *models.SystemMetrics) error {
	if systemMetrics == nil {
		return fmt.Errorf("received nil system metrics")
	}

	// FIX: Check if metricsRepo is nil
	if cs.metricsRepo == nil {
		return fmt.Errorf("metrics repository is not available")
	}

	// Convert SystemMetrics to database Metric model
	dbMetric := models.ConvertSystemMetricsToDBMetric(systemMetrics)

	// FIX: Check if conversion returned nil
	if dbMetric == nil {
		return fmt.Errorf("failed to convert system metrics to database model")
	}

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

	// FIX: Check if config is nil
	if cs.config == nil {
		log.Println("❌ Config is nil, cannot start cleanup routine")
		return
	}

	log.Printf("🧹 Starting cleanup routine: will delete metrics older than %d days", cs.config.Metrics.RetentionDays)

	for {
		select {
		case <-ticker.C:
			if err := cs.cleanupOldMetrics(); err != nil {
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
	// FIX: Check if config is nil
	if cs.config == nil {
		return fmt.Errorf("configuration is not available")
	}

	// FIX: Check if metricsRepo is nil
	if cs.metricsRepo == nil {
		return fmt.Errorf("metrics repository is not available")
	}

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

// GetStats returns collector service statistics
func (cs *CollectorService) GetStats() map[string]interface{} {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	stats := map[string]interface{}{
		"is_running":           cs.isRunning,
		"collections_count":    cs.collectionsCount,
		"last_collection_time": cs.lastCollectionTime,
		"alert_service_linked": cs.alertService != nil,
	}

	return stats
}
