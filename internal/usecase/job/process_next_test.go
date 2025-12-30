package job

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// MockHandlerResolver is a mock implementation of ports.HandlerResolver
type MockHandlerResolver struct {
	mock.Mock
}

func (m *MockHandlerResolver) Get(jobType string) (ports.JobHandler, bool) {
	args := m.Called(jobType)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(ports.JobHandler), args.Bool(1)
}

// MockBackoffStrategy is a mock implementation of ports.BackoffStrategy
type MockBackoffStrategy struct {
	mock.Mock
}

func (m *MockBackoffStrategy) NextDelay(attempt int) time.Duration {
	args := m.Called(attempt)
	return args.Get(0).(time.Duration)
}

// TestProcessNextJobUseCase_Execute_NoJobAvailable tests when no job is available
func TestProcessNextJobUseCase_Execute_NoJobAvailable(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockResolver := new(MockHandlerResolver)
	mockBackoff := new(MockBackoffStrategy)
	mockClock := new(MockClock)

	mockRepo.On("ClaimNextJobDue", ctx, "worker-1").Return(nil, false, nil)

	uc := NewProcessNextJobUseCase(mockRepo, mockResolver, mockBackoff, mockClock)

	found, err := uc.Execute(ctx, "worker-1")

	require.NoError(t, err)
	assert.False(t, found)
	mockRepo.AssertExpectations(t)
}

// TestProcessNextJobUseCase_Execute_ClaimError tests error while claiming job
func TestProcessNextJobUseCase_Execute_ClaimError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockResolver := new(MockHandlerResolver)
	mockBackoff := new(MockBackoffStrategy)
	mockClock := new(MockClock)

	expectedErr := errors.New("database error")
	mockRepo.On("ClaimNextJobDue", ctx, "worker-1").Return(nil, false, expectedErr)

	uc := NewProcessNextJobUseCase(mockRepo, mockResolver, mockBackoff, mockClock)

	found, err := uc.Execute(ctx, "worker-1")

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.False(t, found)
	mockRepo.AssertExpectations(t)
}

// TestProcessNextJobUseCase_Execute_SuccessfulJob tests successful job execution
func TestProcessNextJobUseCase_Execute_SuccessfulJob(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockResolver := new(MockHandlerResolver)
	mockBackoff := new(MockBackoffStrategy)
	mockClock := new(MockClock)

	job := &domain.Job{
		ID:          1,
		Type:        "test-job",
		Status:      shared.StatusRunning,
		Attempts:    1,
		MaxAttempts: 3,
	}

	startTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	endTime := startTime.Add(100 * time.Millisecond)

	mockRepo.On("ClaimNextJobDue", ctx, "worker-1").Return(job, true, nil)
	mockClock.On("Now").Return(startTime).Once()
	mockClock.On("Now").Return(endTime).Once()

	handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		return nil // Success
	})
	mockResolver.On("Get", "test-job").Return(handler, true)
	mockRepo.On("MarkSuccess", mock.Anything, int64(1)).Return(nil)

	uc := NewProcessNextJobUseCase(mockRepo, mockResolver, mockBackoff, mockClock)

	found, err := uc.Execute(ctx, "worker-1")

	require.NoError(t, err)
	assert.True(t, found)
	mockRepo.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestProcessNextJobUseCase_Execute_HandlerNotFound tests when handler is not registered
func TestProcessNextJobUseCase_Execute_HandlerNotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockResolver := new(MockHandlerResolver)
	mockBackoff := new(MockBackoffStrategy)
	mockClock := new(MockClock)

	job := &domain.Job{
		ID:          1,
		Type:        "unknown-job",
		Status:      shared.StatusRunning,
		Attempts:    1,
		MaxAttempts: 3,
	}

	mockRepo.On("ClaimNextJobDue", ctx, "worker-1").Return(job, true, nil)
	mockResolver.On("Get", "unknown-job").Return(nil, false)
	mockRepo.On("MarkDead", mock.Anything, int64(1), "no handler registered").Return(nil)

	uc := NewProcessNextJobUseCase(mockRepo, mockResolver, mockBackoff, mockClock)

	found, err := uc.Execute(ctx, "worker-1")

	require.NoError(t, err)
	assert.True(t, found)
	mockRepo.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
}

