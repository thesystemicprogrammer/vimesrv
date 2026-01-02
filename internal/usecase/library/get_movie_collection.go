package library

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetMovieCollectionInput contains the input parameters for getting collection info
type GetMovieCollectionInput struct {
	CollectionID  int
	CurrentTMDBID int // TMDB ID of the current movie being viewed
	Language      string
}

// GetMovieCollectionUseCase retrieves movie collection info with caching and TTL support
type GetMovieCollectionUseCase struct {
	collectionRepo    ports.CollectionRepository
	movieMetadataRepo ports.MovieMetadataRepository
	libraryRepo       ports.LibraryRepository
	tmdbClient        ports.TMDBClient
	cacheTTL          time.Duration
}

// NewGetMovieCollectionUseCase creates a new GetMovieCollectionUseCase instance
func NewGetMovieCollectionUseCase(
	collectionRepo ports.CollectionRepository,
	movieMetadataRepo ports.MovieMetadataRepository,
	libraryRepo ports.LibraryRepository,
	tmdbClient ports.TMDBClient,
	tmdbConfig config.TMDBConfig,
) *GetMovieCollectionUseCase {
	return &GetMovieCollectionUseCase{
		collectionRepo:    collectionRepo,
		movieMetadataRepo: movieMetadataRepo,
		libraryRepo:       libraryRepo,
		tmdbClient:        tmdbClient,
		cacheTTL:          time.Duration(tmdbConfig.CacheTTLHours) * time.Hour,
	}
}

// Execute retrieves collection info, fetching from TMDB if cache is expired or missing
func (uc *GetMovieCollectionUseCase) Execute(ctx context.Context, input GetMovieCollectionInput) (*ports.MovieCollectionInfo, error) {
	language := input.Language
	if language == "" {
		language = "en"
	}

	// Check cache freshness for metadata
	metadata, err := uc.collectionRepo.GetCollectionMetadata(ctx, input.CollectionID)
	if err != nil {
		return nil, fmt.Errorf("check collection cache: %w", err)
	}

	needsFetch := metadata == nil || time.Since(metadata.FetchedAt) > uc.cacheTTL

	if needsFetch {
		// Fetch from TMDB and update cache
		if err := uc.fetchAndCacheCollection(ctx, input.CollectionID, language); err != nil {
			// Log error but continue with stale cache if available
			logger.Warn().Err(err).
				Int("collection_id", input.CollectionID).
				Msg("Failed to fetch collection from TMDB, using cached data if available")

			// If we have no cache at all, return the error
			if metadata == nil {
				return nil, fmt.Errorf("fetch collection: %w", err)
			}
		}

		// Re-fetch metadata after cache update
		metadata, err = uc.collectionRepo.GetCollectionMetadata(ctx, input.CollectionID)
		if err != nil {
			return nil, fmt.Errorf("get collection metadata after update: %w", err)
		}
	}

	if metadata == nil {
		return nil, nil // No collection data available
	}

	// Get translation if language is not English
	name := metadata.Name
	overview := metadata.Overview
	if language != "en" {
		translation, err := uc.collectionRepo.GetCollectionTranslation(ctx, input.CollectionID, language)
		if err != nil {
			logger.Warn().Err(err).
				Int("collection_id", input.CollectionID).
				Str("language", language).
				Msg("Failed to get collection translation")
		} else if translation != nil {
			if translation.Name != "" {
				name = translation.Name
			}
			if translation.Overview != "" {
				overview = translation.Overview
			}
		}
	}

	// Get movies in collection
	cachedMovies, err := uc.collectionRepo.GetCollectionMovies(ctx, input.CollectionID)
	if err != nil {
		return nil, fmt.Errorf("get collection movies from cache: %w", err)
	}

	// Get movie translations for the requested language
	movieTranslations, err := uc.collectionRepo.GetCollectionMovieTranslations(ctx, input.CollectionID, language)
	if err != nil {
		logger.Warn().Err(err).
			Int("collection_id", input.CollectionID).
			Str("language", language).
			Msg("Failed to get collection movie translations")
		movieTranslations = make(map[int64]string) // Continue without translations
	}

	// Sort by release date
	sort.Slice(cachedMovies, func(i, j int) bool {
		return cachedMovies[i].ReleaseDate < cachedMovies[j].ReleaseDate
	})

	// Convert to display format and check library presence
	movies := make([]ports.CollectionMovieDisplay, 0, len(cachedMovies))
	position := 0
	for i, m := range cachedMovies {
		title := m.Title
		// Apply movie translation if available
		if translatedTitle, ok := movieTranslations[m.ID]; ok && translatedTitle != "" {
			title = translatedTitle
		}

		item := ports.CollectionMovieDisplay{
			TMDBID:      m.TMDBMovieID,
			Title:       title,
			PosterPath:  m.PosterPath,
			ReleaseDate: m.ReleaseDate,
			Year:        extractYear(m.ReleaseDate),
			VoteAverage: m.VoteAverage,
			IsCurrent:   m.TMDBMovieID == input.CurrentTMDBID,
		}

		if item.IsCurrent {
			position = i + 1 // 1-based position
		}

		// Check if this movie exists in our library
		if mediaID, err := uc.findMovieInLibrary(ctx, m.TMDBMovieID, language); err == nil && mediaID != "" {
			item.InLibrary = true
			item.MediaID = mediaID
		}

		movies = append(movies, item)
	}

	return &ports.MovieCollectionInfo{
		CollectionID: input.CollectionID,
		Name:         name,
		Overview:     overview,
		PosterPath:   metadata.PosterPath,
		BackdropPath: metadata.BackdropPath,
		Movies:       movies,
		Position:     position,
		TotalMovies:  len(movies),
	}, nil
}

