package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ListGenresOutput contains the list of available genres
type ListGenresOutput struct {
	MovieGenres  []string `json:"movie_genres"`
	SeriesGenres []string `json:"series_genres"`
}

// ListGenresUseCase retrieves all available genres from the library
type ListGenresUseCase struct {
	libraryRepository ports.LibraryRepository
}

// NewListGenresUseCase creates a new ListGenresUseCase instance
func NewListGenresUseCase(libraryRepository ports.LibraryRepository) *ListGenresUseCase {
	return &ListGenresUseCase{
		libraryRepository: libraryRepository,
	}
}

// Execute retrieves all available genres from the library
func (uc *ListGenresUseCase) Execute(ctx context.Context) (*ListGenresOutput, error) {
	movieGenres, err := uc.libraryRepository.ListMovieGenres(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list movie genres: %w", err)
	}

	seriesGenres, err := uc.libraryRepository.ListSeriesGenres(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list series genres: %w", err)
	}

	return &ListGenresOutput{
		MovieGenres:  movieGenres,
		SeriesGenres: seriesGenres,
	}, nil
}
