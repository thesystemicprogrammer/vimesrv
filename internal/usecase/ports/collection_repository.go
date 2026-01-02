package ports

import (
	"context"
	"time"
)

// CollectionMetadata represents cached collection metadata from TMDB
type CollectionMetadata struct {
	ID           int64
	CollectionID int
	Name         string
	Overview     string
	PosterPath   string
	BackdropPath string
	FetchedAt    time.Time
}

// CollectionTranslation represents cached collection translation from TMDB
type CollectionTranslation struct {
	ID           int64
	CollectionID int
	Language     string
	Name         string
	Overview     string
	FetchedAt    time.Time
}

// CollectionMovieItem represents a movie within a collection from TMDB
type CollectionMovieItem struct {
	ID            int64
	CollectionID  int
	TMDBMovieID   int
	Title         string
	OriginalTitle string
	PosterPath    string
	ReleaseDate   string
	VoteAverage   float64
	DisplayOrder  int
	FetchedAt     time.Time
}

// CollectionMovieTranslation represents a translation for a movie within a collection
type CollectionMovieTranslation struct {
	CollectionMovieID int64
	Language          string
	Title             string
}

// CollectionRepository provides access to cached collection data
type CollectionRepository interface {
	// Metadata
	GetCollectionMetadata(ctx context.Context, collectionID int) (*CollectionMetadata, error)
	SaveCollectionMetadata(ctx context.Context, metadata *CollectionMetadata) error

	// Translations
	GetCollectionTranslation(ctx context.Context, collectionID int, language string) (*CollectionTranslation, error)
	SaveCollectionTranslation(ctx context.Context, translation *CollectionTranslation) error

	// Movies in collection
	GetCollectionMovies(ctx context.Context, collectionID int) ([]CollectionMovieItem, error)
	SaveCollectionMovies(ctx context.Context, collectionID int, movies []CollectionMovieItem) error

	// Movie translations
	GetCollectionMovieTranslation(ctx context.Context, collectionMovieID int64, language string) (*CollectionMovieTranslation, error)
	GetCollectionMovieTranslations(ctx context.Context, collectionID int, language string) (map[int64]string, error) // Returns map[collectionMovieID]title
	SaveCollectionMovieTranslation(ctx context.Context, translation *CollectionMovieTranslation) error
	SaveCollectionMovieTranslations(ctx context.Context, translations []CollectionMovieTranslation) error

	// Cache management
	DeleteExpiredData(ctx context.Context, maxAge time.Duration) error
}
