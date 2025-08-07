package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eyzaun/godash/internal/config"
	"github.com/eyzaun/godash/internal/models"
)

// WebhookSender interface for sending webhook notifications
type WebhookSender interface {
	SendAlert(alert *models.Alert, history *models.AlertHistory) error
	SendTestWebhook(url string, payload map[string]interface{}) error
	ValidateConfiguration() error
}

// HTTPWebhookSender implements WebhookSender using HTTP client
type HTTPWebhookSender struct {
	client *http.Client
	config *config.WebhookConfig
}

// NewWebhookSender creates a new webhook sender
func NewWebhookSender(webhookConfig *config.WebhookConfig) WebhookSender {
	client := &http.Client{
		Timeout: webhookConfig.DefaultTimeout,
	}

	return &HTTPWebhookSender{
		client: client,
		config: webhookConfig,
	}
}

// WebhookPayload represents the payload sent to webhooks
type WebhookPayload struct {
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Alert     AlertWebhookData  `json:"alert"`
	System    SystemWebhookData `json:"system"`
}

// AlertWebhookData represents alert data in webhook payload
type AlertWebhookData struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	MetricType   string    `json:"metric_type"`
	Condition    string    `json:"condition"`
	Threshold    float64   `json:"threshold"`
	CurrentValue float64   `json:"current_value"`
	Severity     string    `json:"severity"`
	Message      string    `json:"message"`
	Hostname     string    `json:"hostname"`
	TriggeredAt  time.Time `json:"triggered_at"`
}

// SystemWebhookData represents system data in webhook payload
type SystemWebhookData struct {
	Hostname     string `json:"hostname"`
	DashboardURL string `json:"dashboard_url"`
	Platform     string `json:"platform"`
}

// SlackPayload represents Slack-specific webhook format
type SlackPayload struct {
	Text        string            `json:"text"`
	Username    string            `json:"username"`
	IconEmoji   string            `json:"icon_emoji"`
	Channel     string            `json:"channel,omitempty"`
	Attachments []SlackAttachment `json:"attachments"`
}

// SlackAttachment represents Slack message attachment
type SlackAttachment struct {
	Color     string       `json:"color"`
	Title     string       `json:"title"`
	Text      string       `json:"text"`
	Fields    []SlackField `json:"fields"`
	Timestamp int64        `json:"ts"`
}

// SlackField represents Slack attachment field
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// DiscordPayload represents Discord-specific webhook format
type DiscordPayload struct {
	Username  string         `json:"username"`
	AvatarURL string         `json:"avatar_url"`
	Content   string         `json:"content"`
	Embeds    []DiscordEmbed `json:"embeds"`
}

// DiscordEmbed represents Discord message embed
type DiscordEmbed struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Color       int                 `json:"color"`
	Fields      []DiscordEmbedField `json:"fields"`
	Footer      DiscordEmbedFooter  `json:"footer"`
	Timestamp   time.Time           `json:"timestamp"`
}

// DiscordEmbedField represents Discord embed field
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// DiscordEmbedFooter represents Discord embed footer
type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

// SendAlert sends webhook notification for an alert
func (w *HTTPWebhookSender) SendAlert(alert *models.Alert, history *models.AlertHistory) error {
	if w.config == nil {
		return fmt.Errorf("webhook configuration is not available")
	}

	if !alert.WebhookEnabled || alert.WebhookURL == "" {
		return fmt.Errorf("webhook not enabled or URL not configured for alert: %s", alert.Name)
	}

	// Detect webhook type and create appropriate payload
	payload, err := w.createPayload(alert, history)
	if err != nil {
		return fmt.Errorf("failed to create webhook payload: %w", err)
	}

	// Send webhook with retry mechanism
	return w.sendWithRetry(alert.WebhookURL, payload)
}

// SendTestWebhook sends a test webhook
func (w *HTTPWebhookSender) SendTestWebhook(url string, payload map[string]interface{}) error {
	if w.config == nil {
		return fmt.Errorf("webhook configuration is not available")
	}

	testPayload := map[string]interface{}{
		"type":      "test",
		"timestamp": time.Now(),
		"message":   "This is a test webhook from GoDash System Monitor",
		"system": map[string]interface{}{
			"hostname":      "test-host",
			"dashboard_url": "http://localhost:8080",
		},
	}

	// Merge with provided payload
	for k, v := range payload {
		testPayload[k] = v
	}

	return w.sendWebhook(url, testPayload)
}

// ValidateConfiguration validates webhook configuration
func (w *HTTPWebhookSender) ValidateConfiguration() error {
	if w.config == nil {
		return fmt.Errorf("webhook configuration is nil")
	}

	if w.config.DefaultTimeout <= 0 {
		return fmt.Errorf("webhook timeout must be positive")
	}

	if w.config.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	return nil
}

// createPayload creates webhook payload based on URL type
func (w *HTTPWebhookSender) createPayload(alert *models.Alert, history *models.AlertHistory) (interface{}, error) {
	url := strings.ToLower(alert.WebhookURL)

	// Detect webhook type
	switch {
	case strings.Contains(url, "slack.com"):
		return w.createSlackPayload(alert, history), nil
	case strings.Contains(url, "discord.com") || strings.Contains(url, "discordapp.com"):
		return w.createDiscordPayload(alert, history), nil
	default:
		return w.createGenericPayload(alert, history), nil
	}
}

