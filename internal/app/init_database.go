package app

import (
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

func initializeDatabase(dbConfig config.DatabaseConfig) (*database.DB, error) {
	logger.Info().Str("path", dbConfig.Path).Msg("initializing database")
	db, err := database.New(database.Config{
		Path:            dbConfig.Path,
		MaxOpenConns:    1, // SQLite performs best with single writer
		MaxIdleConns:    1, // Keep one connection alive
		ConnMaxLifetime: 0, // Unlimited (connections don't expire)
		ConnMaxIdleTime: 0, // Unlimited
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Run migrations
	logger.Info().Msg("running database migrations")
	databaseMigration := database.NewDatabaseMigration(db.DB)
	if err := databaseMigration.Migrate(); err != nil {
		db.Close() // Clean up database connection before returning error
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	return db, nil
}
