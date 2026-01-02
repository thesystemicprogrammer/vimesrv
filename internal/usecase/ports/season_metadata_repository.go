package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// SeasonMetadataRepository defines the interface for season metadata persistence
type SeasonMetadataRepository interface {
	// Create inserts a new season metadata record
	Create(ctx context.Context, metadata *domain.SeasonMetadata) error

	// Get retrieves a season metadata by its internal ID
	Get(ctx context.Context, id int64) (*domain.SeasonMetadata, error)

	// GetBySeriesAndNumber retrieves a season by series ID and season number
	GetBySeriesAndNumber(ctx context.Context, seriesID int64, seasonNumber int) (*domain.SeasonMetadata, error)

	// GetWithEpisodes retrieves a season with all its episodes loaded
	GetWithEpisodes(ctx context.Context, id int64) (*domain.SeasonMetadata, error)

	// ListBySeriesID retrieves all seasons for a given series
	ListBySeriesID(ctx context.Context, seriesID int64) ([]domain.SeasonMetadata, error)

	// Update updates an existing season metadata record
	Update(ctx context.Context, metadata *domain.SeasonMetadata) error

	// Delete removes a season metadata record
	Delete(ctx context.Context, id int64) error

	// CreateTranslation creates a new translation for a season
	CreateTranslation(ctx context.Context, translation *domain.SeasonMetadataTranslation) error

	// GetTranslation retrieves a specific translation for a season
	GetTranslation(ctx context.Context, seasonMetadataID int64, language string) (*domain.SeasonMetadataTranslation, error)

	// GetTranslations retrieves all translations for a season
	GetTranslations(ctx context.Context, seasonMetadataID int64) ([]domain.SeasonMetadataTranslation, error)

	// UpsertTranslation creates or updates a translation
	UpsertTranslation(ctx context.Context, translation *domain.SeasonMetadataTranslation) error
}
