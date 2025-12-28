package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

type Application struct {
	config     *config.Config
	db         *database.DB
	httpServer *server.HttpServer
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

	logger.Debug().Msg("creating HTTP server")
	app.httpServer = server.NewHttpServer(server.HttpServerConfig{
		Host: app.config.Server.Host,
		Port: app.config.Server.Port,
	})

	return nil
}

func initializeDatabase(dbConfig config.DatabaseConfig) (*database.DB, error) {
	logger.Info().Str("path", dbConfig.Path).Msg("initializing database")
	db, err := database.New(database.Config{
		Path: dbConfig.Path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Run migrations
	logger.Info().Msg("running database migrations")
	databaseMigration := database.NewDatabaseMigration(db.DB)
	if err := databaseMigration.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	return db, nil
}

func (app *Application) Start() error {
	// Channel to listen for errors from server
	serverErrors := make(chan error, 1)

	go func() {
		logger.Info().Str("address", app.httpServer.Addr()).Msg("HTTP server listening")
		serverErrors <- app.httpServer.Start()
	}()

	// Channel to listen for interrupt signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logger.Info().Str("signal", sig.String()).Msg("received shutdown signal")
		return app.Shutdown()
	}
}

// Shutdown gracefully shuts down the application
func (app *Application) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info().Msg("shutting down HTTP server")
	if err := app.httpServer.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("error during HTTP server shutdown")
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	logger.Info().Msg("closing database connection")
	if err := app.db.Close(); err != nil {
		logger.Error().Err(err).Msg("error closing database")
		// Don't return error, continue cleanup
	}

	logger.Info().Msg("graceful shutdown complete")
	return nil
}
