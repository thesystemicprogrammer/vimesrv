package watch_progress

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// TestGetProgress_SuccessMovie tests successful progress retrieval for a movie
func TestGetProgress_SuccessMovie(t *testing.T) {
	expectedProgress := &domain.WatchProgress{
		ID:              "progress123",
		UserID:          "user123",
		MediaID:         sql.NullString{String: "media456", Valid: true},
		PositionSeconds: 120,
		DurationSeconds: 300,
		ProgressPercent: 40.0,
		Completed:       false,
		LastWatchedAt:   time.Now().UTC(),
	}

	mockRepo := &mockWatchProgressRepository{
		GetProgressFn: func(ctx context.Context, userID, mediaID string, episodeID *int64) (*domain.WatchProgress, error) {
			assert.Equal(t, "user123", userID)
			assert.Equal(t, "media456", mediaID)
			assert.Nil(t, episodeID)
			return expectedProgress, nil
		},
	}

	uc := NewGetWatchProgressUseCase(mockRepo)

	progress, err := uc.Execute(context.Background(), "user123", "media456", nil)
	require.NoError(t, err)
	require.NotNil(t, progress)
	assert.Equal(t, expectedProgress.ID, progress.ID)
	assert.Equal(t, expectedProgress.UserID, progress.UserID)
	assert.Equal(t, expectedProgress.PositionSeconds, progress.PositionSeconds)
}

// TestGetProgress_SuccessEpisode tests progress retrieval for a TV episode
func TestGetProgress_SuccessEpisode(t *testing.T) {
	episodeID := int64(789)
	expectedProgress := &domain.WatchProgress{
		ID:                "progress123",
		UserID:            "user123",
		MediaID:           sql.NullString{String: "media456", Valid: true},
		EpisodeMetadataID: sql.NullInt64{Int64: episodeID, Valid: true},
		PositionSeconds:   600,
		DurationSeconds:   1800,
		ProgressPercent:   33.33,
		Completed:         false,
		LastWatchedAt:     time.Now().UTC(),
	}

	mockRepo := &mockWatchProgressRepository{
		GetProgressFn: func(ctx context.Context, userID, mediaID string, episodeIDPtr *int64) (*domain.WatchProgress, error) {
			assert.Equal(t, "user123", userID)
			assert.Equal(t, "media456", mediaID)
			require.NotNil(t, episodeIDPtr)
			assert.Equal(t, int64(789), *episodeIDPtr)
			return expectedProgress, nil
		},
	}

	uc := NewGetWatchProgressUseCase(mockRepo)

	progress, err := uc.Execute(context.Background(), "user123", "media456", &episodeID)
	require.NoError(t, err)
	require.NotNil(t, progress)
	assert.Equal(t, expectedProgress.ID, progress.ID)
	assert.True(t, progress.EpisodeMetadataID.Valid)
	assert.Equal(t, int64(789), progress.EpisodeMetadataID.Int64)
}

// TestGetProgress_NotFound tests when progress is not found
func TestGetProgress_NotFound(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		GetProgressFn: func(ctx context.Context, userID, mediaID string, episodeID *int64) (*domain.WatchProgress, error) {
			return nil, nil
		},
	}

	uc := NewGetWatchProgressUseCase(mockRepo)

	progress, err := uc.Execute(context.Background(), "user123", "media456", nil)
	require.NoError(t, err)
	assert.Nil(t, progress)
}

// TestGetProgress_InvalidInput tests validation
func TestGetProgress_InvalidInput(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{}
	uc := NewGetWatchProgressUseCase(mockRepo)

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
			_, err := uc.Execute(context.Background(), tt.userID, tt.mediaID, tt.episodeID)
			assert.Error(t, err)
			assert.Equal(t, ErrInvalidInput, err)
		})
	}
}

// TestGetProgress_RepositoryError tests repository error handling
func TestGetProgress_RepositoryError(t *testing.T) {
	expectedErr := assert.AnError
	mockRepo := &mockWatchProgressRepository{
		GetProgressFn: func(ctx context.Context, userID, mediaID string, episodeID *int64) (*domain.WatchProgress, error) {
			return nil, expectedErr
		},
	}

	uc := NewGetWatchProgressUseCase(mockRepo)

	_, err := uc.Execute(context.Background(), "user123", "media456", nil)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}
