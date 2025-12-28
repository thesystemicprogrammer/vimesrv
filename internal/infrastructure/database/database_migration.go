package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

type DatabaseMigration struct {
	db            *sql.DB
	migrationJobs []MigrationJob
}

type MigrationJob struct {
	version int
	name    string
	up      string
	down    string
}

// migrations contains all database migrations in correct order
var migrationJobs = []MigrationJob{
	{
		version: 1,
		name:    "create_schema_migrations_table",
		up: `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`,
		down: `
DROP TABLE IF EXISTS schema_migrations;
`,
	},
	{
		version: 2,
		name:    "create_media_table",
		up: `
CREATE TABLE IF NOT EXISTS media (
    id TEXT PRIMARY KEY,
    file_path TEXT NOT NULL UNIQUE,
    filename TEXT NOT NULL,
    title TEXT,
    duration INTEGER DEFAULT 0,
    file_size INTEGER DEFAULT 0,
    format TEXT,
    video_codec TEXT,
    audio_codec TEXT,
    resolution TEXT,
    width INTEGER DEFAULT 0,
    height INTEGER DEFAULT 0,
    bitrate INTEGER DEFAULT 0,
    audio_tracks INTEGER DEFAULT 0,
    subtitle_tracks INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    scanned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_media_filename ON media(filename);
CREATE INDEX IF NOT EXISTS idx_media_file_path ON media(file_path);
CREATE INDEX IF NOT EXISTS idx_media_created_at ON media(created_at);

CREATE TABLE IF NOT EXISTS jobs (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	type          TEXT NOT NULL,
	payload       TEXT,
	status        TEXT NOT NULL CHECK (status IN ('queued','running','succeeded','dead')),
	priority      INTEGER NOT NULL DEFAULT 0,
	run_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	attempt       INTEGER NOT NULL DEFAULT 0,
	max_attempts  INTEGER NOT NULL DEFAULT 25,
	last_error    TEXT,
	worker_id     TEXT,
	scheduled_id  INTEGER,
	created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	started_at    DATETIME,
	finished_at   DATETIME,
	updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
		
CREATE INDEX IF NOT EXISTS idx_jobs_status_runat ON jobs(status, run_at, priority);
CREATE INDEX IF NOT EXISTS idx_jobs_priority ON jobs(priority);
CREATE INDEX IF NOT EXISTS idx_jobs_scheduled_id ON jobs(scheduled_id);

CREATE TABLE IF NOT EXISTS schedules (
	id                INTEGER PRIMARY KEY AUTOINCREMENT,
	name              TEXT NOT NULL UNIQUE,
	cron_spec         TEXT NOT NULL,
	job_type          TEXT NOT NULL,
	payload           TEXT,
	priority          INTEGER NOT NULL DEFAULT 0,
	max_attempts      INTEGER NOT NULL DEFAULT 25,
	enabled           INTEGER NOT NULL DEFAULT 1,
	next_run_at       DATETIME,
	last_enqueued_at  DATETIME,
	updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sched_enabled_next ON schedules(enabled, next_run_at);
`,
		down: `
DROP INDEX IF EXISTS idx_media_created_at;
DROP INDEX IF EXISTS idx_media_file_path;
DROP INDEX IF EXISTS idx_media_filename;
DROP TABLE IF EXISTS media;
`,
	},
}

func NewDatabaseMigration(db *sql.DB) *DatabaseMigration {
	databaseMigration := &DatabaseMigration{
		db:            db,
		migrationJobs: migrationJobs,
	}

	return databaseMigration
}

func (databaseMigration DatabaseMigration) Migrate() error {
	ctx := context.Background()

	currentVersion, err := databaseMigration.getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	for _, migration := range databaseMigration.migrationJobs {

		if migration.version <= currentVersion {
			continue
		}

		err = databaseMigration.writeMigrationToDatabase(ctx, databaseMigration.db, migration)
		if err != nil {
			return err
		}

		logger.Info().Int("version", migration.version).Str("name", migration.name).Msg("Migration successfully conducted")
	}

	return nil
}

func (databaseMigration DatabaseMigration) writeMigrationToDatabase(ctx context.Context, db *sql.DB, migrationJob MigrationJob) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error().Err(err).Msg("failed to begin transaction")
		return err
	}

	if _, err := tx.ExecContext(ctx, migrationJob.up); err != nil {
		tx.Rollback()
		logger.Error().Err(err).Int("version", migrationJob.version).Str("name", migrationJob.name).Msg("failed to execute transaction")
		return err
	}

	err = databaseMigration.recordMigration(ctx, tx, migrationJob)
	if err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		logger.Error().Err(err).Int("version", migrationJob.version).Str("name", migrationJob.name).Msg("failed to commit migration")
		return err
	}

	return nil
}

func (DatabaseMigration) recordMigration(ctx context.Context, tx *sql.Tx, migrationJob MigrationJob) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, name) VALUES (?, ?)", migrationJob.version, migrationJob.name)
	if err != nil {
		tx.Rollback()
		logger.Error().Err(err).Int("version", migrationJob.version).Str("name", migrationJob.name).Msg("failed to execute migration recording")
		return err
	}

	return nil
}

func (databaseMigration DatabaseMigration) getCurrentVersion() (int, error) {
	var version int
	err := databaseMigration.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		// Table doesn't exist yet (SQLite returns "no such table" error)
		matched, _ := regexp.MatchString(`no such table:\s*schema_migrations`, err.Error())
		if matched {
			return 0, nil
		}
		logger.Error().Err(err).Msg("failed to get current version")
		return 0, err
	}
	return version, nil
}
