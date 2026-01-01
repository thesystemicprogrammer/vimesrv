package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

type Application struct {
	config     *config.Config
	db         *database.DB
	httpServer *server.HTTPServer
	jobManager *job.JobManager
	adapters   *Adapters
	useCases   *UseCases
}

func NewApplication(cfg *config.Config) (*Application, error) {
	app := &Application{
		config: cfg,
	}

	if err := app.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize application: %w", err)
	}

	return app, nil
}

func (app *Application) initialize() error {
	db, err := initializeDatabase(app.config.Database)
	if err != nil {
		return err
	}
	app.db = db

	// Ensure cleanup on failure
	initSuccess := false
	defer func() {
		if !initSuccess && app.db != nil {
			logger.Warn().Msg("initialization failed, cleaning up database connection")
			if closeErr := app.db.Close(); closeErr != nil {
				logger.Error().Err(closeErr).Msg("failed to close database during cleanup")
			}
			app.db = nil
		}
	}()

	app.adapters = initAdapters(app.config, app.db)

	// Validate external dependencies are available
	if err := validateExternalDependencies(app.adapters); err != nil {
		return fmt.Errorf("external dependency validation failed: %w", err)
	}

	app.useCases = initUseCases(app.config, app.adapters)
	app.httpServer = initializeHTTPServer(app.config.Server)

	jobManager, err := initializeJobManager(app.config, app.adapters, app.useCases)
	if err != nil {
		return fmt.Errorf("failed to initialize job manager: %w", err)
	}
	app.jobManager = jobManager

	registerJobs(app.useCases, app.adapters)

	// Validate that all required job handlers are registered
	if err := validateJobHandlers(app.adapters.HandlerRegistry); err != nil {
		return fmt.Errorf("job handler validation failed: %w", err)
	}

	registerHTTPHandlers(app.useCases, app.httpServer, app.config)

	initSuccess = true // Mark initialization as successful
	return nil
}

func (app *Application) Start() error {
	// Channel to listen for errors from HTTP server
	serverErrors := make(chan error, 1)

	// Start HTTP server in goroutine
	go func() {
		logger.Info().Str("address", app.httpServer.Addr()).Msg("HTTP server listening")
		serverErrors <- app.httpServer.Start()
	}()

	// Start job manager (returns immediately after launching workers)
	if err := app.jobManager.Start(); err != nil {
		return fmt.Errorf("job manager startup error: %w", err)
	}

	// Initialize startup schedules (library scan, etc.)
	// This must happen after job manager starts so scheduler can process schedules
	ctx := context.Background()
	if err := initStartupSchedules(ctx, app.config, app.useCases, app.adapters); err != nil {
		// Fatal error - invalid configuration should prevent server from starting
		logger.Error().Err(err).Msg("failed to initialize startup schedules")
		return fmt.Errorf("startup schedules initialization failed: %w", err)
	}

	// Channel to listen for interrupt signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("HTTP server error: %w", err)

	case sig := <-shutdown:
		logger.Info().Str("signal", sig.String()).Msg("received shutdown signal")
		return app.Shutdown()
	}
}

// Shutdown gracefully shuts down the application
func (app *Application) Shutdown() error {
	// Use configurable timeout
	shutdownTimeout := time.Duration(app.config.Server.ShutdownTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Track shutdown errors
	var shutdownErrors []error

	// Step 1: Shutdown HTTP server (prevents new requests/job creation)
	logger.Info().Msg("shutting down HTTP server")
	if err := app.httpServer.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("error during HTTP server shutdown")
		shutdownErrors = append(shutdownErrors, fmt.Errorf("HTTP server shutdown: %w", err))
	} else {
		logger.Info().Msg("HTTP server stopped successfully")
	}

	// Step 2: Stop job manager (let workers finish in-flight jobs)
	logger.Info().Msg("stopping job manager")
	if err := app.jobManager.Stop(ctx); err != nil {
		logger.Error().Err(err).Msg("error during job manager shutdown")
		shutdownErrors = append(shutdownErrors, fmt.Errorf("job manager shutdown: %w", err))
	} else {
		logger.Info().Msg("job manager stopped successfully")
	}

	// Step 3: Close database connection
	logger.Info().Msg("closing database connection")
	if err := app.db.Close(); err != nil {
		logger.Error().Err(err).Msg("error closing database")
		shutdownErrors = append(shutdownErrors, fmt.Errorf("database close: %w", err))
	} else {
		logger.Info().Msg("database closed successfully")
	}

	// Return combined errors if any
	if len(shutdownErrors) > 0 {
		logger.Error().Int("errorCount", len(shutdownErrors)).Msg("graceful shutdown completed with errors")
		return fmt.Errorf("shutdown encountered %d error(s): %v", len(shutdownErrors), shutdownErrors)
	}

	logger.Info().Msg("graceful shutdown complete")
	return nil
}

// validateJobHandlers ensures all required job handlers are registered
func validateJobHandlers(registry *job.HandlerRegistry) error {
	requiredHandlers := []string{
		shared.JobTypeScanLibrary,
		// Add more job types here as they are implemented
	}

	var missingHandlers []string
	for _, jobType := range requiredHandlers {
		if _, exists := registry.Get(jobType); !exists {
			missingHandlers = append(missingHandlers, jobType)
		}
	}

	if len(missingHandlers) > 0 {
		return fmt.Errorf("missing job handlers: %v", missingHandlers)
	}

	logger.Debug().Int("count", len(requiredHandlers)).Msg("all required job handlers registered")
	return nil
}

// validateExternalDependencies ensures all required external dependencies are available
func validateExternalDependencies(adapters *Adapters) error {
	logger.Info().Msg("validating external dependencies")

	// Validate ffprobe is available
	if err := adapters.FFProbeService.IsAvailable(); err != nil {
		return fmt.Errorf("ffprobe validation failed: %w", err)
	}
	logger.Info().Msg("ffprobe is available")

	// Add more external dependency checks here as needed

	return nil
}
