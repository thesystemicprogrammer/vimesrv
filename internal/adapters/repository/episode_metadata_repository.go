package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SQLiteEpisodeMetadataRepository implements EpisodeMetadataRepository using SQLite
type SQLiteEpisodeMetadataRepository struct {
	db *sql.DB
}

// NewSQLiteEpisodeMetadataRepository creates a new SQLite episode metadata repository
func NewSQLiteEpisodeMetadataRepository(db *sql.DB) *SQLiteEpisodeMetadataRepository {
	return &SQLiteEpisodeMetadataRepository{db: db}
}

// Create inserts a new episode metadata record
func (r *SQLiteEpisodeMetadataRepository) Create(ctx context.Context, metadata *domain.EpisodeMetadata) error {
	query := `
		INSERT INTO episode_metadata (
			season_id, tmdb_id, episode_number, air_date, still_path,
			runtime, vote_average, vote_count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		metadata.SeasonID,
		metadata.TMDBID,
		metadata.EpisodeNumber,
		metadata.AirDate,
		metadata.StillPath,
		metadata.Runtime,
		metadata.VoteAverage,
		metadata.VoteCount,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert episode metadata: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	metadata.ID = id
	metadata.CreatedAt = now
	metadata.UpdatedAt = now

	return nil
}

// Get retrieves an episode metadata by its internal ID
func (r *SQLiteEpisodeMetadataRepository) Get(ctx context.Context, id int64) (*domain.EpisodeMetadata, error) {
	return r.scanEpisode(ctx, "SELECT id, season_id, tmdb_id, episode_number, air_date, still_path, runtime, vote_average, vote_count, created_at, updated_at FROM episode_metadata WHERE id = ?", id)
}

// GetBySeasonAndNumber retrieves an episode by season ID and episode number
func (r *SQLiteEpisodeMetadataRepository) GetBySeasonAndNumber(ctx context.Context, seasonID int64, episodeNumber int) (*domain.EpisodeMetadata, error) {
	return r.scanEpisode(ctx, "SELECT id, season_id, tmdb_id, episode_number, air_date, still_path, runtime, vote_average, vote_count, created_at, updated_at FROM episode_metadata WHERE season_id = ? AND episode_number = ?", seasonID, episodeNumber)
}

// ListBySeasonID retrieves all episodes for a given season
func (r *SQLiteEpisodeMetadataRepository) ListBySeasonID(ctx context.Context, seasonID int64) ([]domain.EpisodeMetadata, error) {
	query := `
		SELECT id, season_id, tmdb_id, episode_number, air_date, still_path, runtime, vote_average, vote_count, created_at, updated_at
		FROM episode_metadata
		WHERE season_id = ?
		ORDER BY episode_number
	`

	rows, err := r.db.QueryContext(ctx, query, seasonID)
	if err != nil {
		return nil, fmt.Errorf("query episodes: %w", err)
	}
	defer rows.Close()

	var episodes []domain.EpisodeMetadata
	for rows.Next() {
		var e domain.EpisodeMetadata
		if err := rows.Scan(
			&e.ID,
			&e.SeasonID,
			&e.TMDBID,
			&e.EpisodeNumber,
			&e.AirDate,
			&e.StillPath,
			&e.Runtime,
			&e.VoteAverage,
			&e.VoteCount,
			&e.CreatedAt,
			&e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan episode: %w", err)
		}
		episodes = append(episodes, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate episodes: %w", err)
	}

	return episodes, nil
}

// scanEpisode scans a single episode from the database
func (r *SQLiteEpisodeMetadataRepository) scanEpisode(ctx context.Context, query string, args ...interface{}) (*domain.EpisodeMetadata, error) {
	metadata := &domain.EpisodeMetadata{}

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&metadata.ID,
		&metadata.SeasonID,
		&metadata.TMDBID,
		&metadata.EpisodeNumber,
		&metadata.AirDate,
		&metadata.StillPath,
		&metadata.Runtime,
		&metadata.VoteAverage,
		&metadata.VoteCount,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("query episode metadata: %w", err)
	}

	return metadata, nil
}

// Update updates an existing episode metadata record
func (r *SQLiteEpisodeMetadataRepository) Update(ctx context.Context, metadata *domain.EpisodeMetadata) error {
	query := `
		UPDATE episode_metadata SET
			season_id = ?, tmdb_id = ?, episode_number = ?, air_date = ?,
			still_path = ?, runtime = ?, vote_average = ?, vote_count = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		metadata.SeasonID,
		metadata.TMDBID,
		metadata.EpisodeNumber,
		metadata.AirDate,
		metadata.StillPath,
		metadata.Runtime,
		metadata.VoteAverage,
		metadata.VoteCount,
		now,
		metadata.ID,
	)
	if err != nil {
		return fmt.Errorf("update episode metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrNotFound
	}

	metadata.UpdatedAt = now
	return nil
}

// Delete removes an episode metadata record
func (r *SQLiteEpisodeMetadataRepository) Delete(ctx context.Context, id int64) error {
	// Delete translations first
	_, err := r.db.ExecContext(ctx, "DELETE FROM episode_metadata_translations WHERE episode_metadata_id = ?", id)
	if err != nil {
		return fmt.Errorf("delete episode translations: %w", err)
	}

	result, err := r.db.ExecContext(ctx, "DELETE FROM episode_metadata WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete episode metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrNotFound
	}

	return nil
}

