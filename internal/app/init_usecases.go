package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/library"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/media"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/transcode"
)

type UseCases struct {
	EnqueueJobUseCase          *job.EnqueueJobUseCase
	ProcessNextJobUseCase      *job.ProcessNextJobUseCase
	RecoverStuckJobsUseCase    *job.RecoverStuckJobsUseCase
	SchedulerTickUseCase       *job.SchedulerTickUseCase
	UpsertScheduleUseCase      *job.UpsertScheduleUseCase
	ScanLibraryUseCase         *library.ScanLibraryUseCase
	CreateTranscodeJobsUseCase *transcode.CreateTranscodeJobsUseCase
	ProcessTranscodeUseCase    *transcode.ProcessTranscodeUseCase
	GetMediaUseCase            *media.GetMediaUseCase
	ListMediaUseCase           *media.ListMediaUseCase
}

func initUseCases(config *config.Config, adapters *Adapters) *UseCases {
	// Initialize CreateTranscodeJobsUseCase first
	createTranscodeJobsUseCase := transcode.NewCreateTranscodeJobsUseCase(
		adapters.MediaRepository,
		adapters.TranscodeRepository,
		adapters.JobRepository,
		adapters.AudioStreamRepository,
		adapters.SubtitleStreamRepository,
		adapters.FFProbeService,
		config,
	)

	return &UseCases{
		EnqueueJobUseCase:       job.NewEnqueueJobUseCase(config.Job, adapters.JobRepository, ports.RealClock{}),
		ProcessNextJobUseCase:   job.NewProcessNextJobUseCase(adapters.JobRepository, adapters.HandlerRegistry, adapters.BackoffStrategy, ports.RealClock{}),
		RecoverStuckJobsUseCase: job.NewRecoverStuckJobsUseCase(config.Job, adapters.JobRepository, ports.RealClock{}),
		SchedulerTickUseCase:    job.NewSchedulerTickUseCase(config.Job, adapters.ScheduleRepository, adapters.CronParser, ports.RealClock{}),
		UpsertScheduleUseCase:   job.NewUpsertScheduleUseCase(config.Job, adapters.ScheduleRepository, adapters.CronParser, ports.RealClock{}),
		ScanLibraryUseCase: library.NewScanLibraryUseCase(
			config.Media,
			adapters.FileHasher,
			adapters.FFProbeService,
			adapters.FileSystemService,
			adapters.MediaRepository,
			createTranscodeJobsUseCase,
		),
		CreateTranscodeJobsUseCase: createTranscodeJobsUseCase,
		ProcessTranscodeUseCase: transcode.NewProcessTranscodeUseCase(
			adapters.TranscodeRepository,
			adapters.MediaRepository,
			adapters.Transcoder,
			adapters.FileSystemService,
			config,
		),
		GetMediaUseCase: media.NewGetMediaUseCase(
			adapters.MediaRepository,
			adapters.TranscodeRepository,
			adapters.AudioStreamRepository,
			adapters.SubtitleStreamRepository,
		),
		ListMediaUseCase: media.NewListMediaUseCase(adapters.MediaRepository),
	}
}
