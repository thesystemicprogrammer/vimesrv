package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type EnqueueJobInput struct {
	Type        string
	Payload     any
	RunAt       time.Time // if zero => now
	Priority    int
	MaxAttempts int // if <=0 => default provided by caller composition
}

type EnqueueJobUseCase struct {
	config        config.JobConfig
	jobRepository ports.JobRepository
	clock         ports.Clock
}

func NewEnqueueJobUseCase(config config.JobConfig, jobRepository ports.JobRepository, clock ports.Clock) *EnqueueJobUseCase {
	enqueueJobUseCase := &EnqueueJobUseCase{
		config:        config,
		jobRepository: jobRepository,
		clock:         clock,
	}

	return enqueueJobUseCase
}

func (uc *EnqueueJobUseCase) Execute(ctx context.Context, jobInput EnqueueJobInput) error {
	if jobInput.Type == "" {
		return fmt.Errorf("job type must not be empty")
	}

	var payload json.RawMessage
	if jobInput.Payload != nil {
		b, err := json.Marshal(jobInput.Payload)
		if err != nil {
			return fmt.Errorf("cannot marshalling json payload: %w", err)
		}
		payload = b
	}

	runAt := jobInput.RunAt
	if runAt.IsZero() {
		runAt = uc.clock.Now()
	}

	job := &domain.Job{
		Type:        jobInput.Type,
		Payload:     payload,
		Status:      shared.StatusQueued,
		Priority:    jobInput.Priority,
		RunAt:       runAt,
		Attempts:    0,
		MaxAttempts: uc.config.MaxAttempts,
	}

	_, err := uc.jobRepository.Enqueue(ctx, job)
	if err != nil {
		logger.Error().Err(err).Msg("error enqueuing job")
	}

	return err
}
