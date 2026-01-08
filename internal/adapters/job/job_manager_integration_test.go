package job

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	usecasejob "github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ===============================================
// Group 1: Basic Job Lifecycle Tests
// ===============================================

func TestJobManager_StartStop(t *testing.T) {
	cfg := testJobConfig()
	manager, _ := setupTestJobManager(t, cfg, nil)

	// Start manager
	err := manager.Start()
	require.NoError(t, err, "manager should start successfully")

	// Try to start again - should fail
	err = manager.Start()
	require.Error(t, err, "starting already running manager should fail")

	// Stop manager
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = manager.Stop(ctx)
	require.NoError(t, err, "manager should stop successfully")
}

func TestJobManager_SingleJobLifecycle(t *testing.T) {
	cfg := testJobConfig()
	handlers := map[string]ports.JobHandler{
		"test-job": newSuccessHandler(),
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Enqueue a job
	ctx := context.Background()
	_, err = deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
		Type:     "test-job",
		Priority: 0,
	})
	require.NoError(t, err)

	// Wait for job to complete
	waitForJobCount(t, deps.DB, string(shared.StatusSucceeded), 1, 5*time.Second)

	// Assert job reached succeeded status
	assertJobCount(t, deps.DB, string(shared.StatusSucceeded), 1)
	assertJobCount(t, deps.DB, string(shared.StatusQueued), 0)
	assertJobCount(t, deps.DB, string(shared.StatusRunning), 0)
}

func TestJobManager_MultipleWorkersConcurrency(t *testing.T) {
	cfg := testJobConfig()

	counter := &atomic.Int32{}
	handlers := map[string]ports.JobHandler{
		"counter-job": newCounterHandler(counter),
	}
	manager, deps := setupTestJobManagerWithWorkerCount(t, cfg, handlers, 3)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Enqueue 15 jobs
	ctx := context.Background()
	jobCount := 15
	for i := 0; i < jobCount; i++ {
		_, err = deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
			Type:     "counter-job",
			Priority: 0,
		})
		require.NoError(t, err)
	}

	// Wait for all jobs to complete
	waitForJobCount(t, deps.DB, string(shared.StatusSucceeded), jobCount, 10*time.Second)

	// Assert all jobs succeeded
	assertJobCount(t, deps.DB, string(shared.StatusSucceeded), jobCount)
	assert.Equal(t, int32(jobCount), counter.Load(), "all jobs should have been processed")
}

func TestJobManager_JobPriorityOrdering(t *testing.T) {
	cfg := testJobConfig()

	var executionOrder []int
	var mu sync.Mutex
	handler := func(ctx context.Context, job *domain.Job) error {
		mu.Lock()
		defer mu.Unlock()
		// Extract priority from payload or use job priority
		executionOrder = append(executionOrder, job.Priority)
		time.Sleep(50 * time.Millisecond) // Small delay to ensure ordering
		return nil
	}

	handlers := map[string]ports.JobHandler{
		"priority-job": handler,
	}
	manager, deps := setupTestJobManagerWithWorkerCount(t, cfg, handlers, 1) // Single worker to ensure ordering

	// Enqueue jobs with different priorities (enqueue low priority first)
	ctx := context.Background()
	priorities := []int{1, 5, 3, 10, 2}
	for _, priority := range priorities {
		_, err := deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
			Type:     "priority-job",
			Priority: priority,
		})
		require.NoError(t, err)
	}

	// Start manager AFTER enqueuing (to ensure jobs are queued)
	err := manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Wait for all jobs to complete
	waitForJobCount(t, deps.DB, string(shared.StatusSucceeded), len(priorities), 10*time.Second)

	// Assert jobs were executed in priority order (highest first)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 10, executionOrder[0], "highest priority job should execute first")
	assert.Equal(t, 5, executionOrder[1], "second highest priority job should execute second")
}

// ===============================================
// Group 2: Error Handling Tests
// ===============================================

