package usecase

import (
	"context"
	"database/sql"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type SchedulerTickUseCase struct {
	config             config.JobConfig
	scheduleRepository ports.ScheduleRepository
	cronParser         ports.CronParser
	clock              ports.Clock
}

func NewSchedulerTickUseCase(config config.JobConfig, scheduleRepository ports.ScheduleRepository, cronParser ports.CronParser, clock ports.Clock) *SchedulerTickUseCase {
	schedulerTickUseCase := &SchedulerTickUseCase{
		config:             config,
		scheduleRepository: scheduleRepository,
		cronParser:         cronParser,
		clock:              clock,
	}

	return schedulerTickUseCase
}

func (uc *SchedulerTickUseCase) Execute(ctx context.Context) error {
	due, err := uc.scheduleRepository.ListDue(ctx, uc.config.SchedulerBatch)
	if err != nil {
		logger.Error().Err(err).Msg("error list due jobs")
		return err
	}

	now := uc.clock.Now()

	for _, scheduled := range due {
		cronSpec, err := uc.cronParser.Parse(scheduled.CronSpec)
		if err != nil {
			logger.Error().Err(err).Int64("Id", scheduled.ID).Str("name", scheduled.Name).Msg("scheduler: invalid cron for schedule")
			continue
		}
		next := cronSpec.Next(now)

		job := &domain.Job{
			Type:        scheduled.JobType,
			Payload:     scheduled.Payload,
			Status:      shared.StatusQueued,
			Priority:    scheduled.Priority,
			RunAt:       now,
			Attempts:    0,
			MaxAttempts: uc.config.MaxAttempts,
			ScheduledID: sql.NullInt64{Valid: true, Int64: scheduled.ID},
		}
		err = uc.scheduleRepository.AdvanceAndEnqueue(ctx, scheduled.ID, next, job)
		if err != nil {
			logger.Error().Err(err).Int64("ID", scheduled.ID).Msg("scheduler: advance/enqueue schedule %d: %v")
			return err
		}
	}
	return nil
}
