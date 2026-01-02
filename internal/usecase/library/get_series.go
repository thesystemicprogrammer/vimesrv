package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetSeriesInput contains the input parameters for getting a series
type GetSeriesInput struct {
	SeriesID int64
	Language string
}

// GetSeriesUseCase retrieves a single series with all seasons and episodes
type GetSeriesUseCase struct {
	libraryRepository ports.LibraryRepository
}

// NewGetSeriesUseCase creates a new GetSeriesUseCase instance
func NewGetSeriesUseCase(libraryRepository ports.LibraryRepository) *GetSeriesUseCase {
	return &GetSeriesUseCase{
		libraryRepository: libraryRepository,
	}
}

// Execute retrieves a single series with all seasons and episodes
func (uc *GetSeriesUseCase) Execute(ctx context.Context, input GetSeriesInput) (*ports.SeriesDetail, error) {
	language := input.Language
	if language == "" {
		language = "en"
	}

	series, err := uc.libraryRepository.GetSeriesDetail(ctx, input.SeriesID, language)
	if err != nil {
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	if series == nil {
		return nil, shared.ErrNotFound
	}

	return series, nil
}