func TestJobManager_JobRetryWithBackoff(t *testing.T) {
	cfg := testJobConfig()
	cfg.MaxAttempts = 3

	// Handler that fails twice then succeeds
	handlers := map[string]ports.JobHandler{
		"flaky-job": newFailNTimesHandler(2),
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Enqueue job
	ctx := context.Background()
	_, err = deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
		Type: "flaky-job",
	})
	require.NoError(t, err)

	// Wait for job to eventually succeed (with retries)
	waitForJobCount(t, deps.DB, string(shared.StatusSucceeded), 1, 15*time.Second)

	// Get the job and verify it was retried
	var job domain.Job
	query := "SELECT attempt FROM jobs WHERE type = 'flaky-job'"
	err = deps.DB.QueryRow(query).Scan(&job.Attempts)
	require.NoError(t, err)
	assert.Equal(t, 3, job.Attempts, "job should have been attempted 3 times")
}

func TestJobManager_MaxAttemptsExceeded(t *testing.T) {
	cfg := testJobConfig()
	cfg.MaxAttempts = 3

	// Handler that always fails
	handlers := map[string]ports.JobHandler{
		"failing-job": newErrorHandler("permanent failure"),
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Enqueue job
	ctx := context.Background()
	_, err = deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
		Type: "failing-job",
	})
	require.NoError(t, err)

	// Wait for job to become dead (after max attempts)
	waitForJobCount(t, deps.DB, string(shared.StatusDead), 1, 15*time.Second)

	// Verify job is dead and has correct attempt count
	var job domain.Job
	query := "SELECT attempt, last_error FROM jobs WHERE type = 'failing-job'"
	err = deps.DB.QueryRow(query).Scan(&job.Attempts, &job.LastError)
	require.NoError(t, err)
	assert.Equal(t, 3, job.Attempts, "job should have max attempts")
	assert.True(t, job.LastError.Valid, "last_error should be set")
	assert.Contains(t, job.LastError.String, "permanent failure")
}

func TestJobManager_UnregisteredHandler(t *testing.T) {
	cfg := testJobConfig()
	// No handlers registered
	manager, deps := setupTestJobManager(t, cfg, nil)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Enqueue job with unregistered type
	ctx := context.Background()
	_, err = deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
		Type: "unknown-job",
	})
	require.NoError(t, err)

	// Wait for job to become dead immediately
	waitForJobCount(t, deps.DB, string(shared.StatusDead), 1, 5*time.Second)

	// Verify last_error mentions missing handler
	var lastError sql.NullString
	query := "SELECT last_error FROM jobs WHERE type = 'unknown-job'"
	err = deps.DB.QueryRow(query).Scan(&lastError)
	require.NoError(t, err)
	assert.True(t, lastError.Valid)
	assert.Contains(t, lastError.String, "no handler registered")
}

func TestJobManager_HandlerPanic(t *testing.T) {
	cfg := testJobConfig()
	handlers := map[string]ports.JobHandler{
		"panic-job": newPanicHandler(),
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Enqueue job that will panic
	ctx := context.Background()
	_, err = deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
		Type: "panic-job",
	})
	require.NoError(t, err)

	// Wait for job to become dead (panic should be recovered)
	waitForJobCount(t, deps.DB, string(shared.StatusDead), 1, 5*time.Second)

	// Verify last_error mentions panic
	var lastError sql.NullString
	query := "SELECT last_error FROM jobs WHERE type = 'panic-job'"
	err = deps.DB.QueryRow(query).Scan(&lastError)
	require.NoError(t, err)
	assert.True(t, lastError.Valid)
	assert.Contains(t, lastError.String, "panic")
}

// ===============================================
// Group 3: Scheduling & Timing Tests
// ===============================================

func TestJobManager_FutureJobScheduling(t *testing.T) {
	cfg := testJobConfig()
	handlers := map[string]ports.JobHandler{
		"future-job": newSuccessHandler(),
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Enqueue job scheduled for 3 seconds in the future
	ctx := context.Background()
	futureTime := time.Now().Add(3 * time.Second)
	_, err = deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
		Type:  "future-job",
		RunAt: futureTime,
	})
	require.NoError(t, err)

	// Job should NOT be processed immediately
	time.Sleep(1 * time.Second)
	assertJobCount(t, deps.DB, string(shared.StatusQueued), 1)
	assertJobCount(t, deps.DB, string(shared.StatusSucceeded), 0)

	// Wait for job to become due and be processed
	waitForJobCount(t, deps.DB, string(shared.StatusSucceeded), 1, 10*time.Second)
}

