package job

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
)

// TestUpsertScheduleUseCase_Execute_Success tests successful schedule creation
func TestUpsertScheduleUseCase_Execute_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)
	mockCronSchedule := new(MockCronSchedule)

	cfg := config.JobConfig{
		MaxAttempts: 5,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	nextRun := now.Add(1 * time.Hour)

	input := UpsertScheduleInput{
		Name:     "test-schedule",
		CronSpec: "0 * * * *",
		JobType:  "test-job",
		Payload:  map[string]string{"key": "value"},
		Priority: 5,
		Enabled:  true,
	}

	// Validate cron
	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil).Once()

	// Upsert schedule
	mockRepo.On("Upsert", ctx, mock.MatchedBy(func(s *domain.Schedule) bool {
		return s.Name == "test-schedule" &&
			s.CronSpec == "0 * * * *" &&
			s.JobType == "test-job" &&
			s.Priority == 5 &&
			s.MaxAttempts == 5 &&
			s.Enabled == true &&
			string(s.Payload) == `{"key":"value"}`
	})).Return(int64(1), nil)

	// GetByName - return schedule with NextRunAt not set
	mockRepo.On("GetByName", ctx, "test-schedule").Return(&domain.Schedule{
		ID:          1,
		Name:        "test-schedule",
		CronSpec:    "0 * * * *",
		JobType:     "test-job",
		Priority:    5,
		MaxAttempts: 5,
		Enabled:     true,
		NextRunAt:   sql.NullTime{Valid: false}, // Not set yet
	}, nil)

	// Calculate next run time
	mockClock.On("Now").Return(now)
	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil).Once()
	mockCronSchedule.On("Next", now).Return(nextRun)

	// Set next run time
	mockRepo.On("SetNextRunIfNull", ctx, int64(1), nextRun).Return(nil)

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	id, err := uc.Execute(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
	mockRepo.AssertExpectations(t)
	mockParser.AssertExpectations(t)
	mockClock.AssertExpectations(t)
	mockCronSchedule.AssertExpectations(t)
}

// TestUpsertScheduleUseCase_Execute_EmptyName tests validation for empty name
func TestUpsertScheduleUseCase_Execute_EmptyName(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)

	cfg := config.JobConfig{}

	input := UpsertScheduleInput{
		Name:     "",
		CronSpec: "0 * * * *",
		JobType:  "test-job",
	}

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	id, err := uc.Execute(ctx, input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "schedule name required")
	assert.Equal(t, int64(0), id)
}

// TestUpsertScheduleUseCase_Execute_EmptyJobType tests validation for empty job type
func TestUpsertScheduleUseCase_Execute_EmptyJobType(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)

	cfg := config.JobConfig{}

	input := UpsertScheduleInput{
		Name:     "test-schedule",
		CronSpec: "0 * * * *",
		JobType:  "",
	}

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	id, err := uc.Execute(ctx, input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "job type required")
	assert.Equal(t, int64(0), id)
}

// TestUpsertScheduleUseCase_Execute_EmptyCronSpec tests validation for empty cron spec
func TestUpsertScheduleUseCase_Execute_EmptyCronSpec(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)

	cfg := config.JobConfig{}

	input := UpsertScheduleInput{
		Name:     "test-schedule",
		CronSpec: "",
		JobType:  "test-job",
	}

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	id, err := uc.Execute(ctx, input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cron specification required")
	assert.Equal(t, int64(0), id)
}

// TestUpsertScheduleUseCase_Execute_InvalidCronSpec tests validation for invalid cron spec
func TestUpsertScheduleUseCase_Execute_InvalidCronSpec(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)

	cfg := config.JobConfig{}

	input := UpsertScheduleInput{
		Name:     "test-schedule",
		CronSpec: "invalid cron",
		JobType:  "test-job",
	}

	mockParser.On("Parse", "invalid cron").Return(nil, errors.New("invalid cron syntax"))

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	id, err := uc.Execute(ctx, input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron specification")
	assert.Equal(t, int64(0), id)
	mockParser.AssertExpectations(t)
}

