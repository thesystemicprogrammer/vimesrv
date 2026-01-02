package ports

import (
	"context"
	"time"
)

// SimilarMovie represents a similar movie from TMDB stored in cache
type SimilarMovie struct {
	ID          int64 // Database ID for linking translations
	TMDBID      int
	Title       string
	PosterPath  string
	ReleaseDate string
	VoteAverage float64
}

// SimilarMovieTranslation represents a translation for a similar movie
type SimilarMovieTranslation struct {
	SimilarMovieID int64
	Language       string
	Title          string
}

// SimilarSeries represents a similar series from TMDB stored in cache
type SimilarSeries struct {
	ID           int64 // Database ID for linking translations
	TMDBID       int
	Name         string
	PosterPath   string
	FirstAirDate string
	VoteAverage  float64
}

// SimilarSeriesTranslation represents a translation for a similar series
type SimilarSeriesTranslation struct {
	SimilarSeriesID int64
	Language        string
	Name            string
}

// SimilarContentRepository provides access to cached similar content
type SimilarContentRepository interface {
	// Movies
	GetSimilarMovies(ctx context.Context, movieMetadataID int64, limit int) ([]SimilarMovie, error)
	SaveSimilarMovies(ctx context.Context, movieMetadataID int64, movies []SimilarMovie) error
	GetSimilarMoviesFetchedAt(ctx context.Context, movieMetadataID int64) (*time.Time, error)
	DeleteSimilarMovies(ctx context.Context, movieMetadataID int64) error

	// Movie translations
	GetSimilarMovieTranslation(ctx context.Context, similarMovieID int64, language string) (*SimilarMovieTranslation, error)
	GetSimilarMovieTranslations(ctx context.Context, movieMetadataID int64, language string) (map[int64]string, error) // Returns map[similarMovieID]title
	SaveSimilarMovieTranslation(ctx context.Context, translation *SimilarMovieTranslation) error
	SaveSimilarMovieTranslations(ctx context.Context, translations []SimilarMovieTranslation) error

	// Series
	GetSimilarSeries(ctx context.Context, seriesMetadataID int64, limit int) ([]SimilarSeries, error)
	SaveSimilarSeries(ctx context.Context, seriesMetadataID int64, series []SimilarSeries) error
	GetSimilarSeriesFetchedAt(ctx context.Context, seriesMetadataID int64) (*time.Time, error)
	DeleteSimilarSeries(ctx context.Context, seriesMetadataID int64) error

	// Series translations
	GetSimilarSeriesTranslation(ctx context.Context, similarSeriesID int64, language string) (*SimilarSeriesTranslation, error)
	GetSimilarSeriesTranslations(ctx context.Context, seriesMetadataID int64, language string) (map[int64]string, error) // Returns map[similarSeriesID]name
	SaveSimilarSeriesTranslation(ctx context.Context, translation *SimilarSeriesTranslation) error
	SaveSimilarSeriesTranslations(ctx context.Context, translations []SimilarSeriesTranslation) error
}
