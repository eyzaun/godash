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
	Alerts   AlertsConfig   `json:"alerts" yaml:"alerts"`
	Logging  LoggingConfig  `json:"logging" yaml:"logging"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string        `json:"host" yaml:"host"`
	Port         int           `json:"port" yaml:"port"`
	Mode         string        `json:"mode" yaml:"mode"` // debug, release, test
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	CORS         CORSConfig    `json:"cors" yaml:"cors"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string      `json:"allow_origins" yaml:"allow_origins"`
	AllowMethods     []string      `json:"allow_methods" yaml:"allow_methods"`
	AllowHeaders     []string      `json:"allow_headers" yaml:"allow_headers"`
	ExposeHeaders    []string      `json:"expose_headers" yaml:"expose_headers"`
	AllowCredentials bool          `json:"allow_credentials" yaml:"allow_credentials"`
	MaxAge           time.Duration `json:"max_age" yaml:"max_age"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `json:"host" yaml:"host"`
	Port            int           `json:"port" yaml:"port"`
	User            string        `json:"user" yaml:"user"`
	Password        string        `json:"password" yaml:"password"`
	Name            string        `json:"name" yaml:"name"`
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

// AlertsConfig holds alerting configuration
type AlertsConfig struct {
	Enable          bool          `json:"enable" yaml:"enable"`
	SMTPHost        string        `json:"smtp_host" yaml:"smtp_host"`
	SMTPPort        int           `json:"smtp_port" yaml:"smtp_port"`
	SMTPUser        string        `json:"smtp_user" yaml:"smtp_user"`
	SMTPPassword    string        `json:"smtp_password" yaml:"smtp_password"`
	SMTPFrom        string        `json:"smtp_from" yaml:"smtp_from"`
	WebhookURL      string        `json:"webhook_url" yaml:"webhook_url"`
	WebhookTimeout  time.Duration `json:"webhook_timeout" yaml:"webhook_timeout"`
	CooldownPeriod  time.Duration `json:"cooldown_period" yaml:"cooldown_period"`
	CPUThreshold    float64       `json:"cpu_threshold" yaml:"cpu_threshold"`
	MemoryThreshold float64       `json:"memory_threshold" yaml:"memory_threshold"`
	DiskThreshold   float64       `json:"disk_threshold" yaml:"disk_threshold"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"` // json, text
	Output     string `json:"output" yaml:"output"` // stdout, stderr, file
	FilePath   string `json:"file_path" yaml:"file_path"`
	MaxSize    int    `json:"max_size" yaml:"max_size"`       // MB
	MaxBackups int    `json:"max_backups" yaml:"max_backups"` // number of backup files
	MaxAge     int    `json:"max_age" yaml:"max_age"`         // days
	Compress   bool   `json:"compress" yaml:"compress"`
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		Server:   loadServerConfig(),
		Database: loadDatabaseConfig(),
		Metrics:  loadMetricsConfig(),
		Alerts:   loadAlertsConfig(),
		Logging:  loadLoggingConfig(),
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
		Host:         getEnvString("SERVER_HOST", "0.0.0.0"),
		Port:         getEnvInt("SERVER_PORT", 8082), // Changed to 8082 to avoid conflicts
		Mode:         getEnvString("SERVER_MODE", "debug"),
		ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		CORS: CORSConfig{
			AllowOrigins: []string{
				"http://localhost:8080",
				"http://127.0.0.1:8080",
				"http://localhost:3000", // For development
			},
			AllowMethods: []string{
				"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD",
			},
			AllowHeaders: []string{
				"Origin", "Content-Type", "Accept", "Authorization",
				"X-Requested-With", "X-Client-ID",
			},
			ExposeHeaders: []string{
				"Content-Length", "X-Request-ID",
			},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		},
	}
}

