package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/eyzaun/godash/internal/config"
	"github.com/eyzaun/godash/internal/models"
	"github.com/eyzaun/godash/internal/repository"
)

// AlertService manages alert checking and notifications
type AlertService struct {
	alertRepo        repository.AlertRepository
	emailSender      EmailSender
	webhookSender    WebhookSender
	websocketHandler interface{} // WebSocket handler for broadcasting alerts
	config           *config.AlertConfig

	// Alert state management
	lastAlerts     map[string]time.Time // Key: alert_id:hostname, Value: last triggered time
	alertDurations map[string]time.Time // Key: alert_id:hostname, Value: first triggered time
	isRunning      bool
	stopChan       chan bool
	ctx            context.Context
	cancel         context.CancelFunc
	mutex          sync.RWMutex

	// Statistics
	checkedCount   int64
	triggeredCount int64
	lastCheckTime  time.Time
}

// NewAlertService creates a new alert service
func NewAlertService(
	cfg *config.Config,
	alertRepo repository.AlertRepository,
	emailSender EmailSender,
	webhookSender WebhookSender,
) *AlertService {
	// Extract alert config or use defaults
	var alertConfig *config.AlertConfig
	if cfg != nil && cfg.Alerts != nil {
		alertConfig = cfg.Alerts
	} else {
		alertConfig = &config.AlertConfig{
			EnableAlerts:   true,
			CheckInterval:  30 * time.Second,
			CooldownPeriod: 5 * time.Minute,
		}
	}

	return &AlertService{
		alertRepo:      alertRepo,
		emailSender:    emailSender,
		webhookSender:  webhookSender,
		config:         alertConfig,
		lastAlerts:     make(map[string]time.Time),
		alertDurations: make(map[string]time.Time),
		stopChan:       make(chan bool, 1),
	}
}

// Start starts the alert checking service
func (as *AlertService) Start(ctx context.Context) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if as.isRunning {
		return fmt.Errorf("alert service is already running")
	}

	if !as.config.EnableAlerts {
		log.Println("🔕 Alert service is disabled in configuration")
		return nil
	}

	log.Printf("🚨 Starting alert service with %v check interval", as.config.CheckInterval)

	// Create context for this service
	as.ctx, as.cancel = context.WithCancel(ctx)

	// Start the alert checking routine
	go as.alertCheckingRoutine()

	as.isRunning = true
	log.Println("✅ Alert service started successfully")

	return nil
}

// Stop stops the alert checking service
func (as *AlertService) Stop() error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if !as.isRunning {
		return fmt.Errorf("alert service is not running")
	}

	log.Println("🛑 Stopping alert service...")

	// Cancel context to stop checking
	if as.cancel != nil {
		as.cancel()
	}

	// Send stop signal
	select {
	case as.stopChan <- true:
	default:
		// Channel might be full or closed
	}

	as.isRunning = false
	log.Println("✅ Alert service stopped successfully")

	return nil
}

// CheckMetrics checks metrics against alert thresholds
func (as *AlertService) CheckMetrics(metrics *models.SystemMetrics) {
	if !as.config.EnableAlerts {
		return
	}

	if metrics == nil {
		log.Printf("❌ Received nil metrics for alert checking")
		return
	}

	as.mutex.Lock()
	as.checkedCount++
	as.lastCheckTime = time.Now()
	as.mutex.Unlock()

	// Get active alerts
	alerts, err := as.alertRepo.GetActiveAlerts()
	if err != nil {
		log.Printf("❌ Failed to get active alerts: %v", err)
		return
	}

	if len(alerts) == 0 {
		return // No alerts configured
	}

	// Check each alert
	for _, alert := range alerts {
		if err := as.checkSingleAlert(alert, metrics); err != nil {
			// Debug log removed - too verbose in production
		}
	}
}

