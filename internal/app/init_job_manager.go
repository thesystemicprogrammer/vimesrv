package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

func initializeJobManager(cfg *config.Config, adapters *Adapters, useCases *UseCases) (*job.JobManager, error) {
	logger.Info().Msg("initializing job manager")

	jobManager := job.NewJobManager(job.JobManagerInput{
		Config:                  cfg.Job,
		ProcessNextJobUseCase:   useCases.ProcessNextJobUseCase,
		SchedulerTickUseCase:    useCases.SchedulerTickUseCase,
		RecoverStuckJobsUseCase: useCases.RecoverStuckJobsUseCase,
		JobRepository:           adapters.JobRepository,
		ScheduleRepository:      adapters.ScheduleRepository,
		Handlers:                adapters.HandlerRegistry,
		BackoffStrategy:         adapters.BackoffStrategy,
		Cron:                    adapters.CronParser,
		Clock:                   ports.RealClock{},
	})

	logger.Debug().Msg("job manager initialized successfully")
	return jobManager, nil
}
