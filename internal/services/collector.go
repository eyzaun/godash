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

// CollectorService handles background metrics collection
type CollectorService struct {
	config         *config.Config
	systemCollector collector.Collector
	metricsRepo    repository.MetricsRepository
	
	// Service state
	running        bool
	mutex          sync.RWMutex
	stopChan       chan struct{}
	errorChan      chan error
	
	// Metrics collection state
	collectionCount int64
	lastCollection  time.Time
	errors          []error
	maxErrors       int
	
	// Batch processing
	batchBuffer    []*models.Metric
	batchMutex     sync.Mutex
	batchTicker    *time.Ticker
}

// NewCollectorService creates a new collector service
func NewCollectorService(cfg *config.Config, metricsRepo repository.MetricsRepository) *CollectorService {
	// Create collector config
	collectorConfig := &collector.CollectorConfig{
		CollectInterval: cfg.Metrics.CollectionInterval,
		EnableCPU:       cfg.Metrics.EnableCPU,
		EnableMemory:    cfg.Metrics.EnableMemory,
		EnableDisk:      cfg.Metrics.EnableDisk,
		EnableNetwork:   cfg.Metrics.EnableNetwork,
		EnableProcesses: cfg.Metrics.EnableProcesses,
	}

	systemCollector := collector.NewSystemCollector(collectorConfig)

	return &CollectorService{
		config:          cfg,
		systemCollector: systemCollector,
		metricsRepo:     metricsRepo,
		running:         false,
		stopChan:        make(chan struct{}),
		errorChan:       make(chan error, 100), // Buffered error channel
		maxErrors:       10,
		batchBuffer:     make([]*models.Metric, 0, cfg.Metrics.BatchSize),
	}
}

// Start begins the background metrics collection
func (cs *CollectorService) Start(ctx context.Context) error {
	cs.mutex.Lock()
	if cs.running {
		cs.mutex.Unlock()
		return fmt.Errorf("collector service is already running")
	}
	cs.running = true
	cs.mutex.Unlock()

	log.Printf("Starting metrics collection service with interval: %v", cs.config.Metrics.CollectionInterval)

	// Start batch processing ticker
	cs.batchTicker = time.NewTicker(30 * time.Second) // Process batch every 30 seconds

	// Start collection goroutines
	go cs.collectionWorker(ctx)
	go cs.batchProcessor(ctx)
	go cs.errorHandler(ctx)
	go cs.cleanupWorker(ctx)

	log.Println("Metrics collection service started successfully")
	return nil
}

// Stop stops the background metrics collection
func (cs *CollectorService) Stop() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if !cs.running {
		return fmt.Errorf("collector service is not running")
	}

	log.Println("Stopping metrics collection service...")

	// Signal stop to all goroutines
	close(cs.stopChan)
	
	// Stop batch ticker
	if cs.batchTicker != nil {
		cs.batchTicker.Stop()
	}

	// Process any remaining metrics in buffer
	cs.flushBatchBuffer()

	cs.running = false
	log.Println("Metrics collection service stopped")
	return nil
}

// collectionWorker performs continuous metrics collection
func (cs *CollectorService) collectionWorker(ctx context.Context) {
	ticker := time.NewTicker(cs.config.Metrics.CollectionInterval)
	defer ticker.Stop()

	// Collect initial metrics immediately
	cs.collectAndStore()

	for {
		select {
		case <-ticker.C:
			cs.collectAndStore()
		case <-cs.stopChan:
			log.Println("Collection worker stopped")
			return
		case <-ctx.Done():
			log.Println("Collection worker stopped due to context cancellation")
			return
		}
	}
}

