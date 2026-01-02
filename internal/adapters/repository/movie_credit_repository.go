package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SQLiteMovieCreditRepository implements MovieCreditRepository using SQLite
type SQLiteMovieCreditRepository struct {
	db *sql.DB
}

// NewSQLiteMovieCreditRepository creates a new SQLite movie credit repository
func NewSQLiteMovieCreditRepository(db *sql.DB) *SQLiteMovieCreditRepository {
	return &SQLiteMovieCreditRepository{db: db}
}

// Create inserts a new movie credit record
func (r *SQLiteMovieCreditRepository) Create(ctx context.Context, credit *domain.MovieCredit) error {
	query := `
		INSERT INTO movie_credits (
			movie_metadata_id, credit_type, tmdb_person_id, name, character,
			job, department, profile_path, display_order, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		credit.MovieMetadataID,
		string(credit.CreditType),
		credit.TMDBPersonID,
		credit.Name,
		credit.Character,
		credit.Job,
		credit.Department,
		credit.ProfilePath,
		credit.DisplayOrder,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert movie credit: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	credit.ID = id
	credit.CreatedAt = now

	return nil
}

// CreateBatch inserts multiple movie credits in a single transaction
func (r *SQLiteMovieCreditRepository) CreateBatch(ctx context.Context, credits []*domain.MovieCredit) error {
	if len(credits) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO movie_credits (
			movie_metadata_id, credit_type, tmdb_person_id, name, character,
			job, department, profile_path, display_order, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, credit := range credits {
		result, err := stmt.ExecContext(ctx,
			credit.MovieMetadataID,
			string(credit.CreditType),
			credit.TMDBPersonID,
			credit.Name,
			credit.Character,
			credit.Job,
			credit.Department,
			credit.ProfilePath,
			credit.DisplayOrder,
			now,
		)
		if err != nil {
			return fmt.Errorf("insert movie credit: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("get last insert id: %w", err)
		}

		credit.ID = id
		credit.CreatedAt = now
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetByMovieMetadataID retrieves all credits for a movie
func (r *SQLiteMovieCreditRepository) GetByMovieMetadataID(ctx context.Context, movieMetadataID int64) ([]domain.MovieCredit, error) {
	query := `
		SELECT id, movie_metadata_id, credit_type, tmdb_person_id, name, character,
			   job, department, profile_path, display_order, created_at
		FROM movie_credits
		WHERE movie_metadata_id = ?
		ORDER BY credit_type, display_order
	`

	return r.queryCredits(ctx, query, movieMetadataID)
}

// GetCastByMovieMetadataID retrieves cast credits for a movie, ordered by display_order
func (r *SQLiteMovieCreditRepository) GetCastByMovieMetadataID(ctx context.Context, movieMetadataID int64, limit int) ([]domain.MovieCredit, error) {
	query := `
		SELECT id, movie_metadata_id, credit_type, tmdb_person_id, name, character,
			   job, department, profile_path, display_order, created_at
		FROM movie_credits
		WHERE movie_metadata_id = ? AND credit_type = 'cast'
		ORDER BY display_order
		LIMIT ?
	`

	return r.queryCredits(ctx, query, movieMetadataID, limit)
}

// GetCrewByMovieMetadataID retrieves crew credits for a movie, ordered by display_order
func (r *SQLiteMovieCreditRepository) GetCrewByMovieMetadataID(ctx context.Context, movieMetadataID int64) ([]domain.MovieCredit, error) {
	query := `
		SELECT id, movie_metadata_id, credit_type, tmdb_person_id, name, character,
			   job, department, profile_path, display_order, created_at
		FROM movie_credits
		WHERE movie_metadata_id = ? AND credit_type = 'crew'
		ORDER BY display_order
	`

	return r.queryCredits(ctx, query, movieMetadataID)
}

// GetDirectorsByMovieMetadataID retrieves director(s) for a movie
func (r *SQLiteMovieCreditRepository) GetDirectorsByMovieMetadataID(ctx context.Context, movieMetadataID int64) ([]domain.MovieCredit, error) {
	query := `
		SELECT id, movie_metadata_id, credit_type, tmdb_person_id, name, character,
			   job, department, profile_path, display_order, created_at
		FROM movie_credits
		WHERE movie_metadata_id = ? AND credit_type = 'crew' AND job = 'Director'
		ORDER BY display_order
	`

	return r.queryCredits(ctx, query, movieMetadataID)
}

// DeleteByMovieMetadataID removes all credits for a movie
func (r *SQLiteMovieCreditRepository) DeleteByMovieMetadataID(ctx context.Context, movieMetadataID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM movie_credits WHERE movie_metadata_id = ?", movieMetadataID)
	if err != nil {
		return fmt.Errorf("delete movie credits: %w", err)
	}

	return nil
}

// queryCredits is a helper to execute credit queries and scan results
func (r *SQLiteMovieCreditRepository) queryCredits(ctx context.Context, query string, args ...interface{}) ([]domain.MovieCredit, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []domain.MovieCredit{}, nil
		}
		return nil, fmt.Errorf("query movie credits: %w", err)
	}
	defer rows.Close()

	var credits []domain.MovieCredit
	for rows.Next() {
		var c domain.MovieCredit
		var creditType string
		if err := rows.Scan(
			&c.ID,
			&c.MovieMetadataID,
			&creditType,
			&c.TMDBPersonID,
			&c.Name,
			&c.Character,
			&c.Job,
			&c.Department,
			&c.ProfilePath,
			&c.DisplayOrder,
			&c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan movie credit: %w", err)
		}
		c.CreditType = domain.CreditType(creditType)
		credits = append(credits, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movie credits: %w", err)
	}

	if credits == nil {
		credits = []domain.MovieCredit{}
	}

	return credits, nil
}

// Ensure SQLiteMovieCreditRepository implements MovieCreditRepository
var _ ports.MovieCreditRepository = (*SQLiteMovieCreditRepository)(nil)
