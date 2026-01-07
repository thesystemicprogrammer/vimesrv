package recommendation

import (
	"context"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// TopNRecommendations is the number of similar items to pre-compute per content
const TopNRecommendations = 50

// BuildRecommendationModelInput contains parameters for building recommendation models
type BuildRecommendationModelInput struct {
	ModelType string // "movie", "series", or "all"
}

// BuildRecommendationModelOutput contains the result of building recommendation models
type BuildRecommendationModelOutput struct {
	MovieModelBuilt  bool
	SeriesModelBuilt bool
	MovieItems       int
	SeriesItems      int
	MovieDurationMs  int
	SeriesDurationMs int
}

// BuildRecommendationModelUseCase orchestrates building the recommendation models
type BuildRecommendationModelUseCase struct {
	featureRepo        ports.FeatureExtractionRepository
	recommendationRepo ports.RecommendationRepository
}

// NewBuildRecommendationModelUseCase creates a new use case for building recommendation models
func NewBuildRecommendationModelUseCase(
	featureRepo ports.FeatureExtractionRepository,
	recommendationRepo ports.RecommendationRepository,
) *BuildRecommendationModelUseCase {
	return &BuildRecommendationModelUseCase{
		featureRepo:        featureRepo,
		recommendationRepo: recommendationRepo,
	}
}

// Execute builds the recommendation models based on the input parameters
func (uc *BuildRecommendationModelUseCase) Execute(ctx context.Context, input BuildRecommendationModelInput) (*BuildRecommendationModelOutput, error) {
	output := &BuildRecommendationModelOutput{}

	// Build movie model
	if input.ModelType == "movie" || input.ModelType == "all" || input.ModelType == "" {
		items, durationMs, err := uc.buildMovieModel(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("failed to build movie recommendation model")
			return nil, fmt.Errorf("build movie model: %w", err)
		}
		output.MovieModelBuilt = true
		output.MovieItems = items
		output.MovieDurationMs = durationMs
		logger.Info().Int("items", items).Int("duration_ms", durationMs).Msg("movie recommendation model built")
	}

	// Build series model
	if input.ModelType == "series" || input.ModelType == "all" || input.ModelType == "" {
		items, durationMs, err := uc.buildSeriesModel(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("failed to build series recommendation model")
			return nil, fmt.Errorf("build series model: %w", err)
		}
		output.SeriesModelBuilt = true
		output.SeriesItems = items
		output.SeriesDurationMs = durationMs
		logger.Info().Int("items", items).Int("duration_ms", durationMs).Msg("series recommendation model built")
	}

	return output, nil
}

// buildMovieModel builds the movie recommendation model
func (uc *BuildRecommendationModelUseCase) buildMovieModel(ctx context.Context) (int, int, error) {
	startTime := time.Now()

	// 1. Extract features for all movies in library
	moviesData, err := uc.featureRepo.GetMoviesWithFeatures(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("get movies with features: %w", err)
	}

	if len(moviesData) == 0 {
		logger.Info().Msg("no movies in library, skipping movie model build")
		return 0, 0, nil
	}

	logger.Info().Int("count", len(moviesData)).Msg("extracted movie features")

	// 2. Convert to content features
	features := make([]domain.ContentFeatures, len(moviesData))
	for i, movie := range moviesData {
		features[i] = ExtractMovieFeatures(movie)
	}

	// 3. Build TF-IDF model
	recommender := NewTFIDFRecommender("movie")
	if err := recommender.Build(features); err != nil {
		return 0, 0, fmt.Errorf("build tfidf model: %w", err)
	}

	// 4. Clear existing recommendations before rebuilding
	if err := uc.recommendationRepo.DeleteAllMovieRecommendations(ctx); err != nil {
		return 0, 0, fmt.Errorf("delete old recommendations: %w", err)
	}

	// 5. Pre-compute similar items for each movie
	for _, movie := range moviesData {
		similar, err := recommender.GetSimilar(movie.ID, TopNRecommendations)
		if err != nil {
			logger.Warn().Err(err).Int64("movie_id", movie.ID).Msg("failed to get similar movies")
			continue
		}

		if len(similar) == 0 {
			continue
		}

		// Convert to domain recommendations
		recommendations := make([]domain.MovieRecommendation, len(similar))
		for i, s := range similar {
			recommendations[i] = domain.MovieRecommendation{
				RecommendedMovieMetadataID: s.ID,
				SimilarityScore:            s.SimilarityScore,
				RankOrder:                  i + 1,
			}
		}

		// Save recommendations for this movie
		if err := uc.recommendationRepo.SaveMovieRecommendations(ctx, movie.ID, recommendations); err != nil {
			logger.Warn().Err(err).Int64("movie_id", movie.ID).Msg("failed to save movie recommendations")
		}
	}

	// 6. Save model metadata
	durationMs := int(time.Since(startTime).Milliseconds())
	metadata := domain.RecommendationModelMetadata{
		ModelType:       "movie",
		TotalItems:      recommender.GetItemCount(),
		FeatureCount:    recommender.GetFeatureCount(),
		LastBuiltAt:     time.Now().UTC(),
		BuildDurationMs: durationMs,
	}
	if err := uc.recommendationRepo.SaveModelMetadata(ctx, metadata); err != nil {
		logger.Warn().Err(err).Msg("failed to save movie model metadata")
	}

	return recommender.GetItemCount(), durationMs, nil
}

// buildSeriesModel builds the series recommendation model
func (uc *BuildRecommendationModelUseCase) buildSeriesModel(ctx context.Context) (int, int, error) {
	startTime := time.Now()

	// 1. Extract features for all series in library
	seriesData, err := uc.featureRepo.GetSeriesWithFeatures(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("get series with features: %w", err)
	}

	if len(seriesData) == 0 {
		logger.Info().Msg("no series in library, skipping series model build")
		return 0, 0, nil
	}

	logger.Info().Int("count", len(seriesData)).Msg("extracted series features")

	// 2. Convert to content features
	features := make([]domain.ContentFeatures, len(seriesData))
	for i, series := range seriesData {
		features[i] = ExtractSeriesFeatures(series)
	}

	// 3. Build TF-IDF model
	recommender := NewTFIDFRecommender("series")
	if err := recommender.Build(features); err != nil {
		return 0, 0, fmt.Errorf("build tfidf model: %w", err)
	}

	// 4. Clear existing recommendations before rebuilding
	if err := uc.recommendationRepo.DeleteAllSeriesRecommendations(ctx); err != nil {
		return 0, 0, fmt.Errorf("delete old recommendations: %w", err)
	}

	// 5. Pre-compute similar items for each series
	for _, series := range seriesData {
		similar, err := recommender.GetSimilar(series.ID, TopNRecommendations)
		if err != nil {
			logger.Warn().Err(err).Int64("series_id", series.ID).Msg("failed to get similar series")
			continue
		}

		if len(similar) == 0 {
			continue
		}

		// Convert to domain recommendations
		recommendations := make([]domain.SeriesRecommendation, len(similar))
		for i, s := range similar {
			recommendations[i] = domain.SeriesRecommendation{
				RecommendedSeriesMetadataID: s.ID,
				SimilarityScore:             s.SimilarityScore,
				RankOrder:                   i + 1,
			}
		}

		// Save recommendations for this series
		if err := uc.recommendationRepo.SaveSeriesRecommendations(ctx, series.ID, recommendations); err != nil {
			logger.Warn().Err(err).Int64("series_id", series.ID).Msg("failed to save series recommendations")
		}
	}

	// 6. Save model metadata
	durationMs := int(time.Since(startTime).Milliseconds())
	metadata := domain.RecommendationModelMetadata{
		ModelType:       "series",
		TotalItems:      recommender.GetItemCount(),
		FeatureCount:    recommender.GetFeatureCount(),
		LastBuiltAt:     time.Now().UTC(),
		BuildDurationMs: durationMs,
	}
	if err := uc.recommendationRepo.SaveModelMetadata(ctx, metadata); err != nil {
		logger.Warn().Err(err).Msg("failed to save series model metadata")
	}

	return recommender.GetItemCount(), durationMs, nil
}
