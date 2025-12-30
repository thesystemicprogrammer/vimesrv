package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// MediaRepository provides persistence operations for media files
type MediaRepository interface {
	// Create inserts a new media file record
	Create(ctx context.Context, media *domain.MediaFile) error

	// FindByFingerprint retrieves a media file by its fingerprint
	// Returns nil if not found
	FindByFingerprint(ctx context.Context, fingerprint string) (*domain.MediaFile, error)

	// ExistsByFingerprint checks if a media file with the given fingerprint exists
	ExistsByFingerprint(ctx context.Context, fingerprint string) (bool, error)

	// Update updates an existing media file record
	Update(ctx context.Context, media *domain.MediaFile) error
}
