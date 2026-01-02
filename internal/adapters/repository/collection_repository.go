package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SQLiteCollectionRepository implements CollectionRepository using SQLite
type SQLiteCollectionRepository struct {
	db *sql.DB
}

// NewSQLiteCollectionRepository creates a new SQLite collection repository
func NewSQLiteCollectionRepository(db *sql.DB) *SQLiteCollectionRepository {
	return &SQLiteCollectionRepository{db: db}
}

// GetCollectionMetadata retrieves cached collection metadata
func (r *SQLiteCollectionRepository) GetCollectionMetadata(ctx context.Context, collectionID int) (*ports.CollectionMetadata, error) {
	query := `
		SELECT id, collection_id, name, overview, poster_path, backdrop_path, fetched_at
		FROM collection_metadata
		WHERE collection_id = ?
	`

	var m ports.CollectionMetadata
	var overview, posterPath, backdropPath sql.NullString
	err := r.db.QueryRowContext(ctx, query, collectionID).Scan(
		&m.ID,
		&m.CollectionID,
		&m.Name,
		&overview,
		&posterPath,
		&backdropPath,
		&m.FetchedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query collection metadata: %w", err)
	}

	m.Overview = overview.String
	m.PosterPath = posterPath.String
	m.BackdropPath = backdropPath.String

	return &m, nil
}

