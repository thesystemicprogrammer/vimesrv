package metadata

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// EnrichMediaFileInput contains the input parameters for enriching a media file
type EnrichMediaFileInput struct {
	MediaID string
}

// EnrichMediaFileOutput contains the result of the enrichment operation
type EnrichMediaFileOutput struct {
	MediaID          string
	EnrichmentStatus string
	MetadataType     string
	CandidateCount   int
	AutoLinked       bool
	Message          string
}

// EnrichMediaFileUseCase orchestrates the metadata enrichment process for a media file
type EnrichMediaFileUseCase struct {
	config                       config.TMDBConfig
	filenameParser               ports.FilenameParser
	tmdbClient                   ports.TMDBClient
	imageDownloader              ports.ImageDownloader
	mediaRepository              ports.MediaRepository
	movieMetadataRepository      ports.MovieMetadataRepository
	seriesMetadataRepository     ports.SeriesMetadataRepository
	seasonMetadataRepository     ports.SeasonMetadataRepository
	episodeMetadataRepository    ports.EpisodeMetadataRepository
	metadataCandidateRepository  ports.MetadataCandidateRepository
	movieCreditRepository        ports.MovieCreditRepository
	movieCertificationRepository ports.MovieCertificationRepository
}

// NewEnrichMediaFileUseCase creates a new instance of EnrichMediaFileUseCase
func NewEnrichMediaFileUseCase(
	config config.TMDBConfig,
	filenameParser ports.FilenameParser,
	tmdbClient ports.TMDBClient,
	imageDownloader ports.ImageDownloader,
	mediaRepository ports.MediaRepository,
	movieMetadataRepository ports.MovieMetadataRepository,
	seriesMetadataRepository ports.SeriesMetadataRepository,
	seasonMetadataRepository ports.SeasonMetadataRepository,
	episodeMetadataRepository ports.EpisodeMetadataRepository,
	metadataCandidateRepository ports.MetadataCandidateRepository,
	movieCreditRepository ports.MovieCreditRepository,
	movieCertificationRepository ports.MovieCertificationRepository,
) *EnrichMediaFileUseCase {
	return &EnrichMediaFileUseCase{
		config:                       config,
		filenameParser:               filenameParser,
		tmdbClient:                   tmdbClient,
		imageDownloader:              imageDownloader,
		mediaRepository:              mediaRepository,
		movieMetadataRepository:      movieMetadataRepository,
		seriesMetadataRepository:     seriesMetadataRepository,
		seasonMetadataRepository:     seasonMetadataRepository,
		episodeMetadataRepository:    episodeMetadataRepository,
		metadataCandidateRepository:  metadataCandidateRepository,
		movieCreditRepository:        movieCreditRepository,
		movieCertificationRepository: movieCertificationRepository,
	}
}

// Execute performs the metadata enrichment for a single media file
func (uc *EnrichMediaFileUseCase) Execute(ctx context.Context, input EnrichMediaFileInput) (*EnrichMediaFileOutput, error) {
	logger.Info().Str("media_id", input.MediaID).Msg("Starting metadata enrichment")

	// Get the media file
	media, err := uc.mediaRepository.Get(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media file: %w", err)
	}
	if media == nil {
		return nil, fmt.Errorf("media file not found: %s", input.MediaID)
	}

	// Skip if already enriched
	if media.EnrichmentStatus != domain.EnrichmentStatusPending {
		logger.Info().
			Str("media_id", input.MediaID).
			Str("status", media.EnrichmentStatus).
			Msg("Media file already processed, skipping")
		return &EnrichMediaFileOutput{
			MediaID:          input.MediaID,
			EnrichmentStatus: media.EnrichmentStatus,
			MetadataType:     media.MetadataType,
			Message:          "Already processed",
		}, nil
	}

	// Parse filename to extract title, year, season/episode info
	filename := filepath.Base(media.FilePath)
	parsed := uc.filenameParser.Parse(filename)

	logger.Debug().
		Str("media_id", input.MediaID).
		Str("filename", filename).
		Str("title", parsed.Title).
		Int("year", parsed.Year).
		Bool("is_series", parsed.IsSeries).
		Int("season", parsed.SeasonNumber).
		Int("episode", parsed.EpisodeNumber).
		Str("edition", parsed.Edition).
		Msg("Parsed filename")

	// Store edition from parsed filename
	if parsed.Edition != "" {
		media.Edition = parsed.Edition
	}

	// Search TMDB based on parsed info
	var output *EnrichMediaFileOutput
	if parsed.IsSeries {
		output, err = uc.enrichAsSeries(ctx, media, parsed)
	} else {
		output, err = uc.enrichAsMovie(ctx, media, parsed)
	}

	if err != nil {
		return nil, err
	}

	return output, nil
}