// createGenericPayload creates a generic webhook payload
func (w *HTTPWebhookSender) createGenericPayload(alert *models.Alert, history *models.AlertHistory) WebhookPayload {
	return WebhookPayload{
		Type:      "alert_triggered",
		Timestamp: time.Now(),
		Alert: AlertWebhookData{
			ID:           alert.ID,
			Name:         alert.Name,
			MetricType:   alert.MetricType,
			Condition:    alert.Condition,
			Threshold:    alert.Threshold,
			CurrentValue: history.MetricValue,
			Severity:     history.Severity,
			Message:      history.Message,
			Hostname:     history.Hostname,
			TriggeredAt:  history.CreatedAt,
		},
		System: SystemWebhookData{
			Hostname:     history.Hostname,
			DashboardURL: "http://localhost:8080", // In production, use config
			Platform:     "GoDash",
		},
	}
}

// createSlackPayload creates Slack-specific webhook payload
func (w *HTTPWebhookSender) createSlackPayload(alert *models.Alert, history *models.AlertHistory) SlackPayload {
	color := "#28a745" // green
	emoji := ":white_check_mark:"

	switch strings.ToLower(history.Severity) {
	case "warning":
		color = "#ffc107"
		emoji = ":warning:"
	case "critical":
		color = "#dc3545"
		emoji = ":rotating_light:"
	}

	unit := "%"
	if !strings.Contains(strings.ToLower(alert.MetricType), "cpu") &&
		!strings.Contains(strings.ToLower(alert.MetricType), "memory") &&
		!strings.Contains(strings.ToLower(alert.MetricType), "disk") {
		unit = ""
	}

	return SlackPayload{
		Text:      fmt.Sprintf("%s *%s Alert*: %s", emoji, strings.Title(history.Severity), alert.Name),
		Username:  "GoDash Monitor",
		IconEmoji: ":chart_with_upwards_trend:",
		Attachments: []SlackAttachment{
			{
				Color: color,
				Title: fmt.Sprintf("%s on %s", strings.Title(alert.MetricType), history.Hostname),
				Text:  history.Message,
				Fields: []SlackField{
					{Title: "Current Value", Value: fmt.Sprintf("%.2f%s", history.MetricValue, unit), Short: true},
					{Title: "Threshold", Value: fmt.Sprintf("%.2f%s", alert.Threshold, unit), Short: true},
					{Title: "Condition", Value: alert.Condition, Short: true},
					{Title: "Hostname", Value: history.Hostname, Short: true},
				},
				Timestamp: history.CreatedAt.Unix(),
			},
		},
	}
}

// createDiscordPayload creates Discord-specific webhook payload
func (w *HTTPWebhookSender) createDiscordPayload(alert *models.Alert, history *models.AlertHistory) DiscordPayload {
	color := 0x28a745 // green

	switch strings.ToLower(history.Severity) {
	case "warning":
		color = 0xffc107 // yellow
	case "critical":
		color = 0xdc3545 // red
	}

	unit := "%"
	if !strings.Contains(strings.ToLower(alert.MetricType), "cpu") &&
		!strings.Contains(strings.ToLower(alert.MetricType), "memory") &&
		!strings.Contains(strings.ToLower(alert.MetricType), "disk") {
		unit = ""
	}

	return DiscordPayload{
		Username:  "GoDash Monitor",
		AvatarURL: "https://cdn.discordapp.com/embed/avatars/0.png",
		Content:   fmt.Sprintf("🚨 **%s Alert**: %s", strings.Title(history.Severity), alert.Name),
		Embeds: []DiscordEmbed{
			{
				Title:       fmt.Sprintf("%s Alert on %s", strings.Title(alert.MetricType), history.Hostname),
				Description: history.Message,
				Color:       color,
				Fields: []DiscordEmbedField{
					{Name: "Current Value", Value: fmt.Sprintf("%.2f%s", history.MetricValue, unit), Inline: true},
					{Name: "Threshold", Value: fmt.Sprintf("%.2f%s", alert.Threshold, unit), Inline: true},
					{Name: "Condition", Value: alert.Condition, Inline: true},
					{Name: "Hostname", Value: history.Hostname, Inline: true},
					{Name: "Severity", Value: strings.Title(history.Severity), Inline: true},
				},
				Footer: DiscordEmbedFooter{
					Text: "GoDash System Monitor",
				},
				Timestamp: history.CreatedAt,
			},
		},
	}
}

// sendWithRetry sends webhook with retry mechanism
func (w *HTTPWebhookSender) sendWithRetry(url string, payload interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			time.Sleep(w.config.RetryDelay * time.Duration(attempt))
		}

		err := w.sendWebhook(url, payload)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Don't retry for client errors (4xx)
		if httpErr, ok := err.(*HTTPError); ok && httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 {
			break
		}
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", w.config.MaxRetries+1, lastErr)
}

// sendWebhook sends a single webhook request
func (w *HTTPWebhookSender) sendWebhook(url string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoDash-Monitor/1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("webhook returned status %d", resp.StatusCode),
		}
	}

	return nil
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}
