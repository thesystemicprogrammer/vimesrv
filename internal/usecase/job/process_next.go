package job

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

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
	return &ProcessNextJobUseCase{
		jobRepository:   jobRepository,
		handlerResolver: handlerResolver,
		backoffStrategy: backoffStrategy,
		clock:           clock,
	}
}

// retryStateTransition attempts a critical state transition operation with retries.
// If the provided context is canceled during state transition, it will use context.Background()
// to ensure the job's final state is persisted to the database.
func (uc *ProcessNextJobUseCase) retryStateTransition(ctx context.Context, operation func(context.Context) error, jobID int64, opName string) error {
	maxRetries := 3
	delays := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}

	var lastErr error
	activeCtx := ctx

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := operation(activeCtx)
		if err == nil {
			if attempt > 0 {
				logger.Info().Int64("jobID", jobID).Str("operation", opName).Int("attempt", attempt+1).Msg("state transition succeeded after retry")
			}
			return nil
		}

		lastErr = err

		// If error is due to context cancellation and we haven't switched contexts yet,
		// switch to background context for remaining attempts to ensure state persistence
		if ctx.Err() != nil && activeCtx == ctx {
			logger.Warn().Int64("jobID", jobID).Str("operation", opName).Msg("original context canceled, switching to background context for state persistence")
			activeCtx = context.Background()
		}

		logger.Error().Err(err).Int64("jobID", jobID).Str("operation", opName).Int("attempt", attempt+1).Msg("state transition failed, will retry")

		if attempt < maxRetries-1 {
			select {
			case <-time.After(delays[attempt]):
			case <-activeCtx.Done():
				// Context canceled again (only possible if background context somehow fails)
				return activeCtx.Err()
			}
		}
	}

	logger.Error().Err(lastErr).Int64("jobID", jobID).Str("operation", opName).Msg("CRITICAL: state transition failed after all retries - job may be stuck")
	return lastErr
}

func (uc *ProcessNextJobUseCase) Execute(ctx context.Context, workerID string) (found bool, err error) {
	job, ok, claimErr := uc.jobRepository.ClaimNextJobDue(ctx, workerID)
	if claimErr != nil {
		logger.Error().Err(claimErr).Msg("error claiming next due job")
		return false, claimErr
	}
	if !ok {
		return false, nil
	}

	// Panic safety
	defer func() {
		if r := recover(); r != nil {
			_ = uc.retryStateTransition(ctx, func(retryCtx context.Context) error {
				return uc.jobRepository.MarkDead(retryCtx, job.ID, fmt.Sprintf("panic: %v", r))
			}, job.ID, "MarkDead")
			logger.Error().Int64("ID", job.ID).Any("recovery", r).Str("stack", string(debug.Stack())).Msg("panic recovered in job handler")
			// Set return values after panic recovery
			found = true
			err = nil
		}
	}()
	// TODO: Rework the Error handling here. Catch SQL errors correctly
	handler, ok := uc.handlerResolver.Get(job.Type)
	if !ok {
		_ = uc.retryStateTransition(ctx, func(retryCtx context.Context) error {
			return uc.jobRepository.MarkDead(retryCtx, job.ID, "no handler registered")
		}, job.ID, "MarkDead")
		logger.Error().Int64("ID", job.ID).Str("type", job.Type).Msg("no handler registered for job")
		return true, nil
	}

	start := uc.clock.Now()
	handlerErr := handler(ctx, job)
	duration := uc.clock.Now().Sub(start) // Use clock for end time to support MockClock

	if handlerErr == nil {
		_ = uc.retryStateTransition(ctx, func(retryCtx context.Context) error {
			return uc.jobRepository.MarkSuccess(retryCtx, job.ID)
		}, job.ID, "MarkSuccess")
		logger.Info().Int64("ID", job.ID).Str("type", job.Type).Dur("duration", duration).Int("attempts", job.Attempts).Msg("job successfully executed")
		return true, nil
	}

	if job.Attempts >= job.MaxAttempts {
		_ = uc.retryStateTransition(ctx, func(retryCtx context.Context) error {
			return uc.jobRepository.MarkDead(retryCtx, job.ID, handlerErr.Error())
		}, job.ID, "MarkDead")
		logger.Error().Int64("ID", job.ID).Int("attempts", job.Attempts).Int("maxAttempts", job.MaxAttempts).Msg("job exceeded max attempts")
		return true, nil
	}

	delay := uc.backoffStrategy.NextDelay(job.Attempts)
	next := uc.clock.Now().Add(delay)
	_ = uc.retryStateTransition(ctx, func(retryCtx context.Context) error {
		return uc.jobRepository.Reschedule(retryCtx, job.ID, next, handlerErr.Error())
	}, job.ID, "Reschedule")
	logger.Error().Err(handlerErr).Int64("ID", job.ID).Int("attempts", job.Attempts).Int("maxAttempts", job.MaxAttempts).Dur("delay", delay).Msg("job failed, rescheduling for retry")
	return true, nil
}