// enrichAsMovie handles enrichment for movie files
func (uc *EnrichMediaFileUseCase) enrichAsMovie(ctx context.Context, media *domain.MediaFile, parsed *ports.ParsedFilename) (*EnrichMediaFileOutput, error) {
	logger.Debug().
		Str("media_id", media.ID).
		Str("title", parsed.CleanTitle).
		Int("year", parsed.Year).
		Msg("Searching TMDB for movie")

	// Search TMDB for movies
	results, err := uc.tmdbClient.SearchMovie(ctx, parsed.CleanTitle, parsed.Year, uc.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to search TMDB: %w", err)
	}

	if len(results) == 0 {
		// No results found - mark as manual required
		logger.Info().
			Str("media_id", media.ID).
			Str("title", parsed.CleanTitle).
			Msg("No TMDB results found, marking as manual required")

		media.SetEnrichmentManualRequired()
		if err := uc.mediaRepository.Update(ctx, media); err != nil {
			return nil, fmt.Errorf("failed to update media file: %w", err)
		}

		return &EnrichMediaFileOutput{
			MediaID:          media.ID,
			EnrichmentStatus: domain.EnrichmentStatusManualRequired,
			MetadataType:     domain.MetadataTypeNone,
			CandidateCount:   0,
			AutoLinked:       false,
			Message:          "No matches found",
		}, nil
	}

	// Calculate confidence scores for each result
	candidates := uc.scoreMovieCandidates(media.ID, parsed, results)

	// Check if we have a high-confidence match
	if len(candidates) > 0 && candidates[0].ConfidenceScore >= uc.config.AutoLinkThreshold {
		// Auto-link to the best match
		return uc.autoLinkMovie(ctx, media, parsed, candidates[0], results[0])
	}

	// Multiple candidates or no high-confidence match - store candidates
	return uc.storeCandidates(ctx, media, candidates)
}

// enrichAsSeries handles enrichment for TV series episode files
func (uc *EnrichMediaFileUseCase) enrichAsSeries(ctx context.Context, media *domain.MediaFile, parsed *ports.ParsedFilename) (*EnrichMediaFileOutput, error) {
	logger.Debug().
		Str("media_id", media.ID).
		Str("title", parsed.CleanTitle).
		Int("year", parsed.Year).
		Int("season", parsed.SeasonNumber).
		Int("episode", parsed.EpisodeNumber).
		Msg("Searching TMDB for TV series")

	// Search TMDB for TV series
	results, err := uc.tmdbClient.SearchTV(ctx, parsed.CleanTitle, parsed.Year, uc.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to search TMDB: %w", err)
	}

	if len(results) == 0 {
		// No results found - mark as manual required
		logger.Info().
			Str("media_id", media.ID).
			Str("title", parsed.CleanTitle).
			Msg("No TMDB TV results found, marking as manual required")

		media.SetEnrichmentManualRequired()
		if err := uc.mediaRepository.Update(ctx, media); err != nil {
			return nil, fmt.Errorf("failed to update media file: %w", err)
		}

		return &EnrichMediaFileOutput{
			MediaID:          media.ID,
			EnrichmentStatus: domain.EnrichmentStatusManualRequired,
			MetadataType:     domain.MetadataTypeNone,
			CandidateCount:   0,
			AutoLinked:       false,
			Message:          "No matches found",
		}, nil
	}

	// Calculate confidence scores for each result
	candidates := uc.scoreSeriesCandidates(media.ID, parsed, results)

	// Check if we have a high-confidence match
	if len(candidates) > 0 && candidates[0].ConfidenceScore >= uc.config.AutoLinkThreshold {
		// Auto-link to the best match
		return uc.autoLinkSeries(ctx, media, parsed, candidates[0], results[0])
	}

	// Multiple candidates or no high-confidence match - store candidates
	return uc.storeCandidates(ctx, media, candidates)
}

