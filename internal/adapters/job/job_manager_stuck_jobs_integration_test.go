package job

import (
	"context"
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
// Stuck Job Recovery Tests
// ===============================================

// TestJobManager_StuckJobRecovery verifies that jobs stuck in "running" state are reset
func TestJobManager_StuckJobRecovery(t *testing.T) {
	cfg := testJobConfig()
	cfg.StuckJobThresholdMinutes = 1 // 1 minute threshold for faster testing
	handlers := map[string]ports.JobHandler{
		"test-job": newSuccessHandler(),
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	ctx := context.Background()

	// Manually create a stuck job that's been running for 2 minutes
	stuckTime := time.Now().Add(-2 * time.Minute)
	jobID := insertJobDirectly(t, deps.DB, &domain.Job{
		Type:        "test-job",
		Payload:     []byte(`{}`),
		Status:      shared.StatusRunning,
		Priority:    0,
		RunAt:       time.Now().Add(-2 * time.Minute),
		Attempts:    1,
		MaxAttempts: 3,
	})

	// Update started_at to be 2 minutes ago (convert to UTC to match repository behavior)
	_, err := deps.DB.Exec("UPDATE jobs SET started_at = ?, worker_id = ? WHERE id = ?", stuckTime.UTC(), "old-worker", jobID)
	require.NoError(t, err)

	// Verify job is stuck
	job := getJobByID(t, deps.DB, jobID)
	require.Equal(t, shared.StatusRunning, job.Status, "job should be running")

	// Manually trigger stuck job recovery
	err = deps.JobRepository.ResetStuckJob(ctx, jobID)
	require.NoError(t, err)

	// Verify job was reset to 'queued' status
	job = getJobByID(t, deps.DB, jobID)
	assert.Equal(t, shared.StatusQueued, job.Status, "stuck job should be reset to queued")
	assert.False(t, job.StartedAt.Valid, "started_at should be NULL after reset")
	assert.False(t, job.WorkerID.Valid, "worker_id should be NULL after reset")

	// Start manager and verify job gets processed successfully
	err = manager.Start()
	require.NoError(t, err)

	// Wait for job to complete
	waitForJobStatus(t, deps.DB, jobID, string(shared.StatusSucceeded), 5*time.Second)

	// Cleanup
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manager.Stop(stopCtx)
}

// TestJobManager_StuckJobRecoveryRespectsThreshold verifies threshold is respected
func TestJobManager_StuckJobRecoveryRespectsThreshold(t *testing.T) {
	cfg := testJobConfig()
	cfg.StuckJobThresholdMinutes = 120 // 2 hours
	manager, deps := setupTestJobManager(t, cfg, nil)

	ctx := context.Background()

	// Create a job that's been running for 1 hour (under threshold)
	runningTime := time.Now().Add(-1 * time.Hour)
	jobID := insertJobDirectly(t, deps.DB, &domain.Job{
		Type:        "test-job",
		Payload:     []byte(`{}`),
		Status:      shared.StatusRunning,
		Priority:    0,
		RunAt:       runningTime,
		Attempts:    1,
		MaxAttempts: 3,
	})

	// Update started_at to be 1 hour ago (convert to UTC to match repository behavior)
	_, err := deps.DB.Exec("UPDATE jobs SET started_at = ?, worker_id = ? WHERE id = ?", runningTime.UTC(), "test-worker", jobID)
	require.NoError(t, err)

	// Try to manually find stuck jobs (should find none because job is under threshold)
	stuckJobs, err := deps.JobRepository.FindStuckJobs(ctx, 120*time.Minute)
	require.NoError(t, err)
	assert.Len(t, stuckJobs, 0, "job under threshold should not be considered stuck")

	// Verify job is still in running status
	job := getJobByID(t, deps.DB, jobID)
	assert.Equal(t, shared.StatusRunning, job.Status)

	// Cleanup
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manager.Stop(stopCtx)
}

// TestJobManager_RestartWithRunningJobs tests stuck job recovery after crash/restart
func TestJobManager_RestartWithRunningJobs(t *testing.T) {
	cfg := testJobConfig()
	handlers := map[string]ports.JobHandler{
		"long-running-job": newSleepHandler(5 * time.Second),
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)

	// Enqueue job
	ctx := context.Background()
	_, err = deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
		Type: "long-running-job",
	})
	require.NoError(t, err)

	// Wait for job to start (status = running)
	waitForJobCount(t, deps.DB, string(shared.StatusRunning), 1, 3*time.Second)

	// Get job ID
	var jobID int64
	err = deps.DB.QueryRow("SELECT id FROM jobs WHERE type = 'long-running-job'").Scan(&jobID)
	require.NoError(t, err)

	// Stop manager abruptly (simulating crash)
	stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	manager.Stop(stopCtx) // May error due to timeout, that's ok

	// Verify job is still in 'running' state after crash
	job := getJobByID(t, deps.DB, jobID)
	assert.Equal(t, shared.StatusRunning, job.Status, "job should remain in running state after crash")

	// Simulate the passage of time - update started_at to be old enough to be considered stuck
	stuckTime := time.Now().Add(-9 * time.Hour).UTC()
	_, err = deps.DB.Exec("UPDATE jobs SET started_at = ? WHERE id = ?", stuckTime, jobID)
	require.NoError(t, err)

	// Manually call stuck job recovery (simulating what happens on restart)
	threshold := 8 * time.Hour // Default threshold is 480 minutes (8 hours)
	stuckJobs, err := deps.JobRepository.FindStuckJobs(ctx, threshold)
	require.NoError(t, err)
	assert.Len(t, stuckJobs, 1, "should find 1 stuck job")

	// Reset the stuck job
	for _, stuckJob := range stuckJobs {
		err = deps.JobRepository.ResetStuckJob(ctx, stuckJob.ID)
		require.NoError(t, err)
	}

	// Verify job is now back in 'queued' status
	job = getJobByID(t, deps.DB, jobID)
	assert.Equal(t, shared.StatusQueued, job.Status, "stuck job should be reset to queued")
	assert.False(t, job.StartedAt.Valid, "started_at should be NULL after reset")
	assert.False(t, job.WorkerID.Valid, "worker_id should be NULL after reset")

	// Register a success handler for the long-running job (instead of sleep)
	deps.HandlerRegistry.Register("long-running-job", newSuccessHandler())

	// Start a new manager (simulating restart)
	manager2 := NewJobManager(JobManagerInput{
		Config:                  cfg,
		ProcessNextJobUseCase:   deps.ProcessNextJobUseCase,
		SchedulerTickUseCase:    deps.SchedulerTickUseCase,
		RecoverStuckJobsUseCase: usecasejob.NewRecoverStuckJobsUseCase(cfg, deps.JobRepository, deps.Clock),
		JobRepository:           deps.JobRepository,
		ScheduleRepository:      deps.ScheduleRepository,
		Handlers:                deps.HandlerRegistry,
		BackoffStrategy:         deps.BackoffStrategy,
		Cron:                    deps.CronParser,
		Clock:                   deps.Clock,
	})

	err = manager2.Start()
	require.NoError(t, err)

	// Job should now be processed successfully
	waitForJobStatus(t, deps.DB, jobID, string(shared.StatusSucceeded), 5*time.Second)

	// Cleanup
	stopCtx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	manager2.Stop(stopCtx2)
}
