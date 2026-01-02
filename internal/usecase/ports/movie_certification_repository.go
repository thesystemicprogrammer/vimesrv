package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// MovieCertificationRepository defines the interface for movie certification persistence
type MovieCertificationRepository interface {
	// Create inserts a new movie certification record
	Create(ctx context.Context, certification *domain.MovieCertification) error

	// CreateBatch inserts multiple movie certifications in a single transaction
	CreateBatch(ctx context.Context, certifications []*domain.MovieCertification) error

	// GetByMovieMetadataID retrieves all certifications for a movie
	GetByMovieMetadataID(ctx context.Context, movieMetadataID int64) ([]domain.MovieCertification, error)

	// GetByMovieMetadataIDAndCountry retrieves a certification for a specific country
	GetByMovieMetadataIDAndCountry(ctx context.Context, movieMetadataID int64, country string) (*domain.MovieCertification, error)

	// DeleteByMovieMetadataID removes all certifications for a movie
	DeleteByMovieMetadataID(ctx context.Context, movieMetadataID int64) error
}