// collectAndStore collects metrics and adds them to batch buffer
func (cs *CollectorService) collectAndStore() {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in collectAndStore: %v", r)
			cs.addError(err)
			log.Printf("Error: %v", err)
		}
	}()

	// Collect system metrics
	systemMetrics, err := cs.systemCollector.GetSystemMetrics()
	if err != nil {
		cs.addError(fmt.Errorf("failed to collect system metrics: %w", err))
		return
	}

	// Convert to database model
	metric := models.ConvertSystemMetricsToDBMetric(systemMetrics)

	// Add to batch buffer
	cs.addToBatch(metric)

	// Update collection stats
	cs.mutex.Lock()
	cs.collectionCount++
	cs.lastCollection = time.Now()
	cs.mutex.Unlock()

	// Log successful collection (less frequently)
	if cs.collectionCount%10 == 0 {
		log.Printf("Collected %d metrics so far. Latest: CPU=%.1f%%, Memory=%.1f%%, Disk=%.1f%%",
			cs.collectionCount,
			systemMetrics.CPU.Usage,
			systemMetrics.Memory.Percent,
			systemMetrics.Disk.Percent,
		)
	}
}

// addToBatch adds a metric to the batch buffer
func (cs *CollectorService) addToBatch(metric *models.Metric) {
	cs.batchMutex.Lock()
	defer cs.batchMutex.Unlock()

	cs.batchBuffer = append(cs.batchBuffer, metric)

	// If batch is full, trigger immediate processing
	if len(cs.batchBuffer) >= cs.config.Metrics.BatchSize {
		go cs.processBatch()
	}
}

// batchProcessor processes batches at regular intervals
func (cs *CollectorService) batchProcessor(ctx context.Context) {
	for {
		select {
		case <-cs.batchTicker.C:
			cs.processBatch()
		case <-cs.stopChan:
			log.Println("Batch processor stopped")
			return
		case <-ctx.Done():
			log.Println("Batch processor stopped due to context cancellation")
			return
		}
	}
}

// processBatch processes the current batch of metrics
func (cs *CollectorService) processBatch() {
	cs.batchMutex.Lock()
	if len(cs.batchBuffer) == 0 {
		cs.batchMutex.Unlock()
		return
	}

	// Get current batch and reset buffer
	batch := make([]*models.Metric, len(cs.batchBuffer))
	copy(batch, cs.batchBuffer)
	cs.batchBuffer = cs.batchBuffer[:0] // Reset slice
	cs.batchMutex.Unlock()

	// Process batch
	start := time.Now()
	if err := cs.metricsRepo.CreateBatch(batch); err != nil {
		cs.addError(fmt.Errorf("failed to save batch of %d metrics: %w", len(batch), err))
		log.Printf("Error saving batch: %v", err)
		return
	}

	duration := time.Since(start)
	log.Printf("Successfully saved batch of %d metrics in %v", len(batch), duration)
}

// flushBatchBuffer processes any remaining metrics in the buffer
func (cs *CollectorService) flushBatchBuffer() {
	cs.batchMutex.Lock()
	defer cs.batchMutex.Unlock()

	if len(cs.batchBuffer) == 0 {
		return
	}

	log.Printf("Flushing remaining %d metrics from buffer", len(cs.batchBuffer))
	
	if err := cs.metricsRepo.CreateBatch(cs.batchBuffer); err != nil {
		log.Printf("Error flushing batch buffer: %v", err)
	}

	cs.batchBuffer = cs.batchBuffer[:0]
}

// errorHandler manages error reporting and recovery
func (cs *CollectorService) errorHandler(ctx context.Context) {
	for {
		select {
		case err := <-cs.errorChan:
			log.Printf("Collection error: %v", err)
			
			// In production, you might want to:
			// - Send errors to monitoring system
			// - Implement exponential backoff
			// - Send alerts for critical errors
			
		case <-cs.stopChan:
			log.Println("Error handler stopped")
			return
		case <-ctx.Done():
			log.Println("Error handler stopped due to context cancellation")
			return
		}
	}
}

// cleanupWorker periodically cleans up old metrics
func (cs *CollectorService) cleanupWorker(ctx context.Context) {
	// Run cleanup every 6 hours
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	// Run initial cleanup after 1 hour
	initialTimer := time.NewTimer(1 * time.Hour)
	defer initialTimer.Stop()

	for {
		select {
		case <-initialTimer.C:
			cs.performCleanup()
		case <-ticker.C:
			cs.performCleanup()
		case <-cs.stopChan:
			log.Println("Cleanup worker stopped")
			return
		case <-ctx.Done():
			log.Println("Cleanup worker stopped due to context cancellation")
			return
		}
	}
}

