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

// SQLiteMovieCertificationRepository implements MovieCertificationRepository using SQLite
type SQLiteMovieCertificationRepository struct {
	db *sql.DB
}

// NewSQLiteMovieCertificationRepository creates a new SQLite movie certification repository
func NewSQLiteMovieCertificationRepository(db *sql.DB) *SQLiteMovieCertificationRepository {
	return &SQLiteMovieCertificationRepository{db: db}
}

// Create inserts a new movie certification record
func (r *SQLiteMovieCertificationRepository) Create(ctx context.Context, certification *domain.MovieCertification) error {
	query := `
		INSERT INTO movie_certifications (
			movie_metadata_id, country, certification, created_at
		) VALUES (?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		certification.MovieMetadataID,
		certification.Country,
		certification.Certification,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert movie certification: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	certification.ID = id
	certification.CreatedAt = now

	return nil
}

// CreateBatch inserts multiple movie certifications in a single transaction
func (r *SQLiteMovieCertificationRepository) CreateBatch(ctx context.Context, certifications []*domain.MovieCertification) error {
	if len(certifications) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO movie_certifications (
			movie_metadata_id, country, certification, created_at
		) VALUES (?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, cert := range certifications {
		result, err := stmt.ExecContext(ctx,
			cert.MovieMetadataID,
			cert.Country,
			cert.Certification,
			now,
		)
		if err != nil {
			return fmt.Errorf("insert movie certification: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("get last insert id: %w", err)
		}

		cert.ID = id
		cert.CreatedAt = now
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetByMovieMetadataID retrieves all certifications for a movie
func (r *SQLiteMovieCertificationRepository) GetByMovieMetadataID(ctx context.Context, movieMetadataID int64) ([]domain.MovieCertification, error) {
	query := `
		SELECT id, movie_metadata_id, country, certification, created_at
		FROM movie_certifications
		WHERE movie_metadata_id = ?
		ORDER BY country
	`

	rows, err := r.db.QueryContext(ctx, query, movieMetadataID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []domain.MovieCertification{}, nil
		}
		return nil, fmt.Errorf("query movie certifications: %w", err)
	}
	defer rows.Close()

	var certifications []domain.MovieCertification
	for rows.Next() {
		var c domain.MovieCertification
		if err := rows.Scan(
			&c.ID,
			&c.MovieMetadataID,
			&c.Country,
			&c.Certification,
			&c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan movie certification: %w", err)
		}
		certifications = append(certifications, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movie certifications: %w", err)
	}

	if certifications == nil {
		certifications = []domain.MovieCertification{}
	}

	return certifications, nil
}

// GetByMovieMetadataIDAndCountry retrieves a certification for a specific country
func (r *SQLiteMovieCertificationRepository) GetByMovieMetadataIDAndCountry(ctx context.Context, movieMetadataID int64, country string) (*domain.MovieCertification, error) {
	query := `
		SELECT id, movie_metadata_id, country, certification, created_at
		FROM movie_certifications
		WHERE movie_metadata_id = ? AND country = ?
	`

	cert := &domain.MovieCertification{}
	err := r.db.QueryRowContext(ctx, query, movieMetadataID, country).Scan(
		&cert.ID,
		&cert.MovieMetadataID,
		&cert.Country,
		&cert.Certification,
		&cert.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("query movie certification: %w", err)
	}

	return cert, nil
}

// DeleteByMovieMetadataID removes all certifications for a movie
func (r *SQLiteMovieCertificationRepository) DeleteByMovieMetadataID(ctx context.Context, movieMetadataID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM movie_certifications WHERE movie_metadata_id = ?", movieMetadataID)
	if err != nil {
		return fmt.Errorf("delete movie certifications: %w", err)
	}

	return nil
}

// Ensure SQLiteMovieCertificationRepository implements MovieCertificationRepository
var _ ports.MovieCertificationRepository = (*SQLiteMovieCertificationRepository)(nil)
