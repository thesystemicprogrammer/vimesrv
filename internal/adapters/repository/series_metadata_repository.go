package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SQLiteSeriesMetadataRepository implements SeriesMetadataRepository using SQLite
type SQLiteSeriesMetadataRepository struct {
	db *sql.DB
}

// NewSQLiteSeriesMetadataRepository creates a new SQLite series metadata repository
func NewSQLiteSeriesMetadataRepository(db *sql.DB) *SQLiteSeriesMetadataRepository {
	return &SQLiteSeriesMetadataRepository{db: db}
}

// Create inserts a new series metadata record
func (r *SQLiteSeriesMetadataRepository) Create(ctx context.Context, metadata *domain.SeriesMetadata) error {
	genresJSON, err := json.Marshal(metadata.Genres)
	if err != nil {
		return fmt.Errorf("marshal genres: %w", err)
	}
	networksJSON, err := json.Marshal(metadata.Networks)
	if err != nil {
		return fmt.Errorf("marshal networks: %w", err)
	}

	query := `
		INSERT INTO series_metadata (
			tmdb_id, original_name, first_air_date, last_air_date, status,
			poster_path, backdrop_path, genres, networks, vote_average, vote_count,
			popularity, number_of_seasons, number_of_episodes, original_lang,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		metadata.TMDBID,
		metadata.OriginalName,
		metadata.FirstAirDate,
		metadata.LastAirDate,
		metadata.Status,
		metadata.PosterPath,
		metadata.BackdropPath,
		string(genresJSON),
		string(networksJSON),
		metadata.VoteAverage,
		metadata.VoteCount,
		metadata.Popularity,
		metadata.NumberOfSeasons,
		metadata.NumberOfEpisodes,
		metadata.OriginalLang,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert series metadata: %w", err)
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

// Get retrieves a series metadata by its internal ID
func (r *SQLiteSeriesMetadataRepository) Get(ctx context.Context, id int64) (*domain.SeriesMetadata, error) {
	return r.scanSeries(ctx, "SELECT id, tmdb_id, original_name, first_air_date, last_air_date, status, poster_path, backdrop_path, genres, networks, vote_average, vote_count, popularity, number_of_seasons, number_of_episodes, original_lang, created_at, updated_at FROM series_metadata WHERE id = ?", id)
}

// GetByTMDBID retrieves a series metadata by its TMDB ID
func (r *SQLiteSeriesMetadataRepository) GetByTMDBID(ctx context.Context, tmdbID int) (*domain.SeriesMetadata, error) {
	return r.scanSeries(ctx, "SELECT id, tmdb_id, original_name, first_air_date, last_air_date, status, poster_path, backdrop_path, genres, networks, vote_average, vote_count, popularity, number_of_seasons, number_of_episodes, original_lang, created_at, updated_at FROM series_metadata WHERE tmdb_id = ?", tmdbID)
}

// GetWithSeasons retrieves a series with all its seasons loaded
func (r *SQLiteSeriesMetadataRepository) GetWithSeasons(ctx context.Context, id int64) (*domain.SeriesMetadata, error) {
	series, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load seasons
	seasonRepo := NewSQLiteSeasonMetadataRepository(r.db)
	seasons, err := seasonRepo.ListBySeriesID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load seasons: %w", err)
	}
	series.Seasons = seasons

	return series, nil
}

// scanSeries scans a single series from the database
func (r *SQLiteSeriesMetadataRepository) scanSeries(ctx context.Context, query string, args ...interface{}) (*domain.SeriesMetadata, error) {
	metadata := &domain.SeriesMetadata{}
	var genresJSON, networksJSON string

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&metadata.ID,
		&metadata.TMDBID,
		&metadata.OriginalName,
		&metadata.FirstAirDate,
		&metadata.LastAirDate,
		&metadata.Status,
		&metadata.PosterPath,
		&metadata.BackdropPath,
		&genresJSON,
		&networksJSON,
		&metadata.VoteAverage,
		&metadata.VoteCount,
		&metadata.Popularity,
		&metadata.NumberOfSeasons,
		&metadata.NumberOfEpisodes,
		&metadata.OriginalLang,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("query series metadata: %w", err)
	}

	if err := json.Unmarshal([]byte(genresJSON), &metadata.Genres); err != nil {
		metadata.Genres = []string{}
	}
	if err := json.Unmarshal([]byte(networksJSON), &metadata.Networks); err != nil {
		metadata.Networks = []string{}
	}

	return metadata, nil
}

// Update updates an existing series metadata record
func (r *SQLiteSeriesMetadataRepository) Update(ctx context.Context, metadata *domain.SeriesMetadata) error {
	genresJSON, err := json.Marshal(metadata.Genres)
	if err != nil {
		return fmt.Errorf("marshal genres: %w", err)
	}
	networksJSON, err := json.Marshal(metadata.Networks)
	if err != nil {
		return fmt.Errorf("marshal networks: %w", err)
	}

	query := `
		UPDATE series_metadata SET
			tmdb_id = ?, original_name = ?, first_air_date = ?, last_air_date = ?,
			status = ?, poster_path = ?, backdrop_path = ?, genres = ?, networks = ?,
			vote_average = ?, vote_count = ?, popularity = ?, number_of_seasons = ?,
			number_of_episodes = ?, original_lang = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		metadata.TMDBID,
		metadata.OriginalName,
		metadata.FirstAirDate,
		metadata.LastAirDate,
		metadata.Status,
		metadata.PosterPath,
		metadata.BackdropPath,
		string(genresJSON),
		string(networksJSON),
		metadata.VoteAverage,
		metadata.VoteCount,
		metadata.Popularity,
		metadata.NumberOfSeasons,
		metadata.NumberOfEpisodes,
		metadata.OriginalLang,
		now,
		metadata.ID,
	)
	if err != nil {
		return fmt.Errorf("update series metadata: %w", err)
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

// Delete removes a series metadata record
func (r *SQLiteSeriesMetadataRepository) Delete(ctx context.Context, id int64) error {
	// Delete translations first
	_, err := r.db.ExecContext(ctx, "DELETE FROM series_metadata_translations WHERE series_metadata_id = ?", id)
	if err != nil {
		return fmt.Errorf("delete series translations: %w", err)
	}

	result, err := r.db.ExecContext(ctx, "DELETE FROM series_metadata WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete series metadata: %w", err)
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

// ExistsByTMDBID checks if a series metadata with the given TMDB ID exists
func (r *SQLiteSeriesMetadataRepository) ExistsByTMDBID(ctx context.Context, tmdbID int) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		"SELECT 1 FROM series_metadata WHERE tmdb_id = ? LIMIT 1",
		tmdbID,
	).Scan(&exists)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check series metadata exists: %w", err)
	}

	return true, nil
}

