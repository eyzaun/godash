package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `json:"server" yaml:"server"`
	Database DatabaseConfig `json:"database" yaml:"database"`
	Metrics  MetricsConfig  `json:"metrics" yaml:"metrics"`
	Alerts   *AlertConfig   `json:"alerts" yaml:"alerts"`
	Email    *EmailConfig   `json:"email" yaml:"email"`
	Webhook  *WebhookConfig `json:"webhook" yaml:"webhook"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string        `json:"host" yaml:"host"`
	Port         int           `json:"port" yaml:"port"`
	Mode         string        `json:"mode" yaml:"mode"` // debug, release, test
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	AutoOpen     bool          `json:"auto_open" yaml:"auto_open"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	// Driver can be "postgres" or "sqlite"
	Driver   string `json:"driver" yaml:"driver"`
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
	Name     string `json:"name" yaml:"name"`
	// For SQLite, use SQLitePath (file path). If empty, Name may be used.
	SQLitePath      string        `json:"sqlite_path" yaml:"sqlite_path"`
	SSLMode         string        `json:"ssl_mode" yaml:"ssl_mode"`
	Timezone        string        `json:"timezone" yaml:"timezone"`
	MaxOpenConns    int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	LogLevel        string        `json:"log_level" yaml:"log_level"`
}

// MetricsConfig holds metrics collection configuration
type MetricsConfig struct {
	CollectionInterval time.Duration `json:"collection_interval" yaml:"collection_interval"`
	RetentionDays      int           `json:"retention_days" yaml:"retention_days"`
	EnableCPU          bool          `json:"enable_cpu" yaml:"enable_cpu"`
	EnableMemory       bool          `json:"enable_memory" yaml:"enable_memory"`
	EnableDisk         bool          `json:"enable_disk" yaml:"enable_disk"`
	EnableNetwork      bool          `json:"enable_network" yaml:"enable_network"`
	EnableProcesses    bool          `json:"enable_processes" yaml:"enable_processes"`
	BufferSize         int           `json:"buffer_size" yaml:"buffer_size"`
}

// AlertConfig holds alert system configuration
type AlertConfig struct {
	EnableAlerts   bool          `json:"enable_alerts" yaml:"enable_alerts"`
	CheckInterval  time.Duration `json:"check_interval" yaml:"check_interval"`
	CooldownPeriod time.Duration `json:"cooldown_period" yaml:"cooldown_period"`
}

// EmailConfig holds email notification configuration
type EmailConfig struct {
	Enabled      bool   `json:"enabled" yaml:"enabled"`
	SMTPHost     string `json:"smtp_host" yaml:"smtp_host"`
	SMTPPort     int    `json:"smtp_port" yaml:"smtp_port"`
	SMTPUsername string `json:"smtp_username" yaml:"smtp_username"`
	SMTPPassword string `json:"smtp_password" yaml:"smtp_password"`
	FromEmail    string `json:"from_email" yaml:"from_email"`
	FromName     string `json:"from_name" yaml:"from_name"`
	UseTLS       bool   `json:"use_tls" yaml:"use_tls"`
}

// WebhookConfig holds webhook notification configuration
type WebhookConfig struct {
	DefaultTimeout time.Duration `json:"default_timeout" yaml:"default_timeout"`
	MaxRetries     int           `json:"max_retries" yaml:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay" yaml:"retry_delay"`
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		Server:   loadServerConfig(),
		Database: loadDatabaseConfig(),
		Metrics:  loadMetricsConfig(),
		Alerts:   loadAlertConfig(),
		Email:    loadEmailConfig(),
		Webhook:  loadWebhookConfig(),
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// loadServerConfig loads server configuration from environment variables
func loadServerConfig() ServerConfig {
	return ServerConfig{
		Host:         getEnvString("SERVER_HOST", "127.0.0.1"),
		Port:         getEnvInt("SERVER_PORT", 8080),
		Mode:         getEnvString("SERVER_MODE", "release"),
		ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		AutoOpen:     getEnvBool("SERVER_AUTO_OPEN", true),
	}
}

// loadDatabaseConfig loads database configuration from environment variables
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Driver:          getEnvString("DB_DRIVER", "sqlite"),
		Host:            getEnvString("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5433),
		User:            getEnvString("DB_USER", "godash"),
		Password:        getEnvString("DB_PASSWORD", "password"),
		Name:            getEnvString("DB_NAME", "godash"), // ignored in sqlite
		SQLitePath:      getEnvString("SQLITE_PATH", "godash.db"),
		SSLMode:         getEnvString("DB_SSL_MODE", "disable"),
		Timezone:        getEnvString("DB_TIMEZONE", "UTC"),
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		LogLevel:        getEnvString("DB_LOG_LEVEL", "warn"),
	}
}

