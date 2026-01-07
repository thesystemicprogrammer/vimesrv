package favorite

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetUserFavoritesUseCase retrieves all favorites for a user
type GetUserFavoritesUseCase struct {
	repo ports.FavoriteRepository
}

func NewGetUserFavoritesUseCase(repo ports.FavoriteRepository) *GetUserFavoritesUseCase {
	return &GetUserFavoritesUseCase{repo: repo}
}

func (uc *GetUserFavoritesUseCase) Execute(ctx context.Context, userID string, limit int) ([]domain.FavoriteItem, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}
	if limit <= 0 {
		limit = 50
	}

	return uc.repo.GetUserFavorites(ctx, userID, limit)
}
