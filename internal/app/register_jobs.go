package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/library"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/metadata"
	rebuildadapter "github.com/thesystemicprogrammer/vimesrv/internal/adapters/rebuild"
	recommendationadapter "github.com/thesystemicprogrammer/vimesrv/internal/adapters/recommendation"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/transcode"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
)

func registerJobs(cfg *config.Config, useCases *UseCases, adapters *Adapters) {
	// Scan Library
	scanLibraryJobHandler := library.NewScanLibraryJobHandler(useCases.ScanLibraryUseCase)
	adapters.HandlerRegistry.Register(shared.JobTypeScanLibrary, scanLibraryJobHandler)

	// Transcode jobs - all use the same handler since they process via transcode_id
	transcodeJobHandler := transcode.NewTranscodeVideoJobHandler(useCases.ProcessTranscodeUseCase)
	adapters.HandlerRegistry.Register(shared.JobTypeTranscodeVideo, transcodeJobHandler)
	adapters.HandlerRegistry.Register(shared.JobTypeTranscodeAudio, transcodeJobHandler)
	adapters.HandlerRegistry.Register(shared.JobTypeTranscodeSubtitle, transcodeJobHandler)

	// Enrich Metadata (only if TMDB is enabled)
	if useCases.EnrichMediaFileUseCase != nil {
		enrichMetadataJobHandler := metadata.NewEnrichMetadataJobHandler(useCases.EnrichMediaFileUseCase)
		adapters.HandlerRegistry.Register(shared.JobTypeEnrichMetadata, enrichMetadataJobHandler)
	}

	// Fetch Translations (only if TMDB is enabled)
	if adapters.TMDBClient != nil {
		fetchTranslationsJobHandler := metadata.NewFetchTranslationsJobHandler(
			adapters.MovieMetadataRepository,
			adapters.SeriesMetadataRepository,
			adapters.SeasonMetadataRepository,
			adapters.EpisodeMetadataRepository,
			adapters.TMDBClient,
			adapters.ImageDownloader,
			cfg.TMDB.DownloadImages,
		)
		adapters.HandlerRegistry.Register(shared.JobTypeFetchTranslations, fetchTranslationsJobHandler)
	}

	// Prepare Rebuild (periodic export of rebuild.json)
	prepareRebuildJobHandler := rebuildadapter.NewPrepareRebuildJobHandler(useCases.PrepareRebuildUseCase)
	adapters.HandlerRegistry.Register(shared.JobTypePrepareRebuild, prepareRebuildJobHandler)

	// Build Recommendations (periodic recommendation model builds)
	if cfg.Recommendations.Enabled && useCases.BuildRecommendationModelUseCase != nil {
		buildRecommendationsJobHandler := recommendationadapter.NewBuildRecommendationsJobHandler(useCases.BuildRecommendationModelUseCase)
		adapters.HandlerRegistry.Register(shared.JobTypeBuildRecommendations, buildRecommendationsJobHandler)
	}
}