// loadDatabaseConfig loads database configuration from environment variables
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:            getEnvString("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5433), // Use port 5433 for Docker PostgreSQL
		User:            getEnvString("DB_USER", "godash"),
		Password:        getEnvString("DB_PASSWORD", "password"),
		Name:            getEnvString("DB_NAME", "godash"),
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

// loadAlertsConfig loads alerts configuration from environment variables
func loadAlertsConfig() AlertsConfig {
	return AlertsConfig{
		Enable:          getEnvBool("ALERTS_ENABLE", false),
		SMTPHost:        getEnvString("SMTP_HOST", ""),
		SMTPPort:        getEnvInt("SMTP_PORT", 587),
		SMTPUser:        getEnvString("SMTP_USER", ""),
		SMTPPassword:    getEnvString("SMTP_PASSWORD", ""),
		SMTPFrom:        getEnvString("SMTP_FROM", ""),
		WebhookURL:      getEnvString("WEBHOOK_URL", ""),
		WebhookTimeout:  getEnvDuration("WEBHOOK_TIMEOUT", 10*time.Second),
		CooldownPeriod:  getEnvDuration("ALERT_COOLDOWN_PERIOD", 5*time.Minute),
		CPUThreshold:    getEnvFloat("ALERT_CPU_THRESHOLD", 80.0),
		MemoryThreshold: getEnvFloat("ALERT_MEMORY_THRESHOLD", 85.0),
		DiskThreshold:   getEnvFloat("ALERT_DISK_THRESHOLD", 90.0),
	}
}

// loadLoggingConfig loads logging configuration from environment variables
func loadLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Level:      getEnvString("LOG_LEVEL", "info"),
		Format:     getEnvString("LOG_FORMAT", "text"),
		Output:     getEnvString("LOG_OUTPUT", "stdout"),
		FilePath:   getEnvString("LOG_FILE_PATH", "godash.log"),
		MaxSize:    getEnvInt("LOG_MAX_SIZE", 100),
		MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 3),
		MaxAge:     getEnvInt("LOG_MAX_AGE", 28),
		Compress:   getEnvBool("LOG_COMPRESS", true),
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
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}

	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate metrics configuration
	if c.Metrics.CollectionInterval < time.Second {
		return fmt.Errorf("collection interval must be at least 1 second")
	}

	if c.Metrics.RetentionDays < 1 {
		return fmt.Errorf("retention days must be at least 1")
	}

	// Validate alert thresholds
	if c.Alerts.CPUThreshold < 0 || c.Alerts.CPUThreshold > 100 {
		return fmt.Errorf("CPU threshold must be between 0 and 100")
	}

	if c.Alerts.MemoryThreshold < 0 || c.Alerts.MemoryThreshold > 100 {
		return fmt.Errorf("memory threshold must be between 0 and 100")
	}

	if c.Alerts.DiskThreshold < 0 || c.Alerts.DiskThreshold > 100 {
		return fmt.Errorf("disk threshold must be between 0 and 100")
	}

	return nil
}

// GetServerAddress returns the full server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetDSN returns the database connection string
func (c *Config) GetDSN() string {
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

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Mode == "debug"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Mode == "release"
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

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
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

// LoadFromFile loads configuration from a YAML or JSON file
func LoadFromFile(filePath string) (*Config, error) {
	// For now, just return the environment-based config
	// In the future, implement file-based configuration loading
	return Load()
}

// GetDefaultConfig returns a configuration with all default values
func GetDefaultConfig() *Config {
	config := &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			Mode:         "debug",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			CORS: CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders:     []string{"*"},
				AllowCredentials: true,
				MaxAge:           12 * time.Hour,
			},
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "godash",
			Password:        "password",
			Name:            "godash",
			SSLMode:         "disable",
			Timezone:        "UTC",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			LogLevel:        "warn",
		},
		Metrics: MetricsConfig{
			CollectionInterval: 30 * time.Second,
			RetentionDays:      30,
			EnableCPU:          true,
			EnableMemory:       true,
			EnableDisk:         true,
			EnableNetwork:      true,
			EnableProcesses:    true,
			BufferSize:         100,
		},
		Alerts: AlertsConfig{
			Enable:          false,
			CooldownPeriod:  5 * time.Minute,
			CPUThreshold:    80.0,
			MemoryThreshold: 85.0,
			DiskThreshold:   90.0,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
	}

	return config
}

// String returns a string representation of the configuration (without sensitive data)
func (c *Config) String() string {
	return fmt.Sprintf("Config{Server: %s:%d, Database: %s@%s:%d/%s, Mode: %s}",
		c.Server.Host, c.Server.Port,
		c.Database.User, c.Database.Host, c.Database.Port, c.Database.Name,
		c.Server.Mode)
}