// scoreMovieCandidates calculates confidence scores for movie search results
func (uc *EnrichMediaFileUseCase) scoreMovieCandidates(mediaID string, parsed *ports.ParsedFilename, results []ports.TMDBSearchResult) []domain.MetadataCandidate {
	candidates := make([]domain.MetadataCandidate, 0, len(results))

	for _, result := range results {
		score := uc.calculateMovieScore(parsed, result)

		candidate := domain.NewMetadataCandidate(
			mediaID,
			domain.CandidateTypeMovie,
			result.ID,
			result.Title,
			score,
		)
		candidate.ReleaseDate = result.ReleaseDate
		candidate.Overview = result.Overview
		candidate.PosterPath = result.PosterPath

		candidates = append(candidates, *candidate)
	}

	// Sort by confidence score (highest first)
	sortCandidatesByScore(candidates)

	// Limit to max candidates
	if len(candidates) > uc.config.MaxCandidates {
		candidates = candidates[:uc.config.MaxCandidates]
	}

	return candidates
}

// scoreSeriesCandidates calculates confidence scores for TV series search results
func (uc *EnrichMediaFileUseCase) scoreSeriesCandidates(mediaID string, parsed *ports.ParsedFilename, results []ports.TMDBSearchResult) []domain.MetadataCandidate {
	candidates := make([]domain.MetadataCandidate, 0, len(results))

	for _, result := range results {
		score := uc.calculateSeriesScore(parsed, result)

		candidate := domain.NewMetadataCandidate(
			mediaID,
			domain.CandidateTypeSeries,
			result.ID,
			result.Title,
			score,
		)
		candidate.ReleaseDate = result.ReleaseDate
		candidate.Overview = result.Overview
		candidate.PosterPath = result.PosterPath
		candidate.SetEpisodeInfo(parsed.SeasonNumber, parsed.EpisodeNumber)

		candidates = append(candidates, *candidate)
	}

	// Sort by confidence score (highest first)
	sortCandidatesByScore(candidates)

	// Limit to max candidates
	if len(candidates) > uc.config.MaxCandidates {
		candidates = candidates[:uc.config.MaxCandidates]
	}

	return candidates
}

// calculateMovieScore calculates a confidence score for a movie match
func (uc *EnrichMediaFileUseCase) calculateMovieScore(parsed *ports.ParsedFilename, result ports.TMDBSearchResult) int {
	score := 0

	// Title similarity (max 60 points)
	titleScore := calculateTitleSimilarity(parsed.CleanTitle, result.Title)
	score += int(titleScore * 60)

	// Also check against original title
	originalTitleScore := calculateTitleSimilarity(parsed.CleanTitle, result.OriginalTitle)
	if int(originalTitleScore*60) > int(titleScore*60) {
		score = int(originalTitleScore * 60)
	}

	// Year match (max 25 points)
	if parsed.Year > 0 && result.ReleaseDate != "" {
		releaseYear := extractYear(result.ReleaseDate)
		if releaseYear == parsed.Year {
			score += 25
		} else if abs(releaseYear-parsed.Year) == 1 {
			// One year off - common for films released late in year
			score += 15
		}
	}

	// Popularity bonus (max 10 points) - popular movies more likely correct
	popularityScore := min(int(result.Popularity/10), 10)
	score += popularityScore

	// Vote count bonus (max 5 points) - well-known movies more likely correct
	voteScore := min(result.VoteCount/1000, 5)
	score += voteScore

	return min(score, 100)
}

