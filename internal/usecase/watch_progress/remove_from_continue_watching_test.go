package watch_progress

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRemoveFromContinueWatching_SuccessMovie tests successful removal for a movie
func TestRemoveFromContinueWatching_SuccessMovie(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		RemoveFromContinueWatchingFn: func(ctx context.Context, userID, mediaID string, episodeID *int64) error {
			assert.Equal(t, "user123", userID)
			assert.Equal(t, "media456", mediaID)
			assert.Nil(t, episodeID)
			return nil
		},
	}

	uc := NewRemoveFromContinueWatchingUseCase(mockRepo)

	err := uc.Execute(context.Background(), "user123", "media456", nil)
	require.NoError(t, err)
}

// TestRemoveFromContinueWatching_SuccessEpisode tests removal for a TV episode
func TestRemoveFromContinueWatching_SuccessEpisode(t *testing.T) {
	episodeID := int64(789)
	mockRepo := &mockWatchProgressRepository{
		RemoveFromContinueWatchingFn: func(ctx context.Context, userID, mediaID string, episodeIDPtr *int64) error {
			assert.Equal(t, "user123", userID)
			assert.Equal(t, "media456", mediaID)
			require.NotNil(t, episodeIDPtr)
			assert.Equal(t, int64(789), *episodeIDPtr)
			return nil
		},
	}

	uc := NewRemoveFromContinueWatchingUseCase(mockRepo)

	err := uc.Execute(context.Background(), "user123", "media456", &episodeID)
	require.NoError(t, err)
}

// TestRemoveFromContinueWatching_InvalidInput tests validation
func TestRemoveFromContinueWatching_InvalidInput(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{}
	uc := NewRemoveFromContinueWatchingUseCase(mockRepo)

	tests := []struct {
		name      string
		userID    string
		mediaID   string
		episodeID *int64
	}{
		{
			name:      "empty user ID",
			userID:    "",
			mediaID:   "media456",
			episodeID: nil,
		},
		{
			name:      "empty media ID",
			userID:    "user123",
			mediaID:   "",
			episodeID: nil,
		},
		{
			name:      "both empty",
			userID:    "",
			mediaID:   "",
			episodeID: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.Execute(context.Background(), tt.userID, tt.mediaID, tt.episodeID)
			assert.Error(t, err)
			assert.Equal(t, ErrInvalidInput, err)
		})
	}
}

// TestRemoveFromContinueWatching_RepositoryError tests repository error handling
func TestRemoveFromContinueWatching_RepositoryError(t *testing.T) {
	expectedErr := assert.AnError
	mockRepo := &mockWatchProgressRepository{
		RemoveFromContinueWatchingFn: func(ctx context.Context, userID, mediaID string, episodeID *int64) error {
			return expectedErr
		},
	}

	uc := NewRemoveFromContinueWatchingUseCase(mockRepo)

	err := uc.Execute(context.Background(), "user123", "media456", nil)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestRemoveFromContinueWatching_NonExistentItem tests removing non-existent item (should succeed)
func TestRemoveFromContinueWatching_NonExistentItem(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		RemoveFromContinueWatchingFn: func(ctx context.Context, userID, mediaID string, episodeID *int64) error {
			// Repository should handle non-existent items gracefully
			return nil
		},
	}

	uc := NewRemoveFromContinueWatchingUseCase(mockRepo)

	err := uc.Execute(context.Background(), "user123", "nonexistent", nil)
	require.NoError(t, err, "Removing non-existent item should succeed (idempotent)")
}

// TestRemoveFromContinueWatching_WithZeroEpisodeID tests with episode ID of 0
func TestRemoveFromContinueWatching_WithZeroEpisodeID(t *testing.T) {
	episodeID := int64(0)
	mockRepo := &mockWatchProgressRepository{
		RemoveFromContinueWatchingFn: func(ctx context.Context, userID, mediaID string, episodeIDPtr *int64) error {
			require.NotNil(t, episodeIDPtr)
			assert.Equal(t, int64(0), *episodeIDPtr)
			return nil
		},
	}

	uc := NewRemoveFromContinueWatchingUseCase(mockRepo)

	err := uc.Execute(context.Background(), "user123", "media456", &episodeID)
	require.NoError(t, err)
}
