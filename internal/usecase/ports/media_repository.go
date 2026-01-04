package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// MediaRepository provides persistence operations for media files
type MediaRepository interface {
	// Create inserts a new media file record
	Create(ctx context.Context, media *domain.MediaFile) error

	// Get retrieves a media file by its ID
	// Returns nil if not found
	Get(ctx context.Context, id string) (*domain.MediaFile, error)

	// List retrieves all media files with pagination
	// Returns a slice of media files, total count, and any error
	List(ctx context.Context, page, perPage int) ([]*domain.MediaFile, int, error)

	// FindByFingerprint retrieves a media file by its fingerprint
	// Returns nil if not found
	FindByFingerprint(ctx context.Context, fingerprint string) (*domain.MediaFile, error)

	// ExistsByFingerprint checks if a media file with the given fingerprint exists
	ExistsByFingerprint(ctx context.Context, fingerprint string) (bool, error)

	// Update updates an existing media file record
	Update(ctx context.Context, media *domain.MediaFile) error

	// Delete removes a media file record by its ID
	// Related records (audio_streams, subtitle_streams, transcodes, metadata_candidates)
	// are automatically deleted via CASCADE
	Delete(ctx context.Context, id string) error

	// FindByEpisodeMetadataIDs retrieves all media files linked to any of the given episode metadata IDs
	FindByEpisodeMetadataIDs(ctx context.Context, episodeMetadataIDs []int64) ([]*domain.MediaFile, error)

	// CountBySeasonMetadataID counts media files linked to episodes in the given season
	CountBySeasonMetadataID(ctx context.Context, seasonMetadataID int64) (int, error)

	// CountBySeriesMetadataID counts media files linked to episodes in any season of the given series
	CountBySeriesMetadataID(ctx context.Context, seriesMetadataID int64) (int, error)
}
