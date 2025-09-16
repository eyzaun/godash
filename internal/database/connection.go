package database

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
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
	var db *gorm.DB
	var err error
	if cfg.IsSQLite() {
		// SQLite path
		path := cfg.Database.SQLitePath
		if path == "" {
			path = cfg.Database.Name
			if path == "" {
				path = "godash.db"
			}
		}
		log.Printf("Connecting to SQLite database at: %s", path)
		db, err = gorm.Open(sqlite.Open(path), gormConfig)
	} else {
		log.Printf("Connecting to PostgreSQL at %s:%d db=%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
		db, err = gorm.Open(postgres.Open(cfg.GetDSN()), gormConfig)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool (some settings ignored by sqlite driver)
	if !cfg.IsSQLite() {
		sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
		sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	}

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		DB:     db,
		config: &cfg.Database,
	}

	if cfg.IsSQLite() {
		log.Printf("Successfully connected to SQLite database: %s", cfg.Database.SQLitePath)
	} else {
		log.Printf("Successfully connected to PostgreSQL database: %s", cfg.Database.Name)
	}
	return database, nil
}

// AutoMigrate runs database migrations
func (d *Database) AutoMigrate() error {
	log.Println("Running database auto-migration...")

	// First, check if we need to migrate from old schema to new schema (PostgreSQL-only)
	if d.config.Driver == "postgres" || d.config.Driver == "Postgres" || d.config.Driver == "POSTGRES" {
		if err := d.migrateFromOldSchema(); err != nil {
			log.Printf("Warning: failed to migrate from old schema: %v", err)
		}
	}

	// Run auto-migration for all models
	log.Println("Migrating Metric model with extended fields...")
	if err := d.DB.AutoMigrate(&models.Metric{}); err != nil {
		return fmt.Errorf("failed to migrate Metric model: %w", err)
	}

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

	// Add missing speed columns manually if they don't exist (PostgreSQL-only)
	if d.config.Driver == "postgres" || d.config.Driver == "Postgres" || d.config.Driver == "POSTGRES" {
		log.Println("Adding missing speed columns...")
		if err := d.addMissingSpeedColumns(); err != nil {
			log.Printf("Warning: failed to add missing speed columns: %v", err)
		}
	}

	log.Println("Database auto-migration completed successfully")

	// Create indexes for better performance (PostgreSQL-only; SQLite has limited support here)
	if d.config.Driver == "postgres" || d.config.Driver == "Postgres" || d.config.Driver == "POSTGRES" {
		if err := d.createIndexes(); err != nil {
			// Log warning but don't fail the migration
			log.Printf("Warning: failed to create indexes: %v", err)
		}
	}

	log.Println("Database initialization completed successfully")
	return nil
}

