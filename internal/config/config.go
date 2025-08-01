package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"` // debug, release, test
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	CORS         CORSConfig    `mapstructure:"cors"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	Timezone        string        `mapstructure:"timezone"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	LogLevel        string        `mapstructure:"log_level"` // silent, error, warn, info
}

// MetricsConfig holds metrics collection configuration
type MetricsConfig struct {
	CollectionInterval time.Duration `mapstructure:"collection_interval"`
	RetentionDays      int           `mapstructure:"retention_days"`
	BatchSize          int           `mapstructure:"batch_size"`
	EnableCPU          bool          `mapstructure:"enable_cpu"`
	EnableMemory       bool          `mapstructure:"enable_memory"`
	EnableDisk         bool          `mapstructure:"enable_disk"`
	EnableNetwork      bool          `mapstructure:"enable_network"`
	EnableProcesses    bool          `mapstructure:"enable_processes"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string      `mapstructure:"allow_origins"`
	AllowMethods     []string      `mapstructure:"allow_methods"`
	AllowHeaders     []string      `mapstructure:"allow_headers"`
	ExposeHeaders    []string      `mapstructure:"expose_headers"`
	AllowCredentials bool          `mapstructure:"allow_credentials"`
	MaxAge           time.Duration `mapstructure:"max_age"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	Format     string `mapstructure:"format"`      // json, text
	Output     string `mapstructure:"output"`      // stdout, stderr, file
	File       string `mapstructure:"file"`        // log file path
	MaxSize    int    `mapstructure:"max_size"`    // megabytes
	MaxBackups int    `mapstructure:"max_backups"` // number of backups
	MaxAge     int    `mapstructure:"max_age"`     // days
	Compress   bool   `mapstructure:"compress"`    // compress rotated files
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			Mode:         "debug",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			CORS: CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders:     []string{"*"},
				AllowCredentials: false,
				MaxAge:           12 * time.Hour,
			},
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5433,
			User:            "godash",
			Password:        "password",
			Name:            "godash",
			SSLMode:         "disable",
			Timezone:        "UTC",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 1 * time.Hour,
			LogLevel:        "warn",
		},
		Metrics: MetricsConfig{
			CollectionInterval: 30 * time.Second,
			RetentionDays:      30,
			BatchSize:          100,
			EnableCPU:          true,
			EnableMemory:       true,
			EnableDisk:         true,
			EnableNetwork:      true,
			EnableProcesses:    false,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
	}
}

// Load loads configuration from various sources
func Load() (*Config, error) {
	// Set default values
	config := DefaultConfig()

	// Configure viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// Environment variable support
	viper.SetEnvPrefix("GODASH")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; using defaults and environment variables
	}

	// Unmarshal into struct
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Server validation
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}

	if config.Server.Mode != "debug" && config.Server.Mode != "release" && config.Server.Mode != "test" {
		return fmt.Errorf("server mode must be debug, release, or test")
	}

	// Database validation
	if config.Database.Port < 1 || config.Database.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}

	if config.Database.Name == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	if config.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("database max_open_conns must be positive")
	}

	if config.Database.MaxIdleConns <= 0 {
		return fmt.Errorf("database max_idle_conns must be positive")
	}

	// Metrics validation
	if config.Metrics.CollectionInterval < time.Second {
		return fmt.Errorf("metrics collection interval must be at least 1 second")
	}

	if config.Metrics.RetentionDays <= 0 {
		return fmt.Errorf("metrics retention days must be positive")
	}

	if config.Metrics.BatchSize <= 0 {
		return fmt.Errorf("metrics batch size must be positive")
	}

	return nil
}

// GetDSN returns database connection string
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

// GetServerAddress returns full server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Mode == "debug"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Mode == "release"
}

// IsTest returns true if running in test mode
func (c *Config) IsTest() bool {
	return c.Server.Mode == "test"
}
