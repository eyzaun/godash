package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/eyzaun/godash/internal/models"
	"github.com/eyzaun/godash/internal/repository"
	"github.com/eyzaun/godash/internal/services"
)

// AlertHandler handles HTTP requests for alerts
type AlertHandler struct {
	alertRepo     repository.AlertRepository
	alertService  *services.AlertService
	emailSender   services.EmailSender
	webhookSender services.WebhookSender
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(
	alertRepo repository.AlertRepository,
	alertService *services.AlertService,
	emailSender services.EmailSender,
	webhookSender services.WebhookSender,
) *AlertHandler {
	return &AlertHandler{
		alertRepo:     alertRepo,
		alertService:  alertService,
		emailSender:   emailSender,
		webhookSender: webhookSender,
	}
}

// CreateAlertRequest represents the request body for creating alerts
type CreateAlertRequest struct {
	Name            string  `json:"name" binding:"required"`
	MetricType      string  `json:"metric_type" binding:"required"`
	Condition       string  `json:"condition" binding:"required"`
	Threshold       float64 `json:"threshold" binding:"required"`
	Duration        int     `json:"duration"`
	Severity        string  `json:"severity" binding:"required"`
	Description     string  `json:"description"`
	EmailEnabled    bool    `json:"email_enabled"`
	EmailRecipients string  `json:"email_recipients"`
	WebhookEnabled  bool    `json:"webhook_enabled"`
	WebhookURL      string  `json:"webhook_url"`
}

// UpdateAlertRequest represents the request body for updating alerts
type UpdateAlertRequest struct {
	Name            string  `json:"name"`
	MetricType      string  `json:"metric_type"`
	Condition       string  `json:"condition"`
	Threshold       float64 `json:"threshold"`
	Duration        int     `json:"duration"`
	Severity        string  `json:"severity"`
	IsActive        *bool   `json:"is_active"`
	Description     string  `json:"description"`
	EmailEnabled    bool    `json:"email_enabled"`
	EmailRecipients string  `json:"email_recipients"`
	WebhookEnabled  bool    `json:"webhook_enabled"`
	WebhookURL      string  `json:"webhook_url"`
}

// CreateAlert creates a new alert configuration
// @Summary Create alert
// @Description Create a new alert configuration
// @Tags alerts
// @Accept json
// @Produce json
// @Param alert body CreateAlertRequest true "Alert configuration"
// @Success 201 {object} APIResponse{data=models.Alert}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/alerts [post]
func (h *AlertHandler) CreateAlert(c *gin.Context) {
	var req CreateAlertRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if err := h.validateAlertRequest(req.MetricType, req.Condition, req.Severity); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	// Create alert model
	alert := &models.Alert{
		Name:            req.Name,
		MetricType:      req.MetricType,
		Condition:       req.Condition,
		Threshold:       req.Threshold,
		Duration:        req.Duration,
		Severity:        req.Severity,
		IsActive:        true,
		Description:     req.Description,
		EmailEnabled:    req.EmailEnabled,
		EmailRecipients: req.EmailRecipients,
		WebhookEnabled:  req.WebhookEnabled,
		WebhookURL:      req.WebhookURL,
	}

	if err := h.alertRepo.CreateAlert(alert); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to create alert",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    alert,
		Message: "Alert created successfully",
	})
}

// GetAlerts retrieves all alert configurations
// @Summary Get alerts
// @Description Retrieve all alert configurations
// @Tags alerts
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=[]models.Alert}
// @Failure 500 {object} APIResponse
// @Router /api/v1/alerts [get]
func (h *AlertHandler) GetAlerts(c *gin.Context) {
	alerts, err := h.alertRepo.GetAlerts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to retrieve alerts",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    alerts,
	})
}

// GetAlert retrieves a specific alert by ID
// @Summary Get alert by ID
// @Description Retrieve a specific alert configuration by ID
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path int true "Alert ID"
// @Success 200 {object} APIResponse{data=models.Alert}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/alerts/{id} [get]
func (h *AlertHandler) GetAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid alert ID",
			Message: "Alert ID must be a valid number",
		})
		return
	}

	alert, err := h.alertRepo.GetAlertByID(uint(id))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "alert not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   "Failed to retrieve alert",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    alert,
	})
}

// UpdateAlert updates an existing alert configuration
// @Summary Update alert
// @Description Update an existing alert configuration
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path int true "Alert ID"
// @Param alert body UpdateAlertRequest true "Alert configuration updates"
// @Success 200 {object} APIResponse{data=models.Alert}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/alerts/{id} [put]
func (h *AlertHandler) UpdateAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid alert ID",
			Message: "Alert ID must be a valid number",
		})
		return
	}

	// Get existing alert
	alert, err := h.alertRepo.GetAlertByID(uint(id))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "alert not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   "Failed to retrieve alert",
			Message: err.Error(),
		})
		return
	}

	var req UpdateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Update fields
	if req.Name != "" {
		alert.Name = req.Name
	}
	if req.MetricType != "" {
		alert.MetricType = req.MetricType
	}
	if req.Condition != "" {
		alert.Condition = req.Condition
	}
	if req.Threshold != 0 {
		alert.Threshold = req.Threshold
	}
	if req.Duration != 0 {
		alert.Duration = req.Duration
	}
	if req.Severity != "" {
		alert.Severity = req.Severity
	}
	if req.IsActive != nil {
		alert.IsActive = *req.IsActive
	}
	if req.Description != "" {
		alert.Description = req.Description
	}

	alert.EmailEnabled = req.EmailEnabled
	alert.EmailRecipients = req.EmailRecipients
	alert.WebhookEnabled = req.WebhookEnabled
	alert.WebhookURL = req.WebhookURL

	// Validate updated alert
	if err := h.validateAlertRequest(alert.MetricType, alert.Condition, alert.Severity); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	if err := h.alertRepo.UpdateAlert(alert); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to update alert",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    alert,
		Message: "Alert updated successfully",
	})
}

