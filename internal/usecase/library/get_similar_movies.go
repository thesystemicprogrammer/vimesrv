package library

import (
	"context"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetSimilarMoviesInput contains the input parameters for getting similar movies
type GetSimilarMoviesInput struct {
	MovieMetadataID int64
	TMDBID          int
	Language        string
}

// GetSimilarMoviesUseCase retrieves similar movies with caching and TTL support
type GetSimilarMoviesUseCase struct {
	similarContentRepo ports.SimilarContentRepository
	movieMetadataRepo  ports.MovieMetadataRepository
	libraryRepo        ports.LibraryRepository
	tmdbClient         ports.TMDBClient
	cacheTTL           time.Duration
	maxSimilar         int
}

// NewGetSimilarMoviesUseCase creates a new GetSimilarMoviesUseCase instance
func NewGetSimilarMoviesUseCase(
	similarContentRepo ports.SimilarContentRepository,
	movieMetadataRepo ports.MovieMetadataRepository,
	libraryRepo ports.LibraryRepository,
	tmdbClient ports.TMDBClient,
	tmdbConfig config.TMDBConfig,
) *GetSimilarMoviesUseCase {
	return &GetSimilarMoviesUseCase{
		similarContentRepo: similarContentRepo,
		movieMetadataRepo:  movieMetadataRepo,
		libraryRepo:        libraryRepo,
		tmdbClient:         tmdbClient,
		cacheTTL:           time.Duration(tmdbConfig.CacheTTLHours) * time.Hour,
		maxSimilar:         tmdbConfig.SimilarContentCount,
	}
}

// Execute retrieves similar movies, fetching from TMDB if cache is expired or missing
func (uc *GetSimilarMoviesUseCase) Execute(ctx context.Context, input GetSimilarMoviesInput) ([]ports.SimilarMovieItem, error) {
	language := input.Language
	if language == "" {
		language = "en"
	}

	// Check cache freshness
	fetchedAt, err := uc.similarContentRepo.GetSimilarMoviesFetchedAt(ctx, input.MovieMetadataID)
	if err != nil {
		return nil, fmt.Errorf("check similar movies cache: %w", err)
	}

	needsFetch := fetchedAt == nil || time.Since(*fetchedAt) > uc.cacheTTL

	if needsFetch {
		// Fetch from TMDB and update cache
		if err := uc.fetchAndCacheSimilarMovies(ctx, input.MovieMetadataID, input.TMDBID, language); err != nil {
			// Log error but continue with stale cache if available
			logger.Warn().Err(err).
				Int64("movie_metadata_id", input.MovieMetadataID).
				Msg("Failed to fetch similar movies from TMDB, using cached data if available")

			// If we have no cache at all, return the error
			if fetchedAt == nil {
				return nil, fmt.Errorf("fetch similar movies: %w", err)
			}
		}
	}

	// Get from cache
	cachedMovies, err := uc.similarContentRepo.GetSimilarMovies(ctx, input.MovieMetadataID, uc.maxSimilar)
	if err != nil {
		return nil, fmt.Errorf("get similar movies from cache: %w", err)
	}

	// Get translations for the requested language
	translations, err := uc.similarContentRepo.GetSimilarMovieTranslations(ctx, input.MovieMetadataID, language)
	if err != nil {
		logger.Warn().Err(err).
			Int64("movie_metadata_id", input.MovieMetadataID).
			Str("language", language).
			Msg("Failed to get similar movie translations")
		translations = make(map[int64]string) // Continue without translations
	}

	// Convert to output format and check library presence
	result := make([]ports.SimilarMovieItem, 0, len(cachedMovies))
	for _, m := range cachedMovies {
		title := m.Title
		// Apply translation if available
		if translatedTitle, ok := translations[m.ID]; ok && translatedTitle != "" {
			title = translatedTitle
		}

		item := ports.SimilarMovieItem{
			TMDBID:      m.TMDBID,
			Title:       title,
			PosterPath:  m.PosterPath,
			ReleaseDate: m.ReleaseDate,
			Year:        extractYear(m.ReleaseDate),
			VoteAverage: m.VoteAverage,
		}

		// Check if this movie exists in our library
		if mediaID, err := uc.findMovieInLibrary(ctx, m.TMDBID, language); err == nil && mediaID != "" {
			item.InLibrary = true
			item.MediaID = mediaID
		}

		result = append(result, item)
	}

	return result, nil
}

// fetchAndCacheSimilarMovies fetches similar movies from TMDB and stores them in cache
func (uc *GetSimilarMoviesUseCase) fetchAndCacheSimilarMovies(ctx context.Context, movieMetadataID int64, tmdbID int, language string) error {
	resp, err := uc.tmdbClient.GetSimilarMovies(ctx, tmdbID, language)
	if err != nil {
		return fmt.Errorf("TMDB get similar movies: %w", err)
	}

	// Convert TMDB response to cache format (limit to configured max)
	movies := make([]ports.SimilarMovie, 0, uc.maxSimilar)
	for i, m := range resp.Results {
		if i >= uc.maxSimilar {
			break
		}
		movies = append(movies, ports.SimilarMovie{
			TMDBID:      m.ID,
			Title:       m.Title,
			PosterPath:  m.PosterPath,
			ReleaseDate: m.ReleaseDate,
			VoteAverage: m.VoteAverage,
		})
	}

	// Save to cache
	if err := uc.similarContentRepo.SaveSimilarMovies(ctx, movieMetadataID, movies); err != nil {
		return fmt.Errorf("save similar movies to cache: %w", err)
	}

	logger.Debug().
		Int64("movie_metadata_id", movieMetadataID).
		Int("count", len(movies)).
		Msg("Cached similar movies from TMDB")

	// Fetch and cache translations for each similar movie
	// We need to re-read the saved movies to get their database IDs
	savedMovies, err := uc.similarContentRepo.GetSimilarMovies(ctx, movieMetadataID, uc.maxSimilar)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to re-read saved similar movies for translation fetching")
		return nil // Don't fail the whole operation, movies are cached
	}

	uc.fetchAndCacheMovieTranslations(ctx, savedMovies)

	return nil
}