// loadMetricsConfig loads metrics configuration from environment variables
func loadMetricsConfig() MetricsConfig {
	return MetricsConfig{
		CollectionInterval: getEnvDuration("METRICS_COLLECTION_INTERVAL", 30*time.Second),
		RetentionDays:      getEnvInt("METRICS_RETENTION_DAYS", 30),
		EnableCPU:          getEnvBool("METRICS_ENABLE_CPU", true),
		EnableMemory:       getEnvBool("METRICS_ENABLE_MEMORY", true),
		EnableDisk:         getEnvBool("METRICS_ENABLE_DISK", true),
		EnableNetwork:      getEnvBool("METRICS_ENABLE_NETWORK", true),
		EnableProcesses:    getEnvBool("METRICS_ENABLE_PROCESSES", true),
		BufferSize:         getEnvInt("METRICS_BUFFER_SIZE", 100),
	}
}

// loadAlertConfig loads alert configuration from environment variables
func loadAlertConfig() *AlertConfig {
	return &AlertConfig{
		EnableAlerts:   getEnvBool("ALERTS_ENABLE", true),
		CheckInterval:  getEnvDuration("ALERTS_CHECK_INTERVAL", 30*time.Second),
		CooldownPeriod: getEnvDuration("ALERTS_COOLDOWN_PERIOD", 5*time.Minute),
	}
}

// loadEmailConfig loads email configuration from environment variables
func loadEmailConfig() *EmailConfig {
	return &EmailConfig{
		Enabled:      getEnvBool("EMAIL_ENABLED", false),
		SMTPHost:     getEnvString("EMAIL_SMTP_HOST", ""),
		SMTPPort:     getEnvInt("EMAIL_SMTP_PORT", 587),
		SMTPUsername: getEnvString("EMAIL_SMTP_USERNAME", ""),
		SMTPPassword: getEnvString("EMAIL_SMTP_PASSWORD", ""),
		FromEmail:    getEnvString("EMAIL_FROM_EMAIL", ""),
		FromName:     getEnvString("EMAIL_FROM_NAME", "GoDash Monitor"),
		UseTLS:       getEnvBool("EMAIL_USE_TLS", true),
	}
}

// loadWebhookConfig loads webhook configuration from environment variables
func loadWebhookConfig() *WebhookConfig {
	return &WebhookConfig{
		DefaultTimeout: getEnvDuration("WEBHOOK_TIMEOUT", 10*time.Second),
		MaxRetries:     getEnvInt("WEBHOOK_MAX_RETRIES", 3),
		RetryDelay:     getEnvDuration("WEBHOOK_RETRY_DELAY", 2*time.Second),
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Server.Mode != "debug" && c.Server.Mode != "release" && c.Server.Mode != "test" {
		return fmt.Errorf("invalid server mode: %s", c.Server.Mode)
	}

	// Validate database configuration
	switch c.Database.Driver {
	case "postgres", "Postgres", "POSTGRES":
		if c.Database.Host == "" {
			return fmt.Errorf("database host is required for postgres")
		}
		if c.Database.Port < 1 || c.Database.Port > 65535 {
			return fmt.Errorf("invalid database port: %d", c.Database.Port)
		}
		if c.Database.Name == "" {
			return fmt.Errorf("database name is required for postgres")
		}
	case "sqlite", "SQLite", "SQLITE":
		if c.Database.SQLitePath == "" && c.Database.Name == "" {
			return fmt.Errorf("sqlite path or name is required for sqlite driver")
		}
	default:
		return fmt.Errorf("unsupported database driver: %s", c.Database.Driver)
	}

	// Validate metrics configuration
	if c.Metrics.CollectionInterval < time.Second {
		return fmt.Errorf("collection interval must be at least 1 second")
	}

	if c.Metrics.RetentionDays < 1 {
		return fmt.Errorf("retention days must be at least 1")
	}

	// Validate alert configuration
	if c.Alerts != nil {
		if c.Alerts.CheckInterval < time.Second {
			return fmt.Errorf("alert check interval must be at least 1 second")
		}

		if c.Alerts.CooldownPeriod < 0 {
			return fmt.Errorf("alert cooldown period cannot be negative")
		}
	}

	// Validate email configuration
	if c.Email != nil && c.Email.Enabled {
		if c.Email.SMTPHost == "" {
			return fmt.Errorf("SMTP host is required when email is enabled")
		}

		if c.Email.SMTPPort < 1 || c.Email.SMTPPort > 65535 {
			return fmt.Errorf("invalid SMTP port: %d", c.Email.SMTPPort)
		}

		if c.Email.FromEmail == "" {
			return fmt.Errorf("from email is required when email is enabled")
		}
	}

	// Validate webhook configuration
	if c.Webhook != nil {
		if c.Webhook.DefaultTimeout <= 0 {
			return fmt.Errorf("webhook timeout must be positive")
		}

		if c.Webhook.MaxRetries < 0 {
			return fmt.Errorf("webhook max retries cannot be negative")
		}

		if c.Webhook.RetryDelay < 0 {
			return fmt.Errorf("webhook retry delay cannot be negative")
		}
	}

	return nil
}

// GetServerAddress returns the full server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetDSN returns the database connection string
func (c *Config) GetDSN() string {
	// For Postgres only; SQLite uses SQLitePath directly
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
		c.Database.Timezone,
	)
}

// IsSQLite indicates whether the configured DB is SQLite
func (c *Config) IsSQLite() bool {
	d := c.Database.Driver
	return d == "sqlite" || d == "SQLite" || d == "SQLITE"
}

// Helper functions for environment variable parsing

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
