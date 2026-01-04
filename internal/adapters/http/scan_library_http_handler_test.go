package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	usecasejob "github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// MockJobRepository is a mock implementation for testing
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

func (m *MockJobRepository) Reschedule(ctx context.Context, jobID int64, runAt time.Time, lastErr string) error {
	args := m.Called(ctx, jobID, runAt, lastErr)
	return args.Error(0)
}

func (m *MockJobRepository) MarkDead(ctx context.Context, jobID int64, lastErr string) error {
	args := m.Called(ctx, jobID, lastErr)
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

func (m *MockJobRepository) ExistsPendingJobByType(ctx context.Context, jobType string, payload string) (bool, error) {
	args := m.Called(ctx, jobType, payload)
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

func (m *MockJobRepository) CountQueuedTranscodeJobs(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

// MockClock for testing
type MockClock struct {
	mock.Mock
}

func (m *MockClock) Now() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

// TestScanLibraryHTTPHandler_Handle_Success tests successful library scan job enqueue
func TestScanLibraryHTTPHandler_Handle_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	// Setup expectations
	now := time.Now()
	expectedJobID := int64(123)
	mockClock.On("Now").Return(now)
	mockRepo.On("Enqueue", mock.Anything, mock.MatchedBy(func(job *domain.Job) bool {
		return job.Type == shared.JobTypeScanLibrary &&
			job.Payload == nil &&
			job.Priority == 0 &&
			job.Status == shared.StatusQueued
	})).Return(expectedJobID, nil)

	// Create use case and handler
	cfg := config.JobConfig{MaxAttempts: 3}
	enqueueUC := usecasejob.NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})
	handler := NewScanLibraryHTTPHandler(enqueueUC)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/library/scan", nil)

	// Execute
	handler.Handle(c)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"job_id":123`)
	assert.Contains(t, w.Body.String(), `"success":true`)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestScanLibraryHTTPHandler_Handle_EnqueueError tests error handling when enqueue fails
func TestScanLibraryHTTPHandler_Handle_EnqueueError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	// Setup expectations - enqueue fails
	now := time.Now()
	mockClock.On("Now").Return(now)
	mockRepo.On("Enqueue", mock.Anything, mock.Anything).
		Return(int64(0), errors.New("database connection failed"))

	cfg := config.JobConfig{MaxAttempts: 3}
	enqueueUC := usecasejob.NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})
	handler := NewScanLibraryHTTPHandler(enqueueUC)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/library/scan", nil)

	handler.Handle(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"success":false`)
	assert.Contains(t, w.Body.String(), `"code":"ENQUEUE_FAILED"`)
	assert.Contains(t, w.Body.String(), "Failed to enqueue scan job")
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestScanLibraryHTTPHandler_Handle_ContextCanceled tests handling of canceled context
func TestScanLibraryHTTPHandler_Handle_ContextCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	// Setup expectations - context canceled
	now := time.Now()
	mockClock.On("Now").Return(now)
	mockRepo.On("Enqueue", mock.Anything, mock.Anything).
		Return(int64(0), context.Canceled)

	cfg := config.JobConfig{MaxAttempts: 3}
	enqueueUC := usecasejob.NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})
	handler := NewScanLibraryHTTPHandler(enqueueUC)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/library/scan", nil)

	handler.Handle(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"success":false`)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestScanLibraryHTTPHandler_Handle_MultipleRequests tests that multiple requests work independently
func TestScanLibraryHTTPHandler_Handle_MultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	// Setup expectations for multiple calls
	now := time.Now()
	mockClock.On("Now").Return(now).Times(3)
	mockRepo.On("Enqueue", mock.Anything, mock.Anything).
		Return(int64(1), nil).Once()
	mockRepo.On("Enqueue", mock.Anything, mock.Anything).
		Return(int64(2), nil).Once()
	mockRepo.On("Enqueue", mock.Anything, mock.Anything).
		Return(int64(3), nil).Once()

	cfg := config.JobConfig{MaxAttempts: 3}
	enqueueUC := usecasejob.NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})
	handler := NewScanLibraryHTTPHandler(enqueueUC)

	// First request
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request, _ = http.NewRequest("POST", "/api/v1/library/scan", nil)
	handler.Handle(c1)

	assert.Equal(t, http.StatusCreated, w1.Code)
	assert.Contains(t, w1.Body.String(), `"job_id":1`)

	// Second request
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request, _ = http.NewRequest("POST", "/api/v1/library/scan", nil)
	handler.Handle(c2)

	assert.Equal(t, http.StatusCreated, w2.Code)
	assert.Contains(t, w2.Body.String(), `"job_id":2`)

	// Third request
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	c3.Request, _ = http.NewRequest("POST", "/api/v1/library/scan", nil)
	handler.Handle(c3)

	assert.Equal(t, http.StatusCreated, w3.Code)
	assert.Contains(t, w3.Body.String(), `"job_id":3`)

	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestScanLibraryHTTPHandler_Handle_JobTypeConstant tests that correct job type is used
func TestScanLibraryHTTPHandler_Handle_JobTypeConstant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	// Verify that the exact job type constant is used
	now := time.Now()
	mockClock.On("Now").Return(now)
	mockRepo.On("Enqueue",
		mock.Anything,
		mock.MatchedBy(func(job *domain.Job) bool {
			// Ensure the job type is exactly the scan library constant
			if job.Type != shared.JobTypeScanLibrary {
				t.Errorf("Expected job type %s, got %s", shared.JobTypeScanLibrary, job.Type)
				return false
			}
			if job.Type != "scan_library" {
				t.Errorf("JobTypeScanLibrary constant changed, expected 'scan_library', got %s", job.Type)
				return false
			}
			return true
		}),
	).Return(int64(999), nil)

	cfg := config.JobConfig{MaxAttempts: 3}
	enqueueUC := usecasejob.NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})
	handler := NewScanLibraryHTTPHandler(enqueueUC)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/library/scan", nil)

	handler.Handle(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestScanLibraryHTTPHandler_Handle_NilPayload tests that payload is explicitly nil
func TestScanLibraryHTTPHandler_Handle_NilPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockJobRepository)
	mockClock := new(MockClock)

	// Verify payload is nil (uses config.Media.LibraryPath instead)
	now := time.Now()
	mockClock.On("Now").Return(now)
	mockRepo.On("Enqueue",
		mock.Anything,
		mock.MatchedBy(func(job *domain.Job) bool {
			if job.Payload != nil {
				t.Errorf("Expected nil payload, got %v", job.Payload)
				return false
			}
			return true
		}),
	).Return(int64(456), nil)

	cfg := config.JobConfig{MaxAttempts: 3}
	enqueueUC := usecasejob.NewEnqueueJobUseCase(cfg, mockRepo, mockClock, &ports.NoOpJobNotifier{})
	handler := NewScanLibraryHTTPHandler(enqueueUC)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/library/scan", nil)

	handler.Handle(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}
