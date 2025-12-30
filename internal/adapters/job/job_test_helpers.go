package job

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/repository"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	usecasejob "github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// JobManagerDependencies holds all dependencies for testing
type JobManagerDependencies struct {
	DB                    *sql.DB
	JobRepository         *repository.JobRepository
	ScheduleRepository    *repository.ScheduleRepository
	UpsertScheduleUseCase *usecasejob.UpsertScheduleUseCase
	EnqueueJobUseCase     *usecasejob.EnqueueJobUseCase
	ProcessNextJobUseCase *usecasejob.ProcessNextJobUseCase
	SchedulerTickUseCase  *usecasejob.SchedulerTickUseCase
	HandlerRegistry       *HandlerRegistry
	BackoffStrategy       ports.BackoffStrategy
	CronParser            ports.CronParser
	Clock                 ports.Clock
}

// ===============================================
// Database Setup
// ===============================================

// setupTestDatabase creates an in-memory SQLite database with migrations
//
// CRITICAL: Uses "file::memory:?cache=shared" instead of plain ":memory:" to ensure
// all connections share the same in-memory database. With plain ":memory:", each
// connection in the pool gets a separate database, causing goroutines to see different
// data (jobs table exists in main thread but not in worker goroutines).
//
// Also sets MaxOpenConns=1 as an additional safeguard to ensure all operations use
// the same connection, preventing any potential race conditions or visibility issues.
func setupTestDatabase(t *testing.T) (*database.DB, *sql.DB) {
	t.Helper()

	cfg := database.Config{
		Path:            "file::memory:?cache=shared",
		MaxOpenConns:    1, // Force single connection for test reliability
		MaxIdleConns:    1,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	db, err := database.New(cfg)
	require.NoError(t, err, "failed to create test database")

	// Run migrations
	migration := database.NewDatabaseMigration(db.DB)
	err = migration.Migrate()
	require.NoError(t, err, "failed to run migrations")

	t.Cleanup(func() {
		db.Close()
	})

	return db, db.DB
}

// ===============================================
// JobManager Setup
// ===============================================

// setupTestJobManager creates a fully configured JobManager with all dependencies
func setupTestJobManager(t *testing.T, cfg config.JobConfig, handlers map[string]ports.JobHandler) (*JobManager, *JobManagerDependencies) {
	t.Helper()

	db, sqlDB := setupTestDatabase(t)

	// Create repositories
	jobRepo := repository.NewJobRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)

	// Create handler registry
	handlerRegistry := NewHandlerRegistry()
	for jobType, handler := range handlers {
		handlerRegistry.Register(jobType, handler)
	}

	// Create components
	backoffStrategy := NewExponentialBackoff(1, 60)
	cronParser := NewRobfigCronParser()
	clock := ports.RealClock{}

	// Create use cases
	enqueueJobUC := usecasejob.NewEnqueueJobUseCase(cfg, jobRepo, clock)
	processNextJobUC := usecasejob.NewProcessNextJobUseCase(jobRepo, handlerRegistry, backoffStrategy, clock)
	schedulerTickUC := usecasejob.NewSchedulerTickUseCase(cfg, scheduleRepo, cronParser, clock)
	upsertScheduleUC := usecasejob.NewUpsertScheduleUseCase(cfg, scheduleRepo, cronParser, clock)
	recoverStuckJobsUC := usecasejob.NewRecoverStuckJobsUseCase(cfg, jobRepo, clock)

	// Create JobManager
	jobManager := NewJobManager(JobManagerInput{
		Config:                  cfg,
		ProcessNextJobUseCase:   processNextJobUC,
		SchedulerTickUseCase:    schedulerTickUC,
		RecoverStuckJobsUseCase: recoverStuckJobsUC,
		JobRepository:           jobRepo,
		ScheduleRepository:      scheduleRepo,
		Handlers:                handlerRegistry,
		BackoffStrategy:         backoffStrategy,
		Cron:                    cronParser,
		Clock:                   clock,
	})

	deps := &JobManagerDependencies{
		DB:                    sqlDB,
		JobRepository:         jobRepo,
		ScheduleRepository:    scheduleRepo,
		UpsertScheduleUseCase: upsertScheduleUC,
		EnqueueJobUseCase:     enqueueJobUC,
		ProcessNextJobUseCase: processNextJobUC,
		SchedulerTickUseCase:  schedulerTickUC,
		HandlerRegistry:       handlerRegistry,
		BackoffStrategy:       backoffStrategy,
		CronParser:            cronParser,
		Clock:                 clock,
	}

	return jobManager, deps
}

