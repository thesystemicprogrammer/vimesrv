package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/library"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type UseCases struct {
	EnqueueJobUseCase       *job.EnqueueJobUseCase
	ProcessNextJobUseCase   *job.ProcessNextJobUseCase
	RecoverStuckJobsUseCase *job.RecoverStuckJobsUseCase
	SchedulerTickUseCase    *job.SchedulerTickUseCase
	UpsertScheduleUseCase   *job.UpsertScheduleUseCase
	ScanLibraryUseCase      *library.ScanLibraryUseCase
}

func initUseCases(config *config.Config, adapters *Adapters) *UseCases {
	return &UseCases{
		EnqueueJobUseCase:       job.NewEnqueueJobUseCase(config.Job, adapters.JobRepository, ports.RealClock{}),
		ProcessNextJobUseCase:   job.NewProcessNextJobUseCase(adapters.JobRepository, adapters.HandlerRegistry, adapters.BackoffStrategy, ports.RealClock{}),
		RecoverStuckJobsUseCase: job.NewRecoverStuckJobsUseCase(config.Job, adapters.JobRepository, ports.RealClock{}),
		SchedulerTickUseCase:    job.NewSchedulerTickUseCase(config.Job, adapters.ScheduleRepository, adapters.CronParser, ports.RealClock{}),
		UpsertScheduleUseCase:   job.NewUpsertScheduleUseCase(config.Job, adapters.ScheduleRepository, adapters.CronParser, ports.RealClock{}),
		ScanLibraryUseCase:      library.NewScanLibraryUseCase(config.Media, adapters.ScanLibraryRepository),
	}
}
