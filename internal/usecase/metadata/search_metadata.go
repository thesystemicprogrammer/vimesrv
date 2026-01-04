package metadata

import (
	"context"
	"fmt"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/metadata/linker"
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
	movieLinker            *linker.MovieLinker
	episodeLinker          *linker.EpisodeLinker
	mediaRepository        ports.MediaRepository
	candidateRepository    ports.MetadataCandidateRepository
	searchRepository       ports.SearchRepository
	movieCreditRepository  ports.MovieCreditRepository
	seriesCreditRepository ports.SeriesCreditRepository
}

// NewLinkFromSearchUseCase creates a new instance of LinkFromSearchUseCase
func NewLinkFromSearchUseCase(
	movieLinker *linker.MovieLinker,
	episodeLinker *linker.EpisodeLinker,
	mediaRepository ports.MediaRepository,
	candidateRepository ports.MetadataCandidateRepository,
	searchRepository ports.SearchRepository,
	movieCreditRepository ports.MovieCreditRepository,
	seriesCreditRepository ports.SeriesCreditRepository,
) *LinkFromSearchUseCase {
	return &LinkFromSearchUseCase{
		movieLinker:            movieLinker,
		episodeLinker:          episodeLinker,
		mediaRepository:        mediaRepository,
		candidateRepository:    candidateRepository,
		searchRepository:       searchRepository,
		movieCreditRepository:  movieCreditRepository,
		seriesCreditRepository: seriesCreditRepository,
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

// linkToMovie uses MovieLinker to create/fetch movie metadata and link the media file
func (uc *LinkFromSearchUseCase) linkToMovie(ctx context.Context, media *domain.MediaFile, tmdbID int) (*LinkFromSearchOutput, error) {
	logger.Debug().
		Str("media_id", media.ID).
		Int("tmdb_id", tmdbID).
		Msg("Linking media to movie from search")

	// Use MovieLinker to handle all movie metadata creation (including credits, certifications, collection)
	result, err := uc.movieLinker.Link(ctx, tmdbID)
	if err != nil {
		return nil, fmt.Errorf("failed to link movie: %w", err)
	}

	// Link media file to movie
	media.LinkToMovie(result.MovieMetadata.ID)
	media.SetEnrichmentLinked()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	// Index movie for full-text search
	uc.indexMovieForSearch(ctx, media.ID, result.MovieMetadata.ID, result.Details.Title, result.Details.OriginalTitle)

	return &LinkFromSearchOutput{
		MediaID:      media.ID,
		MetadataType: domain.MetadataTypeMovie,
		Title:        result.Details.Title,
		Message:      fmt.Sprintf("Linked to movie: %s", result.Details.Title),
	}, nil
}

// linkToSeries uses EpisodeLinker to create/fetch series metadata and link the media file to an episode
func (uc *LinkFromSearchUseCase) linkToSeries(ctx context.Context, media *domain.MediaFile, tmdbID, seasonNumber, episodeNumber int) (*LinkFromSearchOutput, error) {
	if seasonNumber <= 0 {
		seasonNumber = 1
	}
	if episodeNumber <= 0 {
		episodeNumber = 1
	}

	logger.Debug().
		Str("media_id", media.ID).
		Int("tmdb_id", tmdbID).
		Int("season", seasonNumber).
		Int("episode", episodeNumber).
		Msg("Linking media to series episode from search")

	// Use EpisodeLinker to handle all episode metadata creation
	result, err := uc.episodeLinker.Link(ctx, tmdbID, seasonNumber, episodeNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to link episode: %w", err)
	}

	// Link media file to episode
	media.LinkToEpisode(result.EpisodeMetadata.ID)
	media.SetEnrichmentLinked()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	// Index series for full-text search (if it was newly created)
	if result.SeriesCreated {
		uc.indexSeriesForSearch(ctx, result.SeriesMetadata.ID, result.SeriesDetails.Name, result.SeriesDetails.OriginalName)
	}

	seriesName := result.SeriesDetails.Name
	return &LinkFromSearchOutput{
		MediaID:      media.ID,
		MetadataType: domain.MetadataTypeEpisode,
		Title:        fmt.Sprintf("%s S%02dE%02d", seriesName, seasonNumber, episodeNumber),
		Message:      fmt.Sprintf("Linked to %s S%02dE%02d", seriesName, seasonNumber, episodeNumber),
	}, nil
}

// indexMovieForSearch adds the movie to the FTS search index
func (uc *LinkFromSearchUseCase) indexMovieForSearch(ctx context.Context, mediaID string, movieMetadataID int64, title, originalTitle string) {
	if uc.searchRepository == nil {
		return
	}

	// Get credits from the database to build searchable cast/crew strings
	var castNames, crewNames []string
	if uc.movieCreditRepository != nil {
		credits, err := uc.movieCreditRepository.GetByMovieMetadataID(ctx, movieMetadataID)
		if err != nil {
			logger.Debug().Err(err).Int64("movie_id", movieMetadataID).Msg("No credits available for search indexing")
		} else {
			for _, credit := range credits {
				if credit.CreditType == domain.CreditTypeCast {
					castNames = append(castNames, credit.Name)
				} else {
					crewNames = append(crewNames, credit.Name)
				}
			}
		}
	}

	if err := uc.searchRepository.IndexMovie(
		ctx,
		mediaID,
		movieMetadataID,
		title,
		originalTitle,
		strings.Join(castNames, " "),
		strings.Join(crewNames, " "),
	); err != nil {
		logger.Warn().Err(err).Int64("movie_id", movieMetadataID).Msg("Failed to index movie for search")
	}
}

// indexSeriesForSearch adds the series to the FTS search index
func (uc *LinkFromSearchUseCase) indexSeriesForSearch(ctx context.Context, seriesMetadataID int64, name, originalName string) {
	if uc.searchRepository == nil {
		return
	}

	// Get credits from the database to build searchable cast/crew strings
	var castNames, crewNames []string
	if uc.seriesCreditRepository != nil {
		credits, err := uc.seriesCreditRepository.GetBySeriesMetadataID(ctx, seriesMetadataID)
		if err != nil {
			logger.Debug().Err(err).Int64("series_id", seriesMetadataID).Msg("No credits available for search indexing")
		} else {
			for _, credit := range credits {
				if credit.CreditType == domain.CreditTypeCast {
					castNames = append(castNames, credit.Name)
				} else {
					crewNames = append(crewNames, credit.Name)
				}
			}
		}
	}

	if err := uc.searchRepository.IndexSeries(
		ctx,
		seriesMetadataID,
		name,
		originalName,
		strings.Join(castNames, " "),
		strings.Join(crewNames, " "),
	); err != nil {
		logger.Warn().Err(err).Int64("series_id", seriesMetadataID).Msg("Failed to index series for search")
	}
}
