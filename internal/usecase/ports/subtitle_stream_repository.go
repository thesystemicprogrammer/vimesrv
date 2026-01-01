package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// SubtitleStreamRepository defines operations for subtitle stream data access
type SubtitleStreamRepository interface {
	Create(ctx context.Context, stream *domain.SubtitleStream) error
	GetByMediaID(ctx context.Context, mediaID string) ([]*domain.SubtitleStream, error)
	DeleteByMediaID(ctx context.Context, mediaID string) error
}
