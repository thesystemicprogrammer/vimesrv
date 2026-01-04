package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/library"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/media"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/metadata"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/metadata/linker"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/transcode"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/user"
	workeruc "github.com/thesystemicprogrammer/vimesrv/internal/usecase/worker"
)

type UseCases struct {
	EnqueueJobUseCase          *job.EnqueueJobUseCase
	ProcessNextJobUseCase      *job.ProcessNextJobUseCase
	RecoverStuckJobsUseCase    *job.RecoverStuckJobsUseCase
	SchedulerTickUseCase       *job.SchedulerTickUseCase
	UpsertScheduleUseCase      *job.UpsertScheduleUseCase
	ListJobsUseCase            *job.ListJobsUseCase
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
	// Library browsing use cases
	ListMoviesUseCase         *library.ListMoviesUseCase
	GetMovieUseCase           *library.GetMovieUseCase
	GetMovieCreditsUseCase    *library.GetMovieCreditsUseCase
	ListSeriesUseCase         *library.ListSeriesUseCase
	GetSeriesUseCase          *library.GetSeriesUseCase
	GetSeriesCreditsUseCase   *library.GetSeriesCreditsUseCase
	ListRecentUseCase         *library.ListRecentUseCase
	ListUnmatchedUseCase      *library.ListUnmatchedUseCase
	GetSimilarMoviesUseCase   *library.GetSimilarMoviesUseCase
	GetSimilarSeriesUseCase   *library.GetSimilarSeriesUseCase
	GetMovieCollectionUseCase *library.GetMovieCollectionUseCase
	ListGenresUseCase         *library.ListGenresUseCase
	SearchLibraryUseCase      *library.SearchLibraryUseCase
	// User management use cases
	CreateUserUseCase     *user.CreateUserUseCase
	ListUsersUseCase      *user.ListUsersUseCase
	GetUserUseCase        *user.GetUserUseCase
	UpdateUserUseCase     *user.UpdateUserUseCase
	DeleteUserUseCase     *user.DeleteUserUseCase
	ResetPasswordUseCase  *user.ResetPasswordUseCase
	ChangePasswordUseCase *user.ChangePasswordUseCase
	// Worker use cases (for distributed transcoding)
	RegisterWorkerUseCase    *workeruc.RegisterWorkerUseCase
	HeartbeatUseCase         *workeruc.HeartbeatUseCase
	ClaimJobForWorkerUseCase *workeruc.ClaimJobForWorkerUseCase
	CompleteWorkerJobUseCase *workeruc.CompleteWorkerJobUseCase
	FailWorkerJobUseCase     *workeruc.FailWorkerJobUseCase
	ReportProgressUseCase    *workeruc.ReportProgressUseCase
}

