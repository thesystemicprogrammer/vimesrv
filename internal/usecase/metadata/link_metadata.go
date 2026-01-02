package metadata

import (
	"context"
	"errors"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// LinkMetadataInput contains the input parameters for linking a media file to metadata
type LinkMetadataInput struct {
	MediaID     string
	CandidateID int64 // If > 0, link using an existing candidate
}

// LinkMetadataOutput contains the result of the link operation
type LinkMetadataOutput struct {
	MediaID      string
	MetadataType string
	Title        string
	Message      string
}

// LinkMetadataUseCase handles linking a media file to metadata (user selection from candidates)
type LinkMetadataUseCase struct {
	config                      config.TMDBConfig
	tmdbClient                  ports.TMDBClient
	imageDownloader             ports.ImageDownloader
	mediaRepository             ports.MediaRepository
	movieMetadataRepository     ports.MovieMetadataRepository
	seriesMetadataRepository    ports.SeriesMetadataRepository
	seasonMetadataRepository    ports.SeasonMetadataRepository
	episodeMetadataRepository   ports.EpisodeMetadataRepository
	metadataCandidateRepository ports.MetadataCandidateRepository
}

// NewLinkMetadataUseCase creates a new instance of LinkMetadataUseCase
func NewLinkMetadataUseCase(
	config config.TMDBConfig,
	tmdbClient ports.TMDBClient,
	imageDownloader ports.ImageDownloader,
	mediaRepository ports.MediaRepository,
	movieMetadataRepository ports.MovieMetadataRepository,
	seriesMetadataRepository ports.SeriesMetadataRepository,
	seasonMetadataRepository ports.SeasonMetadataRepository,
	episodeMetadataRepository ports.EpisodeMetadataRepository,
	metadataCandidateRepository ports.MetadataCandidateRepository,
) *LinkMetadataUseCase {
	return &LinkMetadataUseCase{
		config:                      config,
		tmdbClient:                  tmdbClient,
		imageDownloader:             imageDownloader,
		mediaRepository:             mediaRepository,
		movieMetadataRepository:     movieMetadataRepository,
		seriesMetadataRepository:    seriesMetadataRepository,
		seasonMetadataRepository:    seasonMetadataRepository,
		episodeMetadataRepository:   episodeMetadataRepository,
		metadataCandidateRepository: metadataCandidateRepository,
	}
}

// Execute links a media file to metadata using a candidate
func (uc *LinkMetadataUseCase) Execute(ctx context.Context, input LinkMetadataInput) (*LinkMetadataOutput, error) {
	logger.Info().
		Str("media_id", input.MediaID).
		Int64("candidate_id", input.CandidateID).
		Msg("Linking metadata to media file")

	// Get the media file
	media, err := uc.mediaRepository.Get(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media file: %w", err)
	}
	if media == nil {
		return nil, fmt.Errorf("media file not found: %s", input.MediaID)
	}

	// Get the candidate
	candidate, err := uc.metadataCandidateRepository.Get(ctx, input.CandidateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate: %w", err)
	}
	if candidate == nil {
		return nil, fmt.Errorf("candidate not found: %d", input.CandidateID)
	}

	// Verify candidate belongs to this media file
	if candidate.MediaFileID != input.MediaID {
		return nil, fmt.Errorf("candidate %d does not belong to media file %s", input.CandidateID, input.MediaID)
	}

	// Mark candidate as selected and reject others
	if err := uc.metadataCandidateRepository.MarkSelected(ctx, input.CandidateID); err != nil {
		return nil, fmt.Errorf("failed to mark candidate as selected: %w", err)
	}

	// Link based on candidate type
	var output *LinkMetadataOutput
	if candidate.IsMovie() {
		output, err = uc.linkToMovie(ctx, media, candidate)
	} else {
		output, err = uc.linkToSeries(ctx, media, candidate)
	}

	if err != nil {
		return nil, err
	}

	return output, nil
}

// linkToMovie creates/fetches movie metadata and links the media file
func (uc *LinkMetadataUseCase) linkToMovie(ctx context.Context, media *domain.MediaFile, candidate *domain.MetadataCandidate) (*LinkMetadataOutput, error) {
	logger.Debug().
		Str("media_id", media.ID).
		Int("tmdb_id", candidate.TMDBID).
		Msg("Linking media to movie")

	// Get detailed movie info from TMDB
	details, err := uc.tmdbClient.GetMovieDetails(ctx, candidate.TMDBID, uc.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie details: %w", err)
	}

	// Check if we already have this movie in our database
	existingMovie, err := uc.movieMetadataRepository.GetByTMDBID(ctx, candidate.TMDBID)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing movie: %w", err)
	}

	var movieMetadata *domain.MovieMetadata
	if existingMovie != nil {
		movieMetadata = existingMovie
		logger.Debug().
			Int64("movie_id", movieMetadata.ID).
			Int("tmdb_id", candidate.TMDBID).
			Msg("Reusing existing movie metadata")
	} else {
		// Create new movie metadata
		movieMetadata = uc.createMovieMetadata(details)
		if err := uc.movieMetadataRepository.Create(ctx, movieMetadata); err != nil {
			return nil, fmt.Errorf("failed to create movie metadata: %w", err)
		}

		// Create translation for configured language
		translation := uc.createMovieTranslation(movieMetadata.ID, details)
		if err := uc.movieMetadataRepository.CreateTranslation(ctx, translation); err != nil {
			return nil, fmt.Errorf("failed to create movie translation: %w", err)
		}

		logger.Debug().
			Int64("movie_id", movieMetadata.ID).
			Int("tmdb_id", candidate.TMDBID).
			Msg("Created new movie metadata")
	}

	// Download images if configured
	if uc.config.DownloadImages && uc.imageDownloader != nil {
		uc.downloadMovieImages(ctx, details)
	}

	// Link media file to movie
	media.LinkToMovie(movieMetadata.ID)
	media.SetEnrichmentLinked()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	return &LinkMetadataOutput{
		MediaID:      media.ID,
		MetadataType: domain.MetadataTypeMovie,
		Title:        details.Title,
		Message:      fmt.Sprintf("Linked to movie: %s", details.Title),
	}, nil
}

