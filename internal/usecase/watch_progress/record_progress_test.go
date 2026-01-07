package watch_progress

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// MockWatchProgressRepository for testing
type mockWatchProgressRepository struct {
	SaveProgressFn               func(ctx context.Context, progress *domain.WatchProgress) error
	GetProgressFn                func(ctx context.Context, userID, mediaID string, episodeID *int64) (*domain.WatchProgress, error)
	GetContinueWatchingFn        func(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error)
	GetWatchHistoryFn            func(ctx context.Context, userID string, page, perPage int) ([]domain.ContinueWatchingItem, int, error)
	RemoveFromContinueWatchingFn func(ctx context.Context, userID, mediaID string, episodeID *int64) error
	DeleteHistoryFn              func(ctx context.Context, userID string) error
	MarkAsCompletedFn            func(ctx context.Context, userID, mediaID string, episodeID *int64) error
}

func (m *mockWatchProgressRepository) SaveProgress(ctx context.Context, progress *domain.WatchProgress) error {
	if m.SaveProgressFn != nil {
		return m.SaveProgressFn(ctx, progress)
	}
	return nil
}

func (m *mockWatchProgressRepository) GetProgress(ctx context.Context, userID, mediaID string, episodeID *int64) (*domain.WatchProgress, error) {
	if m.GetProgressFn != nil {
		return m.GetProgressFn(ctx, userID, mediaID, episodeID)
	}
	return nil, nil
}

func (m *mockWatchProgressRepository) GetContinueWatching(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error) {
	if m.GetContinueWatchingFn != nil {
		return m.GetContinueWatchingFn(ctx, userID, limit)
	}
	return nil, nil
}

func (m *mockWatchProgressRepository) GetWatchHistory(ctx context.Context, userID string, page, perPage int) ([]domain.ContinueWatchingItem, int, error) {
	if m.GetWatchHistoryFn != nil {
		return m.GetWatchHistoryFn(ctx, userID, page, perPage)
	}
	return nil, 0, nil
}

func (m *mockWatchProgressRepository) RemoveFromContinueWatching(ctx context.Context, userID, mediaID string, episodeID *int64) error {
	if m.RemoveFromContinueWatchingFn != nil {
		return m.RemoveFromContinueWatchingFn(ctx, userID, mediaID, episodeID)
	}
	return nil
}

func (m *mockWatchProgressRepository) DeleteHistory(ctx context.Context, userID string) error {
	if m.DeleteHistoryFn != nil {
		return m.DeleteHistoryFn(ctx, userID)
	}
	return nil
}

func (m *mockWatchProgressRepository) MarkAsCompleted(ctx context.Context, userID, mediaID string, episodeID *int64) error {
	if m.MarkAsCompletedFn != nil {
		return m.MarkAsCompletedFn(ctx, userID, mediaID, episodeID)
	}
	return nil
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

// TestRecordProgress_Success tests successful progress recording for a movie
func TestRecordProgress_Success(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		SaveProgressFn: func(ctx context.Context, progress *domain.WatchProgress) error {
			assert.Equal(t, "user123", progress.UserID)
			assert.True(t, progress.MediaID.Valid)
			assert.Equal(t, "media456", progress.MediaID.String)
			assert.Equal(t, 120, progress.PositionSeconds)
			assert.Equal(t, 300, progress.DurationSeconds)
			assert.False(t, progress.Completed)
			assert.InDelta(t, 40.0, progress.ProgressPercent, 0.1)
			return nil
		},
	}

	uc := NewRecordWatchProgressUseCase(mockRepo)

	input := RecordProgressInput{
		UserID:          "user123",
		MediaID:         stringPtr("media456"),
		PositionSeconds: 120,
		DurationSeconds: 300,
	}

	err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
}

// TestRecordProgress_AutoComplete tests auto-completion at 95%
func TestRecordProgress_AutoComplete(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		SaveProgressFn: func(ctx context.Context, progress *domain.WatchProgress) error {
			assert.Equal(t, "user123", progress.UserID)
			assert.True(t, progress.MediaID.Valid)
			assert.Equal(t, "media456", progress.MediaID.String)
			assert.Equal(t, 290, progress.PositionSeconds)
			assert.Equal(t, 300, progress.DurationSeconds)
			assert.True(t, progress.Completed, "Should auto-complete at 95%")
			assert.InDelta(t, 96.67, progress.ProgressPercent, 0.1)
			return nil
		},
	}

	uc := NewRecordWatchProgressUseCase(mockRepo)

	input := RecordProgressInput{
		UserID:          "user123",
		MediaID:         stringPtr("media456"),
		PositionSeconds: 290, // 96.67%
		DurationSeconds: 300,
	}

	err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
}