func TestJobManager_ScheduleUpsertAndExecution(t *testing.T) {
	cfg := testJobConfig()
	counter := &atomic.Int32{}
	handlers := map[string]ports.JobHandler{
		"scheduled-job": newCounterHandler(counter),
	}

	mockClock := NewMockClock(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))
	manager, deps := setupTestJobManagerWithMockClock(t, cfg, handlers, mockClock)

	// Create schedule using UpsertScheduleUseCase
	ctx := context.Background()
	scheduleID, err := deps.UpsertScheduleUseCase.Execute(ctx, usecasejob.UpsertScheduleInput{
		Name:     "test-schedule",
		CronSpec: "0 * * * * *", // Every minute (at :00 seconds)
		JobType:  "scheduled-job",
		Priority: 5,
		Enabled:  true,
	})
	require.NoError(t, err)
	require.NotZero(t, scheduleID)

	// Verify schedule was created
	schedule := getScheduleByName(t, deps.DB, "test-schedule")
	assert.Equal(t, "test-schedule", schedule.Name)
	assert.Equal(t, "0 * * * * *", schedule.CronSpec)
	assert.True(t, schedule.Enabled)
	assert.True(t, schedule.NextRunAt.Valid, "next_run_at should be set")

	// Start manager
	err = manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Advance clock to make schedule due
	mockClock.Advance(2 * time.Minute)

	// Trigger scheduler tick manually
	err = deps.SchedulerTickUseCase.Execute(ctx)
	require.NoError(t, err)

	// Wait for job to be enqueued and processed
	waitForJobCount(t, deps.DB, string(shared.StatusSucceeded), 1, 5*time.Second)

	// Verify job was created from schedule
	jobs := getJobsByScheduleID(t, deps.DB, scheduleID)
	require.Len(t, jobs, 1)
	assert.Equal(t, "scheduled-job", jobs[0].Type)
	assert.Equal(t, 5, jobs[0].Priority)

	// Verify schedule was advanced
	schedule = getScheduleByName(t, deps.DB, "test-schedule")
	assert.True(t, schedule.LastEnqueuedAt.Valid)
}

func TestJobManager_ScheduleMultipleCronExpressions(t *testing.T) {
	// This test uses second-based cron expressions for fast execution with real time.
	// Format: "second minute hour day month weekday"
	// Example: "*/2 * * * * *" = every 2 seconds
	cfg := testJobConfig()
	handlers := map[string]ports.JobHandler{
		"job-a": newSuccessHandler(),
		"job-b": newSuccessHandler(),
		"job-c": newSuccessHandler(),
	}

	manager, deps := setupTestJobManagerWithSecondCron(t, cfg, handlers)

	ctx := context.Background()

	// Create multiple schedules with second-based cron
	_, err := deps.UpsertScheduleUseCase.Execute(ctx, usecasejob.UpsertScheduleInput{
		Name:     "every-2-seconds",
		CronSpec: "*/2 * * * * *", // Every 2 seconds
		JobType:  "job-a",
		Enabled:  true,
	})
	require.NoError(t, err)

	_, err = deps.UpsertScheduleUseCase.Execute(ctx, usecasejob.UpsertScheduleInput{
		Name:     "every-5-seconds",
		CronSpec: "*/5 * * * * *", // Every 5 seconds
		JobType:  "job-b",
		Enabled:  true,
	})
	require.NoError(t, err)

	_, err = deps.UpsertScheduleUseCase.Execute(ctx, usecasejob.UpsertScheduleInput{
		Name:     "every-10-seconds",
		CronSpec: "*/10 * * * * *", // Every 10 seconds
		JobType:  "job-c",
		Enabled:  true,
	})
	require.NoError(t, err)

	// Start manager
	err = manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Wait for first schedule alignment (up to 2 seconds for every-2-seconds to align)
	// Then wait a bit more for the schedule to become due
	time.Sleep(3 * time.Second)
	err = deps.SchedulerTickUseCase.Execute(ctx)
	require.NoError(t, err)

	// Wait for jobs to be processed
	time.Sleep(2 * time.Second)

	// Wait for the 5-second and 10-second schedules to also trigger
	// We need to wait until at least one cycle of each schedule completes
	time.Sleep(10 * time.Second) // Total ~15 seconds from start
	err = deps.SchedulerTickUseCase.Execute(ctx)
	require.NoError(t, err)

	// After ~15 seconds total, we should have multiple jobs from different schedules.
	// The exact count depends on cron alignment timing, but we should have:
	// - At least one job from each schedule (job-a, job-b, job-c)
	// - Multiple jobs total (at least 3, likely more)
	// We'll wait for at least 3 jobs and verify all types are present
	waitForJobCount(t, deps.DB, string(shared.StatusSucceeded), 3, 10*time.Second)

	// Verify all three job types were executed
	var jobTypes []string
	rows, err := deps.DB.Query("SELECT type FROM jobs WHERE status = 'succeeded' ORDER BY created_at")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var jobType string
		rows.Scan(&jobType)
		jobTypes = append(jobTypes, jobType)
	}

	// Must have all three job types
	require.Contains(t, jobTypes, "job-a", "every-2-seconds schedule should have triggered")
	require.Contains(t, jobTypes, "job-b", "every-5-seconds schedule should have triggered")
	require.Contains(t, jobTypes, "job-c", "every-10-seconds schedule should have triggered")

	// Should have at least 3 jobs total (one from each schedule minimum)
	require.GreaterOrEqual(t, len(jobTypes), 3, "should have at least one job from each schedule")
}