// linkToSeries creates/fetches series metadata and links the media file to an episode
func (uc *LinkMetadataUseCase) linkToSeries(ctx context.Context, media *domain.MediaFile, candidate *domain.MetadataCandidate) (*LinkMetadataOutput, error) {
	seasonNumber := 1
	episodeNumber := 1
	if candidate.SeasonNumber != nil {
		seasonNumber = *candidate.SeasonNumber
	}
	if candidate.EpisodeNumber != nil {
		episodeNumber = *candidate.EpisodeNumber
	}

	logger.Debug().
		Str("media_id", media.ID).
		Int("tmdb_id", candidate.TMDBID).
		Int("season", seasonNumber).
		Int("episode", episodeNumber).
		Msg("Linking media to series episode")

	// Get detailed series info from TMDB
	seriesDetails, err := uc.tmdbClient.GetSeriesDetails(ctx, candidate.TMDBID, uc.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get series details: %w", err)
	}

	// Get or create series metadata
	seriesMetadata, err := uc.getOrCreateSeriesMetadata(ctx, seriesDetails)
	if err != nil {
		return nil, err
	}

	// Get season details from TMDB
	seasonDetails, err := uc.tmdbClient.GetSeasonDetails(ctx, candidate.TMDBID, seasonNumber, uc.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get season details: %w", err)
	}

	// Get or create season metadata
	seasonMetadata, err := uc.getOrCreateSeasonMetadata(ctx, seriesMetadata.ID, seasonDetails)
	if err != nil {
		return nil, err
	}

	// Get or create episode metadata
	episodeMetadata, err := uc.getOrCreateEpisodeMetadata(ctx, seasonMetadata.ID, candidate.TMDBID, seasonNumber, episodeNumber)
	if err != nil {
		return nil, err
	}

	// Download images if configured
	if uc.config.DownloadImages && uc.imageDownloader != nil {
		uc.downloadSeriesImages(ctx, seriesDetails, seasonDetails, seasonNumber, episodeNumber)
	}

	// Link media file to episode
	media.LinkToEpisode(episodeMetadata.ID)
	media.SetEnrichmentLinked()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	return &LinkMetadataOutput{
		MediaID:      media.ID,
		MetadataType: domain.MetadataTypeEpisode,
		Title:        fmt.Sprintf("%s S%02dE%02d", seriesDetails.Name, seasonNumber, episodeNumber),
		Message:      fmt.Sprintf("Linked to %s S%02dE%02d", seriesDetails.Name, seasonNumber, episodeNumber),
	}, nil
}

