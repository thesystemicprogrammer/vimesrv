package app

import (
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/media"
	metadataAdapter "github.com/thesystemicprogrammer/vimesrv/internal/adapters/metadata"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/repository"
	workerAdapter "github.com/thesystemicprogrammer/vimesrv/internal/adapters/worker"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/websocket"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type Adapters struct {
	BackoffStrategy              ports.BackoffStrategy
	CronParser                   ports.CronParser
	HandlerRegistry              *job.HandlerRegistry
	JobRepository                ports.JobRepository
	ScheduleRepository           ports.ScheduleRepository
	SettingsRepository           ports.SettingsRepository
	WorkerConfigRepository       ports.WorkerConfigRepository
	FileHasher                   ports.FileHasher
	FFProbeService               ports.FFProbeService
	FileSystemService            ports.FileSystemService
	MediaRepository              ports.MediaRepository
	AudioStreamRepository        ports.AudioStreamRepository
	SubtitleStreamRepository     ports.SubtitleStreamRepository
	TranscodeRepository          ports.TranscodeRepository
	Transcoder                   ports.Transcoder
	FilenameParser               ports.FilenameParser
	TMDBClient                   ports.TMDBClient
	ImageDownloader              ports.ImageDownloader
	MovieMetadataRepository      ports.MovieMetadataRepository
	SeriesMetadataRepository     ports.SeriesMetadataRepository
	SeasonMetadataRepository     ports.SeasonMetadataRepository
	EpisodeMetadataRepository    ports.EpisodeMetadataRepository
	MetadataCandidateRepository  ports.MetadataCandidateRepository
	MovieCreditRepository        ports.MovieCreditRepository
	SeriesCreditRepository       ports.SeriesCreditRepository
	MovieCertificationRepository ports.MovieCertificationRepository
	SimilarContentRepository     ports.SimilarContentRepository
	CollectionRepository         ports.CollectionRepository
	LibraryRepository            ports.LibraryRepository
	SearchRepository             ports.SearchRepository
	UserRepository               ports.UserRepository
	RebuildRepository            ports.RebuildRepository
	WatchProgressRepository      ports.WatchProgressRepository
	FavoriteRepository           ports.FavoriteRepository
	RecommendationRepository     ports.RecommendationRepository
	FeatureExtractionRepository  ports.FeatureExtractionRepository
	UserWatchDataRepository      ports.UserWatchDataRepository
	WebSocketHub                 *websocket.Hub
	ProgressCache                *websocket.ProgressCache
	JobNotifier                  ports.JobNotifier
	WorkerRegistry               *workerAdapter.Registry
}

func initAdapters(cfg *config.Config, db *database.DB) *Adapters {
	adapters := &Adapters{
		CronParser:                   job.NewRobfigCronParser(),
		BackoffStrategy:              job.NewExponentialBackoff(cfg.Job.BackoffBaseSeconds, cfg.Job.BackoffMaxSeconds),
		HandlerRegistry:              job.NewHandlerRegistry(),
		JobRepository:                repository.NewJobRepository(db),
		ScheduleRepository:           repository.NewScheduleRepository(db),
		SettingsRepository:           repository.NewSettingsRepository(db),
		WorkerConfigRepository:       repository.NewWorkerConfigRepository(db),
		FileHasher:                   media.NewBlake2bHasher(),
		FFProbeService:               media.NewFFProbeAdapter(cfg.Media.FFProbeTimeoutSeconds),
		FileSystemService:            media.NewOSFileSystem(),
		MediaRepository:              repository.NewMediaRepository(db),
		AudioStreamRepository:        repository.NewAudioStreamRepository(db),
		SubtitleStreamRepository:     repository.NewSubtitleStreamRepository(db),
		TranscodeRepository:          repository.NewTranscodeRepository(db),
		Transcoder:                   media.NewFFmpegTranscoder(cfg.Media.TranscodeTimeoutSeconds),
		FilenameParser:               metadataAdapter.NewRegexFilenameParser(),
		MovieMetadataRepository:      repository.NewSQLiteMovieMetadataRepository(db.DB),
		SeriesMetadataRepository:     repository.NewSQLiteSeriesMetadataRepository(db.DB),
		SeasonMetadataRepository:     repository.NewSQLiteSeasonMetadataRepository(db.DB),
		EpisodeMetadataRepository:    repository.NewSQLiteEpisodeMetadataRepository(db.DB),
		MetadataCandidateRepository:  repository.NewSQLiteMetadataCandidateRepository(db.DB),
		MovieCreditRepository:        repository.NewSQLiteMovieCreditRepository(db.DB),
		SeriesCreditRepository:       repository.NewSQLiteSeriesCreditRepository(db.DB),
		MovieCertificationRepository: repository.NewSQLiteMovieCertificationRepository(db.DB),
		SimilarContentRepository:     repository.NewSQLiteSimilarContentRepository(db.DB),
		CollectionRepository:         repository.NewSQLiteCollectionRepository(db.DB),
		LibraryRepository:            repository.NewLibraryRepository(db),
		SearchRepository:             repository.NewSearchRepository(db),
		UserRepository:               repository.NewSQLiteUserRepository(db),
		RebuildRepository:            repository.NewSQLiteRebuildRepository(db),
		WatchProgressRepository:      repository.NewWatchProgressRepository(db),
		FavoriteRepository:           repository.NewFavoriteRepository(db),
		RecommendationRepository:     repository.NewRecommendationRepository(db),
		FeatureExtractionRepository:  repository.NewFeatureExtractionRepository(db),
		UserWatchDataRepository:      repository.NewUserWatchDataRepository(db),
	}

	// Initialize WebSocket hub if enabled
	if cfg.WebSocket.Enabled {
		adapters.WebSocketHub = websocket.NewHub()
		adapters.ProgressCache = websocket.NewProgressCache()
		adapters.JobNotifier = websocket.NewWebSocketJobNotifier(adapters.WebSocketHub, adapters.ProgressCache)
	} else {
		// Use no-op notifier when WebSocket is disabled
		adapters.JobNotifier = &ports.NoOpJobNotifier{}
	}

	// Initialize TMDB-related adapters only if TMDB is enabled
	if cfg.TMDB.Enabled {
		adapters.TMDBClient = metadataAdapter.NewTMDBHTTPClient(cfg.TMDB)
		adapters.ImageDownloader = metadataAdapter.NewHTTPImageDownloader(cfg.TMDB, adapters.TMDBClient)
	}

	// Initialize WorkerRegistry if remote worker mode is enabled (auth token configured)
	if cfg.Job.RemoteWorkerAuthToken != "" {
		timeout := time.Duration(cfg.Job.RemoteWorkerHeartbeatTimeoutSeconds) * time.Second
		adapters.WorkerRegistry = workerAdapter.NewRegistry(timeout)
	}

	return adapters
}