// calculateSeriesScore calculates a confidence score for a TV series match
func (uc *EnrichMediaFileUseCase) calculateSeriesScore(parsed *ports.ParsedFilename, result ports.TMDBSearchResult) int {
	score := 0

	// Title similarity (max 60 points)
	titleScore := calculateTitleSimilarity(parsed.CleanTitle, result.Title)
	score += int(titleScore * 60)

	// Also check against original title
	originalTitleScore := calculateTitleSimilarity(parsed.CleanTitle, result.OriginalTitle)
	if int(originalTitleScore*60) > int(titleScore*60) {
		score = int(originalTitleScore * 60)
	}

	// Year match (max 20 points)
	if parsed.Year > 0 && result.ReleaseDate != "" {
		releaseYear := extractYear(result.ReleaseDate)
		if releaseYear == parsed.Year {
			score += 20
		} else if abs(releaseYear-parsed.Year) <= 2 {
			// TV series can run for years, so be more lenient
			score += 10
		}
	}

	// Popularity bonus (max 10 points)
	popularityScore := min(int(result.Popularity/10), 10)
	score += popularityScore

	// Vote count bonus (max 5 points)
	voteScore := min(result.VoteCount/500, 5)
	score += voteScore

	// Season/episode presence bonus (max 5 points)
	// If we have season/episode info, it's more likely a real series match
	if parsed.SeasonNumber > 0 && parsed.EpisodeNumber > 0 {
		score += 5
	}

	return min(score, 100)
}

