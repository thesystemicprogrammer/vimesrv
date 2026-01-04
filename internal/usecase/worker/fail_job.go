package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/worker"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// FailJobInput contains the input for failing a worker job
type FailJobInput struct {
	JobID    int64
	WorkerID string
	Error    string
	Retry    bool // Whether to retry the job
}

// FailWorkerJobUseCase handles marking a worker job as failed
type FailWorkerJobUseCase struct {
	jobRepo        ports.JobRepository
	transcodeRepo  ports.TranscodeRepository
	workerRegistry *worker.Registry
	jobNotifier    ports.JobNotifier
	backoff        ports.BackoffStrategy
}

// NewFailWorkerJobUseCase creates a new FailWorkerJobUseCase
func NewFailWorkerJobUseCase(
	jobRepo ports.JobRepository,
	transcodeRepo ports.TranscodeRepository,
	workerRegistry *worker.Registry,
	jobNotifier ports.JobNotifier,
	backoff ports.BackoffStrategy,
) *FailWorkerJobUseCase {
	return &FailWorkerJobUseCase{
		jobRepo:        jobRepo,
		transcodeRepo:  transcodeRepo,
		workerRegistry: workerRegistry,
		jobNotifier:    jobNotifier,
		backoff:        backoff,
	}
}

// Execute marks a job as failed, optionally rescheduling for retry
func (uc *FailWorkerJobUseCase) Execute(ctx context.Context, input FailJobInput) error {
	// 1. Touch worker timestamp
	uc.workerRegistry.Touch(input.WorkerID)

	// 2. Get job and verify ownership
	job, err := uc.jobRepo.Get(ctx, input.JobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}
	if !job.WorkerID.Valid || job.WorkerID.String != input.WorkerID {
		return fmt.Errorf("job not owned by worker %s", input.WorkerID)
	}

	// 3. Parse payload to get transcode ID
	var payload TranscodeJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		logger.Warn().Err(err).Int64("job_id", job.ID).Msg("failed to parse job payload")
	}

	// 4. Check if we should retry
	if input.Retry && job.Attempts < job.MaxAttempts {
		// Reschedule with backoff
		delay := uc.backoff.NextDelay(job.Attempts)
		nextRun := time.Now().Add(delay)

		if err := uc.jobRepo.Reschedule(ctx, job.ID, nextRun, input.Error); err != nil {
			return fmt.Errorf("failed to reschedule job: %w", err)
		}

		// Mark transcode as pending (back to queue)
		if payload.TranscodeID != "" {
			// Don't update transcode status - it stays as "processing"
			// The next worker will pick it up
		}

		logger.Info().
			Int64("job_id", job.ID).
			Str("worker_id", input.WorkerID).
			Int("attempt", job.Attempts).
			Int("max_attempts", job.MaxAttempts).
			Str("error", input.Error).
			Time("next_run", nextRun).
			Msg("Worker job rescheduled for retry")

		uc.jobNotifier.NotifyJobRetrying(job, job.Attempts, job.MaxAttempts)
	} else {
		// Mark job as dead
		if err := uc.jobRepo.MarkDead(ctx, job.ID, input.Error); err != nil {
			return fmt.Errorf("failed to mark job as dead: %w", err)
		}

		// Mark transcode as failed
		if payload.TranscodeID != "" {
			if err := uc.transcodeRepo.MarkFailed(ctx, payload.TranscodeID); err != nil {
				logger.Warn().Err(err).Str("transcode_id", payload.TranscodeID).Msg("failed to mark transcode as failed")
			}
		}

		logger.Error().
			Int64("job_id", job.ID).
			Str("worker_id", input.WorkerID).
			Int("attempts", job.Attempts).
			Str("error", input.Error).
			Msg("Worker job failed permanently")

		uc.jobNotifier.NotifyJobFailed(job, input.Error)
	}

	// 5. Decrement worker's active job count
	uc.workerRegistry.DecrementActiveJobs(input.WorkerID)

	return nil
}