func TestJobManager_ScheduleDisabledIgnored(t *testing.T) {
	cfg := testJobConfig()
	handlers := map[string]ports.JobHandler{
		"disabled-job": newSuccessHandler(),
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	ctx := context.Background()

	// Create disabled schedule
	scheduleID, err := deps.UpsertScheduleUseCase.Execute(ctx, usecasejob.UpsertScheduleInput{
		Name:     "disabled-schedule",
		CronSpec: "0 * * * * *",
		JobType:  "disabled-job",
		Enabled:  false, // Disabled
	})
	require.NoError(t, err)

	// Start manager
	err = manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Wait a bit and trigger scheduler
	time.Sleep(2 * time.Second)
	err = deps.SchedulerTickUseCase.Execute(ctx)
	require.NoError(t, err)

	// No jobs should have been created
	time.Sleep(1 * time.Second)
	jobs := getJobsByScheduleID(t, deps.DB, scheduleID)
	assert.Len(t, jobs, 0, "disabled schedule should not create jobs")
}

func TestJobManager_ScheduleNextRunAtAdvances(t *testing.T) {
	// This test verifies that after a schedule is executed, its next_run_at is properly advanced
	// and it doesn't double-enqueue. Uses second-based cron for fast execution with real time.
	cfg := testJobConfig()
	handlers := map[string]ports.JobHandler{
		"advance-job": newSuccessHandler(),
	}

	manager, deps := setupTestJobManagerWithSecondCron(t, cfg, handlers)

	ctx := context.Background()

	// Create schedule - every 10 seconds
	scheduleID, err := deps.UpsertScheduleUseCase.Execute(ctx, usecasejob.UpsertScheduleInput{
		Name:     "advance-schedule",
		CronSpec: "*/10 * * * * *", // Every 10 seconds
		JobType:  "advance-job",
		Enabled:  true,
	})
	require.NoError(t, err)

	// Get initial next_run_at
	schedule := getScheduleByName(t, deps.DB, "advance-schedule")
	initialNextRun := schedule.NextRunAt.Time
	t.Logf("Initial next_run_at: %v", initialNextRun)

	// Start manager
	err = manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Wait for schedule to become due (10+ seconds)
	time.Sleep(11 * time.Second)
	err = deps.SchedulerTickUseCase.Execute(ctx)
	require.NoError(t, err)

	// Wait for job to complete
	waitForJobCount(t, deps.DB, string(shared.StatusSucceeded), 1, 5*time.Second)

	// Verify next_run_at was advanced
	schedule = getScheduleByName(t, deps.DB, "advance-schedule")
	t.Logf("After first run - next_run_at: %v", schedule.NextRunAt.Time)
	assert.True(t, schedule.NextRunAt.Time.After(initialNextRun), "next_run_at should be advanced")
	assert.True(t, schedule.LastEnqueuedAt.Valid, "last_enqueued_at should be set")

	// Trigger scheduler again immediately - should NOT enqueue another job
	// because next_run_at has been advanced to the future (10 seconds from last run)
	err = deps.SchedulerTickUseCase.Execute(ctx)
	require.NoError(t, err)

	// Wait a bit to ensure no new jobs are enqueued
	time.Sleep(2 * time.Second)

	// Check how many jobs we have now
	jobs := getJobsByScheduleID(t, deps.DB, scheduleID)
	t.Logf("Jobs found after second tick: %d", len(jobs))
	for i, job := range jobs {
		t.Logf("Job %d: ID=%d, Status=%s, CreatedAt=%v", i+1, job.ID, job.Status, job.CreatedAt)
	}

	// Should still only have 1 job (no double-enqueueing)
	assert.Len(t, jobs, 1, "should not double-enqueue - next_run_at should prevent immediate re-triggering")
}

// ===============================================
// Group 4: Persistence & Restart Tests
// ===============================================

func TestJobManager_RestartWithQueuedJobs(t *testing.T) {
	cfg := testJobConfig()
	handlers := map[string]ports.JobHandler{
		"restart-job": newSuccessHandler(),
	}

	// First manager instance - setup and enqueue jobs WITHOUT starting
	_, deps := setupTestJobManager(t, cfg, handlers)

	// Enqueue jobs directly to DB (simulating jobs from before restart)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
			Type: "restart-job",
		})
		require.NoError(t, err)
	}

	// Verify jobs are queued
	assertJobCount(t, deps.DB, string(shared.StatusQueued), 5)

	// Now create a NEW manager instance (simulating restart) with same DB
	manager2 := NewJobManager(JobManagerInput{
		Config:                cfg,
		ProcessNextJobUseCase: deps.ProcessNextJobUseCase,
		SchedulerTickUseCase:  deps.SchedulerTickUseCase,
		JobRepository:         deps.JobRepository,
		ScheduleRepository:    deps.ScheduleRepository,
		Handlers:              deps.HandlerRegistry,
		BackoffStrategy:       deps.BackoffStrategy,
		Cron:                  deps.CronParser,
		Clock:                 deps.Clock,
	})

	// Start the new manager
	err := manager2.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager2.Stop(ctx)
	}()

	// Jobs should be processed after restart
	waitForJobCount(t, deps.DB, string(shared.StatusSucceeded), 5, 10*time.Second)
	assertJobCount(t, deps.DB, string(shared.StatusQueued), 0)
}