// DeleteAlert deletes an alert configuration
// @Summary Delete alert
// @Description Delete an alert configuration
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path int true "Alert ID"
// @Success 200 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/alerts/{id} [delete]
func (h *AlertHandler) DeleteAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid alert ID",
			Message: "Alert ID must be a valid number",
		})
		return
	}

	if err := h.alertRepo.DeleteAlert(uint(id)); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "alert not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   "Failed to delete alert",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Alert deleted successfully",
	})
}

// GetAlertHistory retrieves alert history with pagination
// @Summary Get alert history
// @Description Retrieve alert history with pagination
// @Tags alerts
// @Accept json
// @Produce json
// @Param limit query int false "Number of records to return" default(50)
// @Param page query int false "Page number" default(1)
// @Success 200 {object} PaginatedResponse{data=[]models.AlertHistory}
// @Failure 500 {object} APIResponse
// @Router /api/v1/alerts/history [get]
func (h *AlertHandler) GetAlertHistory(c *gin.Context) {
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

	history, err := h.alertRepo.GetAlertHistory(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to retrieve alert history",
			Message: err.Error(),
		})
		return
	}

	// For simplicity, using length as total count
	// In production, you might want a separate count query
	total := int64(len(history))
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    history,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// TestAlert tests an alert configuration
// @Summary Test alert
// @Description Send test notifications for an alert configuration
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path int true "Alert ID"
// @Success 200 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/alerts/{id}/test [post]
func (h *AlertHandler) TestAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid alert ID",
			Message: "Alert ID must be a valid number",
		})
		return
	}

	alert, err := h.alertRepo.GetAlertByID(uint(id))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "alert not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   "Failed to retrieve alert",
			Message: err.Error(),
		})
		return
	}

	// Create test alert history
	testHistory := &models.AlertHistory{
		AlertID:     alert.ID,
		Hostname:    "test-hostname",
		MetricValue: alert.Threshold + 10, // Simulate exceeded threshold
		Threshold:   alert.Threshold,
		Severity:    alert.Severity,
		Message:     "This is a test alert notification",
		Resolved:    false,
	}

	var results []string

	// Test email notification
	if alert.EmailEnabled && h.emailSender != nil {
		if err := h.emailSender.SendAlert(alert, testHistory); err != nil {
			results = append(results, fmt.Sprintf("Email test failed: %v", err))
		} else {
			results = append(results, "Email test successful")
		}
	}

	// Test webhook notification
	if alert.WebhookEnabled && h.webhookSender != nil {
		if err := h.webhookSender.SendAlert(alert, testHistory); err != nil {
			results = append(results, fmt.Sprintf("Webhook test failed: %v", err))
		} else {
			results = append(results, "Webhook test successful")
		}
	}

	if len(results) == 0 {
		results = append(results, "No notifications configured for this alert")
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"test_results": results,
			"alert_name":   alert.Name,
		},
		Message: "Alert test completed",
	})
}

// GetAlertStats returns alert statistics
// @Summary Get alert statistics
// @Description Get alert system statistics
// @Tags alerts
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/alerts/stats [get]
func (h *AlertHandler) GetAlertStats(c *gin.Context) {
	stats, err := h.alertRepo.GetAlertStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to get alert statistics",
			Message: err.Error(),
		})
		return
	}

	// Add alert service stats if available
	if h.alertService != nil {
		serviceStats := h.alertService.GetStats()
		for k, v := range serviceStats {
			stats[k] = v
		}
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// ResolveAlert manually resolves an alert
// @Summary Resolve alert
// @Description Manually resolve an alert history entry
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path int true "Alert History ID"
// @Success 200 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/alerts/history/{id}/resolve [post]
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid alert history ID",
			Message: "Alert history ID must be a valid number",
		})
		return
	}

	if err := h.alertRepo.ResolveAlert(uint(id)); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "alert history not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   "Failed to resolve alert",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Alert resolved successfully",
	})
}

// validateAlertRequest validates alert request parameters
func (h *AlertHandler) validateAlertRequest(metricType, condition, severity string) error {
	// Validate metric type
	validMetricTypes := []string{"cpu", "memory", "disk", "load_avg_1", "load_avg_5", "load_avg_15"}
	if !contains(validMetricTypes, strings.ToLower(metricType)) {
		return fmt.Errorf("invalid metric type: %s. Valid types: %s", metricType, strings.Join(validMetricTypes, ", "))
	}

	// Validate condition
	validConditions := []string{">", ">=", "<", "<=", "=", "==", "!=", "gt", "gte", "lt", "lte", "eq", "ne"}
	if !contains(validConditions, strings.ToLower(condition)) {
		return fmt.Errorf("invalid condition: %s. Valid conditions: %s", condition, strings.Join(validConditions, ", "))
	}

	// Validate severity
	validSeverities := []string{"info", "warning", "critical"}
	if !contains(validSeverities, strings.ToLower(severity)) {
		return fmt.Errorf("invalid severity: %s. Valid severities: %s", severity, strings.Join(validSeverities, ", "))
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