// checkSingleAlert processes a single alert against current metrics
func (as *AlertService) checkSingleAlert(alert *models.Alert, metrics *models.SystemMetrics) error {
	as.mutex.Lock()
	as.checkedCount++
	as.mutex.Unlock()

	// Get current metric value based on alert type
	var currentValue float64
	var hostname string

	switch alert.MetricType {
	case "cpu":
		currentValue = metrics.CPU.Usage
		hostname = metrics.Hostname
	case "memory":
		currentValue = metrics.Memory.Percent
		hostname = metrics.Hostname
	case "disk":
		currentValue = metrics.Disk.Percent
		hostname = metrics.Hostname
	case "load_avg_1":
		if len(metrics.CPU.LoadAvg) > 0 {
			currentValue = metrics.CPU.LoadAvg[0]
		}
		hostname = metrics.Hostname
	case "load_avg_5":
		if len(metrics.CPU.LoadAvg) > 1 {
			currentValue = metrics.CPU.LoadAvg[1]
		}
		hostname = metrics.Hostname
	case "load_avg_15":
		if len(metrics.CPU.LoadAvg) > 2 {
			currentValue = metrics.CPU.LoadAvg[2]
		}
		hostname = metrics.Hostname
	default:
		log.Printf("Unknown metric type: %s", alert.MetricType)
		return nil
	}

	alertKey := fmt.Sprintf("%d_%s", alert.ID, hostname)

	// Evaluate condition
	conditionMet, err := as.evaluateCondition(alert.Condition, currentValue, alert.Threshold)
	if err != nil {
		log.Printf("❌ Error evaluating condition for alert %d: %v", alert.ID, err)
		return err
	}

	log.Printf("🔍 Alert %d (%s): %s %s %.2f (current: %.2f) - Condition met: %v",
		alert.ID, alert.Name, alert.MetricType, alert.Condition, alert.Threshold, currentValue, conditionMet)

	if conditionMet {
		// Check if alert should trigger (duration/cooldown logic)
		if as.shouldTriggerAlert(alert, alertKey) {
			log.Printf("🚨 Alert %d TRIGGERING: %s on %s (%.2f %s %.2f)",
				alert.ID, alert.Name, hostname, currentValue, alert.Condition, alert.Threshold)

			// Create alert history entry
			history := &models.AlertHistory{
				AlertID:     alert.ID,
				Hostname:    metrics.Hostname,
				MetricValue: currentValue,
				Threshold:   alert.Threshold,
				Severity:    alert.Severity,
				Message:     as.generateAlertMessage(alert, currentValue, metrics.Hostname),
				Resolved:    false,
			}

			if err := as.alertRepo.CreateAlertHistory(history); err != nil {
				log.Printf("❌ Failed to create alert history: %v", err)
			} else {
				log.Printf("🚨 Alert triggered: %s on %s (%.2f %s threshold)",
					alert.Name, metrics.Hostname, currentValue, alert.Condition)

				as.mutex.Lock()
				as.triggeredCount++
				as.mutex.Unlock()

				// Update alert trigger statistics
				if err := as.alertRepo.UpdateAlertTriggerStats(alert.ID); err != nil {
					log.Printf("❌ Failed to update alert trigger stats: %v", err)
				}

				// Broadcast alert to WebSocket clients
				if as.websocketHandler != nil {
					alertData := map[string]interface{}{
						"alert_id":      alert.ID,
						"alert_name":    alert.Name,
						"hostname":      metrics.Hostname,
						"metric_type":   alert.MetricType,
						"condition":     alert.Condition,
						"threshold":     alert.Threshold,
						"current_value": currentValue,
						"severity":      alert.Severity,
						"message":       history.Message,
						"timestamp":     time.Now(),
					}

					// Try to call BroadcastAlert method if it exists
					if broadcaster, ok := as.websocketHandler.(interface{ BroadcastAlert(map[string]interface{}) }); ok {
						broadcaster.BroadcastAlert(alertData)
					}
				}

				// Send notifications
				go as.sendNotifications(alert, history)
			}

			// Update last alert time
			as.mutex.Lock()
			as.lastAlerts[alertKey] = time.Now()
			as.mutex.Unlock()
		} else {
			// Alert condition met but not triggering due to duration/cooldown
			log.Printf("⏳ Alert %d condition met but not triggering (duration: %ds, cooldown check)",
				alert.ID, alert.Duration)

			as.mutex.Lock()
			if _, exists := as.alertDurations[alertKey]; !exists {
				as.alertDurations[alertKey] = time.Now()
				log.Printf("⏰ Started duration tracking for alert %d", alert.ID)
			}
			as.mutex.Unlock()
		}
	} else {
		// Alert condition not met, clear duration tracking
		as.mutex.Lock()
		delete(as.alertDurations, alertKey)
		as.mutex.Unlock()

		// Check if we should auto-resolve any unresolved alerts
		go as.autoResolveAlerts(alert.ID, metrics.Hostname)
	}

	return nil
}

