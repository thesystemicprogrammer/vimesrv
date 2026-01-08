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
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/user"
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

	registerJobs(app.config, app.useCases, app.adapters)

	// Validate that all required job handlers are registered
	if err := validateJobHandlers(app.adapters.HandlerRegistry); err != nil {
		return fmt.Errorf("job handler validation failed: %w", err)
	}

	registerHTTPHandlers(app.useCases, app.adapters, app.httpServer, app.config, app.jobManager)

	initSuccess = true // Mark initialization as successful
	return nil
}

func (app *Application) Start() error {
	// Channel to listen for errors from HTTP server
	serverErrors := make(chan error, 1)

	// Seed initial admin user if no users exist
	ctx := context.Background()
	if err := seedInitialAdmin(ctx, app.useCases, app.adapters); err != nil {
		logger.Warn().Err(err).Msg("failed to seed initial admin user")
		// Don't fail startup, just log the warning
	}

	// Start WebSocket hub if enabled
	if app.adapters.WebSocketHub != nil {
		go app.adapters.WebSocketHub.Run()
		logger.Info().Msg("WebSocket hub started")
	}

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

	// Step 1: Stop WebSocket hub (close all client connections)
	if app.adapters.WebSocketHub != nil {
		logger.Info().Msg("stopping WebSocket hub")
		app.adapters.WebSocketHub.Stop()
		logger.Info().Msg("WebSocket hub stopped successfully")
	}

	// Step 1b: Stop progress cache cleanup goroutine
	if app.adapters.ProgressCache != nil {
		logger.Info().Msg("stopping progress cache")
		app.adapters.ProgressCache.Stop()
		logger.Info().Msg("progress cache stopped successfully")
	}

	// Step 2: Shutdown HTTP server (prevents new requests/job creation)
	logger.Info().Msg("shutting down HTTP server")
	if err := app.httpServer.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("error during HTTP server shutdown")
		shutdownErrors = append(shutdownErrors, fmt.Errorf("HTTP server shutdown: %w", err))
	} else {
		logger.Info().Msg("HTTP server stopped successfully")
	}

	// Step 3: Stop job manager (let workers finish in-flight jobs)
	logger.Info().Msg("stopping job manager")
	if err := app.jobManager.Stop(ctx); err != nil {
		logger.Error().Err(err).Msg("error during job manager shutdown")
		shutdownErrors = append(shutdownErrors, fmt.Errorf("job manager shutdown: %w", err))
	} else {
		logger.Info().Msg("job manager stopped successfully")
	}

	// Step 4: Close database connection
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
		// Step 5: Close logger (flushes pending writes, stops rotation goroutines)
		// Do this after logging errors but before returning
		if err := logger.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing logger: %v\n", err)
		}
		return fmt.Errorf("shutdown encountered %d error(s): %v", len(shutdownErrors), shutdownErrors)
	}

	logger.Info().Msg("graceful shutdown complete")

	// Step 5: Close logger (flushes pending writes, stops rotation goroutines)
	if err := logger.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "error closing logger: %v\n", err)
	}

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

// seedInitialAdmin creates the initial admin user if no users exist in the database
func seedInitialAdmin(ctx context.Context, useCases *UseCases, adapters *Adapters) error {
	// Check if any users exist
	userCount, err := adapters.UserRepository.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	// If users already exist, no need to seed
	if userCount > 0 {
		logger.Debug().Int("userCount", userCount).Msg("users exist, skipping admin seed")
		return nil
	}

	// Create default admin user with must_change_password flag
	// Default credentials: admin / admin123
	input := user.CreateUserInput{
		Username: "admin",
		Password: "admin123",
		Role:     shared.RoleAdmin,
	}

	_, err = useCases.CreateUserUseCase.Execute(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create initial admin: %w", err)
	}

	logger.Warn().
		Str("username", input.Username).
		Msg("created initial admin user with default password - CHANGE IT IMMEDIATELY")

	return nil
}