// TestUpsertScheduleUseCase_Execute_InvalidPayload tests error when payload cannot be marshaled
func TestUpsertScheduleUseCase_Execute_InvalidPayload(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)
	mockCronSchedule := new(MockCronSchedule)

	cfg := config.JobConfig{}

	// Create an invalid payload (channels cannot be marshaled to JSON)
	ch := make(chan int)
	input := UpsertScheduleInput{
		Name:     "test-schedule",
		CronSpec: "0 * * * *",
		JobType:  "test-job",
		Payload:  ch,
	}

	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil)

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	id, err := uc.Execute(ctx, input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal payload")
	assert.Equal(t, int64(0), id)
	mockParser.AssertExpectations(t)
}

// TestUpsertScheduleUseCase_Execute_NilPayload tests schedule with nil payload
func TestUpsertScheduleUseCase_Execute_NilPayload(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)
	mockCronSchedule := new(MockCronSchedule)

	cfg := config.JobConfig{
		MaxAttempts: 3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	input := UpsertScheduleInput{
		Name:     "test-schedule",
		CronSpec: "0 * * * *",
		JobType:  "test-job",
		Payload:  nil,
		Enabled:  true,
	}

	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil).Once()
	mockRepo.On("Upsert", ctx, mock.MatchedBy(func(s *domain.Schedule) bool {
		return s.Payload == nil
	})).Return(int64(1), nil)

	// Return schedule with NextRunAt already set
	mockRepo.On("GetByName", ctx, "test-schedule").Return(&domain.Schedule{
		ID:        1,
		Name:      "test-schedule",
		CronSpec:  "0 * * * *",
		JobType:   "test-job",
		Enabled:   true,
		NextRunAt: sql.NullTime{Valid: true, Time: now.Add(1 * time.Hour)}, // Already set
	}, nil)

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	id, err := uc.Execute(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
	mockRepo.AssertExpectations(t)
	mockParser.AssertExpectations(t)
	// Clock should not be called if NextRunAt is already set
}

// TestUpsertScheduleUseCase_Execute_UpsertError tests error during upsert
func TestUpsertScheduleUseCase_Execute_UpsertError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)
	mockCronSchedule := new(MockCronSchedule)

	cfg := config.JobConfig{}

	input := UpsertScheduleInput{
		Name:     "test-schedule",
		CronSpec: "0 * * * *",
		JobType:  "test-job",
		Enabled:  true,
	}

	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil)
	expectedErr := errors.New("database error")
	mockRepo.On("Upsert", ctx, mock.Anything).Return(int64(0), expectedErr)

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	id, err := uc.Execute(ctx, input)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, int64(0), id)
	mockRepo.AssertExpectations(t)
}

// TestUpsertScheduleUseCase_Execute_GetByNameError tests error when fetching schedule
func TestUpsertScheduleUseCase_Execute_GetByNameError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)
	mockCronSchedule := new(MockCronSchedule)

	cfg := config.JobConfig{}

	input := UpsertScheduleInput{
		Name:     "test-schedule",
		CronSpec: "0 * * * *",
		JobType:  "test-job",
		Enabled:  true,
	}

	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil)
	mockRepo.On("Upsert", ctx, mock.Anything).Return(int64(1), nil)

	expectedErr := errors.New("database error")
	mockRepo.On("GetByName", ctx, "test-schedule").Return(nil, expectedErr)

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	id, err := uc.Execute(ctx, input)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, int64(0), id)
	mockRepo.AssertExpectations(t)
}

// TestUpsertScheduleUseCase_Execute_SetNextRunError tests error when setting next run time
func TestUpsertScheduleUseCase_Execute_SetNextRunError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)
	mockCronSchedule := new(MockCronSchedule)

	cfg := config.JobConfig{}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	nextRun := now.Add(1 * time.Hour)

	input := UpsertScheduleInput{
		Name:     "test-schedule",
		CronSpec: "0 * * * *",
		JobType:  "test-job",
		Enabled:  true,
	}

	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil).Twice()
	mockRepo.On("Upsert", ctx, mock.Anything).Return(int64(1), nil)
	mockRepo.On("GetByName", ctx, "test-schedule").Return(&domain.Schedule{
		ID:        1,
		Name:      "test-schedule",
		CronSpec:  "0 * * * *",
		JobType:   "test-job",
		Enabled:   true,
		NextRunAt: sql.NullTime{Valid: false},
	}, nil)

	mockClock.On("Now").Return(now)
	mockCronSchedule.On("Next", now).Return(nextRun)

	expectedErr := errors.New("database error")
	mockRepo.On("SetNextRunIfNull", ctx, int64(1), nextRun).Return(expectedErr)

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	id, err := uc.Execute(ctx, input)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, int64(0), id)
	mockRepo.AssertExpectations(t)
}

