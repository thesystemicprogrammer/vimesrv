package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetMovieInput contains the input parameters for getting a movie
type GetMovieInput struct {
	MediaID  string
	Language string
}

// GetMovieUseCase retrieves a single movie with full details
type GetMovieUseCase struct {
	libraryRepository ports.LibraryRepository
	maxCastMembers    int
}

// NewGetMovieUseCase creates a new GetMovieUseCase instance
func NewGetMovieUseCase(libraryRepository ports.LibraryRepository, maxCastMembers int) *GetMovieUseCase {
	return &GetMovieUseCase{
		libraryRepository: libraryRepository,
		maxCastMembers:    maxCastMembers,
	}
}

// Execute retrieves a single movie with full details including credits and certification
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

	return movie, nil
}
