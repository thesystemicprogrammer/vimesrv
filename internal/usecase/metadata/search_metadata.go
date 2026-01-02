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

// SearchMetadataInput contains the input parameters for searching metadata
type SearchMetadataInput struct {
	MediaID    string
	Query      string
	Year       int    // Optional year filter
	MediaType  string // "movie", "tv", or "" for both
	MaxResults int    // Max results to return (default 10)
	Language   string // Optional language code (e.g., "en", "de"). Falls back to config if empty
}

// SearchMetadataResult represents a single search result
type SearchMetadataResult struct {
	TMDBID        int     `json:"tmdb_id"`
	MediaType     string  `json:"media_type"` // "movie" or "tv"
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	ReleaseDate   string  `json:"release_date"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	PosterURL     string  `json:"poster_url"` // Full URL for display
	VoteAverage   float64 `json:"vote_average"`
	Popularity    float64 `json:"popularity"`
}

// SearchMetadataOutput contains the search results
type SearchMetadataOutput struct {
	MediaID string                 `json:"media_id"`
	Query   string                 `json:"query"`
	Results []SearchMetadataResult `json:"results"`
	Count   int                    `json:"count"`
}

// SearchMetadataUseCase handles manual TMDB searches
type SearchMetadataUseCase struct {
	config     config.TMDBConfig
	tmdbClient ports.TMDBClient
}

// NewSearchMetadataUseCase creates a new instance of SearchMetadataUseCase
func NewSearchMetadataUseCase(
	config config.TMDBConfig,
	tmdbClient ports.TMDBClient,
) *SearchMetadataUseCase {
	return &SearchMetadataUseCase{
		config:     config,
		tmdbClient: tmdbClient,
	}
}

// Execute performs a manual TMDB search
func (uc *SearchMetadataUseCase) Execute(ctx context.Context, input SearchMetadataInput) (*SearchMetadataOutput, error) {
	logger.Info().
		Str("media_id", input.MediaID).
		Str("query", input.Query).
		Str("media_type", input.MediaType).
		Int("year", input.Year).
		Msg("Searching TMDB manually")

	if input.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}

	var results []SearchMetadataResult

	// Use input language if provided, otherwise fall back to config
	language := input.Language
	if language == "" {
		language = uc.config.Language
	}

	switch input.MediaType {
	case "movie":
		movieResults, err := uc.tmdbClient.SearchMovie(ctx, input.Query, input.Year, language)
		if err != nil {
			return nil, fmt.Errorf("failed to search movies: %w", err)
		}
		results = uc.convertResults(movieResults, "movie")

	case "tv":
		tvResults, err := uc.tmdbClient.SearchTV(ctx, input.Query, input.Year, language)
		if err != nil {
			return nil, fmt.Errorf("failed to search TV: %w", err)
		}
		results = uc.convertResults(tvResults, "tv")

	default:
		// Search both movies and TV
		multiResults, err := uc.tmdbClient.SearchMulti(ctx, input.Query, language)
		if err != nil {
			return nil, fmt.Errorf("failed to search: %w", err)
		}
		results = uc.convertMultiResults(multiResults)
	}

	// Limit results
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	logger.Debug().
		Str("media_id", input.MediaID).
		Str("query", input.Query).
		Int("result_count", len(results)).
		Msg("TMDB search completed")

	return &SearchMetadataOutput{
		MediaID: input.MediaID,
		Query:   input.Query,
		Results: results,
		Count:   len(results),
	}, nil
}

func (uc *SearchMetadataUseCase) convertResults(results []ports.TMDBSearchResult, mediaType string) []SearchMetadataResult {
	output := make([]SearchMetadataResult, 0, len(results))
	for _, r := range results {
		output = append(output, SearchMetadataResult{
			TMDBID:        r.ID,
			MediaType:     mediaType,
			Title:         r.Title,
			OriginalTitle: r.OriginalTitle,
			ReleaseDate:   r.ReleaseDate,
			Overview:      r.Overview,
			PosterPath:    r.PosterPath,
			PosterURL:     uc.tmdbClient.GetImageURL(r.PosterPath, uc.config.PosterSize),
			VoteAverage:   r.VoteAverage,
			Popularity:    r.Popularity,
		})
	}
	return output
}

func (uc *SearchMetadataUseCase) convertMultiResults(results []ports.TMDBSearchResult) []SearchMetadataResult {
	output := make([]SearchMetadataResult, 0, len(results))
	for _, r := range results {
		// Only include movies and TV shows
		if r.MediaType != "movie" && r.MediaType != "tv" {
			continue
		}
		output = append(output, SearchMetadataResult{
			TMDBID:        r.ID,
			MediaType:     r.MediaType,
			Title:         r.Title,
			OriginalTitle: r.OriginalTitle,
			ReleaseDate:   r.ReleaseDate,
			Overview:      r.Overview,
			PosterPath:    r.PosterPath,
			PosterURL:     uc.tmdbClient.GetImageURL(r.PosterPath, uc.config.PosterSize),
			VoteAverage:   r.VoteAverage,
			Popularity:    r.Popularity,
		})
	}
	return output
}

// LinkFromSearchInput contains the input for linking directly from a search result
type LinkFromSearchInput struct {
	MediaID       string
	TMDBID        int
	MediaType     string // "movie" or "tv"
	SeasonNumber  int    // For TV series
	EpisodeNumber int    // For TV series
}

// LinkFromSearchOutput contains the result of linking from a search result
type LinkFromSearchOutput struct {
	MediaID      string
	MetadataType string
	Title        string
	Message      string
}

// LinkFromSearchUseCase handles linking a media file directly from a search result (skipping candidates)
type LinkFromSearchUseCase struct {
	config                    config.TMDBConfig
	tmdbClient                ports.TMDBClient
	imageDownloader           ports.ImageDownloader
	mediaRepository           ports.MediaRepository
	movieMetadataRepository   ports.MovieMetadataRepository
	seriesMetadataRepository  ports.SeriesMetadataRepository
	seasonMetadataRepository  ports.SeasonMetadataRepository
	episodeMetadataRepository ports.EpisodeMetadataRepository
	candidateRepository       ports.MetadataCandidateRepository
}

// NewLinkFromSearchUseCase creates a new instance of LinkFromSearchUseCase
func NewLinkFromSearchUseCase(
	config config.TMDBConfig,
	tmdbClient ports.TMDBClient,
	imageDownloader ports.ImageDownloader,
	mediaRepository ports.MediaRepository,
	movieMetadataRepository ports.MovieMetadataRepository,
	seriesMetadataRepository ports.SeriesMetadataRepository,
	seasonMetadataRepository ports.SeasonMetadataRepository,
	episodeMetadataRepository ports.EpisodeMetadataRepository,
	candidateRepository ports.MetadataCandidateRepository,
) *LinkFromSearchUseCase {
	return &LinkFromSearchUseCase{
		config:                    config,
		tmdbClient:                tmdbClient,
		imageDownloader:           imageDownloader,
		mediaRepository:           mediaRepository,
		movieMetadataRepository:   movieMetadataRepository,
		seriesMetadataRepository:  seriesMetadataRepository,
		seasonMetadataRepository:  seasonMetadataRepository,
		episodeMetadataRepository: episodeMetadataRepository,
		candidateRepository:       candidateRepository,
	}
}

// Execute links a media file directly to a TMDB result
func (uc *LinkFromSearchUseCase) Execute(ctx context.Context, input LinkFromSearchInput) (*LinkFromSearchOutput, error) {
	logger.Info().
		Str("media_id", input.MediaID).
		Int("tmdb_id", input.TMDBID).
		Str("media_type", input.MediaType).
		Msg("Linking media file directly from search result")

	// Get the media file
	media, err := uc.mediaRepository.Get(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media file: %w", err)
	}
	if media == nil {
		return nil, fmt.Errorf("media file not found: %s", input.MediaID)
	}

	// Reject any existing candidates for this media file
	if err := uc.candidateRepository.RejectAll(ctx, input.MediaID); err != nil {
		logger.Warn().Err(err).Str("media_id", input.MediaID).Msg("Failed to reject existing candidates")
	}

	// Link based on media type
	var output *LinkFromSearchOutput
	if input.MediaType == "movie" {
		output, err = uc.linkToMovie(ctx, media, input.TMDBID)
	} else {
		output, err = uc.linkToSeries(ctx, media, input.TMDBID, input.SeasonNumber, input.EpisodeNumber)
	}

	if err != nil {
		return nil, err
	}

	return output, nil
}

func (uc *LinkFromSearchUseCase) linkToMovie(ctx context.Context, media *domain.MediaFile, tmdbID int) (*LinkFromSearchOutput, error) {
	// Get detailed movie info from TMDB
	details, err := uc.tmdbClient.GetMovieDetails(ctx, tmdbID, uc.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie details: %w", err)
	}

	// Check if we already have this movie in our database
	existingMovie, err := uc.movieMetadataRepository.GetByTMDBID(ctx, tmdbID)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing movie: %w", err)
	}

	var movieMetadata *domain.MovieMetadata
	if existingMovie != nil {
		movieMetadata = existingMovie
	} else {
		movieMetadata = createMovieMetadataFromDetails(details)
		if err := uc.movieMetadataRepository.Create(ctx, movieMetadata); err != nil {
			return nil, fmt.Errorf("failed to create movie metadata: %w", err)
		}

		translation := createMovieTranslationFromDetails(movieMetadata.ID, uc.config.Language, details)
		if err := uc.movieMetadataRepository.CreateTranslation(ctx, translation); err != nil {
			return nil, fmt.Errorf("failed to create movie translation: %w", err)
		}
	}

	// Download images if configured
	if uc.config.DownloadImages && uc.imageDownloader != nil {
		downloadMovieImagesHelper(ctx, uc.imageDownloader, details)
	}

	// Link media file to movie
	media.LinkToMovie(movieMetadata.ID)
	media.SetEnrichmentLinked()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	return &LinkFromSearchOutput{
		MediaID:      media.ID,
		MetadataType: domain.MetadataTypeMovie,
		Title:        details.Title,
		Message:      fmt.Sprintf("Linked to movie: %s", details.Title),
	}, nil
}

func (uc *LinkFromSearchUseCase) linkToSeries(ctx context.Context, media *domain.MediaFile, tmdbID, seasonNumber, episodeNumber int) (*LinkFromSearchOutput, error) {
	if seasonNumber <= 0 {
		seasonNumber = 1
	}
	if episodeNumber <= 0 {
		episodeNumber = 1
	}

	// Get detailed series info from TMDB
	seriesDetails, err := uc.tmdbClient.GetSeriesDetails(ctx, tmdbID, uc.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get series details: %w", err)
	}

	// Get or create series metadata
	seriesMetadata, err := getOrCreateSeriesMetadataHelper(ctx, uc.seriesMetadataRepository, seriesDetails, uc.config.Language)
	if err != nil {
		return nil, err
	}

	// Get season details from TMDB
	seasonDetails, err := uc.tmdbClient.GetSeasonDetails(ctx, tmdbID, seasonNumber, uc.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get season details: %w", err)
	}

	// Get or create season metadata
	seasonMetadata, err := getOrCreateSeasonMetadataHelper(ctx, uc.seasonMetadataRepository, seriesMetadata.ID, seasonDetails, uc.config.Language)
	if err != nil {
		return nil, err
	}

	// Get or create episode metadata
	episodeMetadata, err := getOrCreateEpisodeMetadataHelper(ctx, uc.episodeMetadataRepository, uc.tmdbClient, seasonMetadata.ID, tmdbID, seasonNumber, episodeNumber, uc.config.Language)
	if err != nil {
		return nil, err
	}

	// Download images if configured
	if uc.config.DownloadImages && uc.imageDownloader != nil {
		downloadSeriesImagesHelper(ctx, uc.imageDownloader, seriesDetails, seasonDetails, seasonNumber, episodeNumber)
	}

	// Link media file to episode
	media.LinkToEpisode(episodeMetadata.ID)
	media.SetEnrichmentLinked()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	return &LinkFromSearchOutput{
		MediaID:      media.ID,
		MetadataType: domain.MetadataTypeEpisode,
		Title:        fmt.Sprintf("%s S%02dE%02d", seriesDetails.Name, seasonNumber, episodeNumber),
		Message:      fmt.Sprintf("Linked to %s S%02dE%02d", seriesDetails.Name, seasonNumber, episodeNumber),
	}, nil
}

// Helper functions shared between use cases

func createMovieMetadataFromDetails(details *ports.TMDBMovieDetails) *domain.MovieMetadata {
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

func createMovieTranslationFromDetails(movieID int64, language string, details *ports.TMDBMovieDetails) *domain.MovieMetadataTranslation {
	translation := domain.NewMovieMetadataTranslation(movieID, language, details.Title)
	translation.Tagline = details.Tagline
	translation.Overview = details.Overview
	return translation
}

func getOrCreateSeriesMetadataHelper(ctx context.Context, repo ports.SeriesMetadataRepository, details *ports.TMDBSeriesDetails, language string) (*domain.SeriesMetadata, error) {
	existing, err := repo.GetByTMDBID(ctx, details.ID)
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

	if err := repo.Create(ctx, series); err != nil {
		return nil, fmt.Errorf("failed to create series metadata: %w", err)
	}

	translation := domain.NewSeriesMetadataTranslation(series.ID, language, details.Name)
	translation.Overview = details.Overview
	if err := repo.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create series translation: %w", err)
	}

	return series, nil
}

func getOrCreateSeasonMetadataHelper(ctx context.Context, repo ports.SeasonMetadataRepository, seriesID int64, details *ports.TMDBSeasonDetails, language string) (*domain.SeasonMetadata, error) {
	existing, err := repo.GetBySeriesAndNumber(ctx, seriesID, details.SeasonNumber)
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

	if err := repo.Create(ctx, season); err != nil {
		return nil, fmt.Errorf("failed to create season metadata: %w", err)
	}

	translation := domain.NewSeasonMetadataTranslation(season.ID, language, details.Name)
	translation.Overview = details.Overview
	if err := repo.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create season translation: %w", err)
	}

	return season, nil
}

func getOrCreateEpisodeMetadataHelper(ctx context.Context, repo ports.EpisodeMetadataRepository, tmdbClient ports.TMDBClient, seasonID int64, seriesID int, seasonNumber int, episodeNumber int, language string) (*domain.EpisodeMetadata, error) {
	existing, err := repo.GetBySeasonAndNumber(ctx, seasonID, episodeNumber)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing episode: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	details, err := tmdbClient.GetEpisodeDetails(ctx, seriesID, seasonNumber, episodeNumber, language)
	if err != nil {
		return nil, fmt.Errorf("failed to get episode details: %w", err)
	}

	episode := domain.NewEpisodeMetadata(seasonID, details.ID, episodeNumber)
	episode.AirDate = details.AirDate
	episode.StillPath = details.StillPath
	episode.Runtime = details.Runtime
	episode.VoteAverage = details.VoteAverage
	episode.VoteCount = details.VoteCount

	if err := repo.Create(ctx, episode); err != nil {
		return nil, fmt.Errorf("failed to create episode metadata: %w", err)
	}

	translation := domain.NewEpisodeMetadataTranslation(episode.ID, language, details.Name)
	translation.Overview = details.Overview
	if err := repo.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create episode translation: %w", err)
	}

	return episode, nil
}

func downloadMovieImagesHelper(ctx context.Context, downloader ports.ImageDownloader, details *ports.TMDBMovieDetails) {
	if details.PosterPath != "" {
		if _, err := downloader.DownloadImage(ctx, details.PosterPath, ports.ImageTypeMoviePoster, details.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", details.ID).Msg("Failed to download movie poster")
		}
	}
	if details.BackdropPath != "" {
		if _, err := downloader.DownloadImage(ctx, details.BackdropPath, ports.ImageTypeMovieBackdrop, details.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", details.ID).Msg("Failed to download movie backdrop")
		}
	}
}

func downloadSeriesImagesHelper(ctx context.Context, downloader ports.ImageDownloader, seriesDetails *ports.TMDBSeriesDetails, seasonDetails *ports.TMDBSeasonDetails, seasonNumber, episodeNumber int) {
	if seriesDetails.PosterPath != "" {
		if _, err := downloader.DownloadImage(ctx, seriesDetails.PosterPath, ports.ImageTypeSeriesPoster, seriesDetails.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Msg("Failed to download series poster")
		}
	}
	if seriesDetails.BackdropPath != "" {
		if _, err := downloader.DownloadImage(ctx, seriesDetails.BackdropPath, ports.ImageTypeSeriesBackdrop, seriesDetails.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Msg("Failed to download series backdrop")
		}
	}

	if seasonDetails.PosterPath != "" {
		if _, err := downloader.DownloadSeasonImage(ctx, seasonDetails.PosterPath, seriesDetails.ID, seasonNumber); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Int("season", seasonNumber).Msg("Failed to download season poster")
		}
	}

	for _, ep := range seasonDetails.Episodes {
		if ep.EpisodeNumber == episodeNumber && ep.StillPath != "" {
			if _, err := downloader.DownloadEpisodeImage(ctx, ep.StillPath, seriesDetails.ID, seasonNumber, episodeNumber); err != nil {
				logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Int("season", seasonNumber).Int("episode", episodeNumber).Msg("Failed to download episode still")
			}
			break
		}
	}
}
