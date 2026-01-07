package watch_progress

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetContinueWatchingUseCase retrieves items for the Continue Watching section
type GetContinueWatchingUseCase struct {
	repo ports.WatchProgressRepository
}

func NewGetContinueWatchingUseCase(repo ports.WatchProgressRepository) *GetContinueWatchingUseCase {
	return &GetContinueWatchingUseCase{repo: repo}
}

func (uc *GetContinueWatchingUseCase) Execute(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}
	if limit <= 0 {
		limit = 20
	}

	return uc.repo.GetContinueWatching(ctx, userID, limit)
}