// setupTestJobManagerWithMockClock creates a JobManager with a MockClock for time-based tests
func setupTestJobManagerWithMockClock(t *testing.T, cfg config.JobConfig, handlers map[string]ports.JobHandler, mockClock *MockClock) (*JobManager, *JobManagerDependencies) {
	t.Helper()

	db, sqlDB := setupTestDatabase(t)

	// Create repositories
	jobRepo := repository.NewJobRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)

	// Create handler registry
	handlerRegistry := NewHandlerRegistry()
	for jobType, handler := range handlers {
		handlerRegistry.Register(jobType, handler)
	}

	// Create components
	backoffStrategy := NewExponentialBackoff(1, 60)
	cronParser := NewRobfigCronParser()

	// Create use cases with mock clock
	enqueueJobUC := usecasejob.NewEnqueueJobUseCase(cfg, jobRepo, mockClock)
	processNextJobUC := usecasejob.NewProcessNextJobUseCase(jobRepo, handlerRegistry, backoffStrategy, mockClock)
	schedulerTickUC := usecasejob.NewSchedulerTickUseCase(cfg, scheduleRepo, cronParser, mockClock)
	upsertScheduleUC := usecasejob.NewUpsertScheduleUseCase(cfg, scheduleRepo, cronParser, mockClock)
	recoverStuckJobsUC := usecasejob.NewRecoverStuckJobsUseCase(cfg, jobRepo, mockClock)

	// Create JobManager
	jobManager := NewJobManager(JobManagerInput{
		Config:                  cfg,
		ProcessNextJobUseCase:   processNextJobUC,
		SchedulerTickUseCase:    schedulerTickUC,
		RecoverStuckJobsUseCase: recoverStuckJobsUC,
		JobRepository:           jobRepo,
		ScheduleRepository:      scheduleRepo,
		Handlers:                handlerRegistry,
		BackoffStrategy:         backoffStrategy,
		Cron:                    cronParser,
		Clock:                   mockClock,
	})

	deps := &JobManagerDependencies{
		DB:                    sqlDB,
		JobRepository:         jobRepo,
		ScheduleRepository:    scheduleRepo,
		UpsertScheduleUseCase: upsertScheduleUC,
		EnqueueJobUseCase:     enqueueJobUC,
		ProcessNextJobUseCase: processNextJobUC,
		SchedulerTickUseCase:  schedulerTickUC,
		HandlerRegistry:       handlerRegistry,
		BackoffStrategy:       backoffStrategy,
		CronParser:            cronParser,
		Clock:                 mockClock,
	}

	return jobManager, deps
}

// setupTestJobManagerWithSecondCron creates a JobManager with a second-based cron parser
// for faster integration tests. Instead of waiting minutes for cron schedules, tests can
// use second-level precision (e.g., "*/2 * * * * *" = every 2 seconds).
//
// This uses real time (not MockClock) for realistic timing tests without long waits.
func setupTestJobManagerWithSecondCron(t *testing.T, cfg config.JobConfig, handlers map[string]ports.JobHandler) (*JobManager, *JobManagerDependencies) {
	t.Helper()

	db, sqlDB := setupTestDatabase(t)

	// Create repositories
	jobRepo := repository.NewJobRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)

	// Create handler registry
	handlerRegistry := NewHandlerRegistry()
	for jobType, handler := range handlers {
		handlerRegistry.Register(jobType, handler)
	}

	// Create components with second-based cron parser
	backoffStrategy := NewExponentialBackoff(1, 60)
	cronParser := NewSecondBasedCronParser() // Use second-based parser for fast tests
	clock := ports.RealClock{}

	// Create use cases
	enqueueJobUC := usecasejob.NewEnqueueJobUseCase(cfg, jobRepo, clock)
	processNextJobUC := usecasejob.NewProcessNextJobUseCase(jobRepo, handlerRegistry, backoffStrategy, clock)
	schedulerTickUC := usecasejob.NewSchedulerTickUseCase(cfg, scheduleRepo, cronParser, clock)
	upsertScheduleUC := usecasejob.NewUpsertScheduleUseCase(cfg, scheduleRepo, cronParser, clock)
	recoverStuckJobsUC := usecasejob.NewRecoverStuckJobsUseCase(cfg, jobRepo, clock)

	// Create JobManager
	jobManager := NewJobManager(JobManagerInput{
		Config:                  cfg,
		ProcessNextJobUseCase:   processNextJobUC,
		SchedulerTickUseCase:    schedulerTickUC,
		RecoverStuckJobsUseCase: recoverStuckJobsUC,
		JobRepository:           jobRepo,
		ScheduleRepository:      scheduleRepo,
		Handlers:                handlerRegistry,
		BackoffStrategy:         backoffStrategy,
		Cron:                    cronParser,
		Clock:                   clock,
	})

	deps := &JobManagerDependencies{
		DB:                    sqlDB,
		JobRepository:         jobRepo,
		ScheduleRepository:    scheduleRepo,
		UpsertScheduleUseCase: upsertScheduleUC,
		EnqueueJobUseCase:     enqueueJobUC,
		ProcessNextJobUseCase: processNextJobUC,
		SchedulerTickUseCase:  schedulerTickUC,
		HandlerRegistry:       handlerRegistry,
		BackoffStrategy:       backoffStrategy,
		CronParser:            cronParser,
		Clock:                 clock,
	}

	return jobManager, deps
}

