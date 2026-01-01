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
		name:    "create_media_files_table",
		up: `
CREATE TABLE IF NOT EXISTS media_files (
    id TEXT PRIMARY KEY,
    fingerprint TEXT NOT NULL UNIQUE,
    file_path TEXT NOT NULL UNIQUE,
    original_filename TEXT NOT NULL,
    filename TEXT NOT NULL,
    title TEXT,
    duration INTEGER DEFAULT 0,
    file_size INTEGER DEFAULT 0,
    format TEXT,
    video_codec TEXT,
    audio_codecs TEXT,
    resolution TEXT,
    width INTEGER DEFAULT 0,
    height INTEGER DEFAULT 0,
    bitrate INTEGER DEFAULT 0,
    audio_tracks INTEGER DEFAULT 0,
    subtitle_tracks INTEGER DEFAULT 0,
    subtitle_languages TEXT,
    status TEXT CHECK(status IN ('processing','ready','error')) DEFAULT 'ready',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    scanned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_media_files_fingerprint ON media_files(fingerprint);
CREATE INDEX IF NOT EXISTS idx_media_files_filename ON media_files(filename);
CREATE INDEX IF NOT EXISTS idx_media_files_file_path ON media_files(file_path);
CREATE INDEX IF NOT EXISTS idx_media_files_created_at ON media_files(created_at);
CREATE INDEX IF NOT EXISTS idx_media_files_status ON media_files(status);

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
DROP INDEX IF EXISTS idx_media_files_status;
DROP INDEX IF EXISTS idx_media_files_created_at;
DROP INDEX IF EXISTS idx_media_files_file_path;
DROP INDEX IF EXISTS idx_media_files_filename;
DROP INDEX IF EXISTS idx_media_files_fingerprint;
DROP TABLE IF EXISTS media_files;
`,
	},
	{
		version: 3,
		name:    "create_transcoding_tables",
		up: `
-- Audio stream metadata (extracted during media scan)
CREATE TABLE IF NOT EXISTS audio_streams (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_id TEXT NOT NULL,
    stream_index INTEGER NOT NULL,
    codec TEXT,
    language TEXT,
    channels INTEGER DEFAULT 2,
    channel_layout TEXT,
    sample_rate INTEGER,
    title TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (media_id) REFERENCES media_files(id) ON DELETE CASCADE,
    UNIQUE(media_id, stream_index)
);

CREATE INDEX IF NOT EXISTS idx_audio_streams_media_id ON audio_streams(media_id);

-- Subtitle stream metadata (extracted during media scan)
CREATE TABLE IF NOT EXISTS subtitle_streams (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_id TEXT NOT NULL,
    stream_index INTEGER NOT NULL,
    codec TEXT,
    language TEXT,
    title TEXT,
    forced INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (media_id) REFERENCES media_files(id) ON DELETE CASCADE,
    UNIQUE(media_id, stream_index)
);

CREATE INDEX IF NOT EXISTS idx_subtitle_streams_media_id ON subtitle_streams(media_id);

-- Transcode tracking (individual transcode tasks)
-- Note: Detailed progress, timing, and errors are stored in the jobs table
-- This table only tracks transcode-specific metadata and status
CREATE TABLE IF NOT EXISTS transcodes (
    id TEXT PRIMARY KEY,
    media_id TEXT NOT NULL,
    quality TEXT NOT NULL,
    track_type TEXT NOT NULL CHECK(track_type IN ('video', 'audio', 'subtitle')),
    track_index INTEGER DEFAULT 0,
    status TEXT NOT NULL CHECK(status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    output_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (media_id) REFERENCES media_files(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_transcodes_media_id ON transcodes(media_id);
CREATE INDEX IF NOT EXISTS idx_transcodes_status ON transcodes(status);
CREATE INDEX IF NOT EXISTS idx_transcodes_media_status ON transcodes(media_id, status);
`,
		down: `
DROP INDEX IF EXISTS idx_transcodes_media_status;
DROP INDEX IF EXISTS idx_transcodes_status;
DROP INDEX IF EXISTS idx_transcodes_media_id;
DROP TABLE IF EXISTS transcodes;
DROP INDEX IF EXISTS idx_subtitle_streams_media_id;
DROP TABLE IF EXISTS subtitle_streams;
DROP INDEX IF EXISTS idx_audio_streams_media_id;
DROP TABLE IF EXISTS audio_streams;
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
