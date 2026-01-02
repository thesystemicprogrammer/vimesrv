package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/library"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/metadata"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/transcode"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
)

func registerJobs(useCases *UseCases, adapters *Adapters) {
	// Scan Library
	scanLibraryJobHandler := library.NewScanLibraryJobHandler(useCases.ScanLibraryUseCase)
	adapters.HandlerRegistry.Register(shared.JobTypeScanLibrary, scanLibraryJobHandler)

	// Transcode Video
	transcodeVideoJobHandler := transcode.NewTranscodeVideoJobHandler(useCases.ProcessTranscodeUseCase)
	adapters.HandlerRegistry.Register(shared.JobTypeTranscodeVideo, transcodeVideoJobHandler)

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
		)
		adapters.HandlerRegistry.Register(shared.JobTypeFetchTranslations, fetchTranslationsJobHandler)
	}
}
