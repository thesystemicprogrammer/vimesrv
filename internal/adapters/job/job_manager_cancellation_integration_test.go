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
// Job Cancellation Tests
// ===============================================

// TestJobManager_GracefulShutdown tests that running jobs complete during graceful shutdown
func TestJobManager_GracefulShutdown(t *testing.T) {
	cfg := testJobConfig()
	handlers := map[string]ports.JobHandler{
		"long-job": newSleepHandler(2 * time.Second),
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)

	// Enqueue a long-running job
	ctx := context.Background()
	_, err = deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
		Type: "long-job",
	})
	require.NoError(t, err)

	// Wait for job to actually be running
	waitForJobCount(t, deps.DB, string(shared.StatusRunning), 1, 3*time.Second)

	// Stop manager with reasonable timeout
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = manager.Stop(stopCtx)
	require.NoError(t, err, "manager should stop gracefully")

	// Job should have completed
	assertJobCount(t, deps.DB, string(shared.StatusSucceeded), 1)
}

// TestJobManager_JobCancellationOnShutdown verifies jobs are interrupted during shutdown
func TestJobManager_JobCancellationOnShutdown(t *testing.T) {
	cfg := testJobConfig()

	// Handler that checks if context is canceled
	contextCanceled := false
	handlers := map[string]ports.JobHandler{
		"long-job": func(ctx context.Context, job *domain.Job) error {
			// Sleep for a long time, but check context
			select {
			case <-time.After(60 * time.Second):
				return nil
			case <-ctx.Done():
				contextCanceled = true
				return ctx.Err()
			}
		},
	}
	manager, deps := setupTestJobManager(t, cfg, handlers)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)

	// Enqueue a long-running job
	ctx := context.Background()
	_, err = deps.EnqueueJobUseCase.Execute(ctx, usecasejob.EnqueueJobInput{
		Type: "long-job",
	})
	require.NoError(t, err)

	// Wait for job to start running
	waitForJobCount(t, deps.DB, string(shared.StatusRunning), 1, 3*time.Second)

	// Stop manager - this should cancel the context
	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = manager.Stop(stopCtx)
	require.NoError(t, err)

	// Verify the handler detected context cancellation
	assert.True(t, contextCanceled, "job handler should detect context cancellation")

	// Note: The job may be in 'running' or 'dead' state depending on timing
	// The important thing is that context was canceled
}
