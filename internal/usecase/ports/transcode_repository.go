package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// TranscodeRepository defines operations for transcode data access
type TranscodeRepository interface {
	Create(ctx context.Context, transcode *domain.Transcode) error
	Get(ctx context.Context, id string) (*domain.Transcode, error)
	GetByMediaID(ctx context.Context, mediaID string) ([]*domain.Transcode, error)
	GetProcessingByMediaID(ctx context.Context, mediaID string) ([]*domain.Transcode, error)
	UpdateStatus(ctx context.Context, id string, status domain.TranscodeStatus) error
	MarkProcessing(ctx context.Context, id string, outputPath string) error
	MarkCompleted(ctx context.Context, id string, outputPath string) error
	MarkFailed(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	ListPending(ctx context.Context, limit int) ([]*domain.Transcode, error)
}
