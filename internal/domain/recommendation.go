package domain

import "time"

// MovieRecommendation represents a cached movie recommendation
// computed using TF-IDF and cosine similarity
type MovieRecommendation struct {
	ID                         int64
	SourceMovieMetadataID      int64
	RecommendedMovieMetadataID int64
	SimilarityScore            float64
	RankOrder                  int
	GeneratedAt                time.Time
}

// SeriesRecommendation represents a cached series recommendation
// computed using TF-IDF and cosine similarity
type SeriesRecommendation struct {
	ID                          int64
	SourceSeriesMetadataID      int64
	RecommendedSeriesMetadataID int64
	SimilarityScore             float64
	RankOrder                   int
	GeneratedAt                 time.Time
}

// RecommendationModelMetadata tracks when recommendation models were last built
type RecommendationModelMetadata struct {
	ID              int64
	ModelType       string // "movie" or "series"
	TotalItems      int
	FeatureCount    int
	LastBuiltAt     time.Time
	BuildDurationMs int
}

// ContentFeatures represents extracted features for TF-IDF vectorization
type ContentFeatures struct {
	ID          int64
	Type        string // "movie" or "series"
	FeatureText string // Weighted feature string for TF-IDF
	Metadata    ContentMetadata
}

// ContentMetadata contains display metadata for recommendations
type ContentMetadata struct {
	Title        string
	Year         string
	Popularity   float64
	VoteAverage  float64
	PosterPath   string
	BackdropPath string
	MediaID      string // For movies in library
}

// SimilarItem represents an item with similarity score
type SimilarItem struct {
	ID              int64
	SimilarityScore float64
}

// PersonalizedRecommendation represents a recommendation for a user
type PersonalizedRecommendation struct {
	ItemID       int64
	ItemType     string // "movie" or "series"
	MediaID      string // For navigation (if in library)
	Title        string
	Year         string
	PosterPath   string
	BackdropPath string
	VoteAverage  float64
	Score        float64 // Final weighted score
}

// UserWatchProfile aggregates user's watch history for recommendations
type UserWatchProfile struct {
	MovieWeights  map[int64]float64 // movie_metadata_id -> completion weight
	SeriesWeights map[int64]float64 // series_metadata_id -> completion weight (episodes)
}

// NewUserWatchProfile creates an empty user watch profile
func NewUserWatchProfile() *UserWatchProfile {
	return &UserWatchProfile{
		MovieWeights:  make(map[int64]float64),
		SeriesWeights: make(map[int64]float64),
	}
}

// AddCompletedMovie adds a completed movie with 2x boost
func (p *UserWatchProfile) AddCompletedMovie(movieMetadataID int64) {
	p.MovieWeights[movieMetadataID] = 2.0
}

// AddWatchedEpisode adds an episode completion to series weight
// Each completed episode adds 0.2, capped at 2.0 (10 episodes)
func (p *UserWatchProfile) AddWatchedEpisode(seriesMetadataID int64) {
	current := p.SeriesWeights[seriesMetadataID]
	if current < 2.0 {
		p.SeriesWeights[seriesMetadataID] = min(current+0.2, 2.0)
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
