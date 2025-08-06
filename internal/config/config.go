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
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string        `json:"host" yaml:"host"`
	Port         int           `json:"port" yaml:"port"`
	Mode         string        `json:"mode" yaml:"mode"` // debug, release, test
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
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

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		Server:   loadServerConfig(),
		Database: loadDatabaseConfig(),
		Metrics:  loadMetricsConfig(),
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
		Port:         getEnvInt("SERVER_PORT", 8080),
		Mode:         getEnvString("SERVER_MODE", "debug"),
		ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
	}
}

// loadDatabaseConfig loads database configuration from environment variables
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:            getEnvString("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5433),
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
