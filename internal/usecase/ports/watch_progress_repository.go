package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// WatchProgressRepository provides persistence operations for watch progress
type WatchProgressRepository interface {
	// SaveProgress creates or updates watch progress for a user
	SaveProgress(ctx context.Context, progress *domain.WatchProgress) error

	// GetProgress retrieves watch progress for a specific media item
	// Returns nil if not found
	GetProgress(ctx context.Context, userID, mediaID string, episodeID *int64) (*domain.WatchProgress, error)

	// GetContinueWatching retrieves in-progress items for Continue Watching section
	GetContinueWatching(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error)

	// GetWatchHistory retrieves completed watch history for a user (paginated)
	GetWatchHistory(ctx context.Context, userID string, page, perPage int) ([]domain.ContinueWatchingItem, int, error)

	// RemoveFromContinueWatching marks an item as manually removed
	RemoveFromContinueWatching(ctx context.Context, userID, mediaID string, episodeID *int64) error

	// DeleteHistory hard deletes all watch progress for a user
	DeleteHistory(ctx context.Context, userID string) error

	// MarkAsCompleted explicitly marks an item as completed
	MarkAsCompleted(ctx context.Context, userID, mediaID string, episodeID *int64) error
}
