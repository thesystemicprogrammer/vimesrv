package watch_progress

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// RemoveFromContinueWatchingUseCase removes an item from Continue Watching
type RemoveFromContinueWatchingUseCase struct {
	repo ports.WatchProgressRepository
}

func NewRemoveFromContinueWatchingUseCase(repo ports.WatchProgressRepository) *RemoveFromContinueWatchingUseCase {
	return &RemoveFromContinueWatchingUseCase{repo: repo}
}

func (uc *RemoveFromContinueWatchingUseCase) Execute(ctx context.Context, userID, mediaID string, episodeID *int64) error {
	if userID == "" || mediaID == "" {
		return ErrInvalidInput
	}

	return uc.repo.RemoveFromContinueWatching(ctx, userID, mediaID, episodeID)
}