// shouldTriggerAlert determines if an alert should be triggered
func (as *AlertService) shouldTriggerAlert(alert *models.Alert, alertKey string) bool {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	log.Printf("🔍 Checking alert %d (duration: %d) for key: %s", alert.ID, alert.Duration, alertKey)

	// For immediate alerts (duration 0), skip cooldown check and trigger immediately
	if alert.Duration == 0 {
		log.Printf("✅ Alert %d has duration 0, triggering immediately", alert.ID)
		return true
	}

	log.Printf("⏳ Alert %d has duration > 0, checking cooldown", alert.ID)
	// Check cooldown period
	if lastTriggered, exists := as.lastAlerts[alertKey]; exists {
		if time.Since(lastTriggered) < as.config.CooldownPeriod {
			log.Printf("❌ Alert %d in cooldown", alert.ID)
			return false // Still in cooldown
		}
		log.Printf("✅ Alert %d cooldown passed", alert.ID)
	} else {
		log.Printf("ℹ️ Alert %d no previous trigger", alert.ID)
	}

	// Check duration requirement
	if alert.Duration > 0 {
		if firstTriggered, exists := as.alertDurations[alertKey]; exists {
			requiredDuration := time.Duration(alert.Duration) * time.Second
			if time.Since(firstTriggered) < requiredDuration {
				log.Printf("❌ Alert %d duration not met", alert.ID)
				return false // Haven't been triggered long enough
			}
			log.Printf("✅ Alert %d duration met", alert.ID)
		} else {
			log.Printf("❌ Alert %d first time, starting duration tracking", alert.ID)
			return false // First time seeing this condition
		}
	}

	log.Printf("✅ Alert %d can trigger", alert.ID)
	return true
}

// evaluateCondition evaluates alert condition
func (as *AlertService) evaluateCondition(condition string, value, threshold float64) (bool, error) {
	switch strings.TrimSpace(condition) {
	case ">", "gt", "greater_than":
		return value > threshold, nil
	case ">=", "gte", "greater_than_equal":
		return value >= threshold, nil
	case "<", "lt", "less_than":
		return value < threshold, nil
	case "<=", "lte", "less_than_equal":
		return value <= threshold, nil
	case "=", "==", "eq", "equal":
		return value == threshold, nil
	case "!=", "ne", "not_equal":
		return value != threshold, nil
	default:
		return false, fmt.Errorf("unsupported condition: %s", condition)
	}
}

// generateAlertMessage generates a human-readable alert message
func (as *AlertService) generateAlertMessage(alert *models.Alert, value float64, hostname string) string {
	unit := ""
	switch strings.ToLower(alert.MetricType) {
	case "cpu", "memory", "disk":
		unit = "%"
	}

	return fmt.Sprintf("%s %s %.2f%s (threshold: %.2f%s) on %s",
		strings.Title(strings.ToLower(alert.MetricType)),
		alert.Condition,
		value, unit,
		alert.Threshold, unit,
		hostname)
}