// TestUpsertScheduleUseCase_Execute_UpdateExistingSchedule tests updating an existing schedule
func TestUpsertScheduleUseCase_Execute_UpdateExistingSchedule(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)
	mockCronSchedule := new(MockCronSchedule)

	cfg := config.JobConfig{
		MaxAttempts: 3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	existingNextRun := now.Add(2 * time.Hour)

	input := UpsertScheduleInput{
		Name:     "existing-schedule",
		CronSpec: "*/5 * * * *", // Changed cron spec
		JobType:  "test-job",
		Priority: 10,    // Changed priority
		Enabled:  false, // Disabled
	}

	mockParser.On("Parse", "*/5 * * * *").Return(mockCronSchedule, nil).Once()
	mockRepo.On("Upsert", ctx, mock.Anything).Return(int64(42), nil)

	// Return schedule with NextRunAt already set (from previous version)
	mockRepo.On("GetByName", ctx, "existing-schedule").Return(&domain.Schedule{
		ID:        42,
		Name:      "existing-schedule",
		CronSpec:  "*/5 * * * *",
		JobType:   "test-job",
		Priority:  10,
		Enabled:   false,
		NextRunAt: sql.NullTime{Valid: true, Time: existingNextRun}, // Already has next run
	}, nil)

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	id, err := uc.Execute(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, int64(42), id)
	mockRepo.AssertExpectations(t)
	mockParser.AssertExpectations(t)
	// Clock and SetNextRunIfNull should not be called since NextRunAt is valid
}

// TestUpsertScheduleUseCase_Execute_PayloadMarshaling tests that payload is correctly marshaled
func TestUpsertScheduleUseCase_Execute_PayloadMarshaling(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockScheduleRepository)
	mockParser := new(MockCronParser)
	mockClock := new(MockClock)
	mockCronSchedule := new(MockCronSchedule)

	cfg := config.JobConfig{
		MaxAttempts: 3,
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	complexPayload := map[string]interface{}{
		"string": "value",
		"number": 42,
		"array":  []int{1, 2, 3},
		"nested": map[string]string{"key": "value"},
	}

	input := UpsertScheduleInput{
		Name:     "test-schedule",
		CronSpec: "0 * * * *",
		JobType:  "test-job",
		Payload:  complexPayload,
		Enabled:  true,
	}

	mockParser.On("Parse", "0 * * * *").Return(mockCronSchedule, nil).Once()

	var capturedPayload json.RawMessage
	mockRepo.On("Upsert", ctx, mock.MatchedBy(func(s *domain.Schedule) bool {
		capturedPayload = s.Payload
		return true
	})).Return(int64(1), nil)

	mockRepo.On("GetByName", ctx, "test-schedule").Return(&domain.Schedule{
		ID:        1,
		Name:      "test-schedule",
		CronSpec:  "0 * * * *",
		JobType:   "test-job",
		Enabled:   true,
		NextRunAt: sql.NullTime{Valid: true, Time: now},
	}, nil)

	uc := NewUpsertScheduleUseCase(cfg, mockRepo, mockParser, mockClock)

	_, err := uc.Execute(ctx, input)

	require.NoError(t, err)

	// Verify payload was marshaled correctly
	var unmarshaled map[string]interface{}
	err = json.Unmarshal(capturedPayload, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, "value", unmarshaled["string"])
	assert.Equal(t, float64(42), unmarshaled["number"]) // JSON numbers unmarshal as float64
	mockRepo.AssertExpectations(t)
}
