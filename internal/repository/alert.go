package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/eyzaun/godash/internal/models"
)

// AlertRepository interface defines methods for alert data access
type AlertRepository interface {
	// Alert configuration operations
	CreateAlert(alert *models.Alert) error
	GetAlertByID(id uint) (*models.Alert, error)
	GetAlerts() ([]*models.Alert, error)
	GetActiveAlerts() ([]*models.Alert, error)
	GetAlertsByMetricType(metricType string) ([]*models.Alert, error)
	UpdateAlert(alert *models.Alert) error
	UpdateAlertTriggerStats(alertID uint) error
	DeleteAlert(id uint) error

	// Alert history operations
	CreateAlertHistory(history *models.AlertHistory) error
	GetAlertHistory(limit, offset int) ([]*models.AlertHistory, error)
	GetAlertHistoryByID(alertID uint, limit, offset int) ([]*models.AlertHistory, error)
	GetUnresolvedAlerts() ([]*models.AlertHistory, error)
	ResolveAlert(historyID uint) error
	GetRecentAlerts(since time.Time) ([]*models.AlertHistory, error)

	// Statistics operations
	GetAlertStats() (map[string]interface{}, error)
	GetTriggeredAlertsCount(since time.Time) (int64, error)
}

// alertRepository implements AlertRepository interface
type alertRepository struct {
	db *gorm.DB
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *gorm.DB) AlertRepository {
	return &alertRepository{
		db: db,
	}
}

// CreateAlert creates a new alert configuration
func (r *alertRepository) CreateAlert(alert *models.Alert) error {
	if err := r.db.Create(alert).Error; err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}
	return nil
}

// GetAlertByID retrieves an alert by its ID
func (r *alertRepository) GetAlertByID(id uint) (*models.Alert, error) {
	var alert models.Alert
	if err := r.db.First(&alert, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("alert not found")
		}
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}
	return &alert, nil
}

// GetAlerts retrieves all alert configurations
func (r *alertRepository) GetAlerts() ([]*models.Alert, error) {
	var alerts []*models.Alert
	if err := r.db.Order("created_at DESC").Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("failed to get alerts: %w", err)
	}
	return alerts, nil
}

// GetActiveAlerts retrieves all active alert configurations
func (r *alertRepository) GetActiveAlerts() ([]*models.Alert, error) {
	var alerts []*models.Alert
	if err := r.db.Where("is_active = ?", true).
		Order("created_at DESC").
		Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("failed to get active alerts: %w", err)
	}
	return alerts, nil
}

// GetAlertsByMetricType retrieves alerts for a specific metric type
func (r *alertRepository) GetAlertsByMetricType(metricType string) ([]*models.Alert, error) {
	var alerts []*models.Alert
	if err := r.db.Where("metric_type = ? AND is_active = ?", metricType, true).
		Order("created_at DESC").
		Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("failed to get alerts by metric type: %w", err)
	}
	return alerts, nil
}

// UpdateAlert updates an existing alert configuration
func (r *alertRepository) UpdateAlert(alert *models.Alert) error {
	if err := r.db.Save(alert).Error; err != nil {
		return fmt.Errorf("failed to update alert: %w", err)
	}
	return nil
}

// DeleteAlert deletes an alert configuration
func (r *alertRepository) DeleteAlert(id uint) error {
	// Start a transaction to ensure atomicity
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// First, delete all related alert history records
	if err := tx.Where("alert_id = ?", id).Delete(&models.AlertHistory{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete alert history: %w", err)
	}

	// Then delete the alert itself
	result := tx.Delete(&models.Alert{}, id)
	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete alert: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("alert not found")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreateAlertHistory creates a new alert history entry
func (r *alertRepository) CreateAlertHistory(history *models.AlertHistory) error {
	if err := r.db.Create(history).Error; err != nil {
		return fmt.Errorf("failed to create alert history: %w", err)
	}
	return nil
}

// GetAlertHistory retrieves alert history with pagination
func (r *alertRepository) GetAlertHistory(limit, offset int) ([]*models.AlertHistory, error) {
	var history []*models.AlertHistory

	query := r.db.Preload("Alert").Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&history).Error; err != nil {
		return nil, fmt.Errorf("failed to get alert history: %w", err)
	}

	return history, nil
}

// GetAlertHistoryByID retrieves alert history for a specific alert
func (r *alertRepository) GetAlertHistoryByID(alertID uint, limit, offset int) ([]*models.AlertHistory, error) {
	var history []*models.AlertHistory

	query := r.db.Where("alert_id = ?", alertID).
		Preload("Alert").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&history).Error; err != nil {
		return nil, fmt.Errorf("failed to get alert history by ID: %w", err)
	}

	return history, nil
}

