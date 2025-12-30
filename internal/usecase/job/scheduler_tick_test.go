package job

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// MockScheduleRepository is a mock implementation of ports.ScheduleRepository
type MockScheduleRepository struct {
	mock.Mock
}

func (m *MockScheduleRepository) Upsert(ctx context.Context, s *domain.Schedule) (int64, error) {
	args := m.Called(ctx, s)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockScheduleRepository) GetByName(ctx context.Context, name string) (*domain.Schedule, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Schedule), args.Error(1)
}

func (m *MockScheduleRepository) SetNextRunIfNull(ctx context.Context, id int64, next time.Time) error {
	args := m.Called(ctx, id, next)
	return args.Error(0)
}

func (m *MockScheduleRepository) SetNextRun(ctx context.Context, id int64, next time.Time) error {
	args := m.Called(ctx, id, next)
	return args.Error(0)
}

func (m *MockScheduleRepository) ListDue(ctx context.Context, limit int) ([]*domain.Schedule, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Schedule), args.Error(1)
}

func (m *MockScheduleRepository) AdvanceAndEnqueue(ctx context.Context, scheduleID int64, next time.Time, jobProto *domain.Job) error {
	args := m.Called(ctx, scheduleID, next, jobProto)
	return args.Error(0)
}

// MockCronSchedule is a mock implementation of ports.CronSchedule
type MockCronSchedule struct {
	mock.Mock
}

func (m *MockCronSchedule) Next(from time.Time) time.Time {
	args := m.Called(from)
	return args.Get(0).(time.Time)
}

// MockCronParser is a mock implementation of ports.CronParser
type MockCronParser struct {
	mock.Mock
}

func (m *MockCronParser) Parse(spec string) (ports.CronSchedule, error) {
	args := m.Called(spec)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.CronSchedule), args.Error(1)
}

// TestSchedulerTickUseCase_Execute_NoDueSchedules tests when no schedules are due
func TestSchedulerTickUseCase_Execute_NoDueSchedules(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		SchedulerBatch: 10,
		MaxAttempts:    3,
	}

	mockRepo.On("ListDue", ctx, 10).Return([]*domain.Schedule{}, nil)
	// Clock is called even if no schedules are due (line 37 in scheduler_tick.go)
	mockClock.On("Now").Return(time.Now())

	uc := NewSchedulerTickUseCase(cfg, mockRepo, mockParser, mockClock)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
	// Parser should not be called if no schedules are due
}

// TestSchedulerTickUseCase_Execute_ListDueError tests error when listing due schedules
func TestSchedulerTickUseCase_Execute_ListDueError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		SchedulerBatch: 10,
		MaxAttempts:    3,
	}

	expectedErr := errors.New("database error")
	mockRepo.On("ListDue", ctx, 10).Return(nil, expectedErr)

	uc := NewSchedulerTickUseCase(cfg, mockRepo, mockParser, mockClock)

	err := uc.Execute(ctx)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockRepo.AssertExpectations(t)
}

// TestSchedulerTickUseCase_Execute_SingleSchedule tests processing a single due schedule
func TestSchedulerTickUseCase_Execute_SingleSchedule(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)
	mockCronSchedule := new(MockCronSchedule)

	cfg := config.JobConfig{
		SchedulerBatch: 10,
		MaxAttempts:    5,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	nextRun := now.Add(1 * time.Hour)

	payload := json.RawMessage(`{"key":"value"}`)
	schedule := &domain.Schedule{
		ID:       1,
		Name:     "test-schedule",
		CronSpec: "0 * * * *",
		JobType:  "test-job",
		Payload:  payload,
		Priority: 5,
		Enabled:  true,
	}

	mockRepo.On("ListDue", ctx, 10).Return([]*domain.Schedule{schedule}, nil)
	mockClock.On("Now").Return(now)
	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil)
	mockCronSchedule.On("Next", now).Return(nextRun)

	// Verify the job that will be created
	mockRepo.On("AdvanceAndEnqueue", ctx, int64(1), nextRun, mock.MatchedBy(func(job *domain.Job) bool {
		return job.Type == "test-job" &&
			string(job.Payload) == `{"key":"value"}` &&
			job.Priority == 5 &&
			job.RunAt.Equal(now) &&
			job.Attempts == 0 &&
			job.MaxAttempts == 5 &&
			job.ScheduledID.Valid &&
			job.ScheduledID.Int64 == 1
	})).Return(nil)

	uc := NewSchedulerTickUseCase(cfg, mockRepo, mockParser, mockClock)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockParser.AssertExpectations(t)
	mockClock.AssertExpectations(t)
	mockCronSchedule.AssertExpectations(t)
}

