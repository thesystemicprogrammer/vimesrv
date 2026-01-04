package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SeriesSortField defines valid sort fields for series
type SeriesSortField string

const (
	SeriesSortByDateAdded SeriesSortField = "date_added"
	SeriesSortByName      SeriesSortField = "name"
	SeriesSortByYear      SeriesSortField = "year"
	SeriesSortByRating    SeriesSortField = "rating"
)

// ListSeriesInput contains the input parameters for listing series
type ListSeriesInput struct {
	Language     string
	IncludeEmpty bool // Include series with no available episodes
	Page         int
	PerPage      int

	// Sorting
	SortBy    SeriesSortField
	SortOrder SortOrder

	// Filtering
	Genres    []string // Filter by genre names (AND logic)
	YearFrom  int      // Filter series from this year (inclusive)
	YearTo    int      // Filter series up to this year (inclusive)
	MinRating float64  // Filter series with rating >= this value
}

// ListSeriesOutput contains the paginated series list
type ListSeriesOutput struct {
	Items   []ports.SeriesSummary `json:"items"`
	Total   int                   `json:"total"`
	Page    int                   `json:"page"`
	PerPage int                   `json:"per_page"`
}

// ListSeriesUseCase retrieves a list of series with metadata
type ListSeriesUseCase struct {
	libraryRepository ports.LibraryRepository
}

// NewListSeriesUseCase creates a new ListSeriesUseCase instance
func NewListSeriesUseCase(libraryRepository ports.LibraryRepository) *ListSeriesUseCase {
	return &ListSeriesUseCase{
		libraryRepository: libraryRepository,
	}
}

// Execute retrieves a paginated list of series with metadata
func (uc *ListSeriesUseCase) Execute(ctx context.Context, input ListSeriesInput) (*ListSeriesOutput, error) {
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
		sortBy = SeriesSortByDateAdded
	}
	if sortOrder == "" {
		sortOrder = SortDesc
	}

	// Validate sort field
	validSortFields := map[SeriesSortField]bool{
		SeriesSortByDateAdded: true,
		SeriesSortByName:      true,
		SeriesSortByYear:      true,
		SeriesSortByRating:    true,
	}
	if !validSortFields[sortBy] {
		sortBy = SeriesSortByDateAdded
	}

	// Validate sort order
	if sortOrder != SortAsc && sortOrder != SortDesc {
		sortOrder = SortDesc
	}

	offset := (page - 1) * perPage

	// Build filter options
	filterOpts := ports.SeriesFilterOptions{
		SortBy:    string(sortBy),
		SortOrder: string(sortOrder),
		Genres:    input.Genres,
		YearFrom:  input.YearFrom,
		YearTo:    input.YearTo,
		MinRating: input.MinRating,
	}

	series, total, err := uc.libraryRepository.ListSeries(ctx, language, input.IncludeEmpty, perPage, offset, filterOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list series: %w", err)
	}

	return &ListSeriesOutput{
		Items:   series,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}
