package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// AudioStreamRepository defines operations for audio stream data access
type AudioStreamRepository interface {
	Create(ctx context.Context, stream *domain.AudioStream) error
	GetByMediaID(ctx context.Context, mediaID string) ([]*domain.AudioStream, error)
	DeleteByMediaID(ctx context.Context, mediaID string) error
}