// performCleanup removes old metrics based on retention policy
func (cs *CollectorService) performCleanup() {
	if cs.config.Metrics.RetentionDays <= 0 {
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -cs.config.Metrics.RetentionDays)
	
	log.Printf("Starting cleanup of metrics older than %d days (%v)", 
		cs.config.Metrics.RetentionDays, cutoffTime)

	deletedCount, err := cs.metricsRepo.DeleteOldRecords(cutoffTime)
	if err != nil {
		cs.addError(fmt.Errorf("cleanup failed: %w", err))
		return
	}

	if deletedCount > 0 {
		log.Printf("Cleanup completed: removed %d old metric records", deletedCount)
	} else {
		log.Println("Cleanup completed: no old records to remove")
	}
}

// addError adds an error to the error tracking
func (cs *CollectorService) addError(err error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cs.errors = append(cs.errors, err)

	// Keep only the last maxErrors
	if len(cs.errors) > cs.maxErrors {
		cs.errors = cs.errors[len(cs.errors)-cs.maxErrors:]
	}

	// Send to error channel (non-blocking)
	select {
	case cs.errorChan <- err:
	default:
		// Channel is full, skip this error
	}
}

// GetStats returns collection service statistics
func (cs *CollectorService) GetStats() map[string]interface{} {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	cs.batchMutex.Lock()
	batchSize := len(cs.batchBuffer)
	cs.batchMutex.Unlock()

	return map[string]interface{}{
		"running":            cs.running,
		"collection_count":   cs.collectionCount,
		"last_collection":    cs.lastCollection,
		"collection_interval": cs.config.Metrics.CollectionInterval,
		"batch_size":         batchSize,
		"error_count":        len(cs.errors),
		"enabled_metrics": map[string]bool{
			"cpu":       cs.config.Metrics.EnableCPU,
			"memory":    cs.config.Metrics.EnableMemory,
			"disk":      cs.config.Metrics.EnableDisk,
			"network":   cs.config.Metrics.EnableNetwork,
			"processes": cs.config.Metrics.EnableProcesses,
		},
	}
}

// GetLastErrors returns the most recent errors
func (cs *CollectorService) GetLastErrors() []string {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	errors := make([]string, len(cs.errors))
	for i, err := range cs.errors {
		errors[i] = err.Error()
	}
	return errors
}

// IsRunning returns whether the service is currently running
func (cs *CollectorService) IsRunning() bool {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	return cs.running
}

// GetCollectionCount returns the total number of metrics collected
func (cs *CollectorService) GetCollectionCount() int64 {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	return cs.collectionCount
}

// GetLastCollection returns the timestamp of the last collection
func (cs *CollectorService) GetLastCollection() time.Time {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	return cs.lastCollection
}

// ForceCollection triggers an immediate metrics collection
func (cs *CollectorService) ForceCollection() error {
	if !cs.IsRunning() {
		return fmt.Errorf("collector service is not running")
	}

	go cs.collectAndStore()
	return nil
}

// ForceBatchProcess triggers immediate batch processing
func (cs *CollectorService) ForceBatchProcess() error {
	if !cs.IsRunning() {
		return fmt.Errorf("collector service is not running")
	}

	go cs.processBatch()
	return nil
}

// UpdateConfig updates the service configuration
func (cs *CollectorService) UpdateConfig(newConfig *config.Config) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Update config
	cs.config = newConfig

	// Update system collector configuration
	collectorConfig := &collector.CollectorConfig{
		CollectInterval: newConfig.Metrics.CollectionInterval,
		EnableCPU:       newConfig.Metrics.EnableCPU,
		EnableMemory:    newConfig.Metrics.EnableMemory,
		EnableDisk:      newConfig.Metrics.EnableDisk,
		EnableNetwork:   newConfig.Metrics.EnableNetwork,
		EnableProcesses: newConfig.Metrics.EnableProcesses,
	}

	cs.systemCollector = collector.NewSystemCollector(collectorConfig)

	log.Println("Collector service configuration updated")
	return nil
}