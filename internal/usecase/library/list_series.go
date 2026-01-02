package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ListSeriesInput contains the input parameters for listing series
type ListSeriesInput struct {
	Language     string
	IncludeEmpty bool // Include series with no available episodes
}

// ListSeriesOutput contains the series list
type ListSeriesOutput struct {
	Items []ports.SeriesSummary `json:"items"`
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

// Execute retrieves a list of series with metadata
func (uc *ListSeriesUseCase) Execute(ctx context.Context, input ListSeriesInput) (*ListSeriesOutput, error) {
	language := input.Language
	if language == "" {
		language = "en"
	}

	series, err := uc.libraryRepository.ListSeries(ctx, language, input.IncludeEmpty)
	if err != nil {
		return nil, fmt.Errorf("failed to list series: %w", err)
	}

	return &ListSeriesOutput{
		Items: series,
	}, nil
}
