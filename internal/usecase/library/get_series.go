package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetSeriesInput contains the input parameters for getting a series
type GetSeriesInput struct {
	SeriesID int64
	Language string
}

// GetSeriesUseCase retrieves a single series with all seasons and episodes
type GetSeriesUseCase struct {
	libraryRepository       ports.LibraryRepository
	getSimilarSeriesUC      *GetSimilarSeriesUseCase
	fetchSimilarOnGetSeries bool
}

// NewGetSeriesUseCase creates a new GetSeriesUseCase instance
func NewGetSeriesUseCase(
	libraryRepository ports.LibraryRepository,
	getSimilarSeriesUC *GetSimilarSeriesUseCase,
) *GetSeriesUseCase {
	return &GetSeriesUseCase{
		libraryRepository:       libraryRepository,
		getSimilarSeriesUC:      getSimilarSeriesUC,
		fetchSimilarOnGetSeries: getSimilarSeriesUC != nil,
	}
}

// Execute retrieves a single series with all seasons, episodes, and similar series
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

	// Fetch similar series if enabled and series has TMDB metadata
	if uc.fetchSimilarOnGetSeries && series.SeriesMetadataID > 0 && series.TMDBID > 0 {
		similarSeries, err := uc.getSimilarSeriesUC.Execute(ctx, GetSimilarSeriesInput{
			SeriesMetadataID: series.SeriesMetadataID,
			TMDBID:           series.TMDBID,
			Language:         language,
		})
		if err != nil {
			// Log but don't fail the request - similar series are optional
			logger.Warn().Err(err).
				Int64("series_id", input.SeriesID).
				Msg("Failed to fetch similar series")
		} else {
			series.SimilarSeries = similarSeries
		}
	}

	return series, nil
}
