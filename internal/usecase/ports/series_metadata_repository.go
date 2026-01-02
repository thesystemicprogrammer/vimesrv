package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// SeriesMetadataForTranslation contains info needed to fetch translations
type SeriesMetadataForTranslation struct {
	ID           int64
	TMDBID       int
	OriginalName string
}

// SeriesMetadataRepository defines the interface for series metadata persistence
type SeriesMetadataRepository interface {
	// Create inserts a new series metadata record
	Create(ctx context.Context, metadata *domain.SeriesMetadata) error

	// Get retrieves a series metadata by its internal ID
	Get(ctx context.Context, id int64) (*domain.SeriesMetadata, error)

	// GetByTMDBID retrieves a series metadata by its TMDB ID
	GetByTMDBID(ctx context.Context, tmdbID int) (*domain.SeriesMetadata, error)

	// GetWithSeasons retrieves a series with all its seasons loaded
	GetWithSeasons(ctx context.Context, id int64) (*domain.SeriesMetadata, error)

	// Update updates an existing series metadata record
	Update(ctx context.Context, metadata *domain.SeriesMetadata) error

	// Delete removes a series metadata record
	Delete(ctx context.Context, id int64) error

	// ExistsByTMDBID checks if a series metadata with the given TMDB ID exists
	ExistsByTMDBID(ctx context.Context, tmdbID int) (bool, error)

	// CreateTranslation creates a new translation for a series
	CreateTranslation(ctx context.Context, translation *domain.SeriesMetadataTranslation) error

	// GetTranslation retrieves a specific translation for a series
	GetTranslation(ctx context.Context, seriesMetadataID int64, language string) (*domain.SeriesMetadataTranslation, error)

	// GetTranslations retrieves all translations for a series
	GetTranslations(ctx context.Context, seriesMetadataID int64) ([]domain.SeriesMetadataTranslation, error)

	// UpsertTranslation creates or updates a translation
	UpsertTranslation(ctx context.Context, translation *domain.SeriesMetadataTranslation) error

	// ListIDsWithoutTranslation returns series metadata IDs that don't have a translation for the given language
	ListIDsWithoutTranslation(ctx context.Context, language string) ([]SeriesMetadataForTranslation, error)

	// SetFullCreditsFetched marks that full credits have been fetched from TMDB for this series
	SetFullCreditsFetched(ctx context.Context, seriesMetadataID int64) error

	// HasFullCreditsFetched checks if full credits have been fetched from TMDB for this series
	HasFullCreditsFetched(ctx context.Context, seriesMetadataID int64) (bool, error)

	// GetTMDBIDByID retrieves the TMDB ID for a series metadata record
	GetTMDBIDByID(ctx context.Context, seriesMetadataID int64) (int, error)
}
