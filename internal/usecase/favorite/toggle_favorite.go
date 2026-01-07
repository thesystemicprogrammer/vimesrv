package favorite

import (
	"context"
	"database/sql"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ToggleFavoriteUseCase handles adding/removing favorites
type ToggleFavoriteUseCase struct {
	repo ports.FavoriteRepository
}

func NewToggleFavoriteUseCase(repo ports.FavoriteRepository) *ToggleFavoriteUseCase {
	return &ToggleFavoriteUseCase{repo: repo}
}

type ToggleFavoriteInput struct {
	UserID     string
	MediaType  string // "movie" or "series"
	MetadataID int64
}

func (uc *ToggleFavoriteUseCase) Execute(ctx context.Context, input ToggleFavoriteInput) (bool, error) {
	// Validate input
	if input.UserID == "" {
		return false, ErrInvalidInput
	}
	if input.MediaType != "movie" && input.MediaType != "series" {
		return false, ErrInvalidMediaType
	}
	if input.MetadataID <= 0 {
		return false, ErrInvalidMetadataID
	}

	// Check if already favorited
	isFavorited, err := uc.repo.IsFavorited(ctx, input.UserID, input.MediaType, input.MetadataID)
	if err != nil {
		return false, err
	}

	if isFavorited {
		// Remove from favorites
		err = uc.repo.RemoveFavorite(ctx, input.UserID, input.MediaType, input.MetadataID)
		return false, err
	}

	// Add to favorites
	favorite := &domain.Favorite{
		UserID:    input.UserID,
		MediaType: input.MediaType,
	}

	if input.MediaType == "movie" {
		favorite.MovieMetadataID = sql.NullInt64{Int64: input.MetadataID, Valid: true}
	} else {
		favorite.SeriesMetadataID = sql.NullInt64{Int64: input.MetadataID, Valid: true}
	}

	err = uc.repo.AddFavorite(ctx, favorite)
	return true, err
}