// CreateTranslation creates a new translation for an episode
func (r *SQLiteEpisodeMetadataRepository) CreateTranslation(ctx context.Context, translation *domain.EpisodeMetadataTranslation) error {
	query := `
		INSERT INTO episode_metadata_translations (
			episode_metadata_id, language, name, overview, still_path, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		translation.EpisodeMetadataID,
		translation.Language,
		translation.Name,
		translation.Overview,
		translation.StillPath,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert episode translation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	translation.ID = id
	translation.CreatedAt = now
	translation.UpdatedAt = now

	return nil
}

// GetTranslation retrieves a specific translation for an episode
func (r *SQLiteEpisodeMetadataRepository) GetTranslation(ctx context.Context, episodeMetadataID int64, language string) (*domain.EpisodeMetadataTranslation, error) {
	query := `
		SELECT id, episode_metadata_id, language, name, overview, still_path, created_at, updated_at
		FROM episode_metadata_translations
		WHERE episode_metadata_id = ? AND language = ?
	`

	translation := &domain.EpisodeMetadataTranslation{}
	err := r.db.QueryRowContext(ctx, query, episodeMetadataID, language).Scan(
		&translation.ID,
		&translation.EpisodeMetadataID,
		&translation.Language,
		&translation.Name,
		&translation.Overview,
		&translation.StillPath,
		&translation.CreatedAt,
		&translation.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("query episode translation: %w", err)
	}

	return translation, nil
}

// GetTranslations retrieves all translations for an episode
func (r *SQLiteEpisodeMetadataRepository) GetTranslations(ctx context.Context, episodeMetadataID int64) ([]domain.EpisodeMetadataTranslation, error) {
	query := `
		SELECT id, episode_metadata_id, language, name, overview, still_path, created_at, updated_at
		FROM episode_metadata_translations
		WHERE episode_metadata_id = ?
		ORDER BY language
	`

	rows, err := r.db.QueryContext(ctx, query, episodeMetadataID)
	if err != nil {
		return nil, fmt.Errorf("query episode translations: %w", err)
	}
	defer rows.Close()

	var translations []domain.EpisodeMetadataTranslation
	for rows.Next() {
		var t domain.EpisodeMetadataTranslation
		if err := rows.Scan(
			&t.ID,
			&t.EpisodeMetadataID,
			&t.Language,
			&t.Name,
			&t.Overview,
			&t.StillPath,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan episode translation: %w", err)
		}
		translations = append(translations, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate episode translations: %w", err)
	}

	return translations, nil
}

// UpsertTranslation creates or updates a translation
func (r *SQLiteEpisodeMetadataRepository) UpsertTranslation(ctx context.Context, translation *domain.EpisodeMetadataTranslation) error {
	query := `
		INSERT INTO episode_metadata_translations (
			episode_metadata_id, language, name, overview, still_path, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(episode_metadata_id, language) DO UPDATE SET
			name = excluded.name,
			overview = excluded.overview,
			still_path = excluded.still_path,
			updated_at = excluded.updated_at
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		translation.EpisodeMetadataID,
		translation.Language,
		translation.Name,
		translation.Overview,
		translation.StillPath,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert episode translation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	if id > 0 {
		translation.ID = id
		translation.CreatedAt = now
	}
	translation.UpdatedAt = now

	return nil
}

// ListIDsWithoutTranslation returns episode metadata IDs that don't have a translation for the given language
// or have a translation but are missing image paths
func (r *SQLiteEpisodeMetadataRepository) ListIDsWithoutTranslation(ctx context.Context, language string) ([]ports.EpisodeMetadataForTranslation, error) {
	exact, base := episodeLanguageParams(language)

	// Join with season_metadata and series_metadata to get season_number and series tmdb_id
	query := `
		SELECT ep.id, ep.episode_number, sea.season_number, ser.tmdb_id
		FROM episode_metadata ep
		JOIN season_metadata sea ON ep.season_id = sea.id
		JOIN series_metadata ser ON sea.series_id = ser.id
		WHERE NOT EXISTS (
			SELECT 1 FROM episode_metadata_translations t
			WHERE t.episode_metadata_id = ep.id 
			AND (t.language = ? OR t.language LIKE ? || '%')
		)
		OR EXISTS (
			SELECT 1 FROM episode_metadata_translations t
			WHERE t.episode_metadata_id = ep.id 
			AND (t.language = ? OR t.language LIKE ? || '%')
			AND (t.still_path IS NULL OR t.still_path = '')
		)
		ORDER BY ep.id
	`

	rows, err := r.db.QueryContext(ctx, query, exact, base, exact, base)
	if err != nil {
		return nil, fmt.Errorf("query episodes without translation: %w", err)
	}
	defer rows.Close()

	var results []ports.EpisodeMetadataForTranslation
	for rows.Next() {
		var e ports.EpisodeMetadataForTranslation
		if err := rows.Scan(&e.ID, &e.EpisodeNumber, &e.SeasonNumber, &e.SeriesTMDBID); err != nil {
			return nil, fmt.Errorf("scan episode: %w", err)
		}
		results = append(results, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate episodes: %w", err)
	}

	return results, nil
}

// episodeLanguageParams returns the exact language and base language for fallback queries
func episodeLanguageParams(lang string) (exact string, base string) {
	if lang == "" {
		return "en", "en"
	}
	if idx := strings.IndexAny(lang, "-_"); idx > 0 {
		return lang, lang[:idx]
	}
	return lang, lang
}

// Ensure SQLiteEpisodeMetadataRepository implements EpisodeMetadataRepository
var _ ports.EpisodeMetadataRepository = (*SQLiteEpisodeMetadataRepository)(nil)
