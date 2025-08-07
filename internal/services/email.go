package services

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"

	"github.com/eyzaun/godash/internal/config"
	"github.com/eyzaun/godash/internal/models"
)

// EmailSender interface for sending alert emails
type EmailSender interface {
	SendAlert(alert *models.Alert, history *models.AlertHistory) error
	SendTestEmail(to, subject, message string) error
	ValidateConfiguration() error
}

// SMTPEmailSender implements EmailSender using SMTP
type SMTPEmailSender struct {
	config *config.EmailConfig
}

// NewEmailSender creates a new email sender
func NewEmailSender(emailConfig *config.EmailConfig) EmailSender {
	return &SMTPEmailSender{
		config: emailConfig,
	}
}

// AlertEmailData holds template data for alert emails
type AlertEmailData struct {
	AlertName    string
	Hostname     string
	MetricType   string
	CurrentValue float64
	Threshold    float64
	Severity     string
	Message      string
	Timestamp    time.Time
	DashboardURL string
	Condition    string
}

// SendAlert sends an alert email notification
func (s *SMTPEmailSender) SendAlert(alert *models.Alert, history *models.AlertHistory) error {
	if s.config == nil {
		return fmt.Errorf("email configuration is not available")
	}

	if !s.config.Enabled {
		return fmt.Errorf("email sending is disabled")
	}

	if alert.EmailRecipients == "" {
		return fmt.Errorf("no email recipients configured for alert: %s", alert.Name)
	}

	// Parse recipients
	recipients := strings.Split(alert.EmailRecipients, ",")
	for i, recipient := range recipients {
		recipients[i] = strings.TrimSpace(recipient)
	}

	// Prepare template data
	templateData := AlertEmailData{
		AlertName:    alert.Name,
		Hostname:     history.Hostname,
		MetricType:   alert.MetricType,
		CurrentValue: history.MetricValue,
		Threshold:    history.Threshold,
		Severity:     history.Severity,
		Message:      history.Message,
		Timestamp:    history.CreatedAt,
		DashboardURL: "http://localhost:8080", // In production, use config
		Condition:    alert.Condition,
	}

	// Generate email subject
	subject := fmt.Sprintf("[GoDash Alert] %s - %s %s on %s",
		strings.ToUpper(history.Severity),
		strings.ToUpper(alert.MetricType),
		alert.Condition,
		history.Hostname)

	// Generate email body
	htmlBody, err := s.generateHTMLBody(templateData)
	if err != nil {
		return fmt.Errorf("failed to generate HTML email body: %w", err)
	}

	textBody := s.generateTextBody(templateData)

	// Send email to all recipients
	for _, recipient := range recipients {
		if recipient == "" {
			continue
		}

		if err := s.sendEmail(recipient, subject, htmlBody, textBody); err != nil {
			return fmt.Errorf("failed to send email to %s: %w", recipient, err)
		}
	}

	return nil
}

