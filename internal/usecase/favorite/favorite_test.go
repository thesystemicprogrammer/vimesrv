package favorite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// MockFavoriteRepository for testing
type mockFavoriteRepository struct {
	AddFavoriteFn      func(ctx context.Context, favorite *domain.Favorite) error
	RemoveFavoriteFn   func(ctx context.Context, userID string, mediaType string, metadataID int64) error
	GetUserFavoritesFn func(ctx context.Context, userID string, limit int) ([]domain.FavoriteItem, error)
	IsFavoritedFn      func(ctx context.Context, userID string, mediaType string, metadataID int64) (bool, error)
}

func (m *mockFavoriteRepository) AddFavorite(ctx context.Context, favorite *domain.Favorite) error {
	if m.AddFavoriteFn != nil {
		return m.AddFavoriteFn(ctx, favorite)
	}
	return nil
}

func (m *mockFavoriteRepository) RemoveFavorite(ctx context.Context, userID string, mediaType string, metadataID int64) error {
	if m.RemoveFavoriteFn != nil {
		return m.RemoveFavoriteFn(ctx, userID, mediaType, metadataID)
	}
	return nil
}

func (m *mockFavoriteRepository) GetUserFavorites(ctx context.Context, userID string, limit int) ([]domain.FavoriteItem, error) {
	if m.GetUserFavoritesFn != nil {
		return m.GetUserFavoritesFn(ctx, userID, limit)
	}
	return nil, nil
}

func (m *mockFavoriteRepository) IsFavorited(ctx context.Context, userID string, mediaType string, metadataID int64) (bool, error) {
	if m.IsFavoritedFn != nil {
		return m.IsFavoritedFn(ctx, userID, mediaType, metadataID)
	}
	return false, nil
}

// TestToggleFavorite_AddMovie tests adding a movie to favorites
func TestToggleFavorite_AddMovie(t *testing.T) {
	mockRepo := &mockFavoriteRepository{
		IsFavoritedFn: func(ctx context.Context, userID string, mediaType string, metadataID int64) (bool, error) {
			return false, nil // Not favorited yet
		},
		AddFavoriteFn: func(ctx context.Context, favorite *domain.Favorite) error {
			assert.Equal(t, "user123", favorite.UserID)
			assert.Equal(t, "movie", favorite.MediaType)
			assert.True(t, favorite.MovieMetadataID.Valid)
			assert.Equal(t, int64(456), favorite.MovieMetadataID.Int64)
			assert.False(t, favorite.SeriesMetadataID.Valid)
			return nil
		},
	}

	uc := NewToggleFavoriteUseCase(mockRepo)

	input := ToggleFavoriteInput{
		UserID:     "user123",
		MediaType:  "movie",
		MetadataID: 456,
	}

	added, err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, added, "Should return true when adding favorite")
}

// TestToggleFavorite_AddSeries tests adding a series to favorites
func TestToggleFavorite_AddSeries(t *testing.T) {
	mockRepo := &mockFavoriteRepository{
		IsFavoritedFn: func(ctx context.Context, userID string, mediaType string, metadataID int64) (bool, error) {
			return false, nil
		},
		AddFavoriteFn: func(ctx context.Context, favorite *domain.Favorite) error {
			assert.Equal(t, "user123", favorite.UserID)
			assert.Equal(t, "series", favorite.MediaType)
			assert.True(t, favorite.SeriesMetadataID.Valid)
			assert.Equal(t, int64(789), favorite.SeriesMetadataID.Int64)
			assert.False(t, favorite.MovieMetadataID.Valid)
			return nil
		},
	}

	uc := NewToggleFavoriteUseCase(mockRepo)

	input := ToggleFavoriteInput{
		UserID:     "user123",
		MediaType:  "series",
		MetadataID: 789,
	}

	added, err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, added)
}

