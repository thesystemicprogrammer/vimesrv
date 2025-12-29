package job

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type ProcessNextJobUseCase struct {
	jobRepository   ports.JobRepository
	handlerResolver ports.HandlerResolver
	backoffStrategy ports.BackoffStrategy
	clock           ports.Clock
}

func NewProcessNextJobUseCase(jobRepository ports.JobRepository, handlerResolver ports.HandlerResolver, backoffStrategy ports.BackoffStrategy, clock ports.Clock) *ProcessNextJobUseCase {
	processNextJobUseCase := &ProcessNextJobUseCase{
		jobRepository:   jobRepository,
		handlerResolver: handlerResolver,
		backoffStrategy: backoffStrategy,
		clock:           clock,
	}
	return processNextJobUseCase
}

func (uc *ProcessNextJobUseCase) Execute(ctx context.Context, workerID string) (bool, error) {
	job, ok, err := uc.jobRepository.ClaimNextJobDue(ctx, workerID)
	if err != nil {
		logger.Error().Err(err).Msg("error claiming next due job")
		return false, err
	}
	if !ok {
		return false, nil
	}

	// Panic safety
	defer func() {
		if r := recover(); r != nil {
			_ = uc.jobRepository.MarkDead(ctx, job.ID, fmt.Sprintf("panic: %v", r))
			logger.Error().Int64("ID", job.ID).Any("Recovery", r).Msg("panic recovery failed")
		}
	}()
	// TODO: Rework the Error handling here. Catch SQL errors correctly
	handler, ok := uc.handlerResolver.Get(job.Type)
	if !ok {
		_ = uc.jobRepository.MarkDead(ctx, job.ID, "no handler registered")
		logger.Error().Int64("ID", job.ID).Str("type", job.Type).Msg("no handler registered for job")
		return true, nil
	}

	start := uc.clock.Now()
	err = handler(ctx, job)
	duration := uc.clock.Now().Sub(start) // Use clock for end time to support MockClock

	if err == nil {
		_ = uc.jobRepository.MarkSuccess(ctx, job.ID)
		logger.Info().Int64("ID", job.ID).Str("type", job.Type).Dur("duration", duration).Int("attempts", job.Attempts).Msg("job successfully executed")
		return true, nil
	}

	if job.Attempts >= job.MaxAttempts {
		_ = uc.jobRepository.MarkDead(ctx, job.ID, err.Error())
		logger.Error().Int64("ID", job.ID).Int("attempts", job.Attempts).Int("maxAttempts", job.MaxAttempts).Msg("job exceeded max attempts")
		return true, nil
	}

	delay := uc.backoffStrategy.NextDelay(job.Attempts)
	next := uc.clock.Now().Add(delay)
	_ = uc.jobRepository.Reschedule(ctx, job.ID, next, err.Error())
	logger.Error().Err(err).Int64("ID", job.ID).Int("attempts", job.Attempts).Int("maxAttempts", job.MaxAttempts).Dur("delay", delay).Msg("job exceeded max attempts")
	return true, nil
}