// autoLinkMovie creates metadata records and links the media file to a movie
func (uc *EnrichMediaFileUseCase) autoLinkMovie(ctx context.Context, media *domain.MediaFile, parsed *ports.ParsedFilename, candidate domain.MetadataCandidate, searchResult ports.TMDBSearchResult) (*EnrichMediaFileOutput, error) {
	logger.Info().
		Str("media_id", media.ID).
		Int("tmdb_id", candidate.TMDBID).
		Str("title", candidate.Title).
		Int("score", candidate.ConfidenceScore).
		Msg("Auto-linking movie with high confidence match")

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
		// Reuse existing metadata
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

		// Fetch and store movie credits
		if err := uc.fetchAndStoreMovieCredits(ctx, movieMetadata.ID, candidate.TMDBID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", candidate.TMDBID).Msg("Failed to fetch movie credits")
		}

		// Fetch and store movie certifications
		if err := uc.fetchAndStoreCertifications(ctx, movieMetadata.ID, candidate.TMDBID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", candidate.TMDBID).Msg("Failed to fetch movie certifications")
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
	media.SetEnrichmentAutoLinked()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	return &EnrichMediaFileOutput{
		MediaID:          media.ID,
		EnrichmentStatus: domain.EnrichmentStatusAutoLinked,
		MetadataType:     domain.MetadataTypeMovie,
		CandidateCount:   1,
		AutoLinked:       true,
		Message:          fmt.Sprintf("Auto-linked to movie: %s (%d)", candidate.Title, candidate.ConfidenceScore),
	}, nil
}

// autoLinkSeries creates metadata records and links the media file to an episode
func (uc *EnrichMediaFileUseCase) autoLinkSeries(ctx context.Context, media *domain.MediaFile, parsed *ports.ParsedFilename, candidate domain.MetadataCandidate, searchResult ports.TMDBSearchResult) (*EnrichMediaFileOutput, error) {
	logger.Info().
		Str("media_id", media.ID).
		Int("tmdb_id", candidate.TMDBID).
		Str("title", candidate.Title).
		Int("season", parsed.SeasonNumber).
		Int("episode", parsed.EpisodeNumber).
		Int("score", candidate.ConfidenceScore).
		Msg("Auto-linking series episode with high confidence match")

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
	seasonDetails, err := uc.tmdbClient.GetSeasonDetails(ctx, candidate.TMDBID, parsed.SeasonNumber, uc.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get season details: %w", err)
	}

	// Get or create season metadata
	seasonMetadata, err := uc.getOrCreateSeasonMetadata(ctx, seriesMetadata.ID, seasonDetails)
	if err != nil {
		return nil, err
	}

	// Get or create episode metadata
	episodeMetadata, err := uc.getOrCreateEpisodeMetadata(ctx, seasonMetadata.ID, candidate.TMDBID, parsed.SeasonNumber, parsed.EpisodeNumber)
	if err != nil {
		return nil, err
	}

	// Download images if configured
	if uc.config.DownloadImages && uc.imageDownloader != nil {
		uc.downloadSeriesImages(ctx, seriesDetails, seasonDetails, parsed.SeasonNumber, parsed.EpisodeNumber)
	}

	// Link media file to episode
	media.LinkToEpisode(episodeMetadata.ID)
	media.SetEnrichmentAutoLinked()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	return &EnrichMediaFileOutput{
		MediaID:          media.ID,
		EnrichmentStatus: domain.EnrichmentStatusAutoLinked,
		MetadataType:     domain.MetadataTypeEpisode,
		CandidateCount:   1,
		AutoLinked:       true,
		Message:          fmt.Sprintf("Auto-linked to %s S%02dE%02d (%d)", candidate.Title, parsed.SeasonNumber, parsed.EpisodeNumber, candidate.ConfidenceScore),
	}, nil
}

// storeCandidates stores multiple candidates for user selection
func (uc *EnrichMediaFileUseCase) storeCandidates(ctx context.Context, media *domain.MediaFile, candidates []domain.MetadataCandidate) (*EnrichMediaFileOutput, error) {
	if len(candidates) == 0 {
		media.SetEnrichmentManualRequired()
		if err := uc.mediaRepository.Update(ctx, media); err != nil {
			return nil, fmt.Errorf("failed to update media file: %w", err)
		}

		return &EnrichMediaFileOutput{
			MediaID:          media.ID,
			EnrichmentStatus: domain.EnrichmentStatusManualRequired,
			MetadataType:     domain.MetadataTypeNone,
			CandidateCount:   0,
			AutoLinked:       false,
			Message:          "No candidates found",
		}, nil
	}

	logger.Info().
		Str("media_id", media.ID).
		Int("candidate_count", len(candidates)).
		Int("top_score", candidates[0].ConfidenceScore).
		Msg("Storing candidates for user selection")

	// Delete any existing candidates for this media file
	if err := uc.metadataCandidateRepository.DeleteByMediaFileID(ctx, media.ID); err != nil {
		return nil, fmt.Errorf("failed to delete existing candidates: %w", err)
	}

	// Store new candidates
	if err := uc.metadataCandidateRepository.CreateBatch(ctx, candidates); err != nil {
		return nil, fmt.Errorf("failed to store candidates: %w", err)
	}

	// Update media file status
	media.SetEnrichmentCandidatesFound()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	return &EnrichMediaFileOutput{
		MediaID:          media.ID,
		EnrichmentStatus: domain.EnrichmentStatusCandidatesFound,
		MetadataType:     domain.MetadataTypeNone,
		CandidateCount:   len(candidates),
		AutoLinked:       false,
		Message:          fmt.Sprintf("Found %d candidates, awaiting selection", len(candidates)),
	}, nil
}

// Helper methods for creating metadata records

func (uc *EnrichMediaFileUseCase) createMovieMetadata(details *ports.TMDBMovieDetails) *domain.MovieMetadata {
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

	// Store collection ID if the movie belongs to a collection
	if details.BelongsToCollection != nil {
		collectionID := details.BelongsToCollection.ID
		movie.CollectionID = &collectionID
	}

	return movie
}

func (uc *EnrichMediaFileUseCase) createMovieTranslation(movieID int64, details *ports.TMDBMovieDetails) *domain.MovieMetadataTranslation {
	translation := domain.NewMovieMetadataTranslation(movieID, uc.config.Language, details.Title)
	translation.Tagline = details.Tagline
	translation.Overview = details.Overview
	return translation
}

func (uc *EnrichMediaFileUseCase) getOrCreateSeriesMetadata(ctx context.Context, details *ports.TMDBSeriesDetails) (*domain.SeriesMetadata, error) {
	// Check if we already have this series
	existing, err := uc.seriesMetadataRepository.GetByTMDBID(ctx, details.ID)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing series: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// Create new series metadata
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
		// Handle race condition: if another worker created the series concurrently,
		// fetch and return the existing record instead of failing
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			existing, fetchErr := uc.seriesMetadataRepository.GetByTMDBID(ctx, details.ID)
			if fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch series after concurrent create: %w", fetchErr)
			}
			return existing, nil
		}
		return nil, fmt.Errorf("failed to create series metadata: %w", err)
	}

	// Create translation
	translation := domain.NewSeriesMetadataTranslation(series.ID, uc.config.Language, details.Name)
	translation.Overview = details.Overview
	if err := uc.seriesMetadataRepository.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create series translation: %w", err)
	}

	return series, nil
}

