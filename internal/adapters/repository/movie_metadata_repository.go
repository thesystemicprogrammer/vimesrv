package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SQLiteMovieMetadataRepository implements MovieMetadataRepository using SQLite
type SQLiteMovieMetadataRepository struct {
	db *sql.DB
}

// NewSQLiteMovieMetadataRepository creates a new SQLite movie metadata repository
func NewSQLiteMovieMetadataRepository(db *sql.DB) *SQLiteMovieMetadataRepository {
	return &SQLiteMovieMetadataRepository{db: db}
}

// Create inserts a new movie metadata record
func (r *SQLiteMovieMetadataRepository) Create(ctx context.Context, metadata *domain.MovieMetadata) error {
	genresJSON, err := json.Marshal(metadata.Genres)
	if err != nil {
		return fmt.Errorf("marshal genres: %w", err)
	}

	query := `
		INSERT INTO movie_metadata (
			tmdb_id, imdb_id, original_title, release_date, runtime,
			poster_path, backdrop_path, vote_average, vote_count,
			popularity, status, original_lang, genres, collection_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		metadata.TMDBID,
		metadata.IMDbID,
		metadata.OriginalTitle,
		metadata.ReleaseDate,
		metadata.Runtime,
		metadata.PosterPath,
		metadata.BackdropPath,
		metadata.VoteAverage,
		metadata.VoteCount,
		metadata.Popularity,
		metadata.Status,
		metadata.OriginalLang,
		string(genresJSON),
		metadata.CollectionID,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert movie metadata: %w", err)
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

// Get retrieves a movie metadata by its internal ID
func (r *SQLiteMovieMetadataRepository) Get(ctx context.Context, id int64) (*domain.MovieMetadata, error) {
	query := `
		SELECT id, tmdb_id, imdb_id, original_title, release_date, runtime,
			   poster_path, backdrop_path, vote_average, vote_count,
			   popularity, status, original_lang, genres, collection_id, created_at, updated_at
		FROM movie_metadata
		WHERE id = ?
	`

	metadata := &domain.MovieMetadata{}
	var genresJSON string
	var collectionID sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&metadata.ID,
		&metadata.TMDBID,
		&metadata.IMDbID,
		&metadata.OriginalTitle,
		&metadata.ReleaseDate,
		&metadata.Runtime,
		&metadata.PosterPath,
		&metadata.BackdropPath,
		&metadata.VoteAverage,
		&metadata.VoteCount,
		&metadata.Popularity,
		&metadata.Status,
		&metadata.OriginalLang,
		&genresJSON,
		&collectionID,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("query movie metadata: %w", err)
	}

	if err := json.Unmarshal([]byte(genresJSON), &metadata.Genres); err != nil {
		metadata.Genres = []string{}
	}

	if collectionID.Valid {
		cid := int(collectionID.Int64)
		metadata.CollectionID = &cid
	}

	return metadata, nil
}

// GetByTMDBID retrieves a movie metadata by its TMDB ID
func (r *SQLiteMovieMetadataRepository) GetByTMDBID(ctx context.Context, tmdbID int) (*domain.MovieMetadata, error) {
	query := `
		SELECT id, tmdb_id, imdb_id, original_title, release_date, runtime,
			   poster_path, backdrop_path, vote_average, vote_count,
			   popularity, status, original_lang, genres, collection_id, created_at, updated_at
		FROM movie_metadata
		WHERE tmdb_id = ?
	`

	metadata := &domain.MovieMetadata{}
	var genresJSON string
	var collectionID sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, tmdbID).Scan(
		&metadata.ID,
		&metadata.TMDBID,
		&metadata.IMDbID,
		&metadata.OriginalTitle,
		&metadata.ReleaseDate,
		&metadata.Runtime,
		&metadata.PosterPath,
		&metadata.BackdropPath,
		&metadata.VoteAverage,
		&metadata.VoteCount,
		&metadata.Popularity,
		&metadata.Status,
		&metadata.OriginalLang,
		&genresJSON,
		&collectionID,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("query movie metadata by tmdb_id: %w", err)
	}

	if err := json.Unmarshal([]byte(genresJSON), &metadata.Genres); err != nil {
		metadata.Genres = []string{}
	}

	if collectionID.Valid {
		cid := int(collectionID.Int64)
		metadata.CollectionID = &cid
	}

	return metadata, nil
}

// Update updates an existing movie metadata record
func (r *SQLiteMovieMetadataRepository) Update(ctx context.Context, metadata *domain.MovieMetadata) error {
	genresJSON, err := json.Marshal(metadata.Genres)
	if err != nil {
		return fmt.Errorf("marshal genres: %w", err)
	}

	query := `
		UPDATE movie_metadata SET
			tmdb_id = ?, imdb_id = ?, original_title = ?, release_date = ?,
			runtime = ?, poster_path = ?, backdrop_path = ?, vote_average = ?,
			vote_count = ?, popularity = ?, status = ?, original_lang = ?,
			genres = ?, collection_id = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		metadata.TMDBID,
		metadata.IMDbID,
		metadata.OriginalTitle,
		metadata.ReleaseDate,
		metadata.Runtime,
		metadata.PosterPath,
		metadata.BackdropPath,
		metadata.VoteAverage,
		metadata.VoteCount,
		metadata.Popularity,
		metadata.Status,
		metadata.OriginalLang,
		string(genresJSON),
		metadata.CollectionID,
		now,
		metadata.ID,
	)
	if err != nil {
		return fmt.Errorf("update movie metadata: %w", err)
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

// Delete removes a movie metadata record
func (r *SQLiteMovieMetadataRepository) Delete(ctx context.Context, id int64) error {
	// First delete translations
	_, err := r.db.ExecContext(ctx, "DELETE FROM movie_metadata_translations WHERE movie_metadata_id = ?", id)
	if err != nil {
		return fmt.Errorf("delete movie translations: %w", err)
	}

	result, err := r.db.ExecContext(ctx, "DELETE FROM movie_metadata WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete movie metadata: %w", err)
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

// ExistsByTMDBID checks if a movie metadata with the given TMDB ID exists
func (r *SQLiteMovieMetadataRepository) ExistsByTMDBID(ctx context.Context, tmdbID int) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		"SELECT 1 FROM movie_metadata WHERE tmdb_id = ? LIMIT 1",
		tmdbID,
	).Scan(&exists)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check movie metadata exists: %w", err)
	}

	return true, nil
}

