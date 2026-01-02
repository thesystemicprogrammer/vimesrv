package library

import (
	"context"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetSimilarSeriesInput contains the input parameters for getting similar series
type GetSimilarSeriesInput struct {
	SeriesMetadataID int64
	TMDBID           int
	Language         string
}

// GetSimilarSeriesUseCase retrieves similar series with caching and TTL support
type GetSimilarSeriesUseCase struct {
	similarContentRepo ports.SimilarContentRepository
	seriesMetadataRepo ports.SeriesMetadataRepository
	libraryRepo        ports.LibraryRepository
	tmdbClient         ports.TMDBClient
	cacheTTL           time.Duration
	maxSimilar         int
}

// NewGetSimilarSeriesUseCase creates a new GetSimilarSeriesUseCase instance
func NewGetSimilarSeriesUseCase(
	similarContentRepo ports.SimilarContentRepository,
	seriesMetadataRepo ports.SeriesMetadataRepository,
	libraryRepo ports.LibraryRepository,
	tmdbClient ports.TMDBClient,
	tmdbConfig config.TMDBConfig,
) *GetSimilarSeriesUseCase {
	return &GetSimilarSeriesUseCase{
		similarContentRepo: similarContentRepo,
		seriesMetadataRepo: seriesMetadataRepo,
		libraryRepo:        libraryRepo,
		tmdbClient:         tmdbClient,
		cacheTTL:           time.Duration(tmdbConfig.CacheTTLHours) * time.Hour,
		maxSimilar:         tmdbConfig.SimilarContentCount,
	}
}

// Execute retrieves similar series, fetching from TMDB if cache is expired or missing
func (uc *GetSimilarSeriesUseCase) Execute(ctx context.Context, input GetSimilarSeriesInput) ([]ports.SimilarSeriesItem, error) {
	language := input.Language
	if language == "" {
		language = "en"
	}

	// Check cache freshness
	fetchedAt, err := uc.similarContentRepo.GetSimilarSeriesFetchedAt(ctx, input.SeriesMetadataID)
	if err != nil {
		return nil, fmt.Errorf("check similar series cache: %w", err)
	}

	needsFetch := fetchedAt == nil || time.Since(*fetchedAt) > uc.cacheTTL

	if needsFetch {
		// Fetch from TMDB and update cache
		if err := uc.fetchAndCacheSimilarSeries(ctx, input.SeriesMetadataID, input.TMDBID, language); err != nil {
			// Log error but continue with stale cache if available
			logger.Warn().Err(err).
				Int64("series_metadata_id", input.SeriesMetadataID).
				Msg("Failed to fetch similar series from TMDB, using cached data if available")

			// If we have no cache at all, return the error
			if fetchedAt == nil {
				return nil, fmt.Errorf("fetch similar series: %w", err)
			}
		}
	}

	// Get from cache
	cachedSeries, err := uc.similarContentRepo.GetSimilarSeries(ctx, input.SeriesMetadataID, uc.maxSimilar)
	if err != nil {
		return nil, fmt.Errorf("get similar series from cache: %w", err)
	}

	// Get translations for the requested language
	translations, err := uc.similarContentRepo.GetSimilarSeriesTranslations(ctx, input.SeriesMetadataID, language)
	if err != nil {
		logger.Warn().Err(err).
			Int64("series_metadata_id", input.SeriesMetadataID).
			Str("language", language).
			Msg("Failed to get similar series translations")
		translations = make(map[int64]string) // Continue without translations
	}

	// Convert to output format and check library presence
	result := make([]ports.SimilarSeriesItem, 0, len(cachedSeries))
	for _, s := range cachedSeries {
		name := s.Name
		// Apply translation if available
		if translatedName, ok := translations[s.ID]; ok && translatedName != "" {
			name = translatedName
		}

		item := ports.SimilarSeriesItem{
			TMDBID:       s.TMDBID,
			Name:         name,
			PosterPath:   s.PosterPath,
			FirstAirDate: s.FirstAirDate,
			Year:         extractYear(s.FirstAirDate),
			VoteAverage:  s.VoteAverage,
		}

		// Check if this series exists in our library
		if seriesID, err := uc.findSeriesInLibrary(ctx, s.TMDBID); err == nil && seriesID != 0 {
			item.InLibrary = true
			item.SeriesMetadataID = seriesID
		}

		result = append(result, item)
	}

	return result, nil
}

// fetchAndCacheSimilarSeries fetches similar series from TMDB and stores them in cache
func (uc *GetSimilarSeriesUseCase) fetchAndCacheSimilarSeries(ctx context.Context, seriesMetadataID int64, tmdbID int, language string) error {
	resp, err := uc.tmdbClient.GetSimilarSeries(ctx, tmdbID, language)
	if err != nil {
		return fmt.Errorf("TMDB get similar series: %w", err)
	}

	// Convert TMDB response to cache format (limit to configured max)
	series := make([]ports.SimilarSeries, 0, uc.maxSimilar)
	for i, s := range resp.Results {
		if i >= uc.maxSimilar {
			break
		}
		series = append(series, ports.SimilarSeries{
			TMDBID:       s.ID,
			Name:         s.Name,
			PosterPath:   s.PosterPath,
			FirstAirDate: s.FirstAirDate,
			VoteAverage:  s.VoteAverage,
		})
	}

	// Save to cache
	if err := uc.similarContentRepo.SaveSimilarSeries(ctx, seriesMetadataID, series); err != nil {
		return fmt.Errorf("save similar series to cache: %w", err)
	}

	logger.Debug().
		Int64("series_metadata_id", seriesMetadataID).
		Int("count", len(series)).
		Msg("Cached similar series from TMDB")

	// Fetch and cache translations for each similar series
	// We need to re-read the saved series to get their database IDs
	savedSeries, err := uc.similarContentRepo.GetSimilarSeries(ctx, seriesMetadataID, uc.maxSimilar)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to re-read saved similar series for translation fetching")
		return nil // Don't fail the whole operation, series are cached
	}

	uc.fetchAndCacheSeriesTranslations(ctx, savedSeries)

	return nil
}