// SaveCollectionMetadata saves collection metadata to the cache
func (r *SQLiteCollectionRepository) SaveCollectionMetadata(ctx context.Context, metadata *ports.CollectionMetadata) error {
	query := `
		INSERT INTO collection_metadata (collection_id, name, overview, poster_path, backdrop_path, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(collection_id) DO UPDATE SET
			name = excluded.name,
			overview = excluded.overview,
			poster_path = excluded.poster_path,
			backdrop_path = excluded.backdrop_path,
			fetched_at = excluded.fetched_at
	`

	var overview, posterPath, backdropPath sql.NullString
	if metadata.Overview != "" {
		overview = sql.NullString{String: metadata.Overview, Valid: true}
	}
	if metadata.PosterPath != "" {
		posterPath = sql.NullString{String: metadata.PosterPath, Valid: true}
	}
	if metadata.BackdropPath != "" {
		backdropPath = sql.NullString{String: metadata.BackdropPath, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		metadata.CollectionID,
		metadata.Name,
		overview,
		posterPath,
		backdropPath,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("save collection metadata: %w", err)
	}

	return nil
}

// GetCollectionTranslation retrieves cached collection translation for a language
func (r *SQLiteCollectionRepository) GetCollectionTranslation(ctx context.Context, collectionID int, language string) (*ports.CollectionTranslation, error) {
	query := `
		SELECT id, collection_id, language, name, overview, fetched_at
		FROM collection_translations
		WHERE collection_id = ? AND language = ?
	`

	var t ports.CollectionTranslation
	var name, overview sql.NullString
	err := r.db.QueryRowContext(ctx, query, collectionID, language).Scan(
		&t.ID,
		&t.CollectionID,
		&t.Language,
		&name,
		&overview,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query collection translation: %w", err)
	}

	t.Name = name.String
	t.Overview = overview.String

	return &t, nil
}

// SaveCollectionTranslation saves collection translation to the cache
func (r *SQLiteCollectionRepository) SaveCollectionTranslation(ctx context.Context, translation *ports.CollectionTranslation) error {
	query := `
		INSERT INTO collection_translations (collection_id, language, name, overview, fetched_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(collection_id, language) DO UPDATE SET
			name = excluded.name,
			overview = excluded.overview,
			fetched_at = excluded.fetched_at
	`

	var name, overview sql.NullString
	if translation.Name != "" {
		name = sql.NullString{String: translation.Name, Valid: true}
	}
	if translation.Overview != "" {
		overview = sql.NullString{String: translation.Overview, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		translation.CollectionID,
		translation.Language,
		name,
		overview,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("save collection translation: %w", err)
	}

	return nil
}

// GetCollectionMovies retrieves cached movies for a collection
func (r *SQLiteCollectionRepository) GetCollectionMovies(ctx context.Context, collectionID int) ([]ports.CollectionMovieItem, error) {
	query := `
		SELECT id, collection_id, tmdb_movie_id, title, original_title, poster_path, release_date, vote_average, display_order, fetched_at
		FROM collection_movies
		WHERE collection_id = ?
		ORDER BY display_order
	`

	rows, err := r.db.QueryContext(ctx, query, collectionID)
	if err != nil {
		return nil, fmt.Errorf("query collection movies: %w", err)
	}
	defer rows.Close()

	var movies []ports.CollectionMovieItem
	for rows.Next() {
		var m ports.CollectionMovieItem
		var originalTitle, posterPath, releaseDate sql.NullString
		if err := rows.Scan(
			&m.ID,
			&m.CollectionID,
			&m.TMDBMovieID,
			&m.Title,
			&originalTitle,
			&posterPath,
			&releaseDate,
			&m.VoteAverage,
			&m.DisplayOrder,
			&m.FetchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan collection movie: %w", err)
		}
		m.OriginalTitle = originalTitle.String
		m.PosterPath = posterPath.String
		m.ReleaseDate = releaseDate.String
		movies = append(movies, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate collection movies: %w", err)
	}

	return movies, nil
}

// SaveCollectionMovies saves collection movies to the cache, replacing any existing entries
func (r *SQLiteCollectionRepository) SaveCollectionMovies(ctx context.Context, collectionID int, movies []ports.CollectionMovieItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing entries
	if _, err := tx.ExecContext(ctx, "DELETE FROM collection_movies WHERE collection_id = ?", collectionID); err != nil {
		return fmt.Errorf("delete existing collection movies: %w", err)
	}

	// Insert new entries
	query := `
		INSERT INTO collection_movies (collection_id, tmdb_movie_id, title, original_title, poster_path, release_date, vote_average, display_order, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	for i, m := range movies {
		var originalTitle, posterPath, releaseDate sql.NullString
		if m.OriginalTitle != "" {
			originalTitle = sql.NullString{String: m.OriginalTitle, Valid: true}
		}
		if m.PosterPath != "" {
			posterPath = sql.NullString{String: m.PosterPath, Valid: true}
		}
		if m.ReleaseDate != "" {
			releaseDate = sql.NullString{String: m.ReleaseDate, Valid: true}
		}

		if _, err := tx.ExecContext(ctx, query,
			collectionID,
			m.TMDBMovieID,
			m.Title,
			originalTitle,
			posterPath,
			releaseDate,
			m.VoteAverage,
			i,
			now,
		); err != nil {
			return fmt.Errorf("insert collection movie: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// DeleteExpiredData removes collection data older than maxAge
func (r *SQLiteCollectionRepository) DeleteExpiredData(ctx context.Context, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete expired collection movies
	if _, err := tx.ExecContext(ctx, "DELETE FROM collection_movies WHERE fetched_at < ?", cutoff); err != nil {
		return fmt.Errorf("delete expired collection movies: %w", err)
	}

	// Delete expired collection translations
	if _, err := tx.ExecContext(ctx, "DELETE FROM collection_translations WHERE fetched_at < ?", cutoff); err != nil {
		return fmt.Errorf("delete expired collection translations: %w", err)
	}

	// Delete expired collection metadata
	if _, err := tx.ExecContext(ctx, "DELETE FROM collection_metadata WHERE fetched_at < ?", cutoff); err != nil {
		return fmt.Errorf("delete expired collection metadata: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetCollectionMovieTranslation retrieves a translation for a specific collection movie
func (r *SQLiteCollectionRepository) GetCollectionMovieTranslation(ctx context.Context, collectionMovieID int64, language string) (*ports.CollectionMovieTranslation, error) {
	query := `
		SELECT collection_movie_id, language, title
		FROM collection_movie_translations
		WHERE collection_movie_id = ? AND language = ?
	`

	var t ports.CollectionMovieTranslation
	err := r.db.QueryRowContext(ctx, query, collectionMovieID, language).Scan(
		&t.CollectionMovieID,
		&t.Language,
		&t.Title,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query collection movie translation: %w", err)
	}

	return &t, nil
}

// GetCollectionMovieTranslations retrieves translations for all movies in a collection
// Returns a map of collectionMovieID -> translated title
func (r *SQLiteCollectionRepository) GetCollectionMovieTranslations(ctx context.Context, collectionID int, language string) (map[int64]string, error) {
	query := `
		SELECT cmt.collection_movie_id, cmt.title
		FROM collection_movie_translations cmt
		INNER JOIN collection_movies cm ON cm.id = cmt.collection_movie_id
		WHERE cm.collection_id = ? AND cmt.language = ?
	`

	rows, err := r.db.QueryContext(ctx, query, collectionID, language)
	if err != nil {
		return nil, fmt.Errorf("query collection movie translations: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]string)
	for rows.Next() {
		var id int64
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, fmt.Errorf("scan collection movie translation: %w", err)
		}
		result[id] = title
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate collection movie translations: %w", err)
	}

	return result, nil
}

// SaveCollectionMovieTranslation saves a single translation for a collection movie
func (r *SQLiteCollectionRepository) SaveCollectionMovieTranslation(ctx context.Context, translation *ports.CollectionMovieTranslation) error {
	query := `
		INSERT INTO collection_movie_translations (collection_movie_id, language, title, fetched_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(collection_movie_id, language) DO UPDATE SET
			title = excluded.title,
			fetched_at = excluded.fetched_at
	`

	_, err := r.db.ExecContext(ctx, query,
		translation.CollectionMovieID,
		translation.Language,
		translation.Title,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("save collection movie translation: %w", err)
	}

	return nil
}

// SaveCollectionMovieTranslations saves multiple translations for collection movies
func (r *SQLiteCollectionRepository) SaveCollectionMovieTranslations(ctx context.Context, translations []ports.CollectionMovieTranslation) error {
	if len(translations) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO collection_movie_translations (collection_movie_id, language, title, fetched_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(collection_movie_id, language) DO UPDATE SET
			title = excluded.title,
			fetched_at = excluded.fetched_at
	`

	now := time.Now()
	for _, t := range translations {
		if _, err := tx.ExecContext(ctx, query,
			t.CollectionMovieID,
			t.Language,
			t.Title,
			now,
		); err != nil {
			return fmt.Errorf("insert collection movie translation: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Ensure SQLiteCollectionRepository implements CollectionRepository
var _ ports.CollectionRepository = (*SQLiteCollectionRepository)(nil)
