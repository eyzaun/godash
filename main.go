package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/eyzaun/godash/internal/api"
	"github.com/eyzaun/godash/internal/config"
	"github.com/eyzaun/godash/internal/database"
	"github.com/eyzaun/godash/internal/repository"
	"github.com/eyzaun/godash/internal/services"
)

const (
	AppName    = "GoDash"
	AppVersion = "1.0.0"
)

// Application holds all application dependencies
type Application struct {
	config           *config.Config
	database         *database.Database
	metricsRepo      repository.MetricsRepository
	collectorService *services.CollectorService
	router           *api.Router
	server           *http.Server
}

func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Print welcome message
	fmt.Printf("%s System Monitor v%s\n", AppName, AppVersion)
	fmt.Println("Starting GoDash system monitoring server...")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v. Initiating graceful shutdown...", sig)
		cancel()
	}()

	// Initialize and run the application
	app, err := initializeApplication()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := app.run(ctx); err != nil {
		log.Fatalf("Application error: %v", err)
	}

	fmt.Println("GoDash shutdown complete.")
}

// initializeApplication initializes all application dependencies
func initializeApplication() (*Application, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	log.Printf("Configuration loaded: Server will run on %s", cfg.GetServerAddress())

	// Initialize database
	db, err := database.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Run database migrations
	if err := db.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	} // Initialize repositories
	metricsRepo := repository.NewMetricsRepository(db.DB)

	// Initialize services
	collectorService := services.NewCollectorService(cfg, metricsRepo)

	// Initialize API router
	router := api.New(cfg, metricsRepo)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      router.GetEngine(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &Application{
		config:           cfg,
		database:         db,
		metricsRepo:      metricsRepo,
		collectorService: collectorService,
		router:           router,
		server:           server,
	}, nil
}

// run starts all application services and handles graceful shutdown
func (app *Application) run(ctx context.Context) error {
	// Start collector service
	log.Println("Starting background metrics collection service...")
	if err := app.collectorService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start collector service: %w", err)
	}

	// Start HTTP server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Starting HTTP server on %s", app.config.GetServerAddress())
		log.Printf("API documentation available at: http://%s/api/v1/", app.config.GetServerAddress())
		log.Printf("Health check available at: http://%s/health", app.config.GetServerAddress())

		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Wait for either context cancellation or server error
	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		log.Println("Shutting down gracefully...")
		return app.shutdown()
	}
}

// shutdown gracefully shuts down all application services
func (app *Application) shutdown() error {
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var shutdownErrors []error

	// Shutdown HTTP server
	log.Println("Shutting down HTTP server...")
	if err := app.server.Shutdown(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("HTTP server shutdown error: %w", err))
	} else {
		log.Println("HTTP server shutdown complete")
	}

	// Stop collector service
	log.Println("Stopping collector service...")
	if err := app.collectorService.Stop(); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("collector service stop error: %w", err))
	} else {
		log.Println("Collector service stopped")
	}

	// Close database connection
	log.Println("Closing database connection...")
	if err := app.database.Close(); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("database close error: %w", err))
	} else {
		log.Println("Database connection closed")
	}

	// Return combined errors if any
	if len(shutdownErrors) > 0 {
		return fmt.Errorf("shutdown errors: %v", shutdownErrors)
	}

	log.Println("Graceful shutdown completed successfully")
	return nil
}

// PrintSystemInfo prints system and configuration information
func (app *Application) PrintSystemInfo() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("  %s System Monitor v%s\n", AppName, AppVersion)
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("Server Address: %s\n", app.config.GetServerAddress())
	fmt.Printf("Database: %s:%d/%s\n",
		app.config.Database.Host,
		app.config.Database.Port,
		app.config.Database.Name)
	fmt.Printf("Collection Interval: %v\n", app.config.Metrics.CollectionInterval)
	fmt.Printf("Retention Days: %d\n", app.config.Metrics.RetentionDays)
	fmt.Printf("Environment: %s\n", app.config.Server.Mode)

	fmt.Println("\nEnabled Metrics:")
	fmt.Printf("  CPU: %v\n", app.config.Metrics.EnableCPU)
	fmt.Printf("  Memory: %v\n", app.config.Metrics.EnableMemory)
	fmt.Printf("  Disk: %v\n", app.config.Metrics.EnableDisk)
	fmt.Printf("  Network: %v\n", app.config.Metrics.EnableNetwork)
	fmt.Printf("  Processes: %v\n", app.config.Metrics.EnableProcesses)

	fmt.Println("\nAPI Endpoints:")
	fmt.Printf("  Health Check: http://%s/health\n", app.config.GetServerAddress())
	fmt.Printf("  Current Metrics: http://%s/api/v1/metrics/current\n", app.config.GetServerAddress())
	fmt.Printf("  Metrics History: http://%s/api/v1/metrics/history\n", app.config.GetServerAddress())
	fmt.Printf("  System Status: http://%s/api/v1/system/status\n", app.config.GetServerAddress())

	fmt.Println(strings.Repeat("=", 60) + "\n")
}

// validateEnvironment checks if the environment is properly configured
func validateEnvironment() error {
	// Check if we're running in a container
	if _, err := os.Stat("/.dockerenv"); err == nil {
		log.Println("Running in Docker container")
	}

	// Check available memory
	// In production, you might want to check system resources

	return nil
}

// init function runs before main
func init() {
	// Validate environment on startup
	if err := validateEnvironment(); err != nil {
		log.Fatalf("Environment validation failed: %v", err)
	}
}