// Helper methods - reuse similar logic from EnrichMediaFileUseCase

func (uc *LinkMetadataUseCase) createMovieMetadata(details *ports.TMDBMovieDetails) *domain.MovieMetadata {
	movie := domain.NewMovieMetadata(details.ID, details.OriginalTitle)
	movie.IMDbID = details.IMDbID
	movie.ReleaseDate = details.ReleaseDate
	movie.Runtime = details.Runtime
	movie.PosterPath = details.PosterPath
	movie.BackdropPath = details.BackdropPath
	movie.VoteAverage = details.VoteAverage
	movie.VoteCount = details.VoteCount
	movie.Popularity = details.Popularity
	movie.Status = details.Status
	movie.OriginalLang = details.OriginalLang

	genres := make([]string, len(details.Genres))
	for i, g := range details.Genres {
		genres[i] = g.Name
	}
	movie.Genres = genres

	return movie
}

func (uc *LinkMetadataUseCase) createMovieTranslation(movieID int64, details *ports.TMDBMovieDetails) *domain.MovieMetadataTranslation {
	translation := domain.NewMovieMetadataTranslation(movieID, uc.config.Language, details.Title)
	translation.Tagline = details.Tagline
	translation.Overview = details.Overview
	return translation
}

func (uc *LinkMetadataUseCase) getOrCreateSeriesMetadata(ctx context.Context, details *ports.TMDBSeriesDetails) (*domain.SeriesMetadata, error) {
	existing, err := uc.seriesMetadataRepository.GetByTMDBID(ctx, details.ID)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing series: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	series := domain.NewSeriesMetadata(details.ID, details.OriginalName)
	series.FirstAirDate = details.FirstAirDate
	series.LastAirDate = details.LastAirDate
	series.Status = details.Status
	series.PosterPath = details.PosterPath
	series.BackdropPath = details.BackdropPath
	series.VoteAverage = details.VoteAverage
	series.VoteCount = details.VoteCount
	series.Popularity = details.Popularity
	series.NumberOfSeasons = details.NumberOfSeasons
	series.NumberOfEpisodes = details.NumberOfEpisodes
	series.OriginalLang = details.OriginalLang

	genres := make([]string, len(details.Genres))
	for i, g := range details.Genres {
		genres[i] = g.Name
	}
	series.Genres = genres

	networks := make([]string, len(details.Networks))
	for i, n := range details.Networks {
		networks[i] = n.Name
	}
	series.Networks = networks

	if err := uc.seriesMetadataRepository.Create(ctx, series); err != nil {
		return nil, fmt.Errorf("failed to create series metadata: %w", err)
	}

	translation := domain.NewSeriesMetadataTranslation(series.ID, uc.config.Language, details.Name)
	translation.Overview = details.Overview
	if err := uc.seriesMetadataRepository.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create series translation: %w", err)
	}

	return series, nil
}

