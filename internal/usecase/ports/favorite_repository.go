package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// FavoriteRepository provides persistence operations for favorites
type FavoriteRepository interface {
	// AddFavorite adds a movie or series to favorites
	AddFavorite(ctx context.Context, favorite *domain.Favorite) error

	// RemoveFavorite removes a movie or series from favorites
	RemoveFavorite(ctx context.Context, userID string, mediaType string, metadataID int64) error

	// GetUserFavorites retrieves all favorited items for a user with enriched metadata
	GetUserFavorites(ctx context.Context, userID string, limit int) ([]domain.FavoriteItem, error)

	// IsFavorited checks if a movie or series is favorited by a user
	IsFavorited(ctx context.Context, userID string, mediaType string, metadataID int64) (bool, error)
}
