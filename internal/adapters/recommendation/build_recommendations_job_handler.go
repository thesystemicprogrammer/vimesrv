package recommendation

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/recommendation"
)

// NewBuildRecommendationsJobHandler creates a job handler for periodic recommendation model builds.
// This handler is registered with the job manager to process "build_recommendations" job types.
// It rebuilds the TF-IDF recommendation models for movies and series.
func NewBuildRecommendationsJobHandler(useCase *recommendation.BuildRecommendationModelUseCase) ports.JobHandler {
	return func(ctx context.Context, job *domain.Job) error {
		logger.Info().Int64("job_id", job.ID).Msg("starting scheduled recommendation model build")

		input := recommendation.BuildRecommendationModelInput{
			ModelType: "", // Build both movie and series models
		}

		output, err := useCase.Execute(ctx, input)
		if err != nil {
			logger.Error().Err(err).Msg("failed to build recommendation models")
			return err
		}

		logger.Info().
			Bool("movie_model_built", output.MovieModelBuilt).
			Bool("series_model_built", output.SeriesModelBuilt).
			Int("movie_items", output.MovieItems).
			Int("series_items", output.SeriesItems).
			Int("movie_duration_ms", output.MovieDurationMs).
			Int("series_duration_ms", output.SeriesDurationMs).
			Msg("recommendation models built successfully")

		return nil
	}
}