// sendNotifications sends email and webhook notifications
func (as *AlertService) sendNotifications(alert *models.Alert, history *models.AlertHistory) {
	// Send email notification
	if alert.EmailEnabled && as.emailSender != nil {
		if err := as.emailSender.SendAlert(alert, history); err != nil {
			log.Printf("❌ Failed to send email alert: %v", err)
		} else {
			log.Printf("📧 Email alert sent successfully")

			// Update history to mark email as sent
			if err := as.alertRepo.CreateAlertHistory(&models.AlertHistory{
				BaseModel:   history.BaseModel,
				AlertID:     history.AlertID,
				Hostname:    history.Hostname,
				MetricValue: history.MetricValue,
				Threshold:   history.Threshold,
				Severity:    history.Severity,
				Message:     history.Message,
				Resolved:    history.Resolved,
				ResolvedAt:  history.ResolvedAt,
				EmailSent:   true,
				WebhookSent: history.WebhookSent,
			}); err != nil {
				log.Printf("❌ Failed to update email sent status: %v", err)
			}
		}
	}

	// Send webhook notification
	if alert.WebhookEnabled && as.webhookSender != nil {
		if err := as.webhookSender.SendAlert(alert, history); err != nil {
			log.Printf("❌ Failed to send webhook alert: %v", err)
		} else {
			log.Printf("🔗 Webhook alert sent successfully")

			// Update history to mark webhook as sent
			if err := as.alertRepo.CreateAlertHistory(&models.AlertHistory{
				BaseModel:   history.BaseModel,
				AlertID:     history.AlertID,
				Hostname:    history.Hostname,
				MetricValue: history.MetricValue,
				Threshold:   history.Threshold,
				Severity:    history.Severity,
				Message:     history.Message,
				Resolved:    history.Resolved,
				ResolvedAt:  history.ResolvedAt,
				EmailSent:   history.EmailSent,
				WebhookSent: true,
			}); err != nil {
				log.Printf("❌ Failed to update webhook sent status: %v", err)
			}
		}
	}
}

// autoResolveAlerts automatically resolves alerts when conditions are no longer met
func (as *AlertService) autoResolveAlerts(alertID uint, hostname string) {
	unresolved, err := as.alertRepo.GetUnresolvedAlerts()
	if err != nil {
		log.Printf("❌ Failed to get unresolved alerts: %v", err)
		return
	}

	for _, history := range unresolved {
		if history.AlertID == alertID && history.Hostname == hostname {
			if err := as.alertRepo.ResolveAlert(history.ID); err != nil {
				log.Printf("❌ Failed to auto-resolve alert: %v", err)
			} else {
				log.Printf("✅ Auto-resolved alert: %s on %s", history.Alert.Name, hostname)
			}
		}
	}
}

// alertCheckingRoutine runs the periodic alert checking
func (as *AlertService) alertCheckingRoutine() {
	log.Printf("🔍 Starting alert checking routine with %v interval", as.config.CheckInterval)

	ticker := time.NewTicker(as.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Alert checking is now primarily driven by metrics collection
			// This routine can be used for cleanup and maintenance tasks
			as.cleanupOldAlertState()

		case <-as.stopChan:
			log.Println("🔍 Alert checking routine stopped")
			return

		case <-as.ctx.Done():
			log.Println("🔍 Alert checking routine cancelled")
			return
		}
	}
}

// cleanupOldAlertState cleans up old alert state data
func (as *AlertService) cleanupOldAlertState() {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)

	// Clean up old duration tracking
	for key, timestamp := range as.alertDurations {
		if timestamp.Before(cutoff) {
			delete(as.alertDurations, key)
		}
	}

	// Clean up old last alert times
	for key, timestamp := range as.lastAlerts {
		if timestamp.Before(cutoff) {
			delete(as.lastAlerts, key)
		}
	}
}

// GetStats returns alert service statistics
func (as *AlertService) GetStats() map[string]interface{} {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	return map[string]interface{}{
		"is_running":       as.isRunning,
		"checked_count":    as.checkedCount,
		"triggered_count":  as.triggeredCount,
		"last_check_time":  as.lastCheckTime,
		"active_durations": len(as.alertDurations),
		"cooldown_alerts":  len(as.lastAlerts),
	}
}

// SetWebSocketHandler sets the WebSocket handler for broadcasting alerts
func (as *AlertService) SetWebSocketHandler(handler interface{}) {
	as.websocketHandler = handler
}
