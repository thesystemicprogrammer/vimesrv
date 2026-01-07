package recommendation

import (
	"context"
	"sort"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// Personalization constants
const (
	// PopularityBoostThreshold - items with vote_average >= this get a popularity boost
	PopularityBoostThreshold = 7.0
	// PopularityBoostFactor - 30% boost for popular items
	PopularityBoostFactor = 1.3
	// CompletedMovieWeight - weight for completed movies (2x)
	CompletedMovieWeight = 2.0
	// EpisodeCompletionWeight - weight added per completed episode
	EpisodeCompletionWeight = 0.2
	// MaxSeriesWeight - maximum weight for a series (capped at 10 episodes worth)
	MaxSeriesWeight = 2.0
	// FavoriteWeight - weight for favorited items (treated same as 2x completion)
	FavoriteWeight = 2.0
	// MaxRecommendationsPerSource - max recommendations to aggregate per watched item
	MaxRecommendationsPerSource = 20
)

// GetPersonalizedRecommendationsInput contains parameters for getting personalized recommendations
type GetPersonalizedRecommendationsInput struct {
	UserID string
	Limit  int
	Type   string // "movie", "series", or "all"
}

// GetPersonalizedRecommendationsUseCase generates personalized recommendations for a user
type GetPersonalizedRecommendationsUseCase struct {
	recommendationRepo ports.RecommendationRepository
	userWatchDataRepo  ports.UserWatchDataRepository
	featureRepo        ports.FeatureExtractionRepository
}

// NewGetPersonalizedRecommendationsUseCase creates a new personalized recommendations use case
func NewGetPersonalizedRecommendationsUseCase(
	recommendationRepo ports.RecommendationRepository,
	userWatchDataRepo ports.UserWatchDataRepository,
	featureRepo ports.FeatureExtractionRepository,
) *GetPersonalizedRecommendationsUseCase {
	return &GetPersonalizedRecommendationsUseCase{
		recommendationRepo: recommendationRepo,
		userWatchDataRepo:  userWatchDataRepo,
		featureRepo:        featureRepo,
	}
}

// Execute generates personalized recommendations for the user
func (uc *GetPersonalizedRecommendationsUseCase) Execute(ctx context.Context, input GetPersonalizedRecommendationsInput) ([]domain.PersonalizedRecommendation, error) {
	// 1. Get user's watch history and favorites
	watchData, err := uc.userWatchDataRepo.GetUserWatchData(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	// Build user watch profile with weights
	profile := domain.NewUserWatchProfile()

	// Add completed movies with 2x weight
	for _, movieID := range watchData.CompletedMovies {
		profile.AddCompletedMovie(movieID)
	}

	// Add favorited movies (also 2x weight, but don't double count)
	for _, movieID := range watchData.FavoritedMovies {
		if _, exists := profile.MovieWeights[movieID]; !exists {
			profile.MovieWeights[movieID] = FavoriteWeight
		}
	}

	// Add watched episodes (0.2 per episode, capped at 2.0)
	for seriesID, count := range watchData.WatchedEpisodesBySeries {
		for i := 0; i < count; i++ {
			profile.AddWatchedEpisode(seriesID)
		}
	}

	// Add favorited series (also 2x weight, but don't double count if already has watch progress)
	for _, seriesID := range watchData.FavoritedSeries {
		if current, exists := profile.SeriesWeights[seriesID]; !exists || current < FavoriteWeight {
			profile.SeriesWeights[seriesID] = FavoriteWeight
		}
	}

	var recommendations []domain.PersonalizedRecommendation

	// 2. Generate movie recommendations
	if input.Type == "movie" || input.Type == "all" || input.Type == "" {
		movieRecs, err := uc.generateMovieRecommendations(ctx, profile, watchData)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to generate movie recommendations")
		} else {
			recommendations = append(recommendations, movieRecs...)
		}
	}

	// 3. Generate series recommendations
	if input.Type == "series" || input.Type == "all" || input.Type == "" {
		seriesRecs, err := uc.generateSeriesRecommendations(ctx, profile, watchData)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to generate series recommendations")
		} else {
			recommendations = append(recommendations, seriesRecs...)
		}
	}

	// 4. Sort by final score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// 5. Limit results
	if input.Limit > 0 && len(recommendations) > input.Limit {
		recommendations = recommendations[:input.Limit]
	}

	return recommendations, nil
}