// ===============================================
// Configuration
// ===============================================

// testJobConfig returns a fast configuration for tests
//
// Note: SchedulerIntervalInSeconds is set to 10 hours (36000 seconds) to effectively
// disable background scheduler ticks. Tests manually trigger SchedulerTickUseCase.Execute()
// for deterministic scheduler behavior. Workers still poll every 1 second for jobs.
//
// Note: StuckJobCheckIntervalMinutes is set to 10 hours (600 minutes) to effectively
// disable background stuck job recovery. Tests can manually trigger RecoverStuckJobsUseCase.Execute()
// for deterministic stuck job recovery behavior.
func testJobConfig() config.JobConfig {
	return config.JobConfig{
		WorkerCount:                  2,
		PollingIntervalInSeconds:     1,
		MaxAttempts:                  3,
		SchedulerIntervalInSeconds:   36000, // 10 hours - effectively disabled for manual testing
		SchedulerBatch:               10,
		BackoffBaseSeconds:           1,
		BackoffMaxSeconds:            60,
		StuckJobThresholdMinutes:     480, // 8 hours
		StuckJobCheckIntervalMinutes: 600, // 10 hours - effectively disabled for manual testing
	}
}

// ===============================================
// Mock Clock
// ===============================================

// MockClock is a controllable clock for testing
type MockClock struct {
	mu   sync.Mutex
	time time.Time
}

// NewMockClock creates a new MockClock starting at the given time
func NewMockClock(t time.Time) *MockClock {
	return &MockClock{time: t}
}

// Now returns the current mock time
func (m *MockClock) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.time
}

// Advance moves the clock forward by the given duration
func (m *MockClock) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.time = m.time.Add(d)
}

// Set sets the clock to a specific time
func (m *MockClock) Set(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.time = t
}

// ===============================================
// Mock Job Handlers
// ===============================================

// newSleepHandler creates a handler that sleeps for the specified duration
func newSleepHandler(duration time.Duration) ports.JobHandler {
	return func(ctx context.Context, job *domain.Job) error {
		time.Sleep(duration)
		return nil
	}
}

// newFailNTimesHandler creates a handler that fails N times then succeeds
func newFailNTimesHandler(failCount int) ports.JobHandler {
	attempts := &atomic.Int32{}
	return func(ctx context.Context, job *domain.Job) error {
		currentAttempt := attempts.Add(1)
		if currentAttempt <= int32(failCount) {
			return fmt.Errorf("intentional failure %d/%d", currentAttempt, failCount)
		}
		return nil
	}
}

// newPanicHandler creates a handler that panics
func newPanicHandler() ports.JobHandler {
	return func(ctx context.Context, job *domain.Job) error {
		panic("intentional panic for testing")
	}
}

// newCounterHandler creates a handler that increments a counter
func newCounterHandler(counter *atomic.Int32) ports.JobHandler {
	return func(ctx context.Context, job *domain.Job) error {
		counter.Add(1)
		return nil
	}
}

// newSuccessHandler creates a handler that always succeeds immediately
func newSuccessHandler() ports.JobHandler {
	return func(ctx context.Context, job *domain.Job) error {
		return nil
	}
}

// newErrorHandler creates a handler that always fails
func newErrorHandler(errMsg string) ports.JobHandler {
	return func(ctx context.Context, job *domain.Job) error {
		return fmt.Errorf("%s", errMsg)
	}
}

// ===============================================
// Assertion Helpers - Jobs
// ===============================================

