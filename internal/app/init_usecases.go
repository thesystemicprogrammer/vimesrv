package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/library"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/media"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/metadata"
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
	// Metadata enrichment use cases
	EnrichMediaFileUseCase *metadata.EnrichMediaFileUseCase
	GetCandidatesUseCase   *metadata.GetCandidatesUseCase
	LinkMetadataUseCase    *metadata.LinkMetadataUseCase
	SearchMetadataUseCase  *metadata.SearchMetadataUseCase
	LinkFromSearchUseCase  *metadata.LinkFromSearchUseCase
	SkipEnrichmentUseCase  *metadata.SkipEnrichmentUseCase
	ResetEnrichmentUseCase *metadata.ResetEnrichmentUseCase
}

func initUseCases(cfg *config.Config, adapters *Adapters) *UseCases {
	// Initialize CreateTranscodeJobsUseCase first
	createTranscodeJobsUseCase := transcode.NewCreateTranscodeJobsUseCase(
		adapters.MediaRepository,
		adapters.TranscodeRepository,
		adapters.JobRepository,
		adapters.AudioStreamRepository,
		adapters.SubtitleStreamRepository,
		adapters.FFProbeService,
		cfg,
	)

	// Initialize EnrichMediaFileUseCase if TMDB is enabled
	var enrichMediaFileUseCase *metadata.EnrichMediaFileUseCase
	var getCandidatesUseCase *metadata.GetCandidatesUseCase
	var linkMetadataUseCase *metadata.LinkMetadataUseCase
	var searchMetadataUseCase *metadata.SearchMetadataUseCase
	var linkFromSearchUseCase *metadata.LinkFromSearchUseCase
	var skipEnrichmentUseCase *metadata.SkipEnrichmentUseCase
	var resetEnrichmentUseCase *metadata.ResetEnrichmentUseCase

	if cfg.TMDB.Enabled {
		enrichMediaFileUseCase = metadata.NewEnrichMediaFileUseCase(
			cfg.TMDB,
			adapters.FilenameParser,
			adapters.TMDBClient,
			adapters.ImageDownloader,
			adapters.MediaRepository,
			adapters.MovieMetadataRepository,
			adapters.SeriesMetadataRepository,
			adapters.SeasonMetadataRepository,
			adapters.EpisodeMetadataRepository,
			adapters.MetadataCandidateRepository,
		)

		getCandidatesUseCase = metadata.NewGetCandidatesUseCase(
			cfg.TMDB,
			adapters.TMDBClient,
			adapters.MediaRepository,
			adapters.MetadataCandidateRepository,
		)

		linkMetadataUseCase = metadata.NewLinkMetadataUseCase(
			cfg.TMDB,
			adapters.TMDBClient,
			adapters.ImageDownloader,
			adapters.MediaRepository,
			adapters.MovieMetadataRepository,
			adapters.SeriesMetadataRepository,
			adapters.SeasonMetadataRepository,
			adapters.EpisodeMetadataRepository,
			adapters.MetadataCandidateRepository,
		)

		searchMetadataUseCase = metadata.NewSearchMetadataUseCase(
			cfg.TMDB,
			adapters.TMDBClient,
		)

		linkFromSearchUseCase = metadata.NewLinkFromSearchUseCase(
			cfg.TMDB,
			adapters.TMDBClient,
			adapters.ImageDownloader,
			adapters.MediaRepository,
			adapters.MovieMetadataRepository,
			adapters.SeriesMetadataRepository,
			adapters.SeasonMetadataRepository,
			adapters.EpisodeMetadataRepository,
			adapters.MetadataCandidateRepository,
		)

		skipEnrichmentUseCase = metadata.NewSkipEnrichmentUseCase(
			adapters.MediaRepository,
			adapters.MetadataCandidateRepository,
		)

		resetEnrichmentUseCase = metadata.NewResetEnrichmentUseCase(
			adapters.MediaRepository,
			adapters.MetadataCandidateRepository,
		)
	}

	scanLibraryUseCase := library.NewScanLibraryUseCase(
		cfg.Media,
		adapters.FileHasher,
		adapters.FFProbeService,
		adapters.FileSystemService,
		adapters.MediaRepository,
		createTranscodeJobsUseCase,
	)

	// Enable enrichment if TMDB is configured
	if cfg.TMDB.Enabled {
		scanLibraryUseCase.WithEnrichment(cfg.TMDB, adapters.JobRepository)
	}

	return &UseCases{
		EnqueueJobUseCase:          job.NewEnqueueJobUseCase(cfg.Job, adapters.JobRepository, ports.RealClock{}),
		ProcessNextJobUseCase:      job.NewProcessNextJobUseCase(adapters.JobRepository, adapters.HandlerRegistry, adapters.BackoffStrategy, ports.RealClock{}),
		RecoverStuckJobsUseCase:    job.NewRecoverStuckJobsUseCase(cfg.Job, adapters.JobRepository, ports.RealClock{}),
		SchedulerTickUseCase:       job.NewSchedulerTickUseCase(cfg.Job, adapters.ScheduleRepository, adapters.CronParser, ports.RealClock{}),
		UpsertScheduleUseCase:      job.NewUpsertScheduleUseCase(cfg.Job, adapters.ScheduleRepository, adapters.CronParser, ports.RealClock{}),
		ScanLibraryUseCase:         scanLibraryUseCase,
		CreateTranscodeJobsUseCase: createTranscodeJobsUseCase,
		ProcessTranscodeUseCase: transcode.NewProcessTranscodeUseCase(
			adapters.TranscodeRepository,
			adapters.MediaRepository,
			adapters.Transcoder,
			adapters.FileSystemService,
			cfg,
		),
		GetMediaUseCase: media.NewGetMediaUseCase(
			adapters.MediaRepository,
			adapters.TranscodeRepository,
			adapters.AudioStreamRepository,
			adapters.SubtitleStreamRepository,
		),
		ListMediaUseCase:       media.NewListMediaUseCase(adapters.MediaRepository),
		EnrichMediaFileUseCase: enrichMediaFileUseCase,
		GetCandidatesUseCase:   getCandidatesUseCase,
		LinkMetadataUseCase:    linkMetadataUseCase,
		SearchMetadataUseCase:  searchMetadataUseCase,
		LinkFromSearchUseCase:  linkFromSearchUseCase,
		SkipEnrichmentUseCase:  skipEnrichmentUseCase,
		ResetEnrichmentUseCase: resetEnrichmentUseCase,
	}
}