func initUseCases(cfg *config.Config, adapters *Adapters) *UseCases {
	// Initialize EnqueueJobUseCase early as it's needed by multiple use cases
	enqueueJobUseCase := job.NewEnqueueJobUseCase(cfg.Job, adapters.JobRepository, ports.RealClock{}, adapters.JobNotifier)

	// Initialize CreateTranscodeJobsUseCase first
	createTranscodeJobsUseCase := transcode.NewCreateTranscodeJobsUseCase(
		adapters.MediaRepository,
		adapters.TranscodeRepository,
		enqueueJobUseCase,
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
		// Create shared linkers for movie and episode metadata
		movieLinker := linker.NewMovieLinker(
			cfg.TMDB,
			adapters.TMDBClient,
			adapters.ImageDownloader,
			adapters.MovieMetadataRepository,
			adapters.MovieCreditRepository,
			adapters.MovieCertificationRepository,
			adapters.SearchRepository,
		)

		episodeLinker := linker.NewEpisodeLinker(
			cfg.TMDB,
			adapters.TMDBClient,
			adapters.ImageDownloader,
			adapters.SeriesMetadataRepository,
			adapters.SeasonMetadataRepository,
			adapters.EpisodeMetadataRepository,
			adapters.SeriesCreditRepository,
			adapters.SearchRepository,
		)

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
			adapters.MovieCreditRepository,
			adapters.MovieCertificationRepository,
			adapters.SearchRepository,
		)

		getCandidatesUseCase = metadata.NewGetCandidatesUseCase(
			cfg.TMDB,
			adapters.TMDBClient,
			adapters.MediaRepository,
			adapters.MetadataCandidateRepository,
		)

		linkMetadataUseCase = metadata.NewLinkMetadataUseCase(
			movieLinker,
			episodeLinker,
			adapters.MediaRepository,
			adapters.MetadataCandidateRepository,
			adapters.SearchRepository,
			adapters.MovieCreditRepository,
			adapters.SeriesCreditRepository,
		)

		searchMetadataUseCase = metadata.NewSearchMetadataUseCase(
			cfg.TMDB,
			adapters.TMDBClient,
		)

		linkFromSearchUseCase = metadata.NewLinkFromSearchUseCase(
			movieLinker,
			episodeLinker,
			adapters.MediaRepository,
			adapters.MetadataCandidateRepository,
			adapters.SearchRepository,
			adapters.MovieCreditRepository,
			adapters.SeriesCreditRepository,
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
		adapters.JobNotifier,
	)

	// Enable enrichment if TMDB is configured
	if cfg.TMDB.Enabled {
		scanLibraryUseCase.WithEnrichment(cfg.TMDB, enqueueJobUseCase)
	}

	// Initialize similar content use cases if TMDB is enabled
	var getSimilarMoviesUseCase *library.GetSimilarMoviesUseCase
	var getSimilarSeriesUseCase *library.GetSimilarSeriesUseCase
	var getMovieCollectionUseCase *library.GetMovieCollectionUseCase
	var getMovieCreditsUseCase *library.GetMovieCreditsUseCase
	var getSeriesCreditsUseCase *library.GetSeriesCreditsUseCase

	if cfg.TMDB.Enabled {
		getSimilarMoviesUseCase = library.NewGetSimilarMoviesUseCase(
			adapters.SimilarContentRepository,
			adapters.MovieMetadataRepository,
			adapters.LibraryRepository,
			adapters.TMDBClient,
			cfg.TMDB,
		)

		getSimilarSeriesUseCase = library.NewGetSimilarSeriesUseCase(
			adapters.SimilarContentRepository,
			adapters.SeriesMetadataRepository,
			adapters.LibraryRepository,
			adapters.TMDBClient,
			cfg.TMDB,
		)

		getMovieCollectionUseCase = library.NewGetMovieCollectionUseCase(
			adapters.CollectionRepository,
			adapters.MovieMetadataRepository,
			adapters.LibraryRepository,
			adapters.TMDBClient,
			cfg.TMDB,
		)

		getMovieCreditsUseCase = library.NewGetMovieCreditsUseCase(
			adapters.MovieMetadataRepository,
			adapters.MovieCreditRepository,
			adapters.TMDBClient,
		)

		getSeriesCreditsUseCase = library.NewGetSeriesCreditsUseCase(
			adapters.SeriesMetadataRepository,
			adapters.SeriesCreditRepository,
			adapters.TMDBClient,
		)
	}

	// Create base ProcessNextJobUseCase
	processNextJobUseCase := job.NewProcessNextJobUseCase(
		adapters.JobRepository,
		adapters.HandlerRegistry,
		adapters.BackoffStrategy,
		ports.RealClock{},
		adapters.JobNotifier,
	)

	// If worker mode is enabled, exclude transcode jobs from local processing
	// (they will be processed exclusively by distributed workers)
	if cfg.Worker.Enabled {
		processNextJobUseCase = processNextJobUseCase.WithExcludedTypes([]string{"transcode_video"})
	}

	useCases := &UseCases{
		EnqueueJobUseCase:          enqueueJobUseCase,
		ProcessNextJobUseCase:      processNextJobUseCase,
		RecoverStuckJobsUseCase:    job.NewRecoverStuckJobsUseCase(cfg.Job, adapters.JobRepository, ports.RealClock{}),
		SchedulerTickUseCase:       job.NewSchedulerTickUseCase(cfg.Job, adapters.ScheduleRepository, adapters.CronParser, ports.RealClock{}, adapters.JobNotifier),
		UpsertScheduleUseCase:      job.NewUpsertScheduleUseCase(cfg.Job, adapters.ScheduleRepository, adapters.CronParser, ports.RealClock{}),
		ListJobsUseCase:            job.NewListJobsUseCase(adapters.JobRepository),
		ScanLibraryUseCase:         scanLibraryUseCase,
		CreateTranscodeJobsUseCase: createTranscodeJobsUseCase,
		ProcessTranscodeUseCase: transcode.NewProcessTranscodeUseCase(
			adapters.TranscodeRepository,
			adapters.MediaRepository,
			adapters.Transcoder,
			adapters.FileSystemService,
			adapters.JobNotifier,
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
		// Library browsing use cases
		ListMoviesUseCase:         library.NewListMoviesUseCase(adapters.LibraryRepository),
		GetMovieUseCase:           library.NewGetMovieUseCase(adapters.LibraryRepository, getSimilarMoviesUseCase, getMovieCollectionUseCase, cfg.TMDB.MaxCastMembers),
		GetMovieCreditsUseCase:    getMovieCreditsUseCase,
		ListSeriesUseCase:         library.NewListSeriesUseCase(adapters.LibraryRepository),
		GetSeriesUseCase:          library.NewGetSeriesUseCase(adapters.LibraryRepository, getSimilarSeriesUseCase),
		GetSeriesCreditsUseCase:   getSeriesCreditsUseCase,
		ListRecentUseCase:         library.NewListRecentUseCase(adapters.LibraryRepository, cfg.Library),
		ListUnmatchedUseCase:      library.NewListUnmatchedUseCase(adapters.LibraryRepository),
		GetSimilarMoviesUseCase:   getSimilarMoviesUseCase,
		GetSimilarSeriesUseCase:   getSimilarSeriesUseCase,
		GetMovieCollectionUseCase: getMovieCollectionUseCase,
		ListGenresUseCase:         library.NewListGenresUseCase(adapters.LibraryRepository),
		SearchLibraryUseCase:      library.NewSearchLibraryUseCase(adapters.SearchRepository, adapters.LibraryRepository),
		// User management use cases
		CreateUserUseCase:     user.NewCreateUserUseCase(adapters.UserRepository),
		ListUsersUseCase:      user.NewListUsersUseCase(adapters.UserRepository),
		GetUserUseCase:        user.NewGetUserUseCase(adapters.UserRepository),
		UpdateUserUseCase:     user.NewUpdateUserUseCase(adapters.UserRepository),
		DeleteUserUseCase:     user.NewDeleteUserUseCase(adapters.UserRepository),
		ResetPasswordUseCase:  user.NewResetPasswordUseCase(adapters.UserRepository),
		ChangePasswordUseCase: user.NewChangePasswordUseCase(adapters.UserRepository),
	}

	// Initialize worker use cases if worker mode is enabled
	if cfg.Worker.Enabled && adapters.WorkerRegistry != nil {
		useCases.RegisterWorkerUseCase = workeruc.NewRegisterWorkerUseCase(adapters.WorkerRegistry)
		useCases.HeartbeatUseCase = workeruc.NewHeartbeatUseCase(adapters.WorkerRegistry, adapters.JobRepository)
		useCases.ClaimJobForWorkerUseCase = workeruc.NewClaimJobForWorkerUseCase(
			adapters.JobRepository,
			adapters.TranscodeRepository,
			adapters.MediaRepository,
			adapters.WorkerRegistry,
			adapters.JobNotifier,
			cfg,
		)
		useCases.CompleteWorkerJobUseCase = workeruc.NewCompleteWorkerJobUseCase(
			adapters.JobRepository,
			adapters.TranscodeRepository,
			adapters.WorkerRegistry,
			adapters.JobNotifier,
			adapters.Transcoder,
			adapters.FileSystemService,
		)
		useCases.FailWorkerJobUseCase = workeruc.NewFailWorkerJobUseCase(
			adapters.JobRepository,
			adapters.TranscodeRepository,
			adapters.WorkerRegistry,
			adapters.JobNotifier,
			adapters.BackoffStrategy,
		)
		useCases.ReportProgressUseCase = workeruc.NewReportProgressUseCase(
			adapters.JobRepository,
			adapters.WorkerRegistry,
			adapters.JobNotifier,
		)
	}

	return useCases
}
