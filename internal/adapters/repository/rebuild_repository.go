package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SQLiteRebuildRepository implements the RebuildRepository interface for SQLite
type SQLiteRebuildRepository struct {
	db *sql.DB
}

// NewSQLiteRebuildRepository creates a new rebuild repository
func NewSQLiteRebuildRepository(db *database.DB) *SQLiteRebuildRepository {
	return &SQLiteRebuildRepository{db: db.DB}
}

// ExportMediaLinks returns all media files that are linked to metadata
func (r *SQLiteRebuildRepository) ExportMediaLinks(ctx context.Context) ([]ports.ExportedMediaLink, error) {
	var links []ports.ExportedMediaLink

	// Export movie links
	movieLinks, err := r.exportMovieLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export movie links: %w", err)
	}
	links = append(links, movieLinks...)

	// Export episode links
	episodeLinks, err := r.exportEpisodeLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export episode links: %w", err)
	}
	links = append(links, episodeLinks...)

	return links, nil
}

// exportMovieLinks exports all media files linked to movie metadata
func (r *SQLiteRebuildRepository) exportMovieLinks(ctx context.Context) ([]ports.ExportedMediaLink, error) {
	const query = `
		SELECT mf.fingerprint, mm.tmdb_id, mf.edition
		FROM media_files mf
		JOIN movie_metadata mm ON mf.movie_metadata_id = mm.id
		WHERE mf.metadata_type = 'movie' AND mf.movie_metadata_id IS NOT NULL
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []ports.ExportedMediaLink
	for rows.Next() {
		var link ports.ExportedMediaLink
		var edition sql.NullString

		if err := rows.Scan(&link.Fingerprint, &link.TMDBID, &edition); err != nil {
			return nil, err
		}

		link.MetadataType = "movie"
		if edition.Valid && edition.String != "" {
			link.Edition = &edition.String
		}

		links = append(links, link)
	}

	return links, rows.Err()
}

// exportEpisodeLinks exports all media files linked to episode metadata
func (r *SQLiteRebuildRepository) exportEpisodeLinks(ctx context.Context) ([]ports.ExportedMediaLink, error) {
	const query = `
		SELECT mf.fingerprint, sm.tmdb_id, sem.season_number, em.episode_number
		FROM media_files mf
		JOIN episode_metadata em ON mf.episode_metadata_id = em.id
		JOIN season_metadata sem ON em.season_id = sem.id
		JOIN series_metadata sm ON sem.series_id = sm.id
		WHERE mf.metadata_type = 'episode' AND mf.episode_metadata_id IS NOT NULL
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []ports.ExportedMediaLink
	for rows.Next() {
		var link ports.ExportedMediaLink

		if err := rows.Scan(&link.Fingerprint, &link.SeriesTMDBID, &link.SeasonNumber, &link.EpisodeNumber); err != nil {
			return nil, err
		}

		link.MetadataType = "episode"
		links = append(links, link)
	}

	return links, rows.Err()
}

// ClearAllTables deletes all data from all tables and runs VACUUM
func (r *SQLiteRebuildRepository) ClearAllTables(ctx context.Context) error {
	// Order matters for FK constraints - delete children before parents
	tables := []string{
		// Jobs and schedules (no deps)
		"jobs",
		"schedules",
		// Transcodes and streams (depend on media_files)
		"transcodes",
		"audio_streams",
		"subtitle_streams",
		"metadata_candidates",
		// Media files (depend on metadata)
		"media_files",
		// Movie metadata hierarchy
		"movie_credits",
		"movie_certifications",
		"movie_metadata_translations",
		"similar_movies",
		"movie_metadata",
		"collection_metadata",
		// Series metadata hierarchy
		"episode_metadata_translations",
		"episode_metadata",
		"season_metadata_translations",
		"season_metadata",
		"series_credits",
		"series_metadata_translations",
		"similar_series",
		"series_metadata",
		// Users
		"users",
	}

	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s", table)
		_, err := r.db.ExecContext(ctx, query)
		if err != nil {
			// Log but continue - table might not exist or be empty
			logger.Warn().Err(err).Str("table", table).Msg("[rebuild] Failed to clear table (may not exist)")
		} else {
			logger.Info().Str("table", table).Msg("[rebuild] Cleared table")
		}
	}

	// Reclaim disk space
	logger.Info().Msg("[rebuild] Running VACUUM to reclaim disk space")
	_, err := r.db.ExecContext(ctx, "VACUUM")
	if err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	return nil
}

// ImportUser imports a user with the exact ID and password hash from the export
func (r *SQLiteRebuildRepository) ImportUser(ctx context.Context, user ports.ExportedUser) error {
	const query = `
		INSERT INTO users (id, username, password_hash, role, must_change_password, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	mustChangePassword := 0
	if user.MustChangePassword {
		mustChangePassword = 1
	}

	// Parse timestamps
	createdAt, err := time.Parse(time.RFC3339, user.CreatedAt)
	if err != nil {
		createdAt = time.Now().UTC()
	}
	updatedAt, err := time.Parse(time.RFC3339, user.UpdatedAt)
	if err != nil {
		updatedAt = time.Now().UTC()
	}

	var createdBy sql.NullString
	if user.CreatedBy != nil {
		createdBy = sql.NullString{String: *user.CreatedBy, Valid: true}
	}

	_, err = r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.Role,
		mustChangePassword,
		createdAt,
		updatedAt,
		createdBy,
	)
	if err != nil {
		return fmt.Errorf("failed to import user %s: %w", user.Username, err)
	}

	return nil
}