// migrateFromOldSchema migrates from old simple schema to new detailed schema
func (d *Database) migrateFromOldSchema() error {
	// Check if we have the old simple schema
	var columnExists bool

	// Check if cpu_cores column exists (new schema indicator)
	err := d.DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'metrics' AND column_name = 'cpu_cores'
		)
	`).Scan(&columnExists).Error

	if err != nil {
		return fmt.Errorf("failed to check schema version: %w", err)
	}

	if columnExists {
		log.Println("✅ Database already has new schema, skipping migration")
		return nil
	}

	log.Println("🔄 Migrating from old schema to new detailed schema...")

	// Add new columns to existing metrics table
	alterQueries := []string{
		// CPU detailed fields
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS cpu_cores INTEGER DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS cpu_frequency_mhz DOUBLE PRECISION DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS cpu_load_avg_1 DOUBLE PRECISION DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS cpu_load_avg_5 DOUBLE PRECISION DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS cpu_load_avg_15 DOUBLE PRECISION DEFAULT 0`,

		// Memory detailed fields
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS memory_total_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS memory_used_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS memory_available_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS memory_free_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS memory_cached_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS memory_buffers_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS memory_swap_total_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS memory_swap_used_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS memory_swap_percent DOUBLE PRECISION DEFAULT 0`,

		// Disk detailed fields
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_total_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_used_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_free_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_read_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_write_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_read_ops BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_write_ops BIGINT DEFAULT 0`,
		// Disk speed fields
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_read_speed_mbps DOUBLE PRECISION DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_write_speed_mbps DOUBLE PRECISION DEFAULT 0`,

		// Network detailed fields
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_total_sent_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_total_recv_bytes BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_packets_sent BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_packets_recv BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_errors BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_drops BIGINT DEFAULT 0`,
		// Network speed fields
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_upload_speed_mbps DOUBLE PRECISION DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_download_speed_mbps DOUBLE PRECISION DEFAULT 0`,

		// System info fields
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS platform VARCHAR(100) DEFAULT ''`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS platform_version VARCHAR(100) DEFAULT ''`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS kernel_arch VARCHAR(50) DEFAULT ''`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS uptime_seconds BIGINT DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS process_count BIGINT DEFAULT 0`,
	}

	// Execute all alter statements
	for _, query := range alterQueries {
		if err := d.DB.Exec(query).Error; err != nil {
			log.Printf("Warning: failed to execute alter query: %s - %v", query, err)
			// Continue with other queries
		}
	}

	log.Println("✅ Schema migration completed")
	return nil
}

// createIndexes creates additional database indexes for better performance
func (d *Database) createIndexes() error {
	log.Println("Creating database indexes for better performance...")

	// Composite index for metrics table (hostname + timestamp) - most important for queries
	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_metrics_hostname_timestamp 
		ON metrics (hostname, "timestamp" DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create hostname_timestamp index: %w", err)
	}

	// Index for time-range queries (critical for historical data)
	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_metrics_timestamp_desc 
		ON metrics ("timestamp" DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create timestamp desc index: %w", err)
	}

	// Index for latest data queries (critical for current metrics)
	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_metrics_id_desc 
		ON metrics (id DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create id desc index: %w", err)
	}

	// Partial index for high usage queries
	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_metrics_high_cpu 
		ON metrics (cpu_usage DESC) WHERE cpu_usage > 80
	`).Error; err != nil {
		log.Printf("Warning: failed to create high CPU index: %v", err)
	}

	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_metrics_high_memory 
		ON metrics (memory_percent DESC) WHERE memory_percent > 80
	`).Error; err != nil {
		log.Printf("Warning: failed to create high memory index: %v", err)
	}

	// System info indexes
	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_system_info_hostname 
		ON system_info (hostname)
	`).Error; err != nil {
		log.Printf("Warning: failed to create system_info hostname index: %v", err)
	}

	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_system_info_last_seen 
		ON system_info (last_seen DESC)
	`).Error; err != nil {
		log.Printf("Warning: failed to create system_info last_seen index: %v", err)
	}

	// Alert indexes
	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_alerts_active 
		ON alerts (metric_type) WHERE is_active = true
	`).Error; err != nil {
		log.Printf("Warning: failed to create active alerts index: %v", err)
	}

	// Index for alert history queries
	if err := d.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_alert_history_hostname_created 
		ON alert_history (hostname, created_at DESC)
	`).Error; err != nil {
		log.Printf("Warning: failed to create alert history index: %v", err)
	}

	log.Println("✅ Database indexes created successfully")
	return nil
}

// HealthCheck checks database connection health
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

// addMissingSpeedColumns adds any missing speed-related columns to the metrics table
func (d *Database) addMissingSpeedColumns() error {
	log.Println("Checking and adding missing speed columns...")

	// Add disk speed columns
	if err := d.DB.Exec(`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_read_speed_mbps DOUBLE PRECISION DEFAULT 0`).Error; err != nil {
		log.Printf("Warning: failed to add disk_read_speed_mbps column: %v", err)
	}

	if err := d.DB.Exec(`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_write_speed_mbps DOUBLE PRECISION DEFAULT 0`).Error; err != nil {
		log.Printf("Warning: failed to add disk_write_speed_mbps column: %v", err)
	}

	// Add network speed columns
	if err := d.DB.Exec(`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_upload_speed_mbps DOUBLE PRECISION DEFAULT 0`).Error; err != nil {
		log.Printf("Warning: failed to add network_upload_speed_mbps column: %v", err)
	}

	if err := d.DB.Exec(`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_download_speed_mbps DOUBLE PRECISION DEFAULT 0`).Error; err != nil {
		log.Printf("Warning: failed to add network_download_speed_mbps column: %v", err)
	}

	log.Println("✅ Speed columns check completed")
	return nil
}

// CleanupOldMetrics removes old metrics based on retention policy
// Note: Removed unused health/stats helper functions to keep API surface minimal.