// CreateTranslation creates a new translation for a series
func (r *SQLiteSeriesMetadataRepository) CreateTranslation(ctx context.Context, translation *domain.SeriesMetadataTranslation) error {
	query := `
		INSERT INTO series_metadata_translations (
			series_metadata_id, language, name, overview, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		translation.SeriesMetadataID,
		translation.Language,
		translation.Name,
		translation.Overview,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert series translation: %w", err)
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

// GetTranslation retrieves a specific translation for a series
func (r *SQLiteSeriesMetadataRepository) GetTranslation(ctx context.Context, seriesMetadataID int64, language string) (*domain.SeriesMetadataTranslation, error) {
	query := `
		SELECT id, series_metadata_id, language, name, overview, created_at, updated_at
		FROM series_metadata_translations
		WHERE series_metadata_id = ? AND language = ?
	`

	translation := &domain.SeriesMetadataTranslation{}
	err := r.db.QueryRowContext(ctx, query, seriesMetadataID, language).Scan(
		&translation.ID,
		&translation.SeriesMetadataID,
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
		return nil, fmt.Errorf("query series translation: %w", err)
	}

	return translation, nil
}

// GetTranslations retrieves all translations for a series
func (r *SQLiteSeriesMetadataRepository) GetTranslations(ctx context.Context, seriesMetadataID int64) ([]domain.SeriesMetadataTranslation, error) {
	query := `
		SELECT id, series_metadata_id, language, name, overview, created_at, updated_at
		FROM series_metadata_translations
		WHERE series_metadata_id = ?
		ORDER BY language
	`

	rows, err := r.db.QueryContext(ctx, query, seriesMetadataID)
	if err != nil {
		return nil, fmt.Errorf("query series translations: %w", err)
	}
	defer rows.Close()

	var translations []domain.SeriesMetadataTranslation
	for rows.Next() {
		var t domain.SeriesMetadataTranslation
		if err := rows.Scan(
			&t.ID,
			&t.SeriesMetadataID,
			&t.Language,
			&t.Name,
			&t.Overview,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan series translation: %w", err)
		}
		translations = append(translations, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate series translations: %w", err)
	}

	return translations, nil
}

// UpsertTranslation creates or updates a translation
func (r *SQLiteSeriesMetadataRepository) UpsertTranslation(ctx context.Context, translation *domain.SeriesMetadataTranslation) error {
	query := `
		INSERT INTO series_metadata_translations (
			series_metadata_id, language, name, overview, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(series_metadata_id, language) DO UPDATE SET
			name = excluded.name,
			overview = excluded.overview,
			updated_at = excluded.updated_at
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		translation.SeriesMetadataID,
		translation.Language,
		translation.Name,
		translation.Overview,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert series translation: %w", err)
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

// Ensure SQLiteSeriesMetadataRepository implements SeriesMetadataRepository
var _ ports.SeriesMetadataRepository = (*SQLiteSeriesMetadataRepository)(nil)