// waitForJobStatus polls until the job reaches the expected status or times out
func waitForJobStatus(t *testing.T, db *sql.DB, jobID int64, expectedStatus string, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			job := getJobByID(t, db, jobID)
			t.Fatalf("timeout waiting for job %d to reach status '%s', current status: '%s'", jobID, expectedStatus, job.Status)
		case <-ticker.C:
			job := getJobByID(t, db, jobID)
			if string(job.Status) == expectedStatus {
				return
			}
		}
	}
}

// waitForJobCount polls until the job count matches the expected count or times out
func waitForJobCount(t *testing.T, db *sql.DB, status string, expectedCount int, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			currentCount := getJobCount(t, db, status)
			t.Fatalf("timeout waiting for job count with status '%s' to reach %d, current count: %d", status, expectedCount, currentCount)
		case <-ticker.C:
			currentCount := getJobCount(t, db, status)
			if currentCount >= expectedCount {
				return
			}
		}
	}
}

// assertJobCount asserts the number of jobs with the given status
func assertJobCount(t *testing.T, db *sql.DB, status string, expectedCount int) {
	t.Helper()

	count := getJobCount(t, db, status)
	require.Equal(t, expectedCount, count, "unexpected job count for status '%s'", status)
}

// getJobCount returns the number of jobs with the given status
func getJobCount(t *testing.T, db *sql.DB, status string) int {
	t.Helper()

	var count int
	query := "SELECT COUNT(*) FROM jobs WHERE status = ?"
	err := db.QueryRow(query, status).Scan(&count)
	require.NoError(t, err, "failed to get job count")
	return count
}

// getJobByID retrieves a job by ID
//
// Note: Payload must be scanned as string because SQLite stores JSON as TEXT type,
// which cannot be directly scanned into json.RawMessage ([]byte)
func getJobByID(t *testing.T, db *sql.DB, jobID int64) *domain.Job {
	t.Helper()

	query := `
		SELECT id, type, payload, status, priority, run_at, attempt, max_attempts, 
		       last_error, worker_id, scheduled_id, created_at, started_at, finished_at, updated_at
		FROM jobs WHERE id = ?
	`
	var job domain.Job
	var payloadStr string // SQLite returns TEXT, scan as string first
	err := db.QueryRow(query, jobID).Scan(
		&job.ID, &job.Type, &payloadStr, &job.Status, &job.Priority, &job.RunAt,
		&job.Attempts, &job.MaxAttempts, &job.LastError, &job.WorkerID, &job.ScheduledID,
		&job.CreatedAt, &job.StartedAt, &job.FinishedAt, &job.UpdatedAt,
	)
	require.NoError(t, err, "failed to get job by ID")
	job.Payload = []byte(payloadStr) // Convert string to []byte
	return &job
}

// getJobsByScheduleID retrieves all jobs created by a specific schedule
func getJobsByScheduleID(t *testing.T, db *sql.DB, scheduleID int64) []*domain.Job {
	t.Helper()

	query := `
		SELECT id, type, payload, status, priority, run_at, attempt, max_attempts, 
		       last_error, worker_id, scheduled_id, created_at, started_at, finished_at, updated_at
		FROM jobs WHERE scheduled_id = ?
	`
	rows, err := db.Query(query, scheduleID)
	require.NoError(t, err, "failed to query jobs by schedule ID")
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		var job domain.Job
		var payloadStr string // SQLite returns TEXT, scan as string first
		err := rows.Scan(
			&job.ID, &job.Type, &payloadStr, &job.Status, &job.Priority, &job.RunAt,
			&job.Attempts, &job.MaxAttempts, &job.LastError, &job.WorkerID, &job.ScheduledID,
			&job.CreatedAt, &job.StartedAt, &job.FinishedAt, &job.UpdatedAt,
		)
		require.NoError(t, err, "failed to scan job")
		job.Payload = []byte(payloadStr) // Convert string to []byte
		jobs = append(jobs, &job)
	}
	require.NoError(t, rows.Err(), "error iterating jobs")
	return jobs
}

// ===============================================
// Assertion Helpers - Schedules
// ===============================================