// SendTestEmail sends a test email
func (s *SMTPEmailSender) SendTestEmail(to, subject, message string) error {
	if s.config == nil {
		return fmt.Errorf("email configuration is not available")
	}

	if !s.config.Enabled {
		return fmt.Errorf("email sending is disabled")
	}

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
</head>
<body>
    <div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
        <h2 style="color: #2c5aa0;">GoDash Test Email</h2>
        <p>%s</p>
        <p style="color: #666; font-size: 12px;">Sent at: %s</p>
        <hr style="border: 1px solid #eee;">
        <p style="color: #999; font-size: 11px;">This is a test email from GoDash System Monitor</p>
    </div>
</body>
</html>`, subject, message, time.Now().Format("2006-01-02 15:04:05"))

	textBody := fmt.Sprintf("GoDash Test Email\n\n%s\n\nSent at: %s", message, time.Now().Format("2006-01-02 15:04:05"))

	return s.sendEmail(to, subject, htmlBody, textBody)
}

// ValidateConfiguration validates email configuration
func (s *SMTPEmailSender) ValidateConfiguration() error {
	if s.config == nil {
		return fmt.Errorf("email configuration is nil")
	}

	if s.config.SMTPHost == "" {
		return fmt.Errorf("SMTP host is required")
	}

	if s.config.SMTPPort == 0 {
		return fmt.Errorf("SMTP port is required")
	}

	if s.config.FromEmail == "" {
		return fmt.Errorf("from email is required")
	}

	// Test connection
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	var auth smtp.Auth
	if s.config.SMTPUsername != "" && s.config.SMTPPassword != "" {
		auth = smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	return nil
}

// sendEmail sends an email using SMTP
func (s *SMTPEmailSender) sendEmail(to, subject, htmlBody, textBody string) error {
	// Setup authentication
	var auth smtp.Auth
	if s.config.SMTPUsername != "" && s.config.SMTPPassword != "" {
		auth = smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	}

	// Build email message
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "multipart/alternative; boundary=boundary123"

	// Build message
	var message bytes.Buffer

	// Headers
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")

	// Text part
	message.WriteString("--boundary123\r\n")
	message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	message.WriteString("\r\n")
	message.WriteString(textBody)
	message.WriteString("\r\n")

	// HTML part
	message.WriteString("--boundary123\r\n")
	message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	message.WriteString("\r\n")
	message.WriteString(htmlBody)
	message.WriteString("\r\n")

	// End boundary
	message.WriteString("--boundary123--\r\n")

	// Send email
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)
	return smtp.SendMail(addr, auth, s.config.FromEmail, []string{to}, message.Bytes())
}

// generateHTMLBody generates HTML email body from template
func (s *SMTPEmailSender) generateHTMLBody(data AlertEmailData) (string, error) {
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>GoDash Alert - {{.AlertName}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background-color: white; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); overflow: hidden; }
        .header { background-color: {{.SeverityColor}}; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .metric-box { background-color: #f8f9fa; border: 1px solid #dee2e6; border-radius: 4px; padding: 15px; margin: 15px 0; }
        .metric-value { font-size: 24px; font-weight: bold; color: {{.SeverityColor}}; }
        .footer { background-color: #f8f9fa; padding: 15px; text-align: center; font-size: 12px; color: #666; }
        .button { display: inline-block; padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 4px; margin: 10px 0; }
        table { width: 100%; border-collapse: collapse; margin: 15px 0; }
        td { padding: 8px; border-bottom: 1px solid #eee; }
        .label { font-weight: bold; width: 120px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚨 System Alert</h1>
            <p>{{.Severity}} Alert Triggered</p>
        </div>
        <div class="content">
            <h2>{{.AlertName}}</h2>
            <p>{{.Message}}</p>
            
            <div class="metric-box">
                <div class="metric-value">{{.CurrentValue}}{{.MetricUnit}}</div>
                <p>Current {{.MetricType}} usage (Threshold: {{.Threshold}}{{.MetricUnit}})</p>
            </div>

            <table>
                <tr><td class="label">Host:</td><td>{{.Hostname}}</td></tr>
                <tr><td class="label">Metric:</td><td>{{.MetricType}}</td></tr>
                <tr><td class="label">Condition:</td><td>{{.Condition}}</td></tr>
                <tr><td class="label">Severity:</td><td>{{.Severity}}</td></tr>
                <tr><td class="label">Time:</td><td>{{.Timestamp.Format "2006-01-02 15:04:05"}}</td></tr>
            </table>

            <div style="text-align: center;">
                <a href="{{.DashboardURL}}" class="button">View Dashboard</a>
            </div>
        </div>
        <div class="footer">
            <p>This alert was generated by GoDash System Monitor</p>
            <p>Timestamp: {{.Timestamp.Format "2006-01-02 15:04:05 UTC"}}</p>
        </div>
    </div>
</body>
</html>`

	// Add template functions
	funcMap := template.FuncMap{
		"SeverityColor": func() string {
			switch strings.ToLower(data.Severity) {
			case "critical":
				return "#dc3545"
			case "warning":
				return "#ffc107"
			default:
				return "#28a745"
			}
		},
		"MetricUnit": func() string {
			switch strings.ToLower(data.MetricType) {
			case "cpu", "memory", "disk":
				return "%"
			default:
				return ""
			}
		},
	}

	tmpl, err := template.New("alert").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return buf.String(), nil
}

// generateTextBody generates plain text email body
func (s *SMTPEmailSender) generateTextBody(data AlertEmailData) string {
	unit := "%"
	if !strings.Contains(strings.ToLower(data.MetricType), "cpu") &&
		!strings.Contains(strings.ToLower(data.MetricType), "memory") &&
		!strings.Contains(strings.ToLower(data.MetricType), "disk") {
		unit = ""
	}

	return fmt.Sprintf(`GoDash System Alert

Alert: %s
Severity: %s

%s

Host: %s
Metric: %s  
Current Value: %.2f%s
Threshold: %.2f%s
Condition: %s
Time: %s

Dashboard: %s

---
This alert was generated by GoDash System Monitor
Timestamp: %s UTC`,
		data.AlertName,
		data.Severity,
		data.Message,
		data.Hostname,
		data.MetricType,
		data.CurrentValue, unit,
		data.Threshold, unit,
		data.Condition,
		data.Timestamp.Format("2006-01-02 15:04:05"),
		data.DashboardURL,
		data.Timestamp.Format("2006-01-02 15:04:05"))
}
