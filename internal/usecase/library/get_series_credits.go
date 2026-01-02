package library

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetSeriesCreditsInput contains the input parameters for getting series credits
type GetSeriesCreditsInput struct {
	SeriesMetadataID int64
}

// SeriesCreditsOutput contains the full cast and crew for a series
type SeriesCreditsOutput struct {
	Cast []domain.SeriesCredit `json:"cast"`
	Crew []domain.SeriesCredit `json:"crew"`
}

// GetSeriesCreditsUseCase retrieves full cast and crew for a series
type GetSeriesCreditsUseCase struct {
	seriesMetadataRepo ports.SeriesMetadataRepository
	seriesCreditRepo   ports.SeriesCreditRepository
	tmdbClient         ports.TMDBClient
}

// NewGetSeriesCreditsUseCase creates a new GetSeriesCreditsUseCase instance
func NewGetSeriesCreditsUseCase(
	seriesMetadataRepo ports.SeriesMetadataRepository,
	seriesCreditRepo ports.SeriesCreditRepository,
	tmdbClient ports.TMDBClient,
) *GetSeriesCreditsUseCase {
	return &GetSeriesCreditsUseCase{
		seriesMetadataRepo: seriesMetadataRepo,
		seriesCreditRepo:   seriesCreditRepo,
		tmdbClient:         tmdbClient,
	}
}

// Execute retrieves full cast and crew for a series
// If full credits haven't been fetched yet, it fetches them from TMDB and stores them
func (uc *GetSeriesCreditsUseCase) Execute(ctx context.Context, input GetSeriesCreditsInput) (*SeriesCreditsOutput, error) {
	// Check if we've already fetched full credits
	hasFullCredits, err := uc.seriesMetadataRepo.HasFullCreditsFetched(ctx, input.SeriesMetadataID)
	if err != nil {
		return nil, fmt.Errorf("check full credits status: %w", err)
	}

	if !hasFullCredits {
		// Fetch full credits from TMDB and store them
		if err := uc.fetchAndStoreCredits(ctx, input.SeriesMetadataID); err != nil {
			return nil, fmt.Errorf("fetch and store credits: %w", err)
		}
	}

	// Retrieve all credits from database
	credits, err := uc.seriesCreditRepo.GetBySeriesMetadataID(ctx, input.SeriesMetadataID)
	if err != nil {
		return nil, fmt.Errorf("get credits from database: %w", err)
	}

	// Separate cast and crew
	output := &SeriesCreditsOutput{
		Cast: make([]domain.SeriesCredit, 0),
		Crew: make([]domain.SeriesCredit, 0),
	}

	for _, credit := range credits {
		if credit.IsCast() {
			output.Cast = append(output.Cast, credit)
		} else {
			output.Crew = append(output.Crew, credit)
		}
	}

	return output, nil
}

// fetchAndStoreCredits fetches credits from TMDB and stores them in the database
func (uc *GetSeriesCreditsUseCase) fetchAndStoreCredits(ctx context.Context, seriesMetadataID int64) error {
	// Get the TMDB ID for this series
	tmdbID, err := uc.seriesMetadataRepo.GetTMDBIDByID(ctx, seriesMetadataID)
	if err != nil {
		return fmt.Errorf("get tmdb id: %w", err)
	}

	// Fetch aggregate credits from TMDB
	tmdbCredits, err := uc.tmdbClient.GetSeriesAggregateCredits(ctx, tmdbID)
	if err != nil {
		return fmt.Errorf("fetch credits from TMDB: %w", err)
	}

	logger.Debug().
		Int64("series_metadata_id", seriesMetadataID).
		Int("tmdb_id", tmdbID).
		Int("cast_count", len(tmdbCredits.Cast)).
		Int("crew_count", len(tmdbCredits.Crew)).
		Msg("Fetched series credits from TMDB")

	// Delete existing credits to avoid duplicates
	if err := uc.seriesCreditRepo.DeleteBySeriesMetadataID(ctx, seriesMetadataID); err != nil {
		return fmt.Errorf("delete existing credits: %w", err)
	}

	// Convert and store all cast members
	var credits []*domain.SeriesCredit
	for _, castMember := range tmdbCredits.Cast {
		// Convert roles to JSON
		rolesJSON, err := json.Marshal(castMember.Roles)
		if err != nil {
			rolesJSON = []byte("[]")
		}

		credit := domain.NewSeriesCastCredit(
			seriesMetadataID,
			castMember.ID,
			castMember.Name,
			string(rolesJSON),
			castMember.ProfilePath,
			castMember.TotalEpisodeCount,
			castMember.Order,
		)
		credits = append(credits, credit)
	}

	// Convert and store all crew members
	for i, crewMember := range tmdbCredits.Crew {
		// Convert jobs to JSON
		jobsJSON, err := json.Marshal(crewMember.Jobs)
		if err != nil {
			jobsJSON = []byte("[]")
		}

		credit := domain.NewSeriesCrewCredit(
			seriesMetadataID,
			crewMember.ID,
			crewMember.Name,
			string(jobsJSON),
			crewMember.Department,
			crewMember.ProfilePath,
			crewMember.TotalEpisodeCount,
			i, // Use index as display order for crew
		)
		credits = append(credits, credit)
	}

	// Batch insert all credits
	if len(credits) > 0 {
		if err := uc.seriesCreditRepo.CreateBatch(ctx, credits); err != nil {
			return fmt.Errorf("store credits: %w", err)
		}
	}

	// Mark as fetched
	if err := uc.seriesMetadataRepo.SetFullCreditsFetched(ctx, seriesMetadataID); err != nil {
		return fmt.Errorf("set full credits fetched: %w", err)
	}

	logger.Info().
		Int64("series_metadata_id", seriesMetadataID).
		Int("credits_stored", len(credits)).
		Msg("Stored full series credits")

	return nil
}