// TestProcessNextJobUseCase_Execute_JobFailsButCanRetry tests job failure with retry
func TestProcessNextJobUseCase_Execute_JobFailsButCanRetry(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockResolver := new(MockHandlerResolver)
	mockBackoff := new(MockBackoffStrategy)
	mockClock := new(MockClock)

	job := &domain.Job{
		ID:          1,
		Type:        "flaky-job",
		Status:      shared.StatusRunning,
		Attempts:    1,
		MaxAttempts: 3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	mockRepo.On("ClaimNextJobDue", ctx, "worker-1").Return(job, true, nil)
	mockClock.On("Now").Return(now).Times(3) // start time, end time, and next run calculation

	jobError := errors.New("temporary failure")
	handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		return jobError
	})
	mockResolver.On("Get", "flaky-job").Return(handler, true)

	mockBackoff.On("NextDelay", 1).Return(5 * time.Second)
	expectedRunAt := now.Add(5 * time.Second)
	mockRepo.On("Reschedule", mock.Anything, int64(1), expectedRunAt, "temporary failure").Return(nil)

	uc := NewProcessNextJobUseCase(mockRepo, mockResolver, mockBackoff, mockClock)

	found, err := uc.Execute(ctx, "worker-1")

	require.NoError(t, err)
	assert.True(t, found)
	mockRepo.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
	mockBackoff.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestProcessNextJobUseCase_Execute_MaxAttemptsExceeded tests job marked dead after max attempts
func TestProcessNextJobUseCase_Execute_MaxAttemptsExceeded(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockResolver := new(MockHandlerResolver)
	mockBackoff := new(MockBackoffStrategy)
	mockClock := new(MockClock)

	job := &domain.Job{
		ID:          1,
		Type:        "failing-job",
		Status:      shared.StatusRunning,
		Attempts:    3, // Already at max
		MaxAttempts: 3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	mockRepo.On("ClaimNextJobDue", ctx, "worker-1").Return(job, true, nil)
	mockClock.On("Now").Return(now).Times(2) // start and end (no next run needed for dead jobs)

	jobError := errors.New("permanent failure")
	handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		return jobError
	})
	mockResolver.On("Get", "failing-job").Return(handler, true)
	mockRepo.On("MarkDead", mock.Anything, int64(1), "permanent failure").Return(nil)

	uc := NewProcessNextJobUseCase(mockRepo, mockResolver, mockBackoff, mockClock)

	found, err := uc.Execute(ctx, "worker-1")

	require.NoError(t, err)
	assert.True(t, found)
	mockRepo.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestProcessNextJobUseCase_Execute_HandlerPanic tests panic recovery
func TestProcessNextJobUseCase_Execute_HandlerPanic(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockResolver := new(MockHandlerResolver)
	mockBackoff := new(MockBackoffStrategy)
	mockClock := new(MockClock)

	job := &domain.Job{
		ID:          1,
		Type:        "panic-job",
		Status:      shared.StatusRunning,
		Attempts:    1,
		MaxAttempts: 3,
	}

	mockRepo.On("ClaimNextJobDue", ctx, "worker-1").Return(job, true, nil)
	mockClock.On("Now").Return(time.Now()).Once() // Only called once for start time, panic prevents end time call

	handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		panic("something went terribly wrong")
	})
	mockResolver.On("Get", "panic-job").Return(handler, true)
	mockRepo.On("MarkDead", mock.Anything, int64(1), mock.MatchedBy(func(msg string) bool {
		return len(msg) > 0 // Check that error message contains panic info
	})).Return(nil)

	uc := NewProcessNextJobUseCase(mockRepo, mockResolver, mockBackoff, mockClock)

	// Should not panic, should recover gracefully
	found, err := uc.Execute(ctx, "worker-1")

	// The function should complete without propagating the panic
	require.NoError(t, err)
	assert.True(t, found)
	mockRepo.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
}