// TestToggleFavorite_Remove tests removing a favorite
func TestToggleFavorite_Remove(t *testing.T) {
	mockRepo := &mockFavoriteRepository{
		IsFavoritedFn: func(ctx context.Context, userID string, mediaType string, metadataID int64) (bool, error) {
			return true, nil // Already favorited
		},
		RemoveFavoriteFn: func(ctx context.Context, userID string, mediaType string, metadataID int64) error {
			assert.Equal(t, "user123", userID)
			assert.Equal(t, "movie", mediaType)
			assert.Equal(t, int64(456), metadataID)
			return nil
		},
	}

	uc := NewToggleFavoriteUseCase(mockRepo)

	input := ToggleFavoriteInput{
		UserID:     "user123",
		MediaType:  "movie",
		MetadataID: 456,
	}

	added, err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, added, "Should return false when removing favorite")
}

// TestToggleFavorite_InvalidInput tests validation
func TestToggleFavorite_InvalidInput(t *testing.T) {
	mockRepo := &mockFavoriteRepository{}
	uc := NewToggleFavoriteUseCase(mockRepo)

	tests := []struct {
		name  string
		input ToggleFavoriteInput
	}{
		{
			name: "empty user ID",
			input: ToggleFavoriteInput{
				UserID:     "",
				MediaType:  "movie",
				MetadataID: 456,
			},
		},
		{
			name: "invalid media type",
			input: ToggleFavoriteInput{
				UserID:     "user123",
				MediaType:  "invalid",
				MetadataID: 456,
			},
		},
		{
			name: "zero metadata ID",
			input: ToggleFavoriteInput{
				UserID:     "user123",
				MediaType:  "movie",
				MetadataID: 0,
			},
		},
		{
			name: "negative metadata ID",
			input: ToggleFavoriteInput{
				UserID:     "user123",
				MediaType:  "movie",
				MetadataID: -1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Execute(context.Background(), tt.input)
			assert.Error(t, err)
		})
	}
}

