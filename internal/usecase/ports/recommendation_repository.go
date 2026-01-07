package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// RecommendationRepository manages cached recommendations
type RecommendationRepository interface {
	// Movie recommendations
	SaveMovieRecommendations(ctx context.Context, sourceID int64, recommendations []domain.MovieRecommendation) error
	GetMovieRecommendations(ctx context.Context, sourceID int64, limit int) ([]domain.MovieRecommendation, error)
	DeleteAllMovieRecommendations(ctx context.Context) error

	// Series recommendations
	SaveSeriesRecommendations(ctx context.Context, sourceID int64, recommendations []domain.SeriesRecommendation) error
	GetSeriesRecommendations(ctx context.Context, sourceID int64, limit int) ([]domain.SeriesRecommendation, error)
	DeleteAllSeriesRecommendations(ctx context.Context) error

	// Model metadata
	SaveModelMetadata(ctx context.Context, metadata domain.RecommendationModelMetadata) error
	GetModelMetadata(ctx context.Context, modelType string) (*domain.RecommendationModelMetadata, error)
}

// MovieFeatureData contains all data needed to build movie features for TF-IDF
type MovieFeatureData struct {
	ID            int64
	OriginalTitle string
	Genres        []string
	ReleaseDate   string
	Popularity    float64
	VoteAverage   float64
	PosterPath    string
	BackdropPath  string
	MediaID       string // From media_files if in library
	Directors     []string
	TopCast       []string // Top 3 actor names
}

// SeriesFeatureData contains all data needed to build series features for TF-IDF
type SeriesFeatureData struct {
	ID           int64
	OriginalName string
	Genres       []string
	FirstAirDate string
	Popularity   float64
	VoteAverage  float64
	PosterPath   string
	BackdropPath string
	Creators     []string // Showrunners/creators
	TopCast      []string // Top 3 actor names
}

// FeatureExtractionRepository provides data for building recommendation models
type FeatureExtractionRepository interface {
	// Get all movies with credits for feature extraction
	GetMoviesWithFeatures(ctx context.Context) ([]MovieFeatureData, error)

	// Get all series with credits for feature extraction
	GetSeriesWithFeatures(ctx context.Context) ([]SeriesFeatureData, error)
}

// EnrichedMovieRecommendation includes recommendation data with display metadata
type EnrichedMovieRecommendation struct {
	MovieMetadataID int64
	MediaID         string // Empty if not in library
	Title           string
	Year            string
	PosterPath      string
	BackdropPath    string
	VoteAverage     float64
	SimilarityScore float64
	InLibrary       bool
}

// EnrichedSeriesRecommendation includes recommendation data with display metadata
type EnrichedSeriesRecommendation struct {
	SeriesMetadataID int64
	Title            string
	Year             string
	PosterPath       string
	BackdropPath     string
	VoteAverage      float64
	SimilarityScore  float64
}

// UserWatchData represents a user's watch history for recommendation scoring
type UserWatchData struct {
	// Completed movies (movie_metadata_id)
	CompletedMovies []int64
	// Watched episode counts per series (series_metadata_id -> count)
	WatchedEpisodesBySeries map[int64]int
	// Favorited movies (movie_metadata_id)
	FavoritedMovies []int64
	// Favorited series (series_metadata_id)
	FavoritedSeries []int64
}

// UserWatchDataRepository provides user watch data for personalized recommendations
type UserWatchDataRepository interface {
	// GetUserWatchData retrieves watch history and favorites for recommendation scoring
	GetUserWatchData(ctx context.Context, userID string) (*UserWatchData, error)
}
