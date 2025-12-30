package job

import (
	"context"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type RecoverStuckJobsUseCase struct {
	config        config.JobConfig
	jobRepository ports.JobRepository
	clock         ports.Clock
}

func NewRecoverStuckJobsUseCase(config config.JobConfig, jobRepository ports.JobRepository, clock ports.Clock) *RecoverStuckJobsUseCase {
	return &RecoverStuckJobsUseCase{
		config:        config,
		jobRepository: jobRepository,
		clock:         clock,
	}
}

func (uc *RecoverStuckJobsUseCase) Execute(ctx context.Context) error {
	threshold := time.Duration(uc.config.StuckJobThresholdMinutes) * time.Minute

	stuckJobs, err := uc.jobRepository.FindStuckJobs(ctx, threshold)
	if err != nil {
		logger.Error().Err(err).Msg("error finding stuck jobs")
		return err
	}

	if len(stuckJobs) == 0 {
		return nil
	}

	logger.Info().Int("count", len(stuckJobs)).Msg("found stuck jobs to recover")

	for _, job := range stuckJobs {
		runningFor := uc.clock.Now().Sub(job.StartedAt.Time)

		logger.Warn().
			Int64("jobID", job.ID).
			Str("type", job.Type).
			Str("workerID", job.WorkerID.String).
			Dur("runningFor", runningFor).
			Int("attempts", job.Attempts).
			Int("maxAttempts", job.MaxAttempts).
			Msg("recovering stuck job - worker likely crashed")

		err := uc.jobRepository.ResetStuckJob(ctx, job.ID)
		if err != nil {
			logger.Error().Err(err).Int64("jobID", job.ID).Msg("failed to reset stuck job")
			// Continue with other stuck jobs
			continue
		}

		logger.Info().Int64("jobID", job.ID).Msg("successfully reset stuck job to queued status")
	}

	return nil
}
