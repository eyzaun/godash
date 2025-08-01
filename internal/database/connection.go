package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/eyzaun/godash/internal/config"
	"github.com/eyzaun/godash/internal/models"
)

// Database wraps GORM database connection
type Database struct {
	DB     *gorm.DB
	config *config.DatabaseConfig
}

// New creates a new database connection
func New(cfg *config.Config) (*Database, error) {
	// Configure GORM logger
	var gormLogger logger.Interface
	switch cfg.Database.LogLevel {
	case "silent":
		gormLogger = logger.Default.LogMode(logger.Silent)
	case "error":
		gormLogger = logger.Default.LogMode(logger.Error)
	case "warn":
		gormLogger = logger.Default.LogMode(logger.Warn)
	case "info":
		gormLogger = logger.Default.LogMode(logger.Info)
	default:
		gormLogger = logger.Default.LogMode(logger.Warn)
	}

	// Configure GORM
	gormConfig := &gorm.Config{
		Logger: gormLogger,
	}

	// Create database connection
	db, err := gorm.Open(postgres.Open(cfg.GetDSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		DB:     db,
		config: &cfg.Database,
	}

	log.Printf("Successfully connected to PostgreSQL database: %s", cfg.Database.Name)
	return database, nil
}

// AutoMigrate runs database migrations
func (d *Database) AutoMigrate() error {
	log.Println("Running database auto-migration...")

	// Run auto-migration for all models one by one
	log.Println("Migrating Metric model...")
	if err := d.DB.AutoMigrate(&models.Metric{}); err != nil {
		return fmt.Errorf("failed to migrate Metric model: %w", err)
	}

	// Temporarily disabled other models to debug
	/*
		log.Println("Migrating DBSystemInfo model...")
		if err := d.DB.AutoMigrate(&models.DBSystemInfo{}); err != nil {
			return fmt.Errorf("failed to migrate DBSystemInfo model: %w", err)
		}

		log.Println("Migrating Alert model...")
		if err := d.DB.AutoMigrate(&models.Alert{}); err != nil {
			return fmt.Errorf("failed to migrate Alert model: %w", err)
		}

		log.Println("Migrating AlertHistory model...")
		if err := d.DB.AutoMigrate(&models.AlertHistory{}); err != nil {
			return fmt.Errorf("failed to migrate AlertHistory model: %w", err)
		}
	*/

	log.Println("Database auto-migration completed successfully")

	// Create indexes for better performance
	if err := d.createIndexes(); err != nil {
		// Log warning but don't fail the migration
		log.Printf("Warning: failed to create indexes: %v", err)
	}

	log.Println("Database initialization completed successfully")
	return nil
}

// createIndexes creates additional database indexes for better performance
func (d *Database) createIndexes() error {
	// Composite index for metrics table (hostname + timestamp)
	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_metrics_hostname_timestamp 
		ON metrics (hostname, "timestamp" DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create hostname_timestamp index: %w", err)
	}

	// Index for time-range queries
	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_metrics_timestamp_desc 
		ON metrics ("timestamp" DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create timestamp desc index: %w", err)
	}

	// Partial index for active alerts
	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_alerts_active 
		ON alerts (metric_type) WHERE is_active = true
	`).Error; err != nil {
		return fmt.Errorf("failed to create active alerts index: %w", err)
	}

	// Index for alert history queries
	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_alert_history_hostname_created 
		ON alert_history (hostname, created_at DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create alert history index: %w", err)
	}

	return nil
}

// HealthCheck checks database connection health
func (d *Database) HealthCheck() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// GetStats returns database connection statistics
func (d *Database) GetStats() map[string]interface{} {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration,
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	log.Println("Database connection closed successfully")
	return nil
}

// Transaction executes a function within a database transaction
func (d *Database) Transaction(fn func(*gorm.DB) error) error {
	return d.DB.Transaction(fn)
}

// CleanupOldMetrics removes old metrics based on retention policy
func (d *Database) CleanupOldMetrics(retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	result := d.DB.Where("created_at < ?", cutoffTime).Delete(&models.Metric{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old metrics: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		log.Printf("Cleaned up %d old metric records older than %d days",
			result.RowsAffected, retentionDays)
	}

	return nil
}

// GetMetricsCount returns total count of metrics in database
func (d *Database) GetMetricsCount() (int64, error) {
	var count int64
	err := d.DB.Model(&models.Metric{}).Count(&count).Error
	return count, err
}

// GetLatestMetricsByHostname returns the latest metrics for each hostname
func (d *Database) GetLatestMetricsByHostname() ([]models.Metric, error) {
	var metrics []models.Metric

	err := d.DB.Raw(`
		SELECT DISTINCT ON (hostname) *
		FROM metrics
		ORDER BY hostname, "timestamp" DESC
	`).Scan(&metrics).Error

	return metrics, err
}

// CheckDatabaseExists checks if database exists
func CheckDatabaseExists(cfg *config.Config) error {
	// Connect to postgres database to check if target database exists
	tempDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s TimeZone=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.SSLMode,
		cfg.Database.Timezone,
	)

	db, err := gorm.Open(postgres.Open(tempDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	defer sqlDB.Close()

	// Check if database exists
	var exists bool
	err = db.Raw("SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = ?)",
		cfg.Database.Name).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("database '%s' does not exist", cfg.Database.Name)
	}

	return nil
}