// TestToggleFavorite_IsFavoritedError tests error handling when checking if favorited
func TestToggleFavorite_IsFavoritedError(t *testing.T) {
	expectedErr := assert.AnError
	mockRepo := &mockFavoriteRepository{
		IsFavoritedFn: func(ctx context.Context, userID string, mediaType string, metadataID int64) (bool, error) {
			return false, expectedErr
		},
	}

	uc := NewToggleFavoriteUseCase(mockRepo)

	input := ToggleFavoriteInput{
		UserID:     "user123",
		MediaType:  "movie",
		MetadataID: 456,
	}

	_, err := uc.Execute(context.Background(), input)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestToggleFavorite_AddError tests error handling when adding
func TestToggleFavorite_AddError(t *testing.T) {
	expectedErr := assert.AnError
	mockRepo := &mockFavoriteRepository{
		IsFavoritedFn: func(ctx context.Context, userID string, mediaType string, metadataID int64) (bool, error) {
			return false, nil
		},
		AddFavoriteFn: func(ctx context.Context, favorite *domain.Favorite) error {
			return expectedErr
		},
	}

	uc := NewToggleFavoriteUseCase(mockRepo)

	input := ToggleFavoriteInput{
		UserID:     "user123",
		MediaType:  "movie",
		MetadataID: 456,
	}

	_, err := uc.Execute(context.Background(), input)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestToggleFavorite_RemoveError tests error handling when removing
func TestToggleFavorite_RemoveError(t *testing.T) {
	expectedErr := assert.AnError
	mockRepo := &mockFavoriteRepository{
		IsFavoritedFn: func(ctx context.Context, userID string, mediaType string, metadataID int64) (bool, error) {
			return true, nil
		},
		RemoveFavoriteFn: func(ctx context.Context, userID string, mediaType string, metadataID int64) error {
			return expectedErr
		},
	}

	uc := NewToggleFavoriteUseCase(mockRepo)

	input := ToggleFavoriteInput{
		UserID:     "user123",
		MediaType:  "movie",
		MetadataID: 456,
	}

	_, err := uc.Execute(context.Background(), input)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestGetUserFavorites_Success tests successful retrieval
func TestGetUserFavorites_Success(t *testing.T) {
	expectedFavorites := []domain.FavoriteItem{
		{
			Favorite: domain.Favorite{
				ID:              "fav1",
				UserID:          "user123",
				MediaType:       "movie",
				MovieMetadataID: sql.NullInt64{Int64: 456, Valid: true},
				AddedAt:         time.Now().UTC(),
			},
			Title:      "Movie 1",
			PosterPath: sql.NullString{String: "/poster1.jpg", Valid: true},
			Year:       sql.NullInt64{Int64: 2020, Valid: true},
			Rating:     sql.NullFloat64{Float64: 8.5, Valid: true},
		},
		{
			Favorite: domain.Favorite{
				ID:               "fav2",
				UserID:           "user123",
				MediaType:        "series",
				SeriesMetadataID: sql.NullInt64{Int64: 789, Valid: true},
				AddedAt:          time.Now().UTC(),
			},
			Title:      "Series 1",
			PosterPath: sql.NullString{String: "/poster2.jpg", Valid: true},
			Year:       sql.NullInt64{Int64: 2019, Valid: true},
			Rating:     sql.NullFloat64{Float64: 9.0, Valid: true},
		},
	}

	mockRepo := &mockFavoriteRepository{
		GetUserFavoritesFn: func(ctx context.Context, userID string, limit int) ([]domain.FavoriteItem, error) {
			assert.Equal(t, "user123", userID)
			assert.Equal(t, 50, limit)
			return expectedFavorites, nil
		},
	}

	uc := NewGetUserFavoritesUseCase(mockRepo)

	favorites, err := uc.Execute(context.Background(), "user123", 50)
	require.NoError(t, err)
	assert.Len(t, favorites, 2)
	assert.Equal(t, expectedFavorites, favorites)
}

// TestGetUserFavorites_Empty tests empty favorites list
func TestGetUserFavorites_Empty(t *testing.T) {
	mockRepo := &mockFavoriteRepository{
		GetUserFavoritesFn: func(ctx context.Context, userID string, limit int) ([]domain.FavoriteItem, error) {
			return []domain.FavoriteItem{}, nil
		},
	}

	uc := NewGetUserFavoritesUseCase(mockRepo)

	favorites, err := uc.Execute(context.Background(), "user123", 50)
	require.NoError(t, err)
	assert.Len(t, favorites, 0)
}

// TestGetUserFavorites_DefaultLimit tests default limit application
func TestGetUserFavorites_DefaultLimit(t *testing.T) {
	mockRepo := &mockFavoriteRepository{
		GetUserFavoritesFn: func(ctx context.Context, userID string, limit int) ([]domain.FavoriteItem, error) {
			assert.Equal(t, 50, limit, "Should use default limit of 50")
			return []domain.FavoriteItem{}, nil
		},
	}

	uc := NewGetUserFavoritesUseCase(mockRepo)

	_, err := uc.Execute(context.Background(), "user123", 0) // 0 should default to 50
	require.NoError(t, err)
}

// TestGetUserFavorites_InvalidInput tests validation
func TestGetUserFavorites_InvalidInput(t *testing.T) {
	mockRepo := &mockFavoriteRepository{}
	uc := NewGetUserFavoritesUseCase(mockRepo)

	_, err := uc.Execute(context.Background(), "", 50)
	assert.Error(t, err)
}

// TestGetUserFavorites_RepositoryError tests repository error handling
func TestGetUserFavorites_RepositoryError(t *testing.T) {
	expectedErr := assert.AnError
	mockRepo := &mockFavoriteRepository{
		GetUserFavoritesFn: func(ctx context.Context, userID string, limit int) ([]domain.FavoriteItem, error) {
			return nil, expectedErr
		},
	}

	uc := NewGetUserFavoritesUseCase(mockRepo)

	_, err := uc.Execute(context.Background(), "user123", 50)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}