// fetchAndCacheCollection fetches collection data from TMDB and stores it in cache
func (uc *GetMovieCollectionUseCase) fetchAndCacheCollection(ctx context.Context, collectionID int, language string) error {
	// Fetch collection details
	details, err := uc.tmdbClient.GetCollectionDetails(ctx, collectionID, language)
	if err != nil {
		return fmt.Errorf("TMDB get collection details: %w", err)
	}

	// Save metadata
	metadata := &ports.CollectionMetadata{
		CollectionID: collectionID,
		Name:         details.Name,
		Overview:     details.Overview,
		PosterPath:   details.PosterPath,
		BackdropPath: details.BackdropPath,
	}
	if err := uc.collectionRepo.SaveCollectionMetadata(ctx, metadata); err != nil {
		return fmt.Errorf("save collection metadata: %w", err)
	}

	// Convert and save movies
	movies := make([]ports.CollectionMovieItem, 0, len(details.Parts))
	for i, part := range details.Parts {
		movies = append(movies, ports.CollectionMovieItem{
			CollectionID:  collectionID,
			TMDBMovieID:   part.ID,
			Title:         part.Title,
			OriginalTitle: part.OriginalTitle,
			PosterPath:    part.PosterPath,
			ReleaseDate:   part.ReleaseDate,
			VoteAverage:   part.VoteAverage,
			DisplayOrder:  i,
		})
	}
	if err := uc.collectionRepo.SaveCollectionMovies(ctx, collectionID, movies); err != nil {
		return fmt.Errorf("save collection movies: %w", err)
	}

	// Fetch and save collection translation if not English
	if language != "en" {
		if err := uc.fetchAndCacheTranslation(ctx, collectionID, language); err != nil {
			logger.Warn().Err(err).
				Int("collection_id", collectionID).
				Str("language", language).
				Msg("Failed to fetch collection translation")
			// Don't fail the whole operation for translation failure
		}
	}

	// Fetch and cache translations for each movie in the collection
	// We need to re-read the saved movies to get their database IDs
	savedMovies, err := uc.collectionRepo.GetCollectionMovies(ctx, collectionID)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to re-read saved collection movies for translation fetching")
	} else {
		uc.fetchAndCacheMovieTranslations(ctx, savedMovies)
	}

	logger.Debug().
		Int("collection_id", collectionID).
		Int("movie_count", len(movies)).
		Msg("Cached collection from TMDB")

	return nil
}

// fetchAndCacheMovieTranslations fetches translations from TMDB for collection movies and caches them
func (uc *GetMovieCollectionUseCase) fetchAndCacheMovieTranslations(ctx context.Context, movies []ports.CollectionMovieItem) {
	for _, movie := range movies {
		translationsResp, err := uc.tmdbClient.GetMovieTranslations(ctx, movie.TMDBMovieID)
		if err != nil {
			logger.Warn().Err(err).
				Int("tmdb_id", movie.TMDBMovieID).
				Msg("Failed to fetch translations for collection movie")
			continue
		}

		// Convert TMDB translations to our format and save
		translations := make([]ports.CollectionMovieTranslation, 0, len(translationsResp.Translations))
		for _, t := range translationsResp.Translations {
			if t.Data.Title == "" {
				continue // Skip empty translations
			}
			translations = append(translations, ports.CollectionMovieTranslation{
				CollectionMovieID: movie.ID,
				Language:          t.ISO639_1,
				Title:             t.Data.Title,
			})
		}

		if len(translations) > 0 {
			if err := uc.collectionRepo.SaveCollectionMovieTranslations(ctx, translations); err != nil {
				logger.Warn().Err(err).
					Int("tmdb_id", movie.TMDBMovieID).
					Int("translation_count", len(translations)).
					Msg("Failed to save translations for collection movie")
			}
		}
	}
}

// fetchAndCacheTranslation fetches and caches a specific translation
func (uc *GetMovieCollectionUseCase) fetchAndCacheTranslation(ctx context.Context, collectionID int, language string) error {
	resp, err := uc.tmdbClient.GetCollectionTranslations(ctx, collectionID)
	if err != nil {
		return fmt.Errorf("TMDB get collection translations: %w", err)
	}

	// Find the matching translation
	for _, t := range resp.Translations {
		if t.ISO639_1 == language {
			translation := &ports.CollectionTranslation{
				CollectionID: collectionID,
				Language:     language,
				Name:         t.Data.Title,
				Overview:     t.Data.Overview,
			}
			if err := uc.collectionRepo.SaveCollectionTranslation(ctx, translation); err != nil {
				return fmt.Errorf("save collection translation: %w", err)
			}
			return nil
		}
	}

	// No translation found for this language - save empty translation to prevent repeated lookups
	translation := &ports.CollectionTranslation{
		CollectionID: collectionID,
		Language:     language,
		Name:         "",
		Overview:     "",
	}
	return uc.collectionRepo.SaveCollectionTranslation(ctx, translation)
}

// findMovieInLibrary checks if a movie with the given TMDB ID exists in the library
// Returns the media ID if found, empty string otherwise
func (uc *GetMovieCollectionUseCase) findMovieInLibrary(ctx context.Context, tmdbID int, language string) (string, error) {
	// Check if we have metadata for this TMDB ID
	metadata, err := uc.movieMetadataRepo.GetByTMDBID(ctx, tmdbID)
	if err != nil {
		return "", nil // Not found or error, treat as not in library
	}

	// Get the movie from library to find the media ID
	movies, _, err := uc.libraryRepo.ListMovies(ctx, language, 1000, 0)
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
