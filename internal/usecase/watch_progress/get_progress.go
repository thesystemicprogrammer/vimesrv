package watch_progress

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetWatchProgressUseCase retrieves watch progress for a specific media item
type GetWatchProgressUseCase struct {
	repo ports.WatchProgressRepository
}

func NewGetWatchProgressUseCase(repo ports.WatchProgressRepository) *GetWatchProgressUseCase {
	return &GetWatchProgressUseCase{repo: repo}
}

func (uc *GetWatchProgressUseCase) Execute(ctx context.Context, userID, mediaID string, episodeID *int64) (*domain.WatchProgress, error) {
	if userID == "" || mediaID == "" {
		return nil, ErrInvalidInput
	}

	return uc.repo.GetProgress(ctx, userID, mediaID, episodeID)
}
