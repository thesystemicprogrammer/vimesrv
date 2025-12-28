package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type UpsertScheduleInput struct {
	Name     string
	CronSpec string
	JobType  string
	Payload  any
	Priority int
	Enabled  bool
}

type UpsertScheduleUseCase struct {
	config             config.JobConfig
	scheduleRepository ports.ScheduleRepository
	cronParser         ports.CronParser
	clock              ports.Clock
}

func NewUpsertScheduleUseCase(config config.JobConfig, scheduleRepository ports.ScheduleRepository, cronParser ports.CronParser, clock ports.Clock) *UpsertScheduleUseCase {
	upsertScheduleUseCase := &UpsertScheduleUseCase{
		config:             config,
		scheduleRepository: scheduleRepository,
		cronParser:         cronParser,
		clock:              clock,
	}

	return upsertScheduleUseCase
}

func (uc *UpsertScheduleUseCase) Execute(ctx context.Context, upsertScheduleInput UpsertScheduleInput) (int64, error) {
	if upsertScheduleInput.Name == "" {
		return 0, fmt.Errorf("schedule name required")
	}
	if upsertScheduleInput.JobType == "" {
		return 0, fmt.Errorf("job type required")
	}
	if upsertScheduleInput.CronSpec == "" {
		return 0, fmt.Errorf("cron specification required")
	}
	// Validate cron.
	if _, err := uc.cronParser.Parse(upsertScheduleInput.CronSpec); err != nil {
		return 0, fmt.Errorf("invalid cron specification: %w", err)
	}

	var payload json.RawMessage
	if upsertScheduleInput.Payload != nil {
		b, err := json.Marshal(upsertScheduleInput.Payload)
		if err != nil {
			return 0, fmt.Errorf("marshal payload: %w", err)
		}
		payload = b
	}

	schedule := &domain.Schedule{
		Name:        upsertScheduleInput.Name,
		CronSpec:    upsertScheduleInput.CronSpec,
		JobType:     upsertScheduleInput.JobType,
		Payload:     payload,
		Priority:    upsertScheduleInput.Priority,
		MaxAttempts: uc.config.MaxAttempts,
		Enabled:     upsertScheduleInput.Enabled,
	}
	id, err := uc.scheduleRepository.Upsert(ctx, schedule)
	if err != nil {
		logger.Error().Err(err).Msg("error scheduling job")
		return 0, err
	}

	saved, err := uc.scheduleRepository.GetByName(ctx, upsertScheduleInput.Name)
	if err != nil {
		logger.Error().Err(err).Msg("error get job by name")
		return 0, err
	}
	if !saved.NextRunAt.Valid {
		now := uc.clock.Now()
		parsed, _ := uc.cronParser.Parse(saved.CronSpec)
		next := parsed.Next(now)
		if err := uc.scheduleRepository.SetNextRunIfNull(ctx, saved.ID, next); err != nil {
			logger.Error().Err(err).Msg("error setting next run")
			return 0, err
		}
	}

	return id, nil
}