// fetchAndCacheSeriesTranslations fetches translations from TMDB for similar series and caches them
func (uc *GetSimilarSeriesUseCase) fetchAndCacheSeriesTranslations(ctx context.Context, series []ports.SimilarSeries) {
	for _, s := range series {
		translationsResp, err := uc.tmdbClient.GetSeriesTranslations(ctx, s.TMDBID)
		if err != nil {
			logger.Warn().Err(err).
				Int("tmdb_id", s.TMDBID).
				Msg("Failed to fetch translations for similar series")
			continue
		}

		// Convert TMDB translations to our format and save
		translations := make([]ports.SimilarSeriesTranslation, 0, len(translationsResp.Translations))
		for _, t := range translationsResp.Translations {
			if t.Data.Name == "" {
				continue // Skip empty translations
			}
			translations = append(translations, ports.SimilarSeriesTranslation{
				SimilarSeriesID: s.ID,
				Language:        t.ISO639_1,
				Name:            t.Data.Name,
			})
		}

		if len(translations) > 0 {
			if err := uc.similarContentRepo.SaveSimilarSeriesTranslations(ctx, translations); err != nil {
				logger.Warn().Err(err).
					Int("tmdb_id", s.TMDBID).
					Int("translation_count", len(translations)).
					Msg("Failed to save translations for similar series")
			}
		}
	}
}

// findSeriesInLibrary checks if a series with the given TMDB ID exists in the library
// Returns the series metadata ID if found, 0 otherwise
func (uc *GetSimilarSeriesUseCase) findSeriesInLibrary(ctx context.Context, tmdbID int) (int64, error) {
	// Check if we have metadata for this TMDB ID
	metadata, err := uc.seriesMetadataRepo.GetByTMDBID(ctx, tmdbID)
	if err != nil {
		return 0, nil // Not found or error, treat as not in library
	}

	return metadata.ID, nil
}
