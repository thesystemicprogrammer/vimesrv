package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SQLiteSeasonMetadataRepository implements SeasonMetadataRepository using SQLite
type SQLiteSeasonMetadataRepository struct {
	db *sql.DB
}

// NewSQLiteSeasonMetadataRepository creates a new SQLite season metadata repository
func NewSQLiteSeasonMetadataRepository(db *sql.DB) *SQLiteSeasonMetadataRepository {
	return &SQLiteSeasonMetadataRepository{db: db}
}

// Create inserts a new season metadata record
func (r *SQLiteSeasonMetadataRepository) Create(ctx context.Context, metadata *domain.SeasonMetadata) error {
	query := `
		INSERT INTO season_metadata (
			series_id, tmdb_id, season_number, air_date, poster_path,
			episode_count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		metadata.SeriesID,
		metadata.TMDBID,
		metadata.SeasonNumber,
		metadata.AirDate,
		metadata.PosterPath,
		metadata.EpisodeCount,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert season metadata: %w", err)
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

// Get retrieves a season metadata by its internal ID
func (r *SQLiteSeasonMetadataRepository) Get(ctx context.Context, id int64) (*domain.SeasonMetadata, error) {
	return r.scanSeason(ctx, "SELECT id, series_id, tmdb_id, season_number, air_date, poster_path, episode_count, created_at, updated_at FROM season_metadata WHERE id = ?", id)
}

// GetBySeriesAndNumber retrieves a season by series ID and season number
func (r *SQLiteSeasonMetadataRepository) GetBySeriesAndNumber(ctx context.Context, seriesID int64, seasonNumber int) (*domain.SeasonMetadata, error) {
	return r.scanSeason(ctx, "SELECT id, series_id, tmdb_id, season_number, air_date, poster_path, episode_count, created_at, updated_at FROM season_metadata WHERE series_id = ? AND season_number = ?", seriesID, seasonNumber)
}

// GetWithEpisodes retrieves a season with all its episodes loaded
func (r *SQLiteSeasonMetadataRepository) GetWithEpisodes(ctx context.Context, id int64) (*domain.SeasonMetadata, error) {
	season, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load episodes
	episodeRepo := NewSQLiteEpisodeMetadataRepository(r.db)
	episodes, err := episodeRepo.ListBySeasonID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load episodes: %w", err)
	}
	season.Episodes = episodes

	return season, nil
}

// ListBySeriesID retrieves all seasons for a given series
func (r *SQLiteSeasonMetadataRepository) ListBySeriesID(ctx context.Context, seriesID int64) ([]domain.SeasonMetadata, error) {
	query := `
		SELECT id, series_id, tmdb_id, season_number, air_date, poster_path, episode_count, created_at, updated_at
		FROM season_metadata
		WHERE series_id = ?
		ORDER BY season_number
	`

	rows, err := r.db.QueryContext(ctx, query, seriesID)
	if err != nil {
		return nil, fmt.Errorf("query seasons: %w", err)
	}
	defer rows.Close()

	var seasons []domain.SeasonMetadata
	for rows.Next() {
		var s domain.SeasonMetadata
		if err := rows.Scan(
			&s.ID,
			&s.SeriesID,
			&s.TMDBID,
			&s.SeasonNumber,
			&s.AirDate,
			&s.PosterPath,
			&s.EpisodeCount,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan season: %w", err)
		}
		seasons = append(seasons, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate seasons: %w", err)
	}

	return seasons, nil
}

// scanSeason scans a single season from the database
func (r *SQLiteSeasonMetadataRepository) scanSeason(ctx context.Context, query string, args ...interface{}) (*domain.SeasonMetadata, error) {
	metadata := &domain.SeasonMetadata{}

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&metadata.ID,
		&metadata.SeriesID,
		&metadata.TMDBID,
		&metadata.SeasonNumber,
		&metadata.AirDate,
		&metadata.PosterPath,
		&metadata.EpisodeCount,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("query season metadata: %w", err)
	}

	return metadata, nil
}

// Update updates an existing season metadata record
func (r *SQLiteSeasonMetadataRepository) Update(ctx context.Context, metadata *domain.SeasonMetadata) error {
	query := `
		UPDATE season_metadata SET
			series_id = ?, tmdb_id = ?, season_number = ?, air_date = ?,
			poster_path = ?, episode_count = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		metadata.SeriesID,
		metadata.TMDBID,
		metadata.SeasonNumber,
		metadata.AirDate,
		metadata.PosterPath,
		metadata.EpisodeCount,
		now,
		metadata.ID,
	)
	if err != nil {
		return fmt.Errorf("update season metadata: %w", err)
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

// Delete removes a season metadata record
func (r *SQLiteSeasonMetadataRepository) Delete(ctx context.Context, id int64) error {
	// Delete translations first
	_, err := r.db.ExecContext(ctx, "DELETE FROM season_metadata_translations WHERE season_metadata_id = ?", id)
	if err != nil {
		return fmt.Errorf("delete season translations: %w", err)
	}

	result, err := r.db.ExecContext(ctx, "DELETE FROM season_metadata WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete season metadata: %w", err)
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

// CreateTranslation creates a new translation for a season
func (r *SQLiteSeasonMetadataRepository) CreateTranslation(ctx context.Context, translation *domain.SeasonMetadataTranslation) error {
	query := `
		INSERT INTO season_metadata_translations (
			season_metadata_id, language, name, overview, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		translation.SeasonMetadataID,
		translation.Language,
		translation.Name,
		translation.Overview,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert season translation: %w", err)
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

// GetTranslation retrieves a specific translation for a season
func (r *SQLiteSeasonMetadataRepository) GetTranslation(ctx context.Context, seasonMetadataID int64, language string) (*domain.SeasonMetadataTranslation, error) {
	query := `
		SELECT id, season_metadata_id, language, name, overview, created_at, updated_at
		FROM season_metadata_translations
		WHERE season_metadata_id = ? AND language = ?
	`

	translation := &domain.SeasonMetadataTranslation{}
	err := r.db.QueryRowContext(ctx, query, seasonMetadataID, language).Scan(
		&translation.ID,
		&translation.SeasonMetadataID,
		&translation.Language,
		&translation.Name,
		&translation.Overview,
		&translation.CreatedAt,
		&translation.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("query season translation: %w", err)
	}

	return translation, nil
}

// GetTranslations retrieves all translations for a season
func (r *SQLiteSeasonMetadataRepository) GetTranslations(ctx context.Context, seasonMetadataID int64) ([]domain.SeasonMetadataTranslation, error) {
	query := `
		SELECT id, season_metadata_id, language, name, overview, created_at, updated_at
		FROM season_metadata_translations
		WHERE season_metadata_id = ?
		ORDER BY language
	`

	rows, err := r.db.QueryContext(ctx, query, seasonMetadataID)
	if err != nil {
		return nil, fmt.Errorf("query season translations: %w", err)
	}
	defer rows.Close()

	var translations []domain.SeasonMetadataTranslation
	for rows.Next() {
		var t domain.SeasonMetadataTranslation
		if err := rows.Scan(
			&t.ID,
			&t.SeasonMetadataID,
			&t.Language,
			&t.Name,
			&t.Overview,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan season translation: %w", err)
		}
		translations = append(translations, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate season translations: %w", err)
	}

	return translations, nil
}

// UpsertTranslation creates or updates a translation
func (r *SQLiteSeasonMetadataRepository) UpsertTranslation(ctx context.Context, translation *domain.SeasonMetadataTranslation) error {
	query := `
		INSERT INTO season_metadata_translations (
			season_metadata_id, language, name, overview, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(season_metadata_id, language) DO UPDATE SET
			name = excluded.name,
			overview = excluded.overview,
			updated_at = excluded.updated_at
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		translation.SeasonMetadataID,
		translation.Language,
		translation.Name,
		translation.Overview,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert season translation: %w", err)
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

// Ensure SQLiteSeasonMetadataRepository implements SeasonMetadataRepository
var _ ports.SeasonMetadataRepository = (*SQLiteSeasonMetadataRepository)(nil)
