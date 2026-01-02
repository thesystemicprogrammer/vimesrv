package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetMovieInput contains the input parameters for getting a movie
type GetMovieInput struct {
	MediaID  string
	Language string
}

// GetMovieUseCase retrieves a single movie with full details
type GetMovieUseCase struct {
	libraryRepository      ports.LibraryRepository
	getSimilarMoviesUC     *GetSimilarMoviesUseCase
	getMovieCollectionUC   *GetMovieCollectionUseCase
	maxCastMembers         int
	fetchSimilarOnGetMovie bool
}

// NewGetMovieUseCase creates a new GetMovieUseCase instance
func NewGetMovieUseCase(
	libraryRepository ports.LibraryRepository,
	getSimilarMoviesUC *GetSimilarMoviesUseCase,
	getMovieCollectionUC *GetMovieCollectionUseCase,
	maxCastMembers int,
) *GetMovieUseCase {
	return &GetMovieUseCase{
		libraryRepository:      libraryRepository,
		getSimilarMoviesUC:     getSimilarMoviesUC,
		getMovieCollectionUC:   getMovieCollectionUC,
		maxCastMembers:         maxCastMembers,
		fetchSimilarOnGetMovie: getSimilarMoviesUC != nil,
	}
}

// Execute retrieves a single movie with full details including credits, certification, similar movies, and collection
func (uc *GetMovieUseCase) Execute(ctx context.Context, input GetMovieInput) (*ports.MovieDetail, error) {
	language := input.Language
	if language == "" {
		language = "en"
	}

	movie, err := uc.libraryRepository.GetMovieDetail(ctx, input.MediaID, language, uc.maxCastMembers)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	if movie == nil {
		return nil, shared.ErrNotFound
	}

	// Fetch similar movies if enabled and movie has TMDB metadata
	if uc.fetchSimilarOnGetMovie && movie.MovieMetadataID != nil && movie.TMDBID > 0 {
		similarMovies, err := uc.getSimilarMoviesUC.Execute(ctx, GetSimilarMoviesInput{
			MovieMetadataID: *movie.MovieMetadataID,
			TMDBID:          movie.TMDBID,
			Language:        language,
		})
		if err != nil {
			// Log but don't fail the request - similar movies are optional
			logger.Warn().Err(err).
				Str("media_id", input.MediaID).
				Msg("Failed to fetch similar movies")
		} else {
			movie.SimilarMovies = similarMovies
		}
	}

	// Fetch collection info if movie belongs to a collection
	if uc.getMovieCollectionUC != nil && movie.CollectionID != nil && movie.TMDBID > 0 {
		collection, err := uc.getMovieCollectionUC.Execute(ctx, GetMovieCollectionInput{
			CollectionID:  *movie.CollectionID,
			CurrentTMDBID: movie.TMDBID,
			Language:      language,
		})
		if err != nil {
			// Log but don't fail the request - collection is optional
			logger.Warn().Err(err).
				Str("media_id", input.MediaID).
				Int("collection_id", *movie.CollectionID).
				Msg("Failed to fetch collection")
		} else {
			movie.Collection = collection
		}
	}

	return movie, nil
}
