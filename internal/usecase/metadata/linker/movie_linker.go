package linker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// MovieLinker handles creating and linking movie metadata from TMDB
type MovieLinker struct {
	config                       config.TMDBConfig
	tmdbClient                   ports.TMDBClient
	imageDownloader              ports.ImageDownloader
	movieMetadataRepository      ports.MovieMetadataRepository
	movieCreditRepository        ports.MovieCreditRepository
	movieCertificationRepository ports.MovieCertificationRepository
}

// NewMovieLinker creates a new MovieLinker instance
func NewMovieLinker(
	config config.TMDBConfig,
	tmdbClient ports.TMDBClient,
	imageDownloader ports.ImageDownloader,
	movieMetadataRepository ports.MovieMetadataRepository,
	movieCreditRepository ports.MovieCreditRepository,
	movieCertificationRepository ports.MovieCertificationRepository,
) *MovieLinker {
	return &MovieLinker{
		config:                       config,
		tmdbClient:                   tmdbClient,
		imageDownloader:              imageDownloader,
		movieMetadataRepository:      movieMetadataRepository,
		movieCreditRepository:        movieCreditRepository,
		movieCertificationRepository: movieCertificationRepository,
	}
}

// LinkResult contains the result of a movie linking operation
type MovieLinkResult struct {
	MovieMetadata *domain.MovieMetadata
	Details       *ports.TMDBMovieDetails
	Created       bool // true if newly created, false if existing was reused
}

// Link fetches movie metadata from TMDB and creates/retrieves the local record
// It also fetches credits, certifications, and downloads images if configured.
// Returns the movie metadata record (existing or newly created).
func (l *MovieLinker) Link(ctx context.Context, tmdbID int) (*MovieLinkResult, error) {
	logger.Debug().Int("tmdb_id", tmdbID).Msg("Linking movie metadata")

	// Get detailed movie info from TMDB
	details, err := l.tmdbClient.GetMovieDetails(ctx, tmdbID, l.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie details: %w", err)
	}

	// Check if we already have this movie in our database
	existingMovie, err := l.movieMetadataRepository.GetByTMDBID(ctx, tmdbID)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing movie: %w", err)
	}

	if existingMovie != nil {
		logger.Debug().
			Int64("movie_id", existingMovie.ID).
			Int("tmdb_id", tmdbID).
			Msg("Reusing existing movie metadata")

		// Still download images in case they were missing
		l.downloadImages(ctx, details)

		return &MovieLinkResult{
			MovieMetadata: existingMovie,
			Details:       details,
			Created:       false,
		}, nil
	}

	// Create new movie metadata
	movieMetadata := l.createMovieMetadata(details)
	if err := l.movieMetadataRepository.Create(ctx, movieMetadata); err != nil {
		// Handle race condition
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			existing, fetchErr := l.movieMetadataRepository.GetByTMDBID(ctx, tmdbID)
			if fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch movie after concurrent create: %w", fetchErr)
			}
			return &MovieLinkResult{
				MovieMetadata: existing,
				Details:       details,
				Created:       false,
			}, nil
		}
		return nil, fmt.Errorf("failed to create movie metadata: %w", err)
	}

	// Create translation for configured language
	translation := l.createMovieTranslation(movieMetadata.ID, details)
	if err := l.movieMetadataRepository.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create movie translation: %w", err)
	}

	// Fetch and store movie credits
	if err := l.fetchAndStoreCredits(ctx, movieMetadata.ID, tmdbID); err != nil {
		logger.Warn().Err(err).Int("tmdb_id", tmdbID).Msg("Failed to fetch movie credits")
	}

	// Fetch and store movie certifications
	if err := l.fetchAndStoreCertifications(ctx, movieMetadata.ID, tmdbID); err != nil {
		logger.Warn().Err(err).Int("tmdb_id", tmdbID).Msg("Failed to fetch movie certifications")
	}

	// Download images if configured
	l.downloadImages(ctx, details)

	logger.Debug().
		Int64("movie_id", movieMetadata.ID).
		Int("tmdb_id", tmdbID).
		Msg("Created new movie metadata")

	return &MovieLinkResult{
		MovieMetadata: movieMetadata,
		Details:       details,
		Created:       true,
	}, nil
}

// createMovieMetadata creates a MovieMetadata domain object from TMDB details
func (l *MovieLinker) createMovieMetadata(details *ports.TMDBMovieDetails) *domain.MovieMetadata {
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

// createMovieTranslation creates a MovieMetadataTranslation from TMDB details
func (l *MovieLinker) createMovieTranslation(movieID int64, details *ports.TMDBMovieDetails) *domain.MovieMetadataTranslation {
	translation := domain.NewMovieMetadataTranslation(movieID, l.config.Language, details.Title)
	translation.Tagline = details.Tagline
	translation.Overview = details.Overview
	return translation
}

// fetchAndStoreCredits fetches cast and crew from TMDB and stores them
func (l *MovieLinker) fetchAndStoreCredits(ctx context.Context, movieMetadataID int64, tmdbID int) error {
	credits, err := l.tmdbClient.GetMovieCredits(ctx, tmdbID)
	if err != nil {
		return fmt.Errorf("failed to get movie credits: %w", err)
	}

	var creditsToStore []*domain.MovieCredit

	// Store top N cast members based on config
	maxCast := l.config.MaxCastMembers
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
		if l.config.DownloadImages && l.imageDownloader != nil && cast.ProfilePath != "" {
			if _, err := l.imageDownloader.DownloadImage(ctx, cast.ProfilePath, ports.ImageTypeProfile, cast.ID); err != nil {
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
		if l.config.DownloadImages && l.imageDownloader != nil && crew.ProfilePath != "" {
			if _, err := l.imageDownloader.DownloadImage(ctx, crew.ProfilePath, ports.ImageTypeProfile, crew.ID); err != nil {
				logger.Debug().Err(err).Int("person_id", crew.ID).Msg("Failed to download profile image")
			}
		}
	}

	if len(creditsToStore) == 0 {
		return nil
	}

	if err := l.movieCreditRepository.CreateBatch(ctx, creditsToStore); err != nil {
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
func (l *MovieLinker) fetchAndStoreCertifications(ctx context.Context, movieMetadataID int64, tmdbID int) error {
	releaseDates, err := l.tmdbClient.GetMovieReleaseDates(ctx, tmdbID)
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

	if err := l.movieCertificationRepository.CreateBatch(ctx, certsToStore); err != nil {
		return fmt.Errorf("failed to store movie certifications: %w", err)
	}

	logger.Debug().
		Int64("movie_id", movieMetadataID).
		Int("certification_count", len(certsToStore)).
		Msg("Stored movie certifications")

	return nil
}

// downloadImages downloads movie poster and backdrop images
func (l *MovieLinker) downloadImages(ctx context.Context, details *ports.TMDBMovieDetails) {
	if !l.config.DownloadImages || l.imageDownloader == nil {
		return
	}

	if details.PosterPath != "" {
		if _, err := l.imageDownloader.DownloadImage(ctx, details.PosterPath, ports.ImageTypeMoviePoster, details.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", details.ID).Msg("Failed to download movie poster")
		}
	}
	if details.BackdropPath != "" {
		if _, err := l.imageDownloader.DownloadImage(ctx, details.BackdropPath, ports.ImageTypeMovieBackdrop, details.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", details.ID).Msg("Failed to download movie backdrop")
		}
	}
}
