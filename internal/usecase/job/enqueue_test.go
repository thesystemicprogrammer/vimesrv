package job

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// MockJobRepository is a mock implementation of ports.JobRepository
type MockJobRepository struct {
	mock.Mock
}

func (m *MockJobRepository) Enqueue(ctx context.Context, job *domain.Job) (int64, error) {
	args := m.Called(ctx, job)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockJobRepository) ClaimNextJobDue(ctx context.Context, workerID string) (*domain.Job, bool, error) {
	args := m.Called(ctx, workerID)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*domain.Job), args.Bool(1), args.Error(2)
}

func (m *MockJobRepository) ClaimNextJobDueExcludingTypes(ctx context.Context, workerID string, excludeTypes []string) (*domain.Job, bool, error) {
	args := m.Called(ctx, workerID, excludeTypes)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*domain.Job), args.Bool(1), args.Error(2)
}

func (m *MockJobRepository) MarkSuccess(ctx context.Context, jobID int64) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockJobRepository) MarkDead(ctx context.Context, jobID int64, lastError string) error {
	args := m.Called(ctx, jobID, lastError)
	return args.Error(0)
}

func (m *MockJobRepository) Reschedule(ctx context.Context, jobID int64, runAt time.Time, lastError string) error {
	args := m.Called(ctx, jobID, runAt, lastError)
	return args.Error(0)
}

func (m *MockJobRepository) FindStuckJobs(ctx context.Context, threshold time.Duration) ([]*domain.Job, error) {
	args := m.Called(ctx, threshold)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Job), args.Error(1)
}

func (m *MockJobRepository) ResetStuckJob(ctx context.Context, jobID int64) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockJobRepository) ExistsPendingJobByType(ctx context.Context, jobType string, language string) (bool, error) {
	args := m.Called(ctx, jobType, language)
	return args.Bool(0), args.Error(1)
}

func (m *MockJobRepository) ListJobs(ctx context.Context, filter ports.JobListFilter) (*ports.JobListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.JobListResult), args.Error(1)
}

func (m *MockJobRepository) Get(ctx context.Context, jobID int64) (*domain.Job, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Job), args.Error(1)
}

func (m *MockJobRepository) ClaimNextTranscodeJob(ctx context.Context, workerID string) (*domain.Job, error) {
	args := m.Called(ctx, workerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Job), args.Error(1)
}

func (m *MockJobRepository) ClaimNextTranscodeJobWithTypes(ctx context.Context, workerID string, allowedTypes []string) (*domain.Job, error) {
	args := m.Called(ctx, workerID, allowedTypes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Job), args.Error(1)
}

func (m *MockJobRepository) CountQueuedTranscodeJobs(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

// MockClock is a mock implementation of ports.Clock
type MockClock struct {
	mock.Mock
}

func (m *MockClock) Now() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

// TestEnqueueJobUseCase_Execute_Success tests successful job enqueueing
func TestEnqueueJobUseCase_Execute_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		MaxAttempts: 3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(now)

	// Mock successful enqueue
	mockRepo.On("Enqueue", ctx, mock.MatchedBy(func(job *domain.Job) bool {
		return job.Type == "test-job" &&
			job.Status == shared.StatusQueued &&
			job.Priority == 10 &&
			job.Attempts == 0 &&
			job.MaxAttempts == 3 &&
			job.RunAt.Equal(now)
	})).Return(int64(123), nil)

	uc := NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})

	jobID, err := uc.Execute(ctx, EnqueueJobInput{
		Type:     "test-job",
		Payload:  map[string]string{"key": "value"},
		Priority: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(123), jobID)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestEnqueueJobUseCase_Execute_EmptyType tests that empty job type is rejected
func TestEnqueueJobUseCase_Execute_EmptyType(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{MaxAttempts: 3}
	uc := NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})

	_, err := uc.Execute(ctx, EnqueueJobInput{
		Type:     "",
		Payload:  map[string]string{"key": "value"},
		Priority: 10,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "job type must not be empty")
}