// generateMovieRecommendations generates movie recommendations based on user profile
func (uc *GetPersonalizedRecommendationsUseCase) generateMovieRecommendations(
	ctx context.Context,
	profile *domain.UserWatchProfile,
	watchData *ports.UserWatchData,
) ([]domain.PersonalizedRecommendation, error) {
	// Aggregate scores for recommended movies
	movieScores := make(map[int64]float64)
	movieMetadata := make(map[int64]movieMeta)

	// Get all movies for metadata lookup
	allMovies, err := uc.featureRepo.GetMoviesWithFeatures(ctx)
	if err != nil {
		return nil, err
	}

	// Build metadata lookup
	for _, m := range allMovies {
		movieMetadata[m.ID] = movieMeta{
			title:        m.OriginalTitle,
			year:         extractYear(m.ReleaseDate),
			posterPath:   m.PosterPath,
			backdropPath: m.BackdropPath,
			voteAverage:  m.VoteAverage,
			mediaID:      m.MediaID,
		}
	}

	// Build set of already watched/favorited movies to exclude
	watched := make(map[int64]bool)
	for _, id := range watchData.CompletedMovies {
		watched[id] = true
	}
	for _, id := range watchData.FavoritedMovies {
		watched[id] = true
	}

	// Aggregate recommendations from each source movie
	for sourceID, weight := range profile.MovieWeights {
		recs, err := uc.recommendationRepo.GetMovieRecommendations(ctx, sourceID, MaxRecommendationsPerSource)
		if err != nil {
			continue
		}

		for _, rec := range recs {
			// Skip already watched movies
			if watched[rec.RecommendedMovieMetadataID] {
				continue
			}

			// Calculate weighted score
			score := rec.SimilarityScore * weight

			// Apply popularity boost
			if meta, ok := movieMetadata[rec.RecommendedMovieMetadataID]; ok {
				if meta.voteAverage >= PopularityBoostThreshold {
					score *= PopularityBoostFactor
				}
			}

			movieScores[rec.RecommendedMovieMetadataID] += score
		}
	}

	// Convert to result slice
	var recommendations []domain.PersonalizedRecommendation
	for movieID, score := range movieScores {
		meta, ok := movieMetadata[movieID]
		if !ok {
			continue
		}

		recommendations = append(recommendations, domain.PersonalizedRecommendation{
			ItemID:       movieID,
			ItemType:     "movie",
			MediaID:      meta.mediaID,
			Title:        meta.title,
			Year:         meta.year,
			PosterPath:   meta.posterPath,
			BackdropPath: meta.backdropPath,
			VoteAverage:  meta.voteAverage,
			Score:        score,
		})
	}

	return recommendations, nil
}

// generateSeriesRecommendations generates series recommendations based on user profile
func (uc *GetPersonalizedRecommendationsUseCase) generateSeriesRecommendations(
	ctx context.Context,
	profile *domain.UserWatchProfile,
	watchData *ports.UserWatchData,
) ([]domain.PersonalizedRecommendation, error) {
	// Aggregate scores for recommended series
	seriesScores := make(map[int64]float64)
	seriesMetadata := make(map[int64]seriesMeta)

	// Get all series for metadata lookup
	allSeries, err := uc.featureRepo.GetSeriesWithFeatures(ctx)
	if err != nil {
		return nil, err
	}

	// Build metadata lookup
	for _, s := range allSeries {
		seriesMetadata[s.ID] = seriesMeta{
			title:        s.OriginalName,
			year:         extractYear(s.FirstAirDate),
			posterPath:   s.PosterPath,
			backdropPath: s.BackdropPath,
			voteAverage:  s.VoteAverage,
		}
	}

	// Build set of already watched/favorited series to exclude
	watched := make(map[int64]bool)
	for id := range watchData.WatchedEpisodesBySeries {
		watched[id] = true
	}
	for _, id := range watchData.FavoritedSeries {
		watched[id] = true
	}

	// Aggregate recommendations from each source series
	for sourceID, weight := range profile.SeriesWeights {
		recs, err := uc.recommendationRepo.GetSeriesRecommendations(ctx, sourceID, MaxRecommendationsPerSource)
		if err != nil {
			continue
		}

		for _, rec := range recs {
			// Skip already watched series
			if watched[rec.RecommendedSeriesMetadataID] {
				continue
			}

			// Calculate weighted score
			score := rec.SimilarityScore * weight

			// Apply popularity boost
			if meta, ok := seriesMetadata[rec.RecommendedSeriesMetadataID]; ok {
				if meta.voteAverage >= PopularityBoostThreshold {
					score *= PopularityBoostFactor
				}
			}

			seriesScores[rec.RecommendedSeriesMetadataID] += score
		}
	}

	// Convert to result slice
	var recommendations []domain.PersonalizedRecommendation
	for seriesID, score := range seriesScores {
		meta, ok := seriesMetadata[seriesID]
		if !ok {
			continue
		}

		recommendations = append(recommendations, domain.PersonalizedRecommendation{
			ItemID:       seriesID,
			ItemType:     "series",
			Title:        meta.title,
			Year:         meta.year,
			PosterPath:   meta.posterPath,
			BackdropPath: meta.backdropPath,
			VoteAverage:  meta.voteAverage,
			Score:        score,
		})
	}

	return recommendations, nil
}

// movieMeta holds movie metadata for recommendations
type movieMeta struct {
	title        string
	year         string
	posterPath   string
	backdropPath string
	voteAverage  float64
	mediaID      string
}

// seriesMeta holds series metadata for recommendations
type seriesMeta struct {
	title        string
	year         string
	posterPath   string
	backdropPath string
	voteAverage  float64
}