// TestProcessNextJobUseCase_Execute_ContextCancellation tests graceful handling of context cancellation
func TestProcessNextJobUseCase_Execute_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mockRepo := new(MockJobRepository)
	mockResolver := new(MockHandlerResolver)
	mockBackoff := new(MockBackoffStrategy)
	mockClock := new(MockClock)

	job := &domain.Job{
		ID:          1,
		Type:        "slow-job",
		Status:      shared.StatusRunning,
		Attempts:    1,
		MaxAttempts: 3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	mockRepo.On("ClaimNextJobDue", ctx, "worker-1").Return(job, true, nil)
	mockClock.On("Now").Return(now).Times(3) // start, end, and next run calculation

	handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		cancel() // Cancel context during execution
		return ctx.Err()
	})
	mockResolver.On("Get", "slow-job").Return(handler, true)

	mockBackoff.On("NextDelay", 1).Return(5 * time.Second)
	expectedRunAt := now.Add(5 * time.Second)

	// First call with canceled context should fail, then retry with background context
	mockRepo.On("Reschedule", mock.Anything, int64(1), expectedRunAt, "context canceled").Return(nil)

	uc := NewProcessNextJobUseCase(mockRepo, mockResolver, mockBackoff, mockClock)

	found, err := uc.Execute(ctx, "worker-1")

	require.NoError(t, err)
	assert.True(t, found)
	mockRepo.AssertExpectations(t)
}

// TestProcessNextJobUseCase_Execute_StateTransitionRetry tests retry mechanism for state transitions
func TestProcessNextJobUseCase_Execute_StateTransitionRetry(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockResolver := new(MockHandlerResolver)
	mockBackoff := new(MockBackoffStrategy)
	mockClock := new(MockClock)

	job := &domain.Job{
		ID:          1,
		Type:        "test-job",
		Status:      shared.StatusRunning,
		Attempts:    1,
		MaxAttempts: 3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	mockRepo.On("ClaimNextJobDue", ctx, "worker-1").Return(job, true, nil)
	mockClock.On("Now").Return(now).Times(2)

	handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		return nil // Success
	})
	mockResolver.On("Get", "test-job").Return(handler, true)

	// First call fails, second succeeds (tests retry logic)
	mockRepo.On("MarkSuccess", mock.Anything, int64(1)).Return(errors.New("transient db error")).Once()
	mockRepo.On("MarkSuccess", mock.Anything, int64(1)).Return(nil).Once()

	uc := NewProcessNextJobUseCase(mockRepo, mockResolver, mockBackoff, mockClock)

	found, err := uc.Execute(ctx, "worker-1")

	require.NoError(t, err)
	assert.True(t, found)
	mockRepo.AssertExpectations(t)
}

// TestProcessNextJobUseCase_Execute_ScheduledJobTracking tests that scheduled_id is preserved
func TestProcessNextJobUseCase_Execute_ScheduledJobTracking(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockResolver := new(MockHandlerResolver)
	mockBackoff := new(MockBackoffStrategy)
	mockClock := new(MockClock)

	job := &domain.Job{
		ID:          1,
		Type:        "scheduled-job",
		Status:      shared.StatusRunning,
		Attempts:    1,
		MaxAttempts: 3,
		ScheduledID: sql.NullInt64{Valid: true, Int64: 42}, // From schedule
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	mockRepo.On("ClaimNextJobDue", ctx, "worker-1").Return(job, true, nil)
	mockClock.On("Now").Return(now).Times(2)

	handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		// Verify the scheduled_id is accessible in handler
		assert.True(t, job.ScheduledID.Valid)
		assert.Equal(t, int64(42), job.ScheduledID.Int64)
		return nil
	})
	mockResolver.On("Get", "scheduled-job").Return(handler, true)
	mockRepo.On("MarkSuccess", mock.Anything, int64(1)).Return(nil)

	uc := NewProcessNextJobUseCase(mockRepo, mockResolver, mockBackoff, mockClock)

	found, err := uc.Execute(ctx, "worker-1")

	require.NoError(t, err)
	assert.True(t, found)
	mockRepo.AssertExpectations(t)
}