func (uc *EnrichMediaFileUseCase) getOrCreateSeasonMetadata(ctx context.Context, seriesID int64, details *ports.TMDBSeasonDetails) (*domain.SeasonMetadata, error) {
	// Check if we already have this season
	existing, err := uc.seasonMetadataRepository.GetBySeriesAndNumber(ctx, seriesID, details.SeasonNumber)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing season: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// Create new season metadata
	season := domain.NewSeasonMetadata(seriesID, details.ID, details.SeasonNumber)
	season.AirDate = details.AirDate
	season.PosterPath = details.PosterPath
	season.EpisodeCount = len(details.Episodes)

	if err := uc.seasonMetadataRepository.Create(ctx, season); err != nil {
		// Handle race condition: if another worker created the season concurrently,
		// fetch and return the existing record instead of failing
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			existing, fetchErr := uc.seasonMetadataRepository.GetBySeriesAndNumber(ctx, seriesID, details.SeasonNumber)
			if fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch season after concurrent create: %w", fetchErr)
			}
			return existing, nil
		}
		return nil, fmt.Errorf("failed to create season metadata: %w", err)
	}

	// Create translation
	translation := domain.NewSeasonMetadataTranslation(season.ID, uc.config.Language, details.Name)
	translation.Overview = details.Overview
	if err := uc.seasonMetadataRepository.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create season translation: %w", err)
	}

	return season, nil
}

func (uc *EnrichMediaFileUseCase) getOrCreateEpisodeMetadata(ctx context.Context, seasonID int64, seriesID int, seasonNumber int, episodeNumber int) (*domain.EpisodeMetadata, error) {
	// Check if we already have this episode
	existing, err := uc.episodeMetadataRepository.GetBySeasonAndNumber(ctx, seasonID, episodeNumber)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing episode: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// Get episode details from TMDB
	details, err := uc.tmdbClient.GetEpisodeDetails(ctx, seriesID, seasonNumber, episodeNumber, uc.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get episode details: %w", err)
	}

	// Create new episode metadata
	episode := domain.NewEpisodeMetadata(seasonID, details.ID, episodeNumber)
	episode.AirDate = details.AirDate
	episode.StillPath = details.StillPath
	episode.Runtime = details.Runtime
	episode.VoteAverage = details.VoteAverage
	episode.VoteCount = details.VoteCount

	if err := uc.episodeMetadataRepository.Create(ctx, episode); err != nil {
		// Handle race condition: if another worker created the episode concurrently,
		// fetch and return the existing record instead of failing
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			existing, fetchErr := uc.episodeMetadataRepository.GetBySeasonAndNumber(ctx, seasonID, episodeNumber)
			if fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch episode after concurrent create: %w", fetchErr)
			}
			return existing, nil
		}
		return nil, fmt.Errorf("failed to create episode metadata: %w", err)
	}

	// Create translation
	translation := domain.NewEpisodeMetadataTranslation(episode.ID, uc.config.Language, details.Name)
	translation.Overview = details.Overview
	if err := uc.episodeMetadataRepository.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create episode translation: %w", err)
	}

	return episode, nil
}

// Image download helpers

