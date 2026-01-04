package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SortField defines valid sort fields for movies
type SortField string

const (
	SortByDateAdded SortField = "date_added"
	SortByTitle     SortField = "title"
	SortByYear      SortField = "year"
	SortByRating    SortField = "rating"
)

// SortOrder defines sort direction
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// ListMoviesInput contains the input parameters for listing movies
type ListMoviesInput struct {
	Language string
	Page     int
	PerPage  int

	// Sorting
	SortBy    SortField
	SortOrder SortOrder

	// Filtering
	Genres    []string // Filter by genre names (AND logic)
	YearFrom  int      // Filter movies from this year (inclusive)
	YearTo    int      // Filter movies up to this year (inclusive)
	MinRating float64  // Filter movies with rating >= this value
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
	sortBy := input.SortBy
	sortOrder := input.SortOrder

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
	if sortBy == "" {
		sortBy = SortByDateAdded
	}
	if sortOrder == "" {
		sortOrder = SortDesc
	}

	// Validate sort field
	validSortFields := map[SortField]bool{
		SortByDateAdded: true,
		SortByTitle:     true,
		SortByYear:      true,
		SortByRating:    true,
	}
	if !validSortFields[sortBy] {
		sortBy = SortByDateAdded
	}

	// Validate sort order
	if sortOrder != SortAsc && sortOrder != SortDesc {
		sortOrder = SortDesc
	}

	offset := (page - 1) * perPage

	// Build filter options
	filterOpts := ports.MovieFilterOptions{
		SortBy:    string(sortBy),
		SortOrder: string(sortOrder),
		Genres:    input.Genres,
		YearFrom:  input.YearFrom,
		YearTo:    input.YearTo,
		MinRating: input.MinRating,
	}

	movies, total, err := uc.libraryRepository.ListMovies(ctx, language, perPage, offset, filterOpts)
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