// TestSchedulerTickUseCase_Execute_MultipleSchedules tests processing multiple due schedules
func TestSchedulerTickUseCase_Execute_MultipleSchedules(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		SchedulerBatch: 10,
		MaxAttempts:    3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	schedules := []*domain.Schedule{
		{
			ID:       1,
			Name:     "schedule-1",
			CronSpec: "0 * * * *",
			JobType:  "job-1",
			Priority: 1,
			Enabled:  true,
		},
		{
			ID:       2,
			Name:     "schedule-2",
			CronSpec: "*/5 * * * *",
			JobType:  "job-2",
			Priority: 10,
			Enabled:  true,
		},
	}

	mockRepo.On("ListDue", ctx, 10).Return(schedules, nil)
	mockClock.On("Now").Return(now)

	// Setup for schedule 1
	mockCronSchedule1 := new(MockCronSchedule)
	nextRun1 := now.Add(1 * time.Hour)
	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule1, nil)
	mockCronSchedule1.On("Next", now).Return(nextRun1)
	mockRepo.On("AdvanceAndEnqueue", ctx, int64(1), nextRun1, mock.Anything).Return(nil)

	// Setup for schedule 2
	mockCronSchedule2 := new(MockCronSchedule)
	nextRun2 := now.Add(5 * time.Minute)
	mockParser.On("Parse", "*/5 * * * *").Return(mockCronSchedule2, nil)
	mockCronSchedule2.On("Next", now).Return(nextRun2)
	mockRepo.On("AdvanceAndEnqueue", ctx, int64(2), nextRun2, mock.Anything).Return(nil)

	uc := NewSchedulerTickUseCase(cfg, mockRepo, mockParser, mockClock)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockParser.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestSchedulerTickUseCase_Execute_InvalidCronSpec tests handling of invalid cron spec
func TestSchedulerTickUseCase_Execute_InvalidCronSpec(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		SchedulerBatch: 10,
		MaxAttempts:    3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	schedules := []*domain.Schedule{
		{
			ID:       1,
			Name:     "bad-schedule",
			CronSpec: "invalid cron",
			JobType:  "job-1",
			Enabled:  true,
		},
		{
			ID:       2,
			Name:     "good-schedule",
			CronSpec: "0 * * * *",
			JobType:  "job-2",
			Enabled:  true,
		},
	}

	mockRepo.On("ListDue", ctx, 10).Return(schedules, nil)
	mockClock.On("Now").Return(now)

	// First schedule has invalid cron - should skip it
	mockParser.On("Parse", "invalid cron").Return(nil, errors.New("invalid cron spec"))

	// Second schedule should be processed normally
	mockCronSchedule := new(MockCronSchedule)
	nextRun := now.Add(1 * time.Hour)
	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil)
	mockCronSchedule.On("Next", now).Return(nextRun)
	mockRepo.On("AdvanceAndEnqueue", ctx, int64(2), nextRun, mock.Anything).Return(nil)

	uc := NewSchedulerTickUseCase(cfg, mockRepo, mockParser, mockClock)

	err := uc.Execute(ctx)

	// Should not return error - invalid cron is logged and skipped
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockParser.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestSchedulerTickUseCase_Execute_AdvanceAndEnqueueError tests error during advance/enqueue
func TestSchedulerTickUseCase_Execute_AdvanceAndEnqueueError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)
	mockCronSchedule := new(MockCronSchedule)

	cfg := config.JobConfig{
		SchedulerBatch: 10,
		MaxAttempts:    3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	nextRun := now.Add(1 * time.Hour)

	schedule := &domain.Schedule{
		ID:       1,
		Name:     "test-schedule",
		CronSpec: "0 * * * *",
		JobType:  "test-job",
		Enabled:  true,
	}

	mockRepo.On("ListDue", ctx, 10).Return([]*domain.Schedule{schedule}, nil)
	mockClock.On("Now").Return(now)
	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil)
	mockCronSchedule.On("Next", now).Return(nextRun)

	expectedErr := errors.New("transaction error")
	mockRepo.On("AdvanceAndEnqueue", ctx, int64(1), nextRun, mock.Anything).Return(expectedErr)

	uc := NewSchedulerTickUseCase(cfg, mockRepo, mockParser, mockClock)

	err := uc.Execute(ctx)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockRepo.AssertExpectations(t)
}

// TestSchedulerTickUseCase_Execute_BatchLimit tests that batch limit is respected
func TestSchedulerTickUseCase_Execute_BatchLimit(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)

	cfg := config.JobConfig{
		SchedulerBatch: 2, // Limit to 2 schedules
		MaxAttempts:    3,
	}

	// Should only request 2 schedules, not more
	mockRepo.On("ListDue", ctx, 2).Return([]*domain.Schedule{}, nil)
	mockClock.On("Now").Return(time.Now())

	uc := NewSchedulerTickUseCase(cfg, mockRepo, mockParser, mockClock)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

// TestSchedulerTickUseCase_Execute_ScheduledIDTracking tests that scheduled_id is properly set
func TestSchedulerTickUseCase_Execute_ScheduledIDTracking(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)
	mockCronSchedule := new(MockCronSchedule)

	cfg := config.JobConfig{
		SchedulerBatch: 10,
		MaxAttempts:    3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	nextRun := now.Add(1 * time.Hour)

	schedule := &domain.Schedule{
		ID:       42,
		Name:     "tracked-schedule",
		CronSpec: "0 * * * *",
		JobType:  "test-job",
		Enabled:  true,
	}

	mockRepo.On("ListDue", ctx, 10).Return([]*domain.Schedule{schedule}, nil)
	mockClock.On("Now").Return(now)
	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil)
	mockCronSchedule.On("Next", now).Return(nextRun)

	// Verify scheduled_id is set to 42
	mockRepo.On("AdvanceAndEnqueue", ctx, int64(42), nextRun, mock.MatchedBy(func(job *domain.Job) bool {
		return job.ScheduledID.Valid && job.ScheduledID.Int64 == 42
	})).Return(nil)

	uc := NewSchedulerTickUseCase(cfg, mockRepo, mockParser, mockClock)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
