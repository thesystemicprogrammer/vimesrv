package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ListMoviesInput contains the input parameters for listing movies
type ListMoviesInput struct {
	Language string
	Page     int
	PerPage  int
}

// ListMoviesOutput contains the paginated movie list
type ListMoviesOutput struct {
	Items   []ports.MovieSummary `json:"items"`
	Total   int                  `json:"total"`
	Page    int                  `json:"page"`
	PerPage int                  `json:"per_page"`
}

// ListMoviesUseCase retrieves a paginated list of movies with metadata
type ListMoviesUseCase struct {
	libraryRepository ports.LibraryRepository
}

// NewListMoviesUseCase creates a new ListMoviesUseCase instance
func NewListMoviesUseCase(libraryRepository ports.LibraryRepository) *ListMoviesUseCase {
	return &ListMoviesUseCase{
		libraryRepository: libraryRepository,
	}
}

// Execute retrieves a paginated list of movies with metadata
func (uc *ListMoviesUseCase) Execute(ctx context.Context, input ListMoviesInput) (*ListMoviesOutput, error) {
	// Set defaults
	page := input.Page
	perPage := input.PerPage
	language := input.Language

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	if language == "" {
		language = "en"
	}

	offset := (page - 1) * perPage

	movies, total, err := uc.libraryRepository.ListMovies(ctx, language, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list movies: %w", err)
	}

	return &ListMoviesOutput{
		Items:   movies,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}
