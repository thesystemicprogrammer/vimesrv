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

// TestGetContinueWatching_Success tests successful continue watching retrieval
func TestGetContinueWatching_Success(t *testing.T) {
	expectedItems := []domain.ContinueWatchingItem{
		{
			WatchProgress: domain.WatchProgress{
				ID:              "progress1",
				UserID:          "user123",
				MediaID:         sql.NullString{String: "media1", Valid: true},
				PositionSeconds: 120,
				DurationSeconds: 300,
				ProgressPercent: 40.0,
				Completed:       false,
				LastWatchedAt:   time.Now().UTC(),
			},
			Title:        "Movie 1",
			PosterPath:   sql.NullString{String: "/poster1.jpg", Valid: true},
			BackdropPath: sql.NullString{String: "/backdrop1.jpg", Valid: true},
			MediaType:    "movie",
		},
		{
			WatchProgress: domain.WatchProgress{
				ID:                "progress2",
				UserID:            "user123",
				MediaID:           sql.NullString{String: "media2", Valid: true},
				EpisodeMetadataID: sql.NullInt64{Int64: 789, Valid: true},
				PositionSeconds:   600,
				DurationSeconds:   1800,
				ProgressPercent:   33.33,
				Completed:         false,
				LastWatchedAt:     time.Now().UTC().Add(-1 * time.Hour),
			},
			Title:         "Series 1 - S01E05",
			PosterPath:    sql.NullString{String: "/poster2.jpg", Valid: true},
			MediaType:     "episode",
			SeriesName:    sql.NullString{String: "Series 1", Valid: true},
			SeasonNumber:  sql.NullInt64{Int64: 1, Valid: true},
			EpisodeNumber: sql.NullInt64{Int64: 5, Valid: true},
		},
	}

	mockRepo := &mockWatchProgressRepository{
		GetContinueWatchingFn: func(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error) {
			assert.Equal(t, "user123", userID)
			assert.Equal(t, 20, limit)
			return expectedItems, nil
		},
	}

	uc := NewGetContinueWatchingUseCase(mockRepo)

	items, err := uc.Execute(context.Background(), "user123", 20)
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, expectedItems[0].Title, items[0].Title)
	assert.Equal(t, expectedItems[1].Title, items[1].Title)
	assert.Equal(t, "movie", items[0].MediaType)
	assert.Equal(t, "episode", items[1].MediaType)
}

// TestGetContinueWatching_Empty tests when user has no continue watching items
func TestGetContinueWatching_Empty(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		GetContinueWatchingFn: func(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error) {
			return []domain.ContinueWatchingItem{}, nil
		},
	}

	uc := NewGetContinueWatchingUseCase(mockRepo)

	items, err := uc.Execute(context.Background(), "user123", 20)
	require.NoError(t, err)
	assert.Len(t, items, 0)
	assert.NotNil(t, items) // Should return empty slice, not nil
}

// TestGetContinueWatching_DefaultLimit tests default limit application
func TestGetContinueWatching_DefaultLimit(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		GetContinueWatchingFn: func(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error) {
			assert.Equal(t, 20, limit, "Should use default limit of 20")
			return []domain.ContinueWatchingItem{}, nil
		},
	}

	uc := NewGetContinueWatchingUseCase(mockRepo)

	_, err := uc.Execute(context.Background(), "user123", 0) // 0 should default to 20
	require.NoError(t, err)
}

// TestGetContinueWatching_NegativeLimit tests negative limit defaults to 20
func TestGetContinueWatching_NegativeLimit(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		GetContinueWatchingFn: func(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error) {
			assert.Equal(t, 20, limit, "Negative limit should default to 20")
			return []domain.ContinueWatchingItem{}, nil
		},
	}

	uc := NewGetContinueWatchingUseCase(mockRepo)

	_, err := uc.Execute(context.Background(), "user123", -5)
	require.NoError(t, err)
}

// TestGetContinueWatching_CustomLimit tests custom limit
func TestGetContinueWatching_CustomLimit(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{
		GetContinueWatchingFn: func(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error) {
			assert.Equal(t, 50, limit)
			return []domain.ContinueWatchingItem{}, nil
		},
	}

	uc := NewGetContinueWatchingUseCase(mockRepo)

	_, err := uc.Execute(context.Background(), "user123", 50)
	require.NoError(t, err)
}

// TestGetContinueWatching_InvalidInput tests validation
func TestGetContinueWatching_InvalidInput(t *testing.T) {
	mockRepo := &mockWatchProgressRepository{}
	uc := NewGetContinueWatchingUseCase(mockRepo)

	_, err := uc.Execute(context.Background(), "", 20)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidInput, err)
}

// TestGetContinueWatching_RepositoryError tests repository error handling
func TestGetContinueWatching_RepositoryError(t *testing.T) {
	expectedErr := assert.AnError
	mockRepo := &mockWatchProgressRepository{
		GetContinueWatchingFn: func(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error) {
			return nil, expectedErr
		},
	}

	uc := NewGetContinueWatchingUseCase(mockRepo)

	_, err := uc.Execute(context.Background(), "user123", 20)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestGetContinueWatching_OrderedByLastWatched tests items are ordered correctly
func TestGetContinueWatching_OrderedByLastWatched(t *testing.T) {
	now := time.Now().UTC()
	expectedItems := []domain.ContinueWatchingItem{
		{
			WatchProgress: domain.WatchProgress{
				ID:              "progress1",
				UserID:          "user123",
				MediaID:         sql.NullString{String: "media1", Valid: true},
				PositionSeconds: 120,
				DurationSeconds: 300,
				LastWatchedAt:   now, // Most recent
			},
			Title: "Most Recent",
		},
		{
			WatchProgress: domain.WatchProgress{
				ID:              "progress2",
				UserID:          "user123",
				MediaID:         sql.NullString{String: "media2", Valid: true},
				PositionSeconds: 60,
				DurationSeconds: 300,
				LastWatchedAt:   now.Add(-2 * time.Hour), // Older
			},
			Title: "Older Item",
		},
	}

	mockRepo := &mockWatchProgressRepository{
		GetContinueWatchingFn: func(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error) {
			return expectedItems, nil
		},
	}

	uc := NewGetContinueWatchingUseCase(mockRepo)

	items, err := uc.Execute(context.Background(), "user123", 20)
	require.NoError(t, err)
	require.Len(t, items, 2)

	// Verify order: most recent first
	assert.Equal(t, "Most Recent", items[0].Title)
	assert.Equal(t, "Older Item", items[1].Title)
	assert.True(t, items[0].LastWatchedAt.After(items[1].LastWatchedAt))
}