func (uc *LinkMetadataUseCase) getOrCreateSeasonMetadata(ctx context.Context, seriesID int64, details *ports.TMDBSeasonDetails) (*domain.SeasonMetadata, error) {
	existing, err := uc.seasonMetadataRepository.GetBySeriesAndNumber(ctx, seriesID, details.SeasonNumber)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing season: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	season := domain.NewSeasonMetadata(seriesID, details.ID, details.SeasonNumber)
	season.AirDate = details.AirDate
	season.PosterPath = details.PosterPath
	season.EpisodeCount = len(details.Episodes)

	if err := uc.seasonMetadataRepository.Create(ctx, season); err != nil {
		return nil, fmt.Errorf("failed to create season metadata: %w", err)
	}

	translation := domain.NewSeasonMetadataTranslation(season.ID, uc.config.Language, details.Name)
	translation.Overview = details.Overview
	if err := uc.seasonMetadataRepository.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create season translation: %w", err)
	}

	return season, nil
}

func (uc *LinkMetadataUseCase) getOrCreateEpisodeMetadata(ctx context.Context, seasonID int64, seriesID int, seasonNumber int, episodeNumber int) (*domain.EpisodeMetadata, error) {
	existing, err := uc.episodeMetadataRepository.GetBySeasonAndNumber(ctx, seasonID, episodeNumber)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing episode: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	details, err := uc.tmdbClient.GetEpisodeDetails(ctx, seriesID, seasonNumber, episodeNumber, uc.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get episode details: %w", err)
	}

	episode := domain.NewEpisodeMetadata(seasonID, details.ID, episodeNumber)
	episode.AirDate = details.AirDate
	episode.StillPath = details.StillPath
	episode.Runtime = details.Runtime
	episode.VoteAverage = details.VoteAverage
	episode.VoteCount = details.VoteCount

	if err := uc.episodeMetadataRepository.Create(ctx, episode); err != nil {
		return nil, fmt.Errorf("failed to create episode metadata: %w", err)
	}

	translation := domain.NewEpisodeMetadataTranslation(episode.ID, uc.config.Language, details.Name)
	translation.Overview = details.Overview
	if err := uc.episodeMetadataRepository.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create episode translation: %w", err)
	}

	return episode, nil
}

func (uc *LinkMetadataUseCase) downloadMovieImages(ctx context.Context, details *ports.TMDBMovieDetails) {
	if details.PosterPath != "" {
		if _, err := uc.imageDownloader.DownloadImage(ctx, details.PosterPath, ports.ImageTypeMoviePoster, details.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", details.ID).Msg("Failed to download movie poster")
		}
	}
	if details.BackdropPath != "" {
		if _, err := uc.imageDownloader.DownloadImage(ctx, details.BackdropPath, ports.ImageTypeMovieBackdrop, details.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", details.ID).Msg("Failed to download movie backdrop")
		}
	}
}

func (uc *LinkMetadataUseCase) downloadSeriesImages(ctx context.Context, seriesDetails *ports.TMDBSeriesDetails, seasonDetails *ports.TMDBSeasonDetails, seasonNumber, episodeNumber int) {
	if seriesDetails.PosterPath != "" {
		if _, err := uc.imageDownloader.DownloadImage(ctx, seriesDetails.PosterPath, ports.ImageTypeSeriesPoster, seriesDetails.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Msg("Failed to download series poster")
		}
	}
	if seriesDetails.BackdropPath != "" {
		if _, err := uc.imageDownloader.DownloadImage(ctx, seriesDetails.BackdropPath, ports.ImageTypeSeriesBackdrop, seriesDetails.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Msg("Failed to download series backdrop")
		}
	}

	if seasonDetails.PosterPath != "" {
		if _, err := uc.imageDownloader.DownloadSeasonImage(ctx, seasonDetails.PosterPath, seriesDetails.ID, seasonNumber); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Int("season", seasonNumber).Msg("Failed to download season poster")
		}
	}

	for _, ep := range seasonDetails.Episodes {
		if ep.EpisodeNumber == episodeNumber && ep.StillPath != "" {
			if _, err := uc.imageDownloader.DownloadEpisodeImage(ctx, ep.StillPath, seriesDetails.ID, seasonNumber, episodeNumber); err != nil {
				logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Int("season", seasonNumber).Int("episode", episodeNumber).Msg("Failed to download episode still")
			}
			break
		}
	}
}
