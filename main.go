package main

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
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
	alertRepo        repository.AlertRepository
	collectorService *services.CollectorService
	alertService     *services.AlertService
	emailSender      services.EmailSender
	webhookSender    services.WebhookSender
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
	}

	// Initialize repositories
	metricsRepo := repository.NewMetricsRepository(db.DB)
	alertRepo := repository.NewAlertRepository(db.DB)

	// Initialize notification services
	var emailSender services.EmailSender
	var webhookSender services.WebhookSender

	if cfg.Email != nil && cfg.Email.Enabled {
		emailSender = services.NewEmailSender(cfg.Email)
		log.Println("📧 Email service initialized")
	}

	if cfg.Webhook != nil {
		webhookSender = services.NewWebhookSender(cfg.Webhook)
		log.Println("🔗 Webhook service initialized")
	}

	// Initialize services
	collectorService := services.NewCollectorService(cfg, metricsRepo)
	alertService := services.NewAlertService(cfg, alertRepo, emailSender, webhookSender)

	// Link alert service to collector service
	collectorService.SetAlertService(alertService)

	// Initialize API router
	// Setup embedded assets if available (single-exe mode)
	var tplFS, statFS fs.FS
	// embeddedAssets is defined in assets_embed.go; it's always present once built with this file.
	// We create sub-FS for templates and static; if this panics in dev, we’ll fall back to disk.
	// To avoid panic in dev when assets not present yet, wrap in a safe parse.
	{
		// Try to obtain sub FS; if it fails, leave nils to use disk paths.
		if sub, err := fs.Sub(embeddedAssets, "web/templates"); err == nil {
			// Sanity parse a template to ensure files exist
			if _, err := template.ParseFS(sub, "*.html"); err == nil {
				tplFS = sub
			}
		}
		if sub, err := fs.Sub(embeddedAssets, "web/static"); err == nil {
			// Check that at least directory opens
			statFS = sub
		}
	}

	router := api.New(cfg, metricsRepo, alertRepo, collectorService, alertService, emailSender, webhookSender, tplFS, statFS)

	// Connect alert service to WebSocket handler for real-time alert broadcasting
	alertService.SetWebSocketHandler(router.GetWebSocketHandler())

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
		alertRepo:        alertRepo,
		collectorService: collectorService,
		alertService:     alertService,
		emailSender:      emailSender,
		webhookSender:    webhookSender,
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

	// Start alert service
	if err := app.alertService.Start(ctx); err != nil {
		log.Printf("⚠️ Failed to start alert service: %v", err)
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

	// Auto-open browser shortly after start (single-exe UX).
	if app.config.Server.AutoOpen {
		go func() {
			time.Sleep(1200 * time.Millisecond)
			url := fmt.Sprintf("http://%s/", app.config.GetServerAddress())

			// If Windows, prefer Edge/Chrome app window to avoid opening a regular tab.
			if runtime.GOOS == "windows" {
				browsers := []string{
					os.Getenv("ProgramFiles") + "\\Microsoft\\Edge\\Application\\msedge.exe",
					os.Getenv("ProgramFiles(x86)") + "\\Microsoft\\Edge\\Application\\msedge.exe",
					os.Getenv("ProgramFiles") + "\\Google\\Chrome\\Application\\chrome.exe",
					os.Getenv("ProgramFiles(x86)") + "\\Google\\Chrome\\Application\\chrome.exe",
				}
				kiosk := os.Getenv("APP_KIOSK") == "1"
				for _, path := range browsers {
					if _, err := os.Stat(path); err == nil {
						var cmd *exec.Cmd
						if kiosk && (path == browsers[0] || path == browsers[1]) {
							// Edge kiosk supports "--kiosk <url>"
							cmd = exec.Command(path, "--new-window", "--kiosk", url)
						} else {
							// App window for both Edge/Chrome
							cmd = exec.Command(path, "--new-window", "--app="+url)
						}
						_ = cmd.Start()
						return
					}
				}

				// Fallback to default browser if no known browser path found
				_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
				return
			}

			// Non-Windows: open default browser (best-effort)
			// macOS: open, Linux: xdg-open
			if runtime.GOOS == "darwin" {
				_ = exec.Command("open", url).Start()
			} else if runtime.GOOS == "linux" {
				_ = exec.Command("xdg-open", url).Start()
			}
		}()
	}

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

	// Stop alert service
	if app.alertService != nil {
		log.Println("Stopping alert service...")
		if err := app.alertService.Stop(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("alert service stop error: %w", err))
		} else {
			log.Println("Alert service stopped")
		}
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
