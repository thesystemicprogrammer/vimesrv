package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
)

// setupTestDatabase creates an in-memory SQLite database with migrations
func setupTestDatabase(t *testing.T) (*database.DB, *sql.DB) {
	t.Helper()

	cfg := database.Config{
		Path:            "file::memory:?cache=shared",
		MaxOpenConns:    1,
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

// TestJobRepository_Enqueue tests enqueueing a job
func TestJobRepository_Enqueue(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	job := &domain.Job{
		Type:        "test_job",
		Payload:     []byte(`{"key":"value"}`),
		Status:      shared.StatusQueued,
		Priority:    5,
		RunAt:       now,
		MaxAttempts: 3,
	}

	jobID, err := repo.Enqueue(ctx, job)

	require.NoError(t, err)
	assert.Greater(t, jobID, int64(0), "job ID should be positive")

	// Verify job was inserted
	var count int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM jobs WHERE id=?", jobID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify job details
	var insertedJob domain.Job
	var payloadStr string
	err = db.DB.QueryRow(`
		SELECT type, payload, status, priority, run_at, attempt, max_attempts
		FROM jobs WHERE id=?
	`, jobID).Scan(
		&insertedJob.Type,
		&payloadStr,
		&insertedJob.Status,
		&insertedJob.Priority,
		&insertedJob.RunAt,
		&insertedJob.Attempts,
		&insertedJob.MaxAttempts,
	)
	require.NoError(t, err)
	assert.Equal(t, "test_job", insertedJob.Type)
	assert.Equal(t, `{"key":"value"}`, payloadStr)
	assert.Equal(t, shared.StatusQueued, insertedJob.Status)
	assert.Equal(t, 5, insertedJob.Priority)
	assert.Equal(t, 0, insertedJob.Attempts)
	assert.Equal(t, 3, insertedJob.MaxAttempts)
}

// TestJobRepository_Enqueue_NilPayload tests enqueueing with nil payload
func TestJobRepository_Enqueue_NilPayload(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	job := &domain.Job{
		Type:        "test_job",
		Payload:     nil,
		Status:      shared.StatusQueued,
		Priority:    0,
		RunAt:       time.Now().UTC(),
		MaxAttempts: 3,
	}

	jobID, err := repo.Enqueue(ctx, job)

	require.NoError(t, err)
	assert.Greater(t, jobID, int64(0))
}

// TestJobRepository_ClaimNextJobDue tests claiming the next due job
func TestJobRepository_ClaimNextJobDue(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	// Enqueue a job that's due
	pastTime := time.Now().UTC().Add(-5 * time.Minute)
	job1 := &domain.Job{
		Type:        "job1",
		Payload:     []byte(`{"test":1}`),
		Status:      shared.StatusQueued,
		Priority:    5,
		RunAt:       pastTime,
		MaxAttempts: 3,
	}
	jobID1, err := repo.Enqueue(ctx, job1)
	require.NoError(t, err)

	// Claim the job
	workerID := "worker-123"
	claimedJob, found, err := repo.ClaimNextJobDue(ctx, workerID)

	require.NoError(t, err)
	assert.True(t, found, "should find a due job")
	require.NotNil(t, claimedJob)
	assert.Equal(t, jobID1, claimedJob.ID)
	assert.Equal(t, "job1", claimedJob.Type)
	assert.JSONEq(t, `{"test":1}`, string(claimedJob.Payload))
	assert.Equal(t, shared.StatusRunning, claimedJob.Status)
	assert.Equal(t, workerID, claimedJob.WorkerID.String)
	assert.Equal(t, 1, claimedJob.Attempts, "attempts should be incremented")
}

// TestJobRepository_ClaimNextJobDue_NoJobs tests claiming when no jobs are available
func TestJobRepository_ClaimNextJobDue_NoJobs(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	claimedJob, found, err := repo.ClaimNextJobDue(ctx, "worker-1")

	require.NoError(t, err)
	assert.False(t, found, "should not find any jobs")
	assert.Nil(t, claimedJob)
}

// TestJobRepository_ClaimNextJobDue_PriorityOrdering tests that jobs are claimed by priority
func TestJobRepository_ClaimNextJobDue_PriorityOrdering(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	pastTime := time.Now().UTC().Add(-1 * time.Hour)

	// Enqueue jobs with different priorities
	lowPriorityJob := &domain.Job{
		Type:        "low_priority",
		Payload:     nil,
		Status:      shared.StatusQueued,
		Priority:    1,
		RunAt:       pastTime,
		MaxAttempts: 3,
	}
	_, err := repo.Enqueue(ctx, lowPriorityJob)
	require.NoError(t, err)

	highPriorityJob := &domain.Job{
		Type:        "high_priority",
		Payload:     nil,
		Status:      shared.StatusQueued,
		Priority:    10,
		RunAt:       pastTime,
		MaxAttempts: 3,
	}
	highPriorityID, err := repo.Enqueue(ctx, highPriorityJob)
	require.NoError(t, err)

	// Claim - should get high priority job first
	claimedJob, found, err := repo.ClaimNextJobDue(ctx, "worker-1")

	require.NoError(t, err)
	assert.True(t, found)
	require.NotNil(t, claimedJob)
	assert.Equal(t, highPriorityID, claimedJob.ID)
	assert.Equal(t, "high_priority", claimedJob.Type)
	assert.Equal(t, 10, claimedJob.Priority)
}

// TestJobRepository_ClaimNextJobDue_FutureJob tests that future jobs are not claimed
func TestJobRepository_ClaimNextJobDue_FutureJob(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	// Enqueue a job scheduled for the future
	futureTime := time.Now().UTC().Add(1 * time.Hour)
	job := &domain.Job{
		Type:        "future_job",
		Payload:     nil,
		Status:      shared.StatusQueued,
		Priority:    5,
		RunAt:       futureTime,
		MaxAttempts: 3,
	}
	_, err := repo.Enqueue(ctx, job)
	require.NoError(t, err)

	// Try to claim - should find nothing
	claimedJob, found, err := repo.ClaimNextJobDue(ctx, "worker-1")

	require.NoError(t, err)
	assert.False(t, found, "should not claim future jobs")
	assert.Nil(t, claimedJob)
}

// TestJobRepository_MarkSuccess tests marking a job as successful
func TestJobRepository_MarkSuccess(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	// Create and claim a job
	job := &domain.Job{
		Type:        "test_job",
		Payload:     nil,
		Status:      shared.StatusQueued,
		Priority:    0,
		RunAt:       time.Now().UTC().Add(-1 * time.Minute),
		MaxAttempts: 3,
	}
	_, err := repo.Enqueue(ctx, job)
	require.NoError(t, err)

	claimedJob, found, err := repo.ClaimNextJobDue(ctx, "worker-1")
	require.NoError(t, err)
	require.True(t, found)

	// Mark as successful
	err = repo.MarkSuccess(ctx, claimedJob.ID)

	require.NoError(t, err)

	// Verify status changed
	var status string
	var finishedAt sql.NullTime
	err = db.DB.QueryRow("SELECT status, finished_at FROM jobs WHERE id=?", claimedJob.ID).Scan(&status, &finishedAt)
	require.NoError(t, err)
	assert.Equal(t, "succeeded", status)
	assert.True(t, finishedAt.Valid, "finished_at should be set")
}

// TestJobRepository_Reschedule tests rescheduling a failed job
func TestJobRepository_Reschedule(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	// Create and claim a job
	job := &domain.Job{
		Type:        "test_job",
		Payload:     nil,
		Status:      shared.StatusQueued,
		Priority:    0,
		RunAt:       time.Now().UTC().Add(-1 * time.Minute),
		MaxAttempts: 3,
	}
	_, err := repo.Enqueue(ctx, job)
	require.NoError(t, err)

	claimedJob, found, err := repo.ClaimNextJobDue(ctx, "worker-1")
	require.NoError(t, err)
	require.True(t, found)

	// Reschedule
	newRunAt := time.Now().UTC().Add(5 * time.Minute).Truncate(time.Second)
	lastError := "connection timeout"
	err = repo.Reschedule(ctx, claimedJob.ID, newRunAt, lastError)

	require.NoError(t, err)

	// Verify changes
	var status string
	var runAt time.Time
	var savedError string
	err = db.DB.QueryRow("SELECT status, run_at, last_error FROM jobs WHERE id=?", claimedJob.ID).Scan(&status, &runAt, &savedError)
	require.NoError(t, err)
	assert.Equal(t, "queued", status)
	assert.Equal(t, newRunAt, runAt.Truncate(time.Second))
	assert.Equal(t, lastError, savedError)
}

// TestJobRepository_Reschedule_LongError tests error message truncation
func TestJobRepository_Reschedule_LongError(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	job := &domain.Job{
		Type:        "test_job",
		Payload:     nil,
		Status:      shared.StatusQueued,
		Priority:    0,
		RunAt:       time.Now().UTC().Add(-1 * time.Minute),
		MaxAttempts: 3,
	}
	_, err := repo.Enqueue(ctx, job)
	require.NoError(t, err)

	claimedJob, found, err := repo.ClaimNextJobDue(ctx, "worker-1")
	require.NoError(t, err)
	require.True(t, found)

	// Create a very long error message (> 2000 chars)
	longError := ""
	for i := 0; i < 300; i++ {
		longError += "This is a very long error message. "
	}
	assert.Greater(t, len(longError), 2000, "test error should be > 2000 chars")

	newRunAt := time.Now().UTC().Add(1 * time.Minute)
	err = repo.Reschedule(ctx, claimedJob.ID, newRunAt, longError)

	require.NoError(t, err)

	// Verify error was truncated to 2000 chars
	var savedError string
	err = db.DB.QueryRow("SELECT last_error FROM jobs WHERE id=?", claimedJob.ID).Scan(&savedError)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(savedError), 2000, "error should be truncated to 2000 chars")
	assert.Equal(t, longError[:2000], savedError)
}

// TestJobRepository_MarkDead tests marking a job as dead after max attempts
func TestJobRepository_MarkDead(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	job := &domain.Job{
		Type:        "test_job",
		Payload:     nil,
		Status:      shared.StatusQueued,
		Priority:    0,
		RunAt:       time.Now().UTC().Add(-1 * time.Minute),
		MaxAttempts: 1,
	}
	_, err := repo.Enqueue(ctx, job)
	require.NoError(t, err)

	claimedJob, found, err := repo.ClaimNextJobDue(ctx, "worker-1")
	require.NoError(t, err)
	require.True(t, found)

	// Mark as dead
	lastError := "max attempts exceeded"
	err = repo.MarkDead(ctx, claimedJob.ID, lastError)

	require.NoError(t, err)

	// Verify status and fields
	var status string
	var finishedAt sql.NullTime
	var savedError string
	err = db.DB.QueryRow("SELECT status, finished_at, last_error FROM jobs WHERE id=?", claimedJob.ID).Scan(&status, &finishedAt, &savedError)
	require.NoError(t, err)
	assert.Equal(t, "dead", status)
	assert.True(t, finishedAt.Valid, "finished_at should be set")
	assert.Equal(t, lastError, savedError)
}

// TestJobRepository_FindStuckJobs tests finding jobs stuck in running state
func TestJobRepository_FindStuckJobs(t *testing.T) {
	db, sqlDB := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	threshold := 5 * time.Minute

	// Insert a stuck job (started 10 minutes ago, still running)
	stuckStartTime := now.Add(-10 * time.Minute)
	_, err := sqlDB.Exec(`
		INSERT INTO jobs (type, payload, status, priority, run_at, attempt, max_attempts, worker_id, started_at, created_at, updated_at)
		VALUES ('stuck_job', '{}', 'running', 0, ?, 1, 3, 'worker-1', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, now.Add(-15*time.Minute), stuckStartTime)
	require.NoError(t, err)

	// Insert a recent running job (started 2 minutes ago, not stuck)
	recentStartTime := now.Add(-2 * time.Minute)
	_, err = sqlDB.Exec(`
		INSERT INTO jobs (type, payload, status, priority, run_at, attempt, max_attempts, worker_id, started_at, created_at, updated_at)
		VALUES ('recent_job', '{}', 'running', 0, ?, 1, 3, 'worker-2', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, now.Add(-10*time.Minute), recentStartTime)
	require.NoError(t, err)

	// Insert a completed job (should not be found)
	_, err = sqlDB.Exec(`
		INSERT INTO jobs (type, payload, status, priority, run_at, attempt, max_attempts, finished_at, created_at, updated_at)
		VALUES ('completed_job', '{}', 'succeeded', 0, ?, 1, 3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, now.Add(-20*time.Minute))
	require.NoError(t, err)

	// Find stuck jobs
	stuckJobs, err := repo.FindStuckJobs(ctx, threshold)

	require.NoError(t, err)
	require.Len(t, stuckJobs, 1, "should find exactly 1 stuck job")
	assert.Equal(t, "stuck_job", stuckJobs[0].Type)
	assert.Equal(t, shared.StatusRunning, stuckJobs[0].Status)
	assert.Equal(t, "worker-1", stuckJobs[0].WorkerID.String)
}

// TestJobRepository_FindStuckJobs_NoStuckJobs tests when no jobs are stuck
func TestJobRepository_FindStuckJobs_NoStuckJobs(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	threshold := 5 * time.Minute
	stuckJobs, err := repo.FindStuckJobs(ctx, threshold)

	require.NoError(t, err)
	assert.Empty(t, stuckJobs, "should find no stuck jobs")
}

// TestJobRepository_ResetStuckJob tests resetting a stuck job back to queued
func TestJobRepository_ResetStuckJob(t *testing.T) {
	db, sqlDB := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()

	// Insert a stuck running job
	result, err := sqlDB.Exec(`
		INSERT INTO jobs (type, payload, status, priority, run_at, attempt, max_attempts, worker_id, started_at, created_at, updated_at)
		VALUES ('stuck_job', '{}', 'running', 0, ?, 2, 3, 'worker-1', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, now.Add(-10*time.Minute), now.Add(-10*time.Minute))
	require.NoError(t, err)
	jobID, err := result.LastInsertId()
	require.NoError(t, err)

	// Reset the stuck job
	err = repo.ResetStuckJob(ctx, jobID)

	require.NoError(t, err)

	// Verify job was reset
	var status string
	var workerID sql.NullString
	var startedAt sql.NullTime
	var attempts int
	err = sqlDB.QueryRow("SELECT status, worker_id, started_at, attempt FROM jobs WHERE id=?", jobID).Scan(&status, &workerID, &startedAt, &attempts)
	require.NoError(t, err)
	assert.Equal(t, "queued", status)
	assert.False(t, workerID.Valid, "worker_id should be NULL")
	assert.False(t, startedAt.Valid, "started_at should be NULL")
	assert.Equal(t, 2, attempts, "attempts should not be reset")
}

// TestJobRepository_ResetStuckJob_AlreadyCompleted tests resetting a job that's already completed
func TestJobRepository_ResetStuckJob_AlreadyCompleted(t *testing.T) {
	db, sqlDB := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()

	// Insert a completed job
	result, err := sqlDB.Exec(`
		INSERT INTO jobs (type, payload, status, priority, run_at, attempt, max_attempts, finished_at, created_at, updated_at)
		VALUES ('completed_job', '{}', 'succeeded', 0, ?, 1, 3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, now.Add(-10*time.Minute))
	require.NoError(t, err)
	jobID, err := result.LastInsertId()
	require.NoError(t, err)

	// Try to reset - should succeed but not affect anything
	err = repo.ResetStuckJob(ctx, jobID)

	require.NoError(t, err)

	// Verify job status didn't change
	var status string
	err = sqlDB.QueryRow("SELECT status FROM jobs WHERE id=?", jobID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "succeeded", status, "status should remain succeeded")
}

// TestJobRepository_ConcurrentClaims tests that concurrent claims don't claim the same job
func TestJobRepository_ConcurrentClaims(t *testing.T) {
	db, _ := setupTestDatabase(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	// Enqueue a single job
	job := &domain.Job{
		Type:        "test_job",
		Payload:     nil,
		Status:      shared.StatusQueued,
		Priority:    0,
		RunAt:       time.Now().UTC().Add(-1 * time.Minute),
		MaxAttempts: 3,
	}
	_, err := repo.Enqueue(ctx, job)
	require.NoError(t, err)

	// Try to claim from multiple workers concurrently
	workers := 5
	claims := make(chan *domain.Job, workers)
	errors := make(chan error, workers)

	for i := 0; i < workers; i++ {
		workerID := "worker-" + string(rune(i+'0'))
		go func(wID string) {
			claimedJob, found, err := repo.ClaimNextJobDue(ctx, wID)
			if err != nil {
				errors <- err
				return
			}
			if found {
				claims <- claimedJob
			}
			errors <- nil
		}(workerID)
	}

	// Collect results
	claimedJobs := []*domain.Job{}
	for i := 0; i < workers; i++ {
		select {
		case job := <-claims:
			claimedJobs = append(claimedJobs, job)
		case err := <-errors:
			require.NoError(t, err)
		}
	}

	// Only one worker should have claimed the job
	assert.Len(t, claimedJobs, 1, "only one worker should claim the job")
}