func TestJobManager_RestartWithDeadJobs(t *testing.T) {
	cfg := testJobConfig()
	cfg.MaxAttempts = 1 // Fail quickly
	handlers := map[string]ports.JobHandler{
		"dead-job": newErrorHandler("fatal error"),
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)

	// Enqueue job that will become dead
	ctx := context.Background()
	_, err = deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
		Type: "dead-job",
	})
	require.NoError(t, err)

	// Wait for job to become dead
	waitForJobCount(t, deps.DB, string(shared.StatusDead), 1, 5*time.Second)

	// Stop manager
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = manager.Stop(stopCtx)
	require.NoError(t, err)

	// Create new manager (restart)
	manager2 := NewJobManager(JobManagerInput{
		Config:                cfg,
		ProcessNextJobUseCase: deps.ProcessNextJobUseCase,
		SchedulerTickUseCase:  deps.SchedulerTickUseCase,
		JobRepository:         deps.JobRepository,
		ScheduleRepository:    deps.ScheduleRepository,
		Handlers:              deps.HandlerRegistry,
		BackoffStrategy:       deps.BackoffStrategy,
		Cron:                  deps.CronParser,
		Clock:                 deps.Clock,
	})

	err = manager2.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager2.Stop(ctx)
	}()

	// Wait a bit to ensure manager has time to process
	time.Sleep(2 * time.Second)

	// Dead job should remain dead (not retried)
	assertJobCount(t, deps.DB, string(shared.StatusDead), 1)
	assertJobCount(t, deps.DB, string(shared.StatusQueued), 0)
}

