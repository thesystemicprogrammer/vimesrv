package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SQLiteSimilarContentRepository implements SimilarContentRepository using SQLite
type SQLiteSimilarContentRepository struct {
	db *sql.DB
}

// NewSQLiteSimilarContentRepository creates a new SQLite similar content repository
func NewSQLiteSimilarContentRepository(db *sql.DB) *SQLiteSimilarContentRepository {
	return &SQLiteSimilarContentRepository{db: db}
}

// GetSimilarMovies retrieves cached similar movies for a movie metadata ID
func (r *SQLiteSimilarContentRepository) GetSimilarMovies(ctx context.Context, movieMetadataID int64, limit int) ([]ports.SimilarMovie, error) {
	query := `
		SELECT id, similar_tmdb_id, title, poster_path, release_date, vote_average
		FROM similar_movies
		WHERE movie_metadata_id = ?
		ORDER BY display_order
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, movieMetadataID, limit)
	if err != nil {
		return nil, fmt.Errorf("query similar movies: %w", err)
	}
	defer rows.Close()

	var movies []ports.SimilarMovie
	for rows.Next() {
		var m ports.SimilarMovie
		var posterPath, releaseDate sql.NullString
		if err := rows.Scan(
			&m.ID,
			&m.TMDBID,
			&m.Title,
			&posterPath,
			&releaseDate,
			&m.VoteAverage,
		); err != nil {
			return nil, fmt.Errorf("scan similar movie: %w", err)
		}
		m.PosterPath = posterPath.String
		m.ReleaseDate = releaseDate.String
		movies = append(movies, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate similar movies: %w", err)
	}

	return movies, nil
}

// SaveSimilarMovies saves similar movies to the cache, replacing any existing entries
func (r *SQLiteSimilarContentRepository) SaveSimilarMovies(ctx context.Context, movieMetadataID int64, movies []ports.SimilarMovie) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing entries
	if _, err := tx.ExecContext(ctx, "DELETE FROM similar_movies WHERE movie_metadata_id = ?", movieMetadataID); err != nil {
		return fmt.Errorf("delete existing similar movies: %w", err)
	}

	// Insert new entries
	query := `
		INSERT INTO similar_movies (movie_metadata_id, similar_tmdb_id, title, poster_path, release_date, vote_average, display_order, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	for i, m := range movies {
		var posterPath, releaseDate sql.NullString
		if m.PosterPath != "" {
			posterPath = sql.NullString{String: m.PosterPath, Valid: true}
		}
		if m.ReleaseDate != "" {
			releaseDate = sql.NullString{String: m.ReleaseDate, Valid: true}
		}

		if _, err := tx.ExecContext(ctx, query,
			movieMetadataID,
			m.TMDBID,
			m.Title,
			posterPath,
			releaseDate,
			m.VoteAverage,
			i,
			now,
		); err != nil {
			return fmt.Errorf("insert similar movie: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetSimilarMoviesFetchedAt returns the timestamp when similar movies were last fetched
func (r *SQLiteSimilarContentRepository) GetSimilarMoviesFetchedAt(ctx context.Context, movieMetadataID int64) (*time.Time, error) {
	var fetchedAt time.Time
	err := r.db.QueryRowContext(ctx,
		"SELECT fetched_at FROM similar_movies WHERE movie_metadata_id = ? LIMIT 1",
		movieMetadataID,
	).Scan(&fetchedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No cache entry exists
		}
		return nil, fmt.Errorf("query similar movies fetched_at: %w", err)
	}

	return &fetchedAt, nil
}

// DeleteSimilarMovies removes all cached similar movies for a movie metadata ID
func (r *SQLiteSimilarContentRepository) DeleteSimilarMovies(ctx context.Context, movieMetadataID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM similar_movies WHERE movie_metadata_id = ?", movieMetadataID)
	if err != nil {
		return fmt.Errorf("delete similar movies: %w", err)
	}
	return nil
}

// GetSimilarSeries retrieves cached similar series for a series metadata ID
func (r *SQLiteSimilarContentRepository) GetSimilarSeries(ctx context.Context, seriesMetadataID int64, limit int) ([]ports.SimilarSeries, error) {
	query := `
		SELECT id, similar_tmdb_id, name, poster_path, first_air_date, vote_average
		FROM similar_series
		WHERE series_metadata_id = ?
		ORDER BY display_order
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, seriesMetadataID, limit)
	if err != nil {
		return nil, fmt.Errorf("query similar series: %w", err)
	}
	defer rows.Close()

	var series []ports.SimilarSeries
	for rows.Next() {
		var s ports.SimilarSeries
		var posterPath, firstAirDate sql.NullString
		if err := rows.Scan(
			&s.ID,
			&s.TMDBID,
			&s.Name,
			&posterPath,
			&firstAirDate,
			&s.VoteAverage,
		); err != nil {
			return nil, fmt.Errorf("scan similar series: %w", err)
		}
		s.PosterPath = posterPath.String
		s.FirstAirDate = firstAirDate.String
		series = append(series, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate similar series: %w", err)
	}

	return series, nil
}

// SaveSimilarSeries saves similar series to the cache, replacing any existing entries
func (r *SQLiteSimilarContentRepository) SaveSimilarSeries(ctx context.Context, seriesMetadataID int64, series []ports.SimilarSeries) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing entries
	if _, err := tx.ExecContext(ctx, "DELETE FROM similar_series WHERE series_metadata_id = ?", seriesMetadataID); err != nil {
		return fmt.Errorf("delete existing similar series: %w", err)
	}

	// Insert new entries
	query := `
		INSERT INTO similar_series (series_metadata_id, similar_tmdb_id, name, poster_path, first_air_date, vote_average, display_order, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	for i, s := range series {
		var posterPath, firstAirDate sql.NullString
		if s.PosterPath != "" {
			posterPath = sql.NullString{String: s.PosterPath, Valid: true}
		}
		if s.FirstAirDate != "" {
			firstAirDate = sql.NullString{String: s.FirstAirDate, Valid: true}
		}

		if _, err := tx.ExecContext(ctx, query,
			seriesMetadataID,
			s.TMDBID,
			s.Name,
			posterPath,
			firstAirDate,
			s.VoteAverage,
			i,
			now,
		); err != nil {
			return fmt.Errorf("insert similar series: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetSimilarSeriesFetchedAt returns the timestamp when similar series were last fetched
func (r *SQLiteSimilarContentRepository) GetSimilarSeriesFetchedAt(ctx context.Context, seriesMetadataID int64) (*time.Time, error) {
	var fetchedAt time.Time
	err := r.db.QueryRowContext(ctx,
		"SELECT fetched_at FROM similar_series WHERE series_metadata_id = ? LIMIT 1",
		seriesMetadataID,
	).Scan(&fetchedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No cache entry exists
		}
		return nil, fmt.Errorf("query similar series fetched_at: %w", err)
	}

	return &fetchedAt, nil
}

// DeleteSimilarSeries removes all cached similar series for a series metadata ID
func (r *SQLiteSimilarContentRepository) DeleteSimilarSeries(ctx context.Context, seriesMetadataID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM similar_series WHERE series_metadata_id = ?", seriesMetadataID)
	if err != nil {
		return fmt.Errorf("delete similar series: %w", err)
	}
	return nil
}

// GetSimilarMovieTranslation retrieves a translation for a specific similar movie
func (r *SQLiteSimilarContentRepository) GetSimilarMovieTranslation(ctx context.Context, similarMovieID int64, language string) (*ports.SimilarMovieTranslation, error) {
	query := `
		SELECT similar_movie_id, language, title
		FROM similar_movie_translations
		WHERE similar_movie_id = ? AND language = ?
	`

	var t ports.SimilarMovieTranslation
	err := r.db.QueryRowContext(ctx, query, similarMovieID, language).Scan(
		&t.SimilarMovieID,
		&t.Language,
		&t.Title,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query similar movie translation: %w", err)
	}

	return &t, nil
}

// GetSimilarMovieTranslations retrieves translations for all similar movies of a movie metadata
// Returns a map of similarMovieID -> translated title
func (r *SQLiteSimilarContentRepository) GetSimilarMovieTranslations(ctx context.Context, movieMetadataID int64, language string) (map[int64]string, error) {
	query := `
		SELECT smt.similar_movie_id, smt.title
		FROM similar_movie_translations smt
		INNER JOIN similar_movies sm ON sm.id = smt.similar_movie_id
		WHERE sm.movie_metadata_id = ? AND smt.language = ?
	`

	rows, err := r.db.QueryContext(ctx, query, movieMetadataID, language)
	if err != nil {
		return nil, fmt.Errorf("query similar movie translations: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]string)
	for rows.Next() {
		var id int64
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, fmt.Errorf("scan similar movie translation: %w", err)
		}
		result[id] = title
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate similar movie translations: %w", err)
	}

	return result, nil
}

// SaveSimilarMovieTranslation saves a single translation for a similar movie
func (r *SQLiteSimilarContentRepository) SaveSimilarMovieTranslation(ctx context.Context, translation *ports.SimilarMovieTranslation) error {
	query := `
		INSERT INTO similar_movie_translations (similar_movie_id, language, title, fetched_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(similar_movie_id, language) DO UPDATE SET
			title = excluded.title,
			fetched_at = excluded.fetched_at
	`

	_, err := r.db.ExecContext(ctx, query,
		translation.SimilarMovieID,
		translation.Language,
		translation.Title,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("save similar movie translation: %w", err)
	}

	return nil
}

// SaveSimilarMovieTranslations saves multiple translations for similar movies
func (r *SQLiteSimilarContentRepository) SaveSimilarMovieTranslations(ctx context.Context, translations []ports.SimilarMovieTranslation) error {
	if len(translations) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO similar_movie_translations (similar_movie_id, language, title, fetched_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(similar_movie_id, language) DO UPDATE SET
			title = excluded.title,
			fetched_at = excluded.fetched_at
	`

	now := time.Now()
	for _, t := range translations {
		if _, err := tx.ExecContext(ctx, query,
			t.SimilarMovieID,
			t.Language,
			t.Title,
			now,
		); err != nil {
			return fmt.Errorf("insert similar movie translation: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetSimilarSeriesTranslation retrieves a translation for a specific similar series
func (r *SQLiteSimilarContentRepository) GetSimilarSeriesTranslation(ctx context.Context, similarSeriesID int64, language string) (*ports.SimilarSeriesTranslation, error) {
	query := `
		SELECT similar_series_id, language, name
		FROM similar_series_translations
		WHERE similar_series_id = ? AND language = ?
	`

	var t ports.SimilarSeriesTranslation
	err := r.db.QueryRowContext(ctx, query, similarSeriesID, language).Scan(
		&t.SimilarSeriesID,
		&t.Language,
		&t.Name,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query similar series translation: %w", err)
	}

	return &t, nil
}

// GetSimilarSeriesTranslations retrieves translations for all similar series of a series metadata
// Returns a map of similarSeriesID -> translated name
func (r *SQLiteSimilarContentRepository) GetSimilarSeriesTranslations(ctx context.Context, seriesMetadataID int64, language string) (map[int64]string, error) {
	query := `
		SELECT sst.similar_series_id, sst.name
		FROM similar_series_translations sst
		INNER JOIN similar_series ss ON ss.id = sst.similar_series_id
		WHERE ss.series_metadata_id = ? AND sst.language = ?
	`

	rows, err := r.db.QueryContext(ctx, query, seriesMetadataID, language)
	if err != nil {
		return nil, fmt.Errorf("query similar series translations: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]string)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan similar series translation: %w", err)
		}
		result[id] = name
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate similar series translations: %w", err)
	}

	return result, nil
}

// SaveSimilarSeriesTranslation saves a single translation for a similar series
func (r *SQLiteSimilarContentRepository) SaveSimilarSeriesTranslation(ctx context.Context, translation *ports.SimilarSeriesTranslation) error {
	query := `
		INSERT INTO similar_series_translations (similar_series_id, language, name, fetched_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(similar_series_id, language) DO UPDATE SET
			name = excluded.name,
			fetched_at = excluded.fetched_at
	`

	_, err := r.db.ExecContext(ctx, query,
		translation.SimilarSeriesID,
		translation.Language,
		translation.Name,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("save similar series translation: %w", err)
	}

	return nil
}

// SaveSimilarSeriesTranslations saves multiple translations for similar series
func (r *SQLiteSimilarContentRepository) SaveSimilarSeriesTranslations(ctx context.Context, translations []ports.SimilarSeriesTranslation) error {
	if len(translations) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO similar_series_translations (similar_series_id, language, name, fetched_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(similar_series_id, language) DO UPDATE SET
			name = excluded.name,
			fetched_at = excluded.fetched_at
	`

	now := time.Now()
	for _, t := range translations {
		if _, err := tx.ExecContext(ctx, query,
			t.SimilarSeriesID,
			t.Language,
			t.Name,
			now,
		); err != nil {
			return fmt.Errorf("insert similar series translation: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Ensure SQLiteSimilarContentRepository implements SimilarContentRepository
var _ ports.SimilarContentRepository = (*SQLiteSimilarContentRepository)(nil)
