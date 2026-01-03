package rebuild

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/metadata/linker"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// Linker handles linking media files to TMDB metadata during rebuild
type Linker struct {
	movieLinker     *linker.MovieLinker
	episodeLinker   *linker.EpisodeLinker
	mediaRepository ports.MediaRepository
}

// NewLinker creates a new rebuild linker
func NewLinker(
	tmdbConfig config.TMDBConfig,
	tmdbClient ports.TMDBClient,
	imageDownloader ports.ImageDownloader,
	movieMetadataRepository ports.MovieMetadataRepository,
	seriesMetadataRepository ports.SeriesMetadataRepository,
	seasonMetadataRepository ports.SeasonMetadataRepository,
	episodeMetadataRepository ports.EpisodeMetadataRepository,
	movieCreditRepository ports.MovieCreditRepository,
	movieCertificationRepository ports.MovieCertificationRepository,
	mediaRepository ports.MediaRepository,
) *Linker {
	return &Linker{
		movieLinker: linker.NewMovieLinker(
			tmdbConfig,
			tmdbClient,
			imageDownloader,
			movieMetadataRepository,
			movieCreditRepository,
			movieCertificationRepository,
		),
		episodeLinker: linker.NewEpisodeLinker(
			tmdbConfig,
			tmdbClient,
			imageDownloader,
			seriesMetadataRepository,
			seasonMetadataRepository,
			episodeMetadataRepository,
		),
		mediaRepository: mediaRepository,
	}
}

// Link links a media file to TMDB metadata based on auto-link data
// On failure, logs a warning and returns the error (caller decides whether to continue)
func (l *Linker) Link(ctx context.Context, media *domain.MediaFile, autoLink AutoLinkData) error {
	switch autoLink.MetadataType {
	case domain.MetadataTypeMovie:
		return l.linkMovie(ctx, media, autoLink.TMDBID)
	case domain.MetadataTypeEpisode:
		return l.linkEpisode(ctx, media, autoLink.SeriesTMDBID, autoLink.SeasonNumber, autoLink.EpisodeNumber)
	default:
		return fmt.Errorf("unknown metadata type: %s", autoLink.MetadataType)
	}
}

// linkMovie links a media file to a movie using the shared MovieLinker
func (l *Linker) linkMovie(ctx context.Context, media *domain.MediaFile, tmdbID int) error {
	logger.Debug().
		Str("media_id", media.ID).
		Int("tmdb_id", tmdbID).
		Msg("[rebuild] Linking to movie")

	result, err := l.movieLinker.Link(ctx, tmdbID)
	if err != nil {
		return fmt.Errorf("failed to link movie: %w", err)
	}

	// Update media file with link
	media.LinkToMovie(result.MovieMetadata.ID)
	media.SetEnrichmentAutoLinked()

	if err := l.mediaRepository.Update(ctx, media); err != nil {
		return fmt.Errorf("failed to update media file: %w", err)
	}

	logger.Info().
		Str("media_id", media.ID).
		Int("tmdb_id", tmdbID).
		Str("title", result.Details.Title).
		Msg("[rebuild] Linked to movie")

	return nil
}

// linkEpisode links a media file to an episode using the shared EpisodeLinker
func (l *Linker) linkEpisode(ctx context.Context, media *domain.MediaFile, seriesTMDBID, seasonNumber, episodeNumber int) error {
	logger.Debug().
		Str("media_id", media.ID).
		Int("series_tmdb_id", seriesTMDBID).
		Int("season", seasonNumber).
		Int("episode", episodeNumber).
		Msg("[rebuild] Linking to episode")

	result, err := l.episodeLinker.Link(ctx, seriesTMDBID, seasonNumber, episodeNumber)
	if err != nil {
		return fmt.Errorf("failed to link episode: %w", err)
	}

	// Update media file with link
	media.LinkToEpisode(result.EpisodeMetadata.ID)
	media.SetEnrichmentAutoLinked()

	if err := l.mediaRepository.Update(ctx, media); err != nil {
		return fmt.Errorf("failed to update media file: %w", err)
	}

	logger.Info().
		Str("media_id", media.ID).
		Int("series_tmdb_id", seriesTMDBID).
		Int("season", seasonNumber).
		Int("episode", episodeNumber).
		Msg("[rebuild] Linked to episode")

	return nil
}