func TestJobManager_RestartWithDueScheduledJobs(t *testing.T) {
	cfg := testJobConfig()
	handlers := map[string]ports.JobHandler{
		"missed-job": newSuccessHandler(),
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	ctx := context.Background()

	// Create schedule
	scheduleID, err := deps.UpsertScheduleUseCase.Execute(ctx, usecasejob.UpsertScheduleInput{
		Name:     "missed-schedule",
		CronSpec: "0 * * * * *",
		JobType:  "missed-job",
		Enabled:  true,
	})
	require.NoError(t, err)

	// Manually set next_run_at to past time (simulating downtime)
	pastTime := time.Now().Add(-10 * time.Minute)
	updateScheduleNextRunAt(t, deps.DB, scheduleID, pastTime)

	// Debug: Verify the update worked
	schedule := getScheduleByName(t, deps.DB, "missed-schedule")
	t.Logf("Schedule ID: %d", schedule.ID)
	t.Logf("Schedule enabled: %v", schedule.Enabled)
	t.Logf("Schedule next_run_at after manual update: %v", schedule.NextRunAt.Time)
	t.Logf("Current time (time.Now()): %v", time.Now())
	t.Logf("Is schedule due? %v", schedule.NextRunAt.Time.Before(time.Now()))

	// Check ALL schedules
	var totalSchedules int
	err = deps.DB.QueryRow("SELECT COUNT(*) FROM schedules").Scan(&totalSchedules)
	require.NoError(t, err)
	t.Logf("Total schedules in database: %d", totalSchedules)

	// Verify schedule is due
	assert.True(t, schedule.NextRunAt.Time.Before(time.Now()))

	// Start manager (simulating restart after downtime)
	err = manager.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Stop(ctx)
	}()

	// Debug: Check what ListDue would return
	var dueCount int
	err = deps.DB.QueryRow("SELECT COUNT(*) FROM schedules WHERE enabled=1 AND (next_run_at IS NULL OR next_run_at <= CURRENT_TIMESTAMP)").Scan(&dueCount)
	require.NoError(t, err)
	t.Logf("Schedules that should be 'due' according to query: %d", dueCount)

	// Debug: Check the actual values in the database
	var dbNextRunAt string
	var dbEnabled int
	err = deps.DB.QueryRow("SELECT next_run_at, enabled FROM schedules WHERE id = ?", scheduleID).Scan(&dbNextRunAt, &dbEnabled)
	require.NoError(t, err)
	t.Logf("Database next_run_at (raw string): %s", dbNextRunAt)
	t.Logf("Database enabled: %d", dbEnabled)

	// Check if the comparison works
	var compareResult int
	err = deps.DB.QueryRow("SELECT CASE WHEN ? <= CURRENT_TIMESTAMP THEN 1 ELSE 0 END", dbNextRunAt).Scan(&compareResult)
	require.NoError(t, err)
	t.Logf("Is next_run_at <= CURRENT_TIMESTAMP? %d", compareResult)

	// Scheduler should detect missed schedule and enqueue job on manual trigger
	t.Log("Calling SchedulerTickUseCase.Execute()...")
	err = deps.SchedulerTickUseCase.Execute(ctx)
	require.NoError(t, err)
	t.Log("SchedulerTickUseCase.Execute() completed")

	// Debug: Check if any jobs were created at all
	var jobCount int
	err = deps.DB.QueryRow("SELECT COUNT(*) FROM jobs").Scan(&jobCount)
	require.NoError(t, err)
	t.Logf("Total jobs in database: %d", jobCount)

	// Wait for job to be created and processed (or at least enqueued)
	// Using a longer timeout in case there are issues
	waitForJobCount(t, deps.DB, string(shared.StatusSucceeded), 1, 10*time.Second)

	// Verify job was created from schedule
	jobs := getJobsByScheduleID(t, deps.DB, scheduleID)
	require.Len(t, jobs, 1)
	assert.Equal(t, "missed-job", jobs[0].Type)

	// Verify next_run_at was updated to future
	schedule = getScheduleByName(t, deps.DB, "missed-schedule")
	assert.True(t, schedule.NextRunAt.Time.After(time.Now()), "next_run_at should be updated to future")
}
