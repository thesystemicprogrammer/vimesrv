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
	{
		version: 4,
		name:    "create_tmdb_metadata_tables",
		up: `
-- Add enrichment fields to media_files
ALTER TABLE media_files ADD COLUMN enrichment_status TEXT DEFAULT 'pending'
    CHECK(enrichment_status IN ('pending', 'auto_linked', 'candidates_found', 
                                 'manual_required', 'linked', 'skipped'));
ALTER TABLE media_files ADD COLUMN metadata_type TEXT DEFAULT ''
    CHECK(metadata_type IN ('', 'movie', 'episode'));
ALTER TABLE media_files ADD COLUMN movie_metadata_id INTEGER;
ALTER TABLE media_files ADD COLUMN episode_metadata_id INTEGER;
ALTER TABLE media_files ADD COLUMN edition TEXT DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_media_files_enrichment_status ON media_files(enrichment_status);
CREATE INDEX IF NOT EXISTS idx_media_files_metadata_type ON media_files(metadata_type);

-- Movie metadata table (language-independent fields)
CREATE TABLE IF NOT EXISTS movie_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tmdb_id INTEGER NOT NULL UNIQUE,
    imdb_id TEXT,
    original_title TEXT NOT NULL,
    release_date TEXT,
    runtime INTEGER DEFAULT 0,
    poster_path TEXT,
    backdrop_path TEXT,
    genres TEXT,
    vote_average REAL DEFAULT 0,
    vote_count INTEGER DEFAULT 0,
    popularity REAL DEFAULT 0,
    status TEXT,
    original_lang TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_movie_metadata_tmdb_id ON movie_metadata(tmdb_id);

-- Movie metadata translations
CREATE TABLE IF NOT EXISTS movie_metadata_translations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    movie_metadata_id INTEGER NOT NULL REFERENCES movie_metadata(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    title TEXT NOT NULL,
    tagline TEXT,
    overview TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(movie_metadata_id, language)
);

CREATE INDEX IF NOT EXISTS idx_movie_translations_movie_id ON movie_metadata_translations(movie_metadata_id);
CREATE INDEX IF NOT EXISTS idx_movie_translations_language ON movie_metadata_translations(language);

-- Series metadata table (language-independent fields)
CREATE TABLE IF NOT EXISTS series_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tmdb_id INTEGER NOT NULL UNIQUE,
    original_name TEXT NOT NULL,
    first_air_date TEXT,
    last_air_date TEXT,
    status TEXT,
    poster_path TEXT,
    backdrop_path TEXT,
    genres TEXT,
    networks TEXT,
    vote_average REAL DEFAULT 0,
    vote_count INTEGER DEFAULT 0,
    popularity REAL DEFAULT 0,
    number_of_seasons INTEGER DEFAULT 0,
    number_of_episodes INTEGER DEFAULT 0,
original_lang TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_series_metadata_tmdb_id ON series_metadata(tmdb_id);

-- Series metadata translations
CREATE TABLE IF NOT EXISTS series_metadata_translations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_metadata_id INTEGER NOT NULL REFERENCES series_metadata(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    name TEXT NOT NULL,
    overview TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(series_metadata_id, language)
);

CREATE INDEX IF NOT EXISTS idx_series_translations_series_id ON series_metadata_translations(series_metadata_id);
CREATE INDEX IF NOT EXISTS idx_series_translations_language ON series_metadata_translations(language);

-- Season metadata table
CREATE TABLE IF NOT EXISTS season_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL REFERENCES series_metadata(id) ON DELETE CASCADE,
    tmdb_id INTEGER NOT NULL,
    season_number INTEGER NOT NULL,
    air_date TEXT,
    poster_path TEXT,
    episode_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(series_id, season_number)
);

CREATE INDEX IF NOT EXISTS idx_season_metadata_series_id ON season_metadata(series_id);

-- Season metadata translations
CREATE TABLE IF NOT EXISTS season_metadata_translations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    season_metadata_id INTEGER NOT NULL REFERENCES season_metadata(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    name TEXT,
    overview TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(season_metadata_id, language)
);

CREATE INDEX IF NOT EXISTS idx_season_translations_season_id ON season_metadata_translations(season_metadata_id);
CREATE INDEX IF NOT EXISTS idx_season_translations_language ON season_metadata_translations(language);

-- Episode metadata table
CREATE TABLE IF NOT EXISTS episode_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    season_id INTEGER NOT NULL REFERENCES season_metadata(id) ON DELETE CASCADE,
    tmdb_id INTEGER NOT NULL,
    episode_number INTEGER NOT NULL,
    air_date TEXT,
    still_path TEXT,
    runtime INTEGER DEFAULT 0,
    vote_average REAL DEFAULT 0,
    vote_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(season_id, episode_number)
);

CREATE INDEX IF NOT EXISTS idx_episode_metadata_season_id ON episode_metadata(season_id);

-- Episode metadata translations
CREATE TABLE IF NOT EXISTS episode_metadata_translations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_metadata_id INTEGER NOT NULL REFERENCES episode_metadata(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    name TEXT,
    overview TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(episode_metadata_id, language)
);

CREATE INDEX IF NOT EXISTS idx_episode_translations_episode_id ON episode_metadata_translations(episode_metadata_id);
CREATE INDEX IF NOT EXISTS idx_episode_translations_language ON episode_metadata_translations(language);

-- Metadata candidates for user selection
CREATE TABLE IF NOT EXISTS metadata_candidates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_file_id TEXT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
    candidate_type TEXT NOT NULL CHECK(candidate_type IN ('movie', 'series')),
    tmdb_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    release_date TEXT,
    overview TEXT,
    poster_path TEXT,
    confidence_score INTEGER NOT NULL,
    season_number INTEGER,
    episode_number INTEGER,
    status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'selected', 'rejected')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(media_file_id, candidate_type, tmdb_id)
);

CREATE INDEX IF NOT EXISTS idx_metadata_candidates_media_file_id ON metadata_candidates(media_file_id);
CREATE INDEX IF NOT EXISTS idx_metadata_candidates_status ON metadata_candidates(status);

-- Add foreign key constraints to media_files (SQLite doesn't enforce these via ALTER, but they're documented)
-- movie_metadata_id REFERENCES movie_metadata(id)
-- episode_metadata_id REFERENCES episode_metadata(id)
`,
		down: `
DROP INDEX IF EXISTS idx_metadata_candidates_status;
DROP INDEX IF EXISTS idx_metadata_candidates_media_file_id;
DROP TABLE IF EXISTS metadata_candidates;

DROP INDEX IF EXISTS idx_episode_translations_language;
DROP INDEX IF EXISTS idx_episode_translations_episode_id;
DROP TABLE IF EXISTS episode_metadata_translations;

DROP INDEX IF EXISTS idx_episode_metadata_season_id;
DROP TABLE IF EXISTS episode_metadata;

DROP INDEX IF EXISTS idx_season_translations_language;
DROP INDEX IF EXISTS idx_season_translations_season_id;
DROP TABLE IF EXISTS season_metadata_translations;

DROP INDEX IF EXISTS idx_season_metadata_series_id;
DROP TABLE IF EXISTS season_metadata;

DROP INDEX IF EXISTS idx_series_translations_language;
DROP INDEX IF EXISTS idx_series_translations_series_id;
DROP TABLE IF EXISTS series_metadata_translations;

DROP INDEX IF EXISTS idx_series_metadata_tmdb_id;
DROP TABLE IF EXISTS series_metadata;

DROP INDEX IF EXISTS idx_movie_translations_language;
DROP INDEX IF EXISTS idx_movie_translations_movie_id;
DROP TABLE IF EXISTS movie_metadata_translations;

DROP INDEX IF EXISTS idx_movie_metadata_tmdb_id;
DROP TABLE IF EXISTS movie_metadata;

-- SQLite doesn't support DROP COLUMN, would need to recreate table
-- For development, we can just leave the columns
`,
	},
	{
		version: 5,
		name:    "create_movie_credits_and_certifications",
		up: `
-- Movie credits table (cast & crew)
CREATE TABLE IF NOT EXISTS movie_credits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    movie_metadata_id INTEGER NOT NULL REFERENCES movie_metadata(id) ON DELETE CASCADE,
    credit_type TEXT NOT NULL CHECK(credit_type IN ('cast', 'crew')),
    tmdb_person_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    character TEXT,
    job TEXT,
    department TEXT,
    profile_path TEXT,
    display_order INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_movie_credits_movie_id ON movie_credits(movie_metadata_id);
CREATE INDEX IF NOT EXISTS idx_movie_credits_type ON movie_credits(credit_type);
CREATE INDEX IF NOT EXISTS idx_movie_credits_order ON movie_credits(movie_metadata_id, credit_type, display_order);

-- Movie certifications table (content ratings by country)
CREATE TABLE IF NOT EXISTS movie_certifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    movie_metadata_id INTEGER NOT NULL REFERENCES movie_metadata(id) ON DELETE CASCADE,
    country TEXT NOT NULL,
    certification TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(movie_metadata_id, country)
);

CREATE INDEX IF NOT EXISTS idx_movie_certifications_movie_id ON movie_certifications(movie_metadata_id);
CREATE INDEX IF NOT EXISTS idx_movie_certifications_country ON movie_certifications(country);
`,
		down: `
DROP INDEX IF EXISTS idx_movie_certifications_country;
DROP INDEX IF EXISTS idx_movie_certifications_movie_id;
DROP TABLE IF EXISTS movie_certifications;

DROP INDEX IF EXISTS idx_movie_credits_order;
DROP INDEX IF EXISTS idx_movie_credits_type;
DROP INDEX IF EXISTS idx_movie_credits_movie_id;
DROP TABLE IF EXISTS movie_credits;
`,
	},
	{
		version: 6,
		name:    "create_similar_content_tables",
		up: `
-- Similar movies cache
CREATE TABLE IF NOT EXISTS similar_movies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    movie_metadata_id INTEGER NOT NULL REFERENCES movie_metadata(id) ON DELETE CASCADE,
    similar_tmdb_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    poster_path TEXT,
    release_date TEXT,
    vote_average REAL DEFAULT 0,
    display_order INTEGER NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(movie_metadata_id, similar_tmdb_id)
);

CREATE INDEX IF NOT EXISTS idx_similar_movies_metadata_id ON similar_movies(movie_metadata_id);
CREATE INDEX IF NOT EXISTS idx_similar_movies_fetched_at ON similar_movies(fetched_at);

-- Similar series cache
CREATE TABLE IF NOT EXISTS similar_series (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_metadata_id INTEGER NOT NULL REFERENCES series_metadata(id) ON DELETE CASCADE,
    similar_tmdb_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    poster_path TEXT,
    first_air_date TEXT,
    vote_average REAL DEFAULT 0,
    display_order INTEGER NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(series_metadata_id, similar_tmdb_id)
);

CREATE INDEX IF NOT EXISTS idx_similar_series_metadata_id ON similar_series(series_metadata_id);
CREATE INDEX IF NOT EXISTS idx_similar_series_fetched_at ON similar_series(fetched_at);
`,
		down: `
DROP INDEX IF EXISTS idx_similar_series_fetched_at;
DROP INDEX IF EXISTS idx_similar_series_metadata_id;
DROP TABLE IF EXISTS similar_series;

DROP INDEX IF EXISTS idx_similar_movies_fetched_at;
DROP INDEX IF EXISTS idx_similar_movies_metadata_id;
DROP TABLE IF EXISTS similar_movies;
`,
	},
	{
		version: 7,
		name:    "create_movie_collections_tables",
		up: `
-- Add collection_id to movie_metadata
ALTER TABLE movie_metadata ADD COLUMN collection_id INTEGER;

-- Collection metadata cache
CREATE TABLE IF NOT EXISTS collection_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    collection_id INTEGER NOT NULL UNIQUE,
    name TEXT NOT NULL,
    overview TEXT,
    poster_path TEXT,
    backdrop_path TEXT,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_collection_metadata_collection_id ON collection_metadata(collection_id);
CREATE INDEX IF NOT EXISTS idx_collection_metadata_fetched_at ON collection_metadata(fetched_at);

-- Collection translations cache
CREATE TABLE IF NOT EXISTS collection_translations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    collection_id INTEGER NOT NULL,
    language TEXT NOT NULL,
    name TEXT,
    overview TEXT,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(collection_id, language)
);

CREATE INDEX IF NOT EXISTS idx_collection_translations_collection_id ON collection_translations(collection_id);
CREATE INDEX IF NOT EXISTS idx_collection_translations_fetched_at ON collection_translations(fetched_at);

-- Collection movies cache (movies within a collection)
CREATE TABLE IF NOT EXISTS collection_movies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    collection_id INTEGER NOT NULL,
    tmdb_movie_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    original_title TEXT NOT NULL,
    poster_path TEXT,
    release_date TEXT,
    vote_average REAL DEFAULT 0,
    display_order INTEGER NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(collection_id, tmdb_movie_id)
);

CREATE INDEX IF NOT EXISTS idx_collection_movies_collection_id ON collection_movies(collection_id);
CREATE INDEX IF NOT EXISTS idx_collection_movies_fetched_at ON collection_movies(fetched_at);
`,
		down: `
DROP INDEX IF EXISTS idx_collection_movies_fetched_at;
DROP INDEX IF EXISTS idx_collection_movies_collection_id;
DROP TABLE IF EXISTS collection_movies;

DROP INDEX IF EXISTS idx_collection_translations_fetched_at;
DROP INDEX IF EXISTS idx_collection_translations_collection_id;
DROP TABLE IF EXISTS collection_translations;

DROP INDEX IF EXISTS idx_collection_metadata_fetched_at;
DROP INDEX IF EXISTS idx_collection_metadata_collection_id;
DROP TABLE IF EXISTS collection_metadata;

-- SQLite doesn't support DROP COLUMN
`,
	},
	{
		version: 8,
		name:    "create_similar_and_collection_movie_translations",
		up: `
-- Similar movie translations cache
CREATE TABLE IF NOT EXISTS similar_movie_translations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    similar_movie_id INTEGER NOT NULL REFERENCES similar_movies(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    title TEXT NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(similar_movie_id, language)
);

CREATE INDEX IF NOT EXISTS idx_similar_movie_translations_movie_id ON similar_movie_translations(similar_movie_id);
CREATE INDEX IF NOT EXISTS idx_similar_movie_translations_language ON similar_movie_translations(language);

-- Similar series translations cache
CREATE TABLE IF NOT EXISTS similar_series_translations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    similar_series_id INTEGER NOT NULL REFERENCES similar_series(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    name TEXT NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(similar_series_id, language)
);

CREATE INDEX IF NOT EXISTS idx_similar_series_translations_series_id ON similar_series_translations(similar_series_id);
CREATE INDEX IF NOT EXISTS idx_similar_series_translations_language ON similar_series_translations(language);

-- Collection movie translations cache
CREATE TABLE IF NOT EXISTS collection_movie_translations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    collection_movie_id INTEGER NOT NULL REFERENCES collection_movies(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    title TEXT NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(collection_movie_id, language)
);

CREATE INDEX IF NOT EXISTS idx_collection_movie_translations_movie_id ON collection_movie_translations(collection_movie_id);
CREATE INDEX IF NOT EXISTS idx_collection_movie_translations_language ON collection_movie_translations(language);
`,
		down: `
DROP INDEX IF EXISTS idx_collection_movie_translations_language;
DROP INDEX IF EXISTS idx_collection_movie_translations_movie_id;
DROP TABLE IF EXISTS collection_movie_translations;

DROP INDEX IF EXISTS idx_similar_series_translations_language;
DROP INDEX IF EXISTS idx_similar_series_translations_series_id;
DROP TABLE IF EXISTS similar_series_translations;

DROP INDEX IF EXISTS idx_similar_movie_translations_language;
DROP INDEX IF EXISTS idx_similar_movie_translations_movie_id;
DROP TABLE IF EXISTS similar_movie_translations;
`,
	},
	{
		version: 9,
		name:    "create_users_table",
		up: `
-- Users table for authentication and authorization
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('admin', 'manager', 'user')),
    must_change_password INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
`,
		down: `
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_username;
DROP TABLE IF EXISTS users;
`,
	},
	{
		version: 10,
		name:    "add_full_credits_support",
		up: `
-- Track if full credits have been fetched for movies
ALTER TABLE movie_metadata ADD COLUMN full_credits_fetched INTEGER DEFAULT 0;

-- Series credits table (aggregate credits across all episodes)
CREATE TABLE IF NOT EXISTS series_credits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_metadata_id INTEGER NOT NULL REFERENCES series_metadata(id) ON DELETE CASCADE,
    credit_type TEXT NOT NULL CHECK(credit_type IN ('cast', 'crew')),
    tmdb_person_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    roles TEXT,
    jobs TEXT,
    department TEXT,
    profile_path TEXT,
    total_episode_count INTEGER DEFAULT 0,
    display_order INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_series_credits_series_id ON series_credits(series_metadata_id);
CREATE INDEX IF NOT EXISTS idx_series_credits_type ON series_credits(credit_type);
CREATE INDEX IF NOT EXISTS idx_series_credits_order ON series_credits(series_metadata_id, credit_type, display_order);

-- Track if full credits have been fetched for series
ALTER TABLE series_metadata ADD COLUMN full_credits_fetched INTEGER DEFAULT 0;
`,
		down: `
DROP INDEX IF EXISTS idx_series_credits_order;
DROP INDEX IF EXISTS idx_series_credits_type;
DROP INDEX IF EXISTS idx_series_credits_series_id;
DROP TABLE IF EXISTS series_credits;

-- SQLite doesn't support DROP COLUMN
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
