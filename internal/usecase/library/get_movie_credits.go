package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetMovieCreditsInput contains the input parameters for getting movie credits
type GetMovieCreditsInput struct {
	MovieMetadataID int64
}

// MovieCreditsOutput contains the full cast and crew for a movie
type MovieCreditsOutput struct {
	Cast []domain.MovieCredit `json:"cast"`
	Crew []domain.MovieCredit `json:"crew"`
}

// GetMovieCreditsUseCase retrieves full cast and crew for a movie
type GetMovieCreditsUseCase struct {
	movieMetadataRepo ports.MovieMetadataRepository
	movieCreditRepo   ports.MovieCreditRepository
	tmdbClient        ports.TMDBClient
}

// NewGetMovieCreditsUseCase creates a new GetMovieCreditsUseCase instance
func NewGetMovieCreditsUseCase(
	movieMetadataRepo ports.MovieMetadataRepository,
	movieCreditRepo ports.MovieCreditRepository,
	tmdbClient ports.TMDBClient,
) *GetMovieCreditsUseCase {
	return &GetMovieCreditsUseCase{
		movieMetadataRepo: movieMetadataRepo,
		movieCreditRepo:   movieCreditRepo,
		tmdbClient:        tmdbClient,
	}
}

// Execute retrieves full cast and crew for a movie
// If full credits haven't been fetched yet, it fetches them from TMDB and stores them
func (uc *GetMovieCreditsUseCase) Execute(ctx context.Context, input GetMovieCreditsInput) (*MovieCreditsOutput, error) {
	// Check if we've already fetched full credits
	hasFullCredits, err := uc.movieMetadataRepo.HasFullCreditsFetched(ctx, input.MovieMetadataID)
	if err != nil {
		return nil, fmt.Errorf("check full credits status: %w", err)
	}

	if !hasFullCredits {
		// Fetch full credits from TMDB and store them
		if err := uc.fetchAndStoreCredits(ctx, input.MovieMetadataID); err != nil {
			return nil, fmt.Errorf("fetch and store credits: %w", err)
		}
	}

	// Retrieve all credits from database
	credits, err := uc.movieCreditRepo.GetByMovieMetadataID(ctx, input.MovieMetadataID)
	if err != nil {
		return nil, fmt.Errorf("get credits from database: %w", err)
	}

	// Separate cast and crew
	output := &MovieCreditsOutput{
		Cast: make([]domain.MovieCredit, 0),
		Crew: make([]domain.MovieCredit, 0),
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
func (uc *GetMovieCreditsUseCase) fetchAndStoreCredits(ctx context.Context, movieMetadataID int64) error {
	// Get the TMDB ID for this movie
	tmdbID, err := uc.movieMetadataRepo.GetTMDBIDByID(ctx, movieMetadataID)
	if err != nil {
		return fmt.Errorf("get tmdb id: %w", err)
	}

	// Fetch credits from TMDB
	tmdbCredits, err := uc.tmdbClient.GetMovieCredits(ctx, tmdbID)
	if err != nil {
		return fmt.Errorf("fetch credits from TMDB: %w", err)
	}

	logger.Debug().
		Int64("movie_metadata_id", movieMetadataID).
		Int("tmdb_id", tmdbID).
		Int("cast_count", len(tmdbCredits.Cast)).
		Int("crew_count", len(tmdbCredits.Crew)).
		Msg("Fetched movie credits from TMDB")

	// Delete existing credits to avoid duplicates
	if err := uc.movieCreditRepo.DeleteByMovieMetadataID(ctx, movieMetadataID); err != nil {
		return fmt.Errorf("delete existing credits: %w", err)
	}

	// Convert and store all cast members
	var credits []*domain.MovieCredit
	for _, castMember := range tmdbCredits.Cast {
		credit := domain.NewCastCredit(
			movieMetadataID,
			castMember.ID,
			castMember.Name,
			castMember.Character,
			castMember.ProfilePath,
			castMember.Order,
		)
		credits = append(credits, credit)
	}

	// Convert and store all crew members
	for _, crewMember := range tmdbCredits.Crew {
		credit := domain.NewCrewCredit(
			movieMetadataID,
			crewMember.ID,
			crewMember.Name,
			crewMember.Job,
			crewMember.Department,
			crewMember.ProfilePath,
		)
		credits = append(credits, credit)
	}

	// Batch insert all credits
	if len(credits) > 0 {
		if err := uc.movieCreditRepo.CreateBatch(ctx, credits); err != nil {
			return fmt.Errorf("store credits: %w", err)
		}
	}

	// Mark as fetched
	if err := uc.movieMetadataRepo.SetFullCreditsFetched(ctx, movieMetadataID); err != nil {
		return fmt.Errorf("set full credits fetched: %w", err)
	}

	logger.Info().
		Int64("movie_metadata_id", movieMetadataID).
		Int("credits_stored", len(credits)).
		Msg("Stored full movie credits")

	return nil
}