// fetchAndCacheMovieTranslations fetches translations from TMDB for similar movies and caches them
func (uc *GetSimilarMoviesUseCase) fetchAndCacheMovieTranslations(ctx context.Context, movies []ports.SimilarMovie) {
	for _, movie := range movies {
		translationsResp, err := uc.tmdbClient.GetMovieTranslations(ctx, movie.TMDBID)
		if err != nil {
			logger.Warn().Err(err).
				Int("tmdb_id", movie.TMDBID).
				Msg("Failed to fetch translations for similar movie")
			continue
		}

		// Convert TMDB translations to our format and save
		translations := make([]ports.SimilarMovieTranslation, 0, len(translationsResp.Translations))
		for _, t := range translationsResp.Translations {
			if t.Data.Title == "" {
				continue // Skip empty translations
			}
			translations = append(translations, ports.SimilarMovieTranslation{
				SimilarMovieID: movie.ID,
				Language:       t.ISO639_1,
				Title:          t.Data.Title,
			})
		}

		if len(translations) > 0 {
			if err := uc.similarContentRepo.SaveSimilarMovieTranslations(ctx, translations); err != nil {
				logger.Warn().Err(err).
					Int("tmdb_id", movie.TMDBID).
					Int("translation_count", len(translations)).
					Msg("Failed to save translations for similar movie")
			}
		}
	}
}

// findMovieInLibrary checks if a movie with the given TMDB ID exists in the library
// Returns the media ID if found, empty string otherwise
func (uc *GetSimilarMoviesUseCase) findMovieInLibrary(ctx context.Context, tmdbID int, language string) (string, error) {
	// Check if we have metadata for this TMDB ID
	metadata, err := uc.movieMetadataRepo.GetByTMDBID(ctx, tmdbID)
	if err != nil {
		return "", nil // Not found or error, treat as not in library
	}

	// Get the movie from library to find the media ID
	movies, _, err := uc.libraryRepo.ListMovies(ctx, language, 1000, 0, ports.MovieFilterOptions{})
	if err != nil {
		return "", err
	}

	for _, m := range movies {
		if m.MovieMetadataID != nil && *m.MovieMetadataID == metadata.ID {
			return m.MediaID, nil
		}
	}

	return "", nil
}

// extractYear extracts the year from a date string (YYYY-MM-DD format)
func extractYear(date string) string {
	if len(date) >= 4 {
		return date[:4]
	}
	return ""
}
