package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// SeriesCreditRepository defines the interface for series credit persistence
type SeriesCreditRepository interface {
	// CreateBatch inserts multiple series credits in a single transaction
	CreateBatch(ctx context.Context, credits []*domain.SeriesCredit) error

	// GetBySeriesMetadataID retrieves all credits for a series
	GetBySeriesMetadataID(ctx context.Context, seriesMetadataID int64) ([]domain.SeriesCredit, error)

	// GetCastBySeriesMetadataID retrieves cast credits for a series, ordered by display_order
	GetCastBySeriesMetadataID(ctx context.Context, seriesMetadataID int64, limit int) ([]domain.SeriesCredit, error)

	// GetCrewBySeriesMetadataID retrieves crew credits for a series, ordered by display_order
	GetCrewBySeriesMetadataID(ctx context.Context, seriesMetadataID int64) ([]domain.SeriesCredit, error)

	// DeleteBySeriesMetadataID removes all credits for a series
	DeleteBySeriesMetadataID(ctx context.Context, seriesMetadataID int64) error

	// HasCredits checks if credits exist for a series
	HasCredits(ctx context.Context, seriesMetadataID int64) (bool, error)
}