func (uc *EnrichMediaFileUseCase) downloadMovieImages(ctx context.Context, details *ports.TMDBMovieDetails) {
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

func (uc *EnrichMediaFileUseCase) downloadSeriesImages(ctx context.Context, seriesDetails *ports.TMDBSeriesDetails, seasonDetails *ports.TMDBSeasonDetails, seasonNumber, episodeNumber int) {
	// Series images
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

	// Season poster
	if seasonDetails.PosterPath != "" {
		if _, err := uc.imageDownloader.DownloadSeasonImage(ctx, seasonDetails.PosterPath, seriesDetails.ID, seasonNumber); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Int("season", seasonNumber).Msg("Failed to download season poster")
		}
	}

	// Episode still
	for _, ep := range seasonDetails.Episodes {
		if ep.EpisodeNumber == episodeNumber && ep.StillPath != "" {
			if _, err := uc.imageDownloader.DownloadEpisodeImage(ctx, ep.StillPath, seriesDetails.ID, seasonNumber, episodeNumber); err != nil {
				logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Int("season", seasonNumber).Int("episode", episodeNumber).Msg("Failed to download episode still")
			}
			break
		}
	}
}

// Utility functions

// calculateTitleSimilarity calculates a similarity score between two titles (0.0 to 1.0)
func calculateTitleSimilarity(parsedTitle, tmdbTitle string) float64 {
	// Normalize both titles
	parsed := normalizeTitle(parsedTitle)
	tmdb := normalizeTitle(tmdbTitle)

	if parsed == "" || tmdb == "" {
		return 0
	}

	// Exact match
	if parsed == tmdb {
		return 1.0
	}

	// Check if one contains the other
	if strings.Contains(parsed, tmdb) || strings.Contains(tmdb, parsed) {
		// Calculate based on length ratio
		shorter := len(parsed)
		longer := len(tmdb)
		if shorter > longer {
			shorter, longer = longer, shorter
		}
		return float64(shorter) / float64(longer)
	}

	// Calculate Levenshtein distance-based similarity
	distance := levenshteinDistance(parsed, tmdb)
	maxLen := max(len(parsed), len(tmdb))
	if maxLen == 0 {
		return 0
	}

	similarity := 1.0 - float64(distance)/float64(maxLen)
	return max(similarity, 0)
}

// normalizeTitle normalizes a title for comparison
func normalizeTitle(title string) string {
	// Convert to lowercase
	normalized := strings.ToLower(title)

	// Remove common articles and punctuation
	normalized = strings.ReplaceAll(normalized, "'", "")
	normalized = strings.ReplaceAll(normalized, "\"", "")
	normalized = strings.ReplaceAll(normalized, ":", "")
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.ReplaceAll(normalized, ".", " ")
	normalized = strings.ReplaceAll(normalized, ",", "")
	normalized = strings.ReplaceAll(normalized, "!", "")
	normalized = strings.ReplaceAll(normalized, "?", "")
	normalized = strings.ReplaceAll(normalized, "&", "and")

	// Remove "the", "a", "an" from start
	prefixes := []string{"the ", "a ", "an "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(normalized, prefix) {
			normalized = strings.TrimPrefix(normalized, prefix)
			break
		}
	}

	// Collapse multiple spaces
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}

	return strings.TrimSpace(normalized)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)

	len1 := len(r1)
	len2 := len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Create matrix
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Initialize first row and column
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 1
			if unicode.ToLower(r1[i-1]) == unicode.ToLower(r2[j-1]) {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len1][len2]
}

// extractYear extracts the year from a date string (YYYY-MM-DD format)
func extractYear(dateStr string) int {
	if len(dateStr) < 4 {
		return 0
	}
	// Parse YYYY-MM-DD format
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		// Try just the year
		t, err = time.Parse("2006", dateStr[:4])
		if err != nil {
			return 0
		}
	}
	return t.Year()
}

