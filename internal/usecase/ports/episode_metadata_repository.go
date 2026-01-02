package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// EpisodeMetadataRepository defines the interface for episode metadata persistence
type EpisodeMetadataRepository interface {
	// Create inserts a new episode metadata record
	Create(ctx context.Context, metadata *domain.EpisodeMetadata) error

	// Get retrieves an episode metadata by its internal ID
	Get(ctx context.Context, id int64) (*domain.EpisodeMetadata, error)

	// GetBySeasonAndNumber retrieves an episode by season ID and episode number
	GetBySeasonAndNumber(ctx context.Context, seasonID int64, episodeNumber int) (*domain.EpisodeMetadata, error)

	// ListBySeasonID retrieves all episodes for a given season
	ListBySeasonID(ctx context.Context, seasonID int64) ([]domain.EpisodeMetadata, error)

	// Update updates an existing episode metadata record
	Update(ctx context.Context, metadata *domain.EpisodeMetadata) error

	// Delete removes an episode metadata record
	Delete(ctx context.Context, id int64) error

	// CreateTranslation creates a new translation for an episode
	CreateTranslation(ctx context.Context, translation *domain.EpisodeMetadataTranslation) error

	// GetTranslation retrieves a specific translation for an episode
	GetTranslation(ctx context.Context, episodeMetadataID int64, language string) (*domain.EpisodeMetadataTranslation, error)

	// GetTranslations retrieves all translations for an episode
	GetTranslations(ctx context.Context, episodeMetadataID int64) ([]domain.EpisodeMetadataTranslation, error)

	// UpsertTranslation creates or updates a translation
	UpsertTranslation(ctx context.Context, translation *domain.EpisodeMetadataTranslation) error
}
