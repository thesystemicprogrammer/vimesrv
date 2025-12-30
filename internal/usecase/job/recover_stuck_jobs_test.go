package job

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
)

// TestRecoverStuckJobsUseCase_Execute_NoStuckJobs tests when no stuck jobs are found
func TestRecoverStuckJobsUseCase_Execute_NoStuckJobs(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		StuckJobThresholdMinutes: 30,
	}

	threshold := 30 * time.Minute
	mockRepo.On("FindStuckJobs", ctx, threshold).Return([]*domain.Job{}, nil)

	uc := NewRecoverStuckJobsUseCase(cfg, mockRepo, mockClock)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestRecoverStuckJobsUseCase_Execute_FindStuckJobsError tests error when finding stuck jobs
func TestRecoverStuckJobsUseCase_Execute_FindStuckJobsError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		StuckJobThresholdMinutes: 30,
	}

	threshold := 30 * time.Minute
	expectedErr := errors.New("database error")
	mockRepo.On("FindStuckJobs", ctx, threshold).Return(nil, expectedErr)

	uc := NewRecoverStuckJobsUseCase(cfg, mockRepo, mockClock)

	err := uc.Execute(ctx)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockRepo.AssertExpectations(t)
}

// TestRecoverStuckJobsUseCase_Execute_SingleStuckJob tests recovering a single stuck job
func TestRecoverStuckJobsUseCase_Execute_SingleStuckJob(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		StuckJobThresholdMinutes: 30,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	startedAt := now.Add(-45 * time.Minute) // Started 45 minutes ago

	stuckJobs := []*domain.Job{
		{
			ID:          1,
			Type:        "stuck-job",
			Status:      shared.StatusRunning,
			WorkerID:    sql.NullString{Valid: true, String: "worker-1"},
			StartedAt:   sql.NullTime{Valid: true, Time: startedAt},
			Attempts:    2,
			MaxAttempts: 3,
		},
	}

	threshold := 30 * time.Minute
	mockRepo.On("FindStuckJobs", ctx, threshold).Return(stuckJobs, nil)
	mockClock.On("Now").Return(now)
	mockRepo.On("ResetStuckJob", ctx, int64(1)).Return(nil)

	uc := NewRecoverStuckJobsUseCase(cfg, mockRepo, mockClock)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestRecoverStuckJobsUseCase_Execute_MultipleStuckJobs tests recovering multiple stuck jobs
func TestRecoverStuckJobsUseCase_Execute_MultipleStuckJobs(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		StuckJobThresholdMinutes: 30,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	stuckJobs := []*domain.Job{
		{
			ID:        1,
			Type:      "job-1",
			Status:    shared.StatusRunning,
			WorkerID:  sql.NullString{Valid: true, String: "worker-1"},
			StartedAt: sql.NullTime{Valid: true, Time: now.Add(-45 * time.Minute)},
			Attempts:  1,
		},
		{
			ID:        2,
			Type:      "job-2",
			Status:    shared.StatusRunning,
			WorkerID:  sql.NullString{Valid: true, String: "worker-2"},
			StartedAt: sql.NullTime{Valid: true, Time: now.Add(-60 * time.Minute)},
			Attempts:  2,
		},
		{
			ID:        3,
			Type:      "job-3",
			Status:    shared.StatusRunning,
			WorkerID:  sql.NullString{Valid: true, String: "worker-3"},
			StartedAt: sql.NullTime{Valid: true, Time: now.Add(-90 * time.Minute)},
			Attempts:  3,
		},
	}

	threshold := 30 * time.Minute
	mockRepo.On("FindStuckJobs", ctx, threshold).Return(stuckJobs, nil)
	mockClock.On("Now").Return(now)
	mockRepo.On("ResetStuckJob", ctx, int64(1)).Return(nil)
	mockRepo.On("ResetStuckJob", ctx, int64(2)).Return(nil)
	mockRepo.On("ResetStuckJob", ctx, int64(3)).Return(nil)

	uc := NewRecoverStuckJobsUseCase(cfg, mockRepo, mockClock)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestRecoverStuckJobsUseCase_Execute_ResetError tests handling of reset errors
func TestRecoverStuckJobsUseCase_Execute_ResetError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		StuckJobThresholdMinutes: 30,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	stuckJobs := []*domain.Job{
		{
			ID:        1,
			Type:      "job-1",
			Status:    shared.StatusRunning,
			WorkerID:  sql.NullString{Valid: true, String: "worker-1"},
			StartedAt: sql.NullTime{Valid: true, Time: now.Add(-45 * time.Minute)},
		},
		{
			ID:        2,
			Type:      "job-2",
			Status:    shared.StatusRunning,
			WorkerID:  sql.NullString{Valid: true, String: "worker-2"},
			StartedAt: sql.NullTime{Valid: true, Time: now.Add(-60 * time.Minute)},
		},
	}

	threshold := 30 * time.Minute
	mockRepo.On("FindStuckJobs", ctx, threshold).Return(stuckJobs, nil)
	mockClock.On("Now").Return(now)

	// First job fails to reset
	mockRepo.On("ResetStuckJob", ctx, int64(1)).Return(errors.New("database error"))

	// Second job should still be attempted and succeed
	mockRepo.On("ResetStuckJob", ctx, int64(2)).Return(nil)

	uc := NewRecoverStuckJobsUseCase(cfg, mockRepo, mockClock)

	err := uc.Execute(ctx)

	// Should not return error - errors are logged and processing continues
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestRecoverStuckJobsUseCase_Execute_ThresholdCalculation tests threshold calculation
func TestRecoverStuckJobsUseCase_Execute_ThresholdCalculation(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		StuckJobThresholdMinutes: 120, // 2 hours
	}

	threshold := 120 * time.Minute
	mockRepo.On("FindStuckJobs", ctx, threshold).Return([]*domain.Job{}, nil)

	uc := NewRecoverStuckJobsUseCase(cfg, mockRepo, mockClock)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestRecoverStuckJobsUseCase_Execute_JobWithoutWorkerID tests stuck job without worker ID
func TestRecoverStuckJobsUseCase_Execute_JobWithoutWorkerID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		StuckJobThresholdMinutes: 30,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	stuckJobs := []*domain.Job{
		{
			ID:        1,
			Type:      "orphaned-job",
			Status:    shared.StatusRunning,
			WorkerID:  sql.NullString{Valid: false}, // No worker ID
			StartedAt: sql.NullTime{Valid: true, Time: now.Add(-45 * time.Minute)},
			Attempts:  1,
		},
	}

	threshold := 30 * time.Minute
	mockRepo.On("FindStuckJobs", ctx, threshold).Return(stuckJobs, nil)
	mockClock.On("Now").Return(now)
	mockRepo.On("ResetStuckJob", ctx, int64(1)).Return(nil)

	uc := NewRecoverStuckJobsUseCase(cfg, mockRepo, mockClock)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestRecoverStuckJobsUseCase_Execute_RunningForCalculation tests that running duration is calculated correctly
func TestRecoverStuckJobsUseCase_Execute_RunningForCalculation(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		StuckJobThresholdMinutes: 30,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	startedAt := now.Add(-2 * time.Hour) // Started 2 hours ago

	stuckJobs := []*domain.Job{
		{
			ID:        1,
			Type:      "long-running-job",
			Status:    shared.StatusRunning,
			WorkerID:  sql.NullString{Valid: true, String: "worker-1"},
			StartedAt: sql.NullTime{Valid: true, Time: startedAt},
			Attempts:  1,
		},
	}

	threshold := 30 * time.Minute
	mockRepo.On("FindStuckJobs", ctx, threshold).Return(stuckJobs, nil)
	mockClock.On("Now").Return(now)
	mockRepo.On("ResetStuckJob", ctx, int64(1)).Return(nil)

	uc := NewRecoverStuckJobsUseCase(cfg, mockRepo, mockClock)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}