// sortCandidatesByScore sorts candidates by confidence score (highest first)
func sortCandidatesByScore(candidates []domain.MetadataCandidate) {
	// Simple bubble sort for small arrays
	n := len(candidates)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if candidates[j].ConfidenceScore < candidates[j+1].ConfidenceScore {
				candidates[j], candidates[j+1] = candidates[j+1], candidates[j]
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// fetchAndStoreMovieCredits fetches cast and crew from TMDB and stores them
func (uc *EnrichMediaFileUseCase) fetchAndStoreMovieCredits(ctx context.Context, movieMetadataID int64, tmdbID int) error {
	credits, err := uc.tmdbClient.GetMovieCredits(ctx, tmdbID)
	if err != nil {
		return fmt.Errorf("failed to get movie credits: %w", err)
	}

	var creditsToStore []*domain.MovieCredit

	// Store top N cast members based on config
	maxCast := uc.config.MaxCastMembers
	if maxCast <= 0 {
		maxCast = 10 // Default
	}
	castCount := min(len(credits.Cast), maxCast)
	for i := 0; i < castCount; i++ {
		cast := credits.Cast[i]
		credit := domain.NewCastCredit(
			movieMetadataID,
			cast.ID,
			cast.Name,
			cast.Character,
			cast.ProfilePath,
			cast.Order,
		)
		creditsToStore = append(creditsToStore, credit)

		// Download profile image if configured
		if uc.config.DownloadImages && uc.imageDownloader != nil && cast.ProfilePath != "" {
			if _, err := uc.imageDownloader.DownloadImage(ctx, cast.ProfilePath, ports.ImageTypeProfile, cast.ID); err != nil {
				logger.Debug().Err(err).Int("person_id", cast.ID).Msg("Failed to download profile image")
			}
		}
	}

	// Store key crew members based on defined roles and limits
	crewCounts := make(map[string]int)
	for _, crew := range credits.Crew {
		maxForJob, isRelevant := domain.MaxCrewPerJob[crew.Job]
		if !isRelevant {
			continue
		}

		if crewCounts[crew.Job] >= maxForJob {
			continue
		}

		credit := domain.NewCrewCredit(
			movieMetadataID,
			crew.ID,
			crew.Name,
			crew.Job,
			crew.Department,
			crew.ProfilePath,
		)
		creditsToStore = append(creditsToStore, credit)
		crewCounts[crew.Job]++

		// Download profile image if configured
		if uc.config.DownloadImages && uc.imageDownloader != nil && crew.ProfilePath != "" {
			if _, err := uc.imageDownloader.DownloadImage(ctx, crew.ProfilePath, ports.ImageTypeProfile, crew.ID); err != nil {
				logger.Debug().Err(err).Int("person_id", crew.ID).Msg("Failed to download profile image")
			}
		}
	}

	if len(creditsToStore) == 0 {
		return nil
	}

	if err := uc.movieCreditRepository.CreateBatch(ctx, creditsToStore); err != nil {
		return fmt.Errorf("failed to store movie credits: %w", err)
	}

	logger.Debug().
		Int64("movie_id", movieMetadataID).
		Int("cast_count", castCount).
		Int("crew_count", len(creditsToStore)-castCount).
		Msg("Stored movie credits")

	return nil
}

// fetchAndStoreCertifications fetches release dates/certifications from TMDB and stores them
func (uc *EnrichMediaFileUseCase) fetchAndStoreCertifications(ctx context.Context, movieMetadataID int64, tmdbID int) error {
	releaseDates, err := uc.tmdbClient.GetMovieReleaseDates(ctx, tmdbID)
	if err != nil {
		return fmt.Errorf("failed to get movie release dates: %w", err)
	}

	var certsToStore []*domain.MovieCertification
	seenCountries := make(map[string]bool)

	for _, country := range releaseDates.Results {
		// Skip if we already have a certification for this country
		if seenCountries[country.ISO3166_1] {
			continue
		}

		// Find the first non-empty certification for this country
		// Prefer theatrical releases (type 3) but accept any
		var certification string
		for _, rd := range country.ReleaseDates {
			if rd.Certification != "" {
				certification = rd.Certification
				if rd.Type == 3 { // Theatrical release
					break
				}
			}
		}

		if certification == "" {
			continue
		}

		cert := domain.NewMovieCertification(movieMetadataID, country.ISO3166_1, certification)
		certsToStore = append(certsToStore, cert)
		seenCountries[country.ISO3166_1] = true
	}

	if len(certsToStore) == 0 {
		return nil
	}

	if err := uc.movieCertificationRepository.CreateBatch(ctx, certsToStore); err != nil {
		return fmt.Errorf("failed to store movie certifications: %w", err)
	}

	logger.Debug().
		Int64("movie_id", movieMetadataID).
		Int("certification_count", len(certsToStore)).
		Msg("Stored movie certifications")

	return nil
}