// CreateTranslation creates a new translation for a movie
func (r *SQLiteMovieMetadataRepository) CreateTranslation(ctx context.Context, translation *domain.MovieMetadataTranslation) error {
	query := `
		INSERT INTO movie_metadata_translations (
			movie_metadata_id, language, title, tagline, overview, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		translation.MovieMetadataID,
		translation.Language,
		translation.Title,
		translation.Tagline,
		translation.Overview,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert movie translation: %w", err)
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

// GetTranslation retrieves a specific translation for a movie
func (r *SQLiteMovieMetadataRepository) GetTranslation(ctx context.Context, movieMetadataID int64, language string) (*domain.MovieMetadataTranslation, error) {
	query := `
		SELECT id, movie_metadata_id, language, title, tagline, overview, created_at, updated_at
		FROM movie_metadata_translations
		WHERE movie_metadata_id = ? AND language = ?
	`

	translation := &domain.MovieMetadataTranslation{}
	err := r.db.QueryRowContext(ctx, query, movieMetadataID, language).Scan(
		&translation.ID,
		&translation.MovieMetadataID,
		&translation.Language,
		&translation.Title,
		&translation.Tagline,
		&translation.Overview,
		&translation.CreatedAt,
		&translation.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("query movie translation: %w", err)
	}

	return translation, nil
}

// GetTranslations retrieves all translations for a movie
func (r *SQLiteMovieMetadataRepository) GetTranslations(ctx context.Context, movieMetadataID int64) ([]domain.MovieMetadataTranslation, error) {
	query := `
		SELECT id, movie_metadata_id, language, title, tagline, overview, created_at, updated_at
		FROM movie_metadata_translations
		WHERE movie_metadata_id = ?
		ORDER BY language
	`

	rows, err := r.db.QueryContext(ctx, query, movieMetadataID)
	if err != nil {
		return nil, fmt.Errorf("query movie translations: %w", err)
	}
	defer rows.Close()

	var translations []domain.MovieMetadataTranslation
	for rows.Next() {
		var t domain.MovieMetadataTranslation
		if err := rows.Scan(
			&t.ID,
			&t.MovieMetadataID,
			&t.Language,
			&t.Title,
			&t.Tagline,
			&t.Overview,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan movie translation: %w", err)
		}
		translations = append(translations, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movie translations: %w", err)
	}

	return translations, nil
}

// UpsertTranslation creates or updates a translation
func (r *SQLiteMovieMetadataRepository) UpsertTranslation(ctx context.Context, translation *domain.MovieMetadataTranslation) error {
	query := `
		INSERT INTO movie_metadata_translations (
			movie_metadata_id, language, title, tagline, overview, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(movie_metadata_id, language) DO UPDATE SET
			title = excluded.title,
			tagline = excluded.tagline,
			overview = excluded.overview,
			updated_at = excluded.updated_at
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		translation.MovieMetadataID,
		translation.Language,
		translation.Title,
		translation.Tagline,
		translation.Overview,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert movie translation: %w", err)
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

// ListIDsWithoutTranslation returns movie metadata IDs that don't have a translation for the given language
func (r *SQLiteMovieMetadataRepository) ListIDsWithoutTranslation(ctx context.Context, language string) ([]ports.MovieMetadataForTranslation, error) {
	exact, base := movieLanguageParams(language)

	query := `
		SELECT mm.id, mm.tmdb_id, mm.original_title
		FROM movie_metadata mm
		WHERE NOT EXISTS (
			SELECT 1 FROM movie_metadata_translations t
			WHERE t.movie_metadata_id = mm.id 
			AND (t.language = ? OR t.language LIKE ? || '%')
		)
		ORDER BY mm.id
	`

	rows, err := r.db.QueryContext(ctx, query, exact, base)
	if err != nil {
		return nil, fmt.Errorf("query movies without translation: %w", err)
	}
	defer rows.Close()

	var results []ports.MovieMetadataForTranslation
	for rows.Next() {
		var m ports.MovieMetadataForTranslation
		if err := rows.Scan(&m.ID, &m.TMDBID, &m.OriginalTitle); err != nil {
			return nil, fmt.Errorf("scan movie: %w", err)
		}
		results = append(results, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movies: %w", err)
	}

	return results, nil
}

// movieLanguageParams returns the exact language and base language for fallback queries
func movieLanguageParams(lang string) (exact string, base string) {
	if lang == "" {
		return "en", "en"
	}
	if idx := strings.IndexAny(lang, "-_"); idx > 0 {
		return lang, lang[:idx]
	}
	return lang, lang
}

// SetFullCreditsFetched marks that full credits have been fetched from TMDB for this movie
func (r *SQLiteMovieMetadataRepository) SetFullCreditsFetched(ctx context.Context, movieMetadataID int64) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE movie_metadata SET full_credits_fetched = 1 WHERE id = ?",
		movieMetadataID,
	)
	if err != nil {
		return fmt.Errorf("set full credits fetched: %w", err)
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

// HasFullCreditsFetched checks if full credits have been fetched from TMDB for this movie
func (r *SQLiteMovieMetadataRepository) HasFullCreditsFetched(ctx context.Context, movieMetadataID int64) (bool, error) {
	var fetched int
	err := r.db.QueryRowContext(ctx,
		"SELECT full_credits_fetched FROM movie_metadata WHERE id = ?",
		movieMetadataID,
	).Scan(&fetched)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, shared.ErrNotFound
		}
		return false, fmt.Errorf("check full credits fetched: %w", err)
	}

	return fetched == 1, nil
}

// GetTMDBIDByID retrieves the TMDB ID for a movie metadata record
func (r *SQLiteMovieMetadataRepository) GetTMDBIDByID(ctx context.Context, movieMetadataID int64) (int, error) {
	var tmdbID int
	err := r.db.QueryRowContext(ctx,
		"SELECT tmdb_id FROM movie_metadata WHERE id = ?",
		movieMetadataID,
	).Scan(&tmdbID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, shared.ErrNotFound
		}
		return 0, fmt.Errorf("get tmdb id: %w", err)
	}

	return tmdbID, nil
}

// Ensure SQLiteMovieMetadataRepository implements MovieMetadataRepository
var _ ports.MovieMetadataRepository = (*SQLiteMovieMetadataRepository)(nil)