// GetUnresolvedAlerts retrieves all unresolved alert entries
func (r *alertRepository) GetUnresolvedAlerts() ([]*models.AlertHistory, error) {
	var history []*models.AlertHistory
	if err := r.db.Where("resolved = ?", false).
		Preload("Alert").
		Order("created_at DESC").
		Find(&history).Error; err != nil {
		return nil, fmt.Errorf("failed to get unresolved alerts: %w", err)
	}
	return history, nil
}

// ResolveAlert marks an alert history entry as resolved
func (r *alertRepository) ResolveAlert(historyID uint) error {
	result := r.db.Model(&models.AlertHistory{}).
		Where("id = ?", historyID).
		Updates(map[string]interface{}{
			"resolved":    true,
			"resolved_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to resolve alert: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("alert history not found")
	}

	return nil
}

// GetRecentAlerts retrieves alerts triggered since a specific time
func (r *alertRepository) GetRecentAlerts(since time.Time) ([]*models.AlertHistory, error) {
	var history []*models.AlertHistory
	if err := r.db.Where("created_at >= ?", since).
		Preload("Alert").
		Order("created_at DESC").
		Find(&history).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent alerts: %w", err)
	}
	return history, nil
}

// GetAlertStats returns statistics about alerts
func (r *alertRepository) GetAlertStats() (map[string]interface{}, error) {
	var stats struct {
		TotalAlerts      int64 `gorm:"column:total_alerts"`
		ActiveAlerts     int64 `gorm:"column:active_alerts"`
		TriggeredToday   int64 `gorm:"column:triggered_today"`
		UnresolvedAlerts int64 `gorm:"column:unresolved_alerts"`
		ResolvedToday    int64 `gorm:"column:resolved_today"`
	}

	today := time.Now().Truncate(24 * time.Hour)

	// Get alert configuration stats
	if err := r.db.Model(&models.Alert{}).Select("count(*) as total_alerts").Scan(&stats.TotalAlerts).Error; err != nil {
		return nil, fmt.Errorf("failed to get total alerts count: %w", err)
	}

	if err := r.db.Model(&models.Alert{}).Where("is_active = ?", true).Select("count(*) as active_alerts").Scan(&stats.ActiveAlerts).Error; err != nil {
		return nil, fmt.Errorf("failed to get active alerts count: %w", err)
	}

	// Get alert history stats
	if err := r.db.Model(&models.AlertHistory{}).Where("created_at >= ?", today).Select("count(*) as triggered_today").Scan(&stats.TriggeredToday).Error; err != nil {
		return nil, fmt.Errorf("failed to get triggered alerts count: %w", err)
	}

	if err := r.db.Model(&models.AlertHistory{}).Where("resolved = ?", false).Select("count(*) as unresolved_alerts").Scan(&stats.UnresolvedAlerts).Error; err != nil {
		return nil, fmt.Errorf("failed to get unresolved alerts count: %w", err)
	}

	if err := r.db.Model(&models.AlertHistory{}).Where("resolved = ? AND resolved_at >= ?", true, today).Select("count(*) as resolved_today").Scan(&stats.ResolvedToday).Error; err != nil {
		return nil, fmt.Errorf("failed to get resolved alerts count: %w", err)
	}

	return map[string]interface{}{
		"total_alerts":      stats.TotalAlerts,
		"active_alerts":     stats.ActiveAlerts,
		"triggered_today":   stats.TriggeredToday,
		"unresolved_alerts": stats.UnresolvedAlerts,
		"resolved_today":    stats.ResolvedToday,
		"timestamp":         time.Now(),
	}, nil
}

// GetTriggeredAlertsCount returns count of alerts triggered since specific time
func (r *alertRepository) GetTriggeredAlertsCount(since time.Time) (int64, error) {
	var count int64
	if err := r.db.Model(&models.AlertHistory{}).Where("created_at >= ?", since).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to get triggered alerts count: %w", err)
	}
	return count, nil
}

// UpdateAlertTriggerStats updates the trigger count and last triggered time for an alert
func (r *alertRepository) UpdateAlertTriggerStats(alertID uint) error {
	// First get the current alert
	var alert models.Alert
	if err := r.db.First(&alert, alertID).Error; err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}

	// Update the fields
	alert.TriggeredCount++
	alert.LastTriggered = time.Now()

	// Save the updated alert
	if err := r.db.Save(&alert).Error; err != nil {
		return fmt.Errorf("failed to update alert trigger stats: %w", err)
	}

	return nil
}