// TestRecordProgress_WithEpisode tests recording progress for a TV episode
func TestRecordProgress_WithEpisode(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		SaveProgressFn: func(ctx context.Context, progress *domain.WatchProgress) error {
			assert.Equal(t, "user123", progress.UserID)
			assert.True(t, progress.EpisodeMetadataID.Valid)
			assert.Equal(t, int64(789), progress.EpisodeMetadataID.Int64)
			assert.False(t, progress.MediaID.Valid, "MediaID should not be set for episodes")
			return nil
		},
	}

	uc := NewRecordWatchProgressUseCase(mockRepo)

	input := RecordProgressInput{
		UserID:          "user123",
		EpisodeID:       int64Ptr(789),
		PositionSeconds: 60,
		DurationSeconds: 1800,
	}

	err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
}

// TestRecordProgress_InvalidInput tests validation
func TestRecordProgress_InvalidInput(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{}
	uc := NewRecordWatchProgressUseCase(mockRepo)

	tests := []struct {
		name  string
		input RecordProgressInput
	}{
		{
			name: "empty user ID",
			input: RecordProgressInput{
				UserID:          "",
				MediaID:         stringPtr("media456"),
				PositionSeconds: 120,
				DurationSeconds: 300,
			},
		},
		{
			name: "no media ID or episode ID",
			input: RecordProgressInput{
				UserID:          "user123",
				PositionSeconds: 120,
				DurationSeconds: 300,
			},
		},
		{
			name: "negative position",
			input: RecordProgressInput{
				UserID:          "user123",
				MediaID:         stringPtr("media456"),
				PositionSeconds: -10,
				DurationSeconds: 300,
			},
		},
		{
			name: "zero duration",
			input: RecordProgressInput{
				UserID:          "user123",
				MediaID:         stringPtr("media456"),
				PositionSeconds: 120,
				DurationSeconds: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.Execute(context.Background(), tt.input)
			assert.Error(t, err)
		})
	}
}

// TestRecordProgress_PositionExceedsDuration tests when position exceeds duration (capped at 100%)
func TestRecordProgress_PositionExceedsDuration(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		SaveProgressFn: func(ctx context.Context, progress *domain.WatchProgress) error {
			// Should cap at 100%
			assert.Equal(t, 400, progress.PositionSeconds)
			assert.InDelta(t, 100.0, progress.ProgressPercent, 0.1)
			assert.True(t, progress.Completed)
			return nil
		},
	}

	uc := NewRecordWatchProgressUseCase(mockRepo)

	input := RecordProgressInput{
		UserID:          "user123",
		MediaID:         stringPtr("media456"),
		PositionSeconds: 400,
		DurationSeconds: 300,
	}

	err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
}

// TestRecordProgress_RepositoryError tests repository error handling
func TestRecordProgress_RepositoryError(t *testing.T) {
	expectedErr := assert.AnError
	mockRepo := &mockWatchProgressRepository{
		SaveProgressFn: func(ctx context.Context, progress *domain.WatchProgress) error {
			return expectedErr
		},
	}

	uc := NewRecordWatchProgressUseCase(mockRepo)

	input := RecordProgressInput{
		UserID:          "user123",
		MediaID:         stringPtr("media456"),
		PositionSeconds: 120,
		DurationSeconds: 300,
	}

	err := uc.Execute(context.Background(), input)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestRecordProgress_ExactThreshold tests the exact 95% threshold
func TestRecordProgress_ExactThreshold(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		SaveProgressFn: func(ctx context.Context, progress *domain.WatchProgress) error {
			assert.True(t, progress.Completed, "Should be completed at exactly 95%")
			assert.InDelta(t, 95.0, progress.ProgressPercent, 0.1)
			return nil
		},
	}

	uc := NewRecordWatchProgressUseCase(mockRepo)

	input := RecordProgressInput{
		UserID:          "user123",
		MediaID:         stringPtr("media456"),
		PositionSeconds: 285, // Exactly 95% of 300
		DurationSeconds: 300,
	}

	err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
}

// TestRecordProgress_BelowThreshold tests just below the 95% threshold
func TestRecordProgress_BelowThreshold(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		SaveProgressFn: func(ctx context.Context, progress *domain.WatchProgress) error {
			assert.False(t, progress.Completed, "Should not be completed below 95%")
			assert.InDelta(t, 94.67, progress.ProgressPercent, 0.1)
			return nil
		},
	}

	uc := NewRecordWatchProgressUseCase(mockRepo)

	input := RecordProgressInput{
		UserID:          "user123",
		MediaID:         stringPtr("media456"),
		PositionSeconds: 284, // Just below 95% of 300
		DurationSeconds: 300,
	}

	err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
}