// getScheduleByName retrieves a schedule by name
func getScheduleByName(t *testing.T, db *sql.DB, name string) *domain.Schedule {
	t.Helper()

	query := `
		SELECT id, name, cron_spec, job_type, payload, priority, max_attempts, 
		       enabled, next_run_at, last_enqueued_at, updated_at
		FROM schedules WHERE name = ?
	`
	var schedule domain.Schedule
	var payload sql.NullString
	var enabledInt int

	err := db.QueryRow(query, name).Scan(
		&schedule.ID, &schedule.Name, &schedule.CronSpec, &schedule.JobType, &payload,
		&schedule.Priority, &schedule.MaxAttempts, &enabledInt, &schedule.NextRunAt,
		&schedule.LastEnqueuedAt, &schedule.UpdatedAt,
	)
	require.NoError(t, err, "failed to get schedule by name")

	if payload.Valid {
		schedule.Payload = []byte(payload.String)
	}
	schedule.Enabled = enabledInt == 1

	return &schedule
}

// assertScheduleNextRunAtInFuture asserts that the schedule's next_run_at is in the future
func assertScheduleNextRunAtInFuture(t *testing.T, db *sql.DB, scheduleID int64) {
	t.Helper()

	var nextRunAt sql.NullTime
	query := "SELECT next_run_at FROM schedules WHERE id = ?"
	err := db.QueryRow(query, scheduleID).Scan(&nextRunAt)
	require.NoError(t, err, "failed to get schedule next_run_at")
	require.True(t, nextRunAt.Valid, "next_run_at should not be NULL")
	require.True(t, nextRunAt.Time.After(time.Now()), "next_run_at should be in the future")
}

// assertScheduleLastEnqueuedAtRecent asserts that the schedule was enqueued recently
func assertScheduleLastEnqueuedAtRecent(t *testing.T, db *sql.DB, scheduleID int64, within time.Duration) {
	t.Helper()

	var lastEnqueuedAt sql.NullTime
	query := "SELECT last_enqueued_at FROM schedules WHERE id = ?"
	err := db.QueryRow(query, scheduleID).Scan(&lastEnqueuedAt)
	require.NoError(t, err, "failed to get schedule last_enqueued_at")
	require.True(t, lastEnqueuedAt.Valid, "last_enqueued_at should not be NULL")
	require.WithinDuration(t, time.Now(), lastEnqueuedAt.Time, within, "last_enqueued_at should be recent")
}

// ===============================================
// Direct DB Manipulation
// ===============================================

// insertJobDirectly inserts a job directly into the database
func insertJobDirectly(t *testing.T, db *sql.DB, job *domain.Job) int64 {
	t.Helper()

	query := `
		INSERT INTO jobs (type, payload, status, priority, run_at, attempt, max_attempts, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	// IMPORTANT: Convert RunAt to UTC to match behavior of JobRepository.Enqueue
	result, err := db.Exec(query, job.Type, string(job.Payload), job.Status, job.Priority, job.RunAt.UTC(), job.Attempts, job.MaxAttempts)
	require.NoError(t, err, "failed to insert job")

	id, err := result.LastInsertId()
	require.NoError(t, err, "failed to get last insert ID")
	return id
}

// insertScheduleDirectly inserts a schedule directly into the database
func insertScheduleDirectly(t *testing.T, db *sql.DB, schedule *domain.Schedule) int64 {
	t.Helper()

	enabledInt := 0
	if schedule.Enabled {
		enabledInt = 1
	}

	query := `
		INSERT INTO schedules (name, cron_spec, job_type, payload, priority, max_attempts, enabled, next_run_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	var nextRunAt interface{}
	if schedule.NextRunAt.Valid {
		nextRunAt = schedule.NextRunAt.Time
	}

	result, err := db.Exec(query, schedule.Name, schedule.CronSpec, schedule.JobType, string(schedule.Payload),
		schedule.Priority, schedule.MaxAttempts, enabledInt, nextRunAt)
	require.NoError(t, err, "failed to insert schedule")

	id, err := result.LastInsertId()
	require.NoError(t, err, "failed to get last insert ID")
	return id
}

// updateJobStatus updates a job's status
func updateJobStatus(t *testing.T, db *sql.DB, jobID int64, status string) {
	t.Helper()

	query := "UPDATE jobs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := db.Exec(query, status, jobID)
	require.NoError(t, err, "failed to update job status")
}

// updateScheduleNextRunAt updates a schedule's next_run_at
// Note: Converts time to UTC to ensure proper comparison with SQLite's CURRENT_TIMESTAMP
func updateScheduleNextRunAt(t *testing.T, db *sql.DB, scheduleID int64, nextRunAt time.Time) {
	t.Helper()

	query := "UPDATE schedules SET next_run_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := db.Exec(query, nextRunAt.UTC(), scheduleID)
	require.NoError(t, err, "failed to update schedule next_run_at")
}