// TestEnqueueJobUseCase_Execute_NilPayload tests enqueueing with nil payload
func TestEnqueueJobUseCase_Execute_NilPayload(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{MaxAttempts: 3}
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(now)

	mockRepo.On("Enqueue", ctx, mock.MatchedBy(func(job *domain.Job) bool {
		return job.Type == "test-job" && job.Payload == nil
	})).Return(int64(123), nil)

	uc := NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})

	jobID, err := uc.Execute(ctx, EnqueueJobInput{
		Type:     "test-job",
		Payload:  nil,
		Priority: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(123), jobID)
	mockRepo.AssertExpectations(t)
}

// TestEnqueueJobUseCase_Execute_InvalidPayload tests that unmarshalable payload is rejected
func TestEnqueueJobUseCase_Execute_InvalidPayload(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{MaxAttempts: 3}
	uc := NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})

	// channels cannot be marshaled to JSON
	invalidPayload := make(chan int)

	_, err := uc.Execute(ctx, EnqueueJobInput{
		Type:     "test-job",
		Payload:  invalidPayload,
		Priority: 10,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot marshalling json payload")
}

// TestEnqueueJobUseCase_Execute_CustomRunAt tests enqueueing with custom run time
func TestEnqueueJobUseCase_Execute_CustomRunAt(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{MaxAttempts: 3}
	futureTime := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	mockRepo.On("Enqueue", ctx, mock.MatchedBy(func(job *domain.Job) bool {
		return job.Type == "future-job" && job.RunAt.Equal(futureTime)
	})).Return(int64(456), nil)

	uc := NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})

	jobID, err := uc.Execute(ctx, EnqueueJobInput{
		Type:     "future-job",
		Payload:  nil,
		RunAt:    futureTime,
		Priority: 5,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(456), jobID)
	mockRepo.AssertExpectations(t)
}

// TestEnqueueJobUseCase_Execute_CustomMaxAttempts tests custom max attempts override
func TestEnqueueJobUseCase_Execute_CustomMaxAttempts(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{MaxAttempts: 3}
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(now)

	mockRepo.On("Enqueue", ctx, mock.MatchedBy(func(job *domain.Job) bool {
		return job.Type == "test-job" && job.MaxAttempts == 5
	})).Return(int64(789), nil)

	uc := NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})

	jobID, err := uc.Execute(ctx, EnqueueJobInput{
		Type:        "test-job",
		Payload:     nil,
		Priority:    10,
		MaxAttempts: 5, // Override default of 3
	})

	require.NoError(t, err)
	assert.Equal(t, int64(789), jobID)
	mockRepo.AssertExpectations(t)
}

// TestEnqueueJobUseCase_Execute_DefaultMaxAttempts tests default max attempts from config
func TestEnqueueJobUseCase_Execute_DefaultMaxAttempts(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{MaxAttempts: 7}
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(now)

	mockRepo.On("Enqueue", ctx, mock.MatchedBy(func(job *domain.Job) bool {
		return job.Type == "test-job" && job.MaxAttempts == 7
	})).Return(int64(111), nil)

	uc := NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})

	jobID, err := uc.Execute(ctx, EnqueueJobInput{
		Type:        "test-job",
		Payload:     nil,
		Priority:    10,
		MaxAttempts: 0, // Use default from config
	})

	require.NoError(t, err)
	assert.Equal(t, int64(111), jobID)
	mockRepo.AssertExpectations(t)
}

// TestEnqueueJobUseCase_Execute_PayloadMarshaling tests that payload is correctly marshaled
func TestEnqueueJobUseCase_Execute_PayloadMarshaling(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	cfg := config.JobConfig{MaxAttempts: 3}
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock.On("Now").Return(now)

	expectedPayload := map[string]interface{}{
		"foo":   "bar",
		"count": float64(42),
		"nested": map[string]interface{}{
			"key": "value",
		},
	}

	mockRepo.On("Enqueue", ctx, mock.MatchedBy(func(job *domain.Job) bool {
		if job.Type != "test-job" {
			return false
		}

		var unmarshaled map[string]interface{}
		err := json.Unmarshal(job.Payload, &unmarshaled)
		if err != nil {
			return false
		}

		return unmarshaled["foo"] == "bar" &&
			unmarshaled["count"] == float64(42)
	})).Return(int64(999), nil)

	uc := NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})

	jobID, err := uc.Execute(ctx, EnqueueJobInput{
		Type:     "test-job",
		Payload:  expectedPayload,
		Priority: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(999), jobID)
	mockRepo.AssertExpectations(t)
}
