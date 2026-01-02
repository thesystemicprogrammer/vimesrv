package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// MovieCreditRepository defines the interface for movie credit persistence
type MovieCreditRepository interface {
	// Create inserts a new movie credit record
	Create(ctx context.Context, credit *domain.MovieCredit) error

	// CreateBatch inserts multiple movie credits in a single transaction
	CreateBatch(ctx context.Context, credits []*domain.MovieCredit) error

	// GetByMovieMetadataID retrieves all credits for a movie
	GetByMovieMetadataID(ctx context.Context, movieMetadataID int64) ([]domain.MovieCredit, error)

	// GetCastByMovieMetadataID retrieves cast credits for a movie, ordered by display_order
	GetCastByMovieMetadataID(ctx context.Context, movieMetadataID int64, limit int) ([]domain.MovieCredit, error)

	// GetCrewByMovieMetadataID retrieves crew credits for a movie, ordered by display_order
	GetCrewByMovieMetadataID(ctx context.Context, movieMetadataID int64) ([]domain.MovieCredit, error)

	// GetDirectorsByMovieMetadataID retrieves director(s) for a movie
	GetDirectorsByMovieMetadataID(ctx context.Context, movieMetadataID int64) ([]domain.MovieCredit, error)

	// DeleteByMovieMetadataID removes all credits for a movie
	DeleteByMovieMetadataID(ctx context.Context, movieMetadataID int64) error
}
