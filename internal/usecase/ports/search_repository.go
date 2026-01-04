package ports

import "context"

// SearchResult represents a search result item
type SearchResult struct {
	Type string  `json:"type"` // "movie" or "series"
	Rank float64 `json:"rank,omitempty"`

	// For movies
	MediaID         string `json:"media_id,omitempty"`
	MovieMetadataID *int64 `json:"movie_metadata_id,omitempty"`

	// For series
	SeriesMetadataID *int64 `json:"series_metadata_id,omitempty"`

	// Common display fields (populated by caller)
	Title       string  `json:"title"`
	Year        string  `json:"year,omitempty"`
	PosterPath  string  `json:"poster_path,omitempty"`
	VoteAverage float64 `json:"vote_average"`
	Genres      string  `json:"genres,omitempty"`
}

// SearchRepository provides full-text search functionality
type SearchRepository interface {
	// SearchMovies searches the movie FTS index
	// Returns media_ids and movie_metadata_ids matching the query
	SearchMovies(ctx context.Context, query string, limit int) ([]SearchResult, error)

	// SearchSeries searches the series FTS index
	// Returns series_metadata_ids matching the query
	SearchSeries(ctx context.Context, query string, limit int) ([]SearchResult, error)

	// IndexMovie adds or updates a movie in the search index
	IndexMovie(ctx context.Context, mediaID string, movieMetadataID int64, title, originalTitle, castNames, crewNames string) error

	// IndexSeries adds or updates a series in the search index
	IndexSeries(ctx context.Context, seriesMetadataID int64, name, originalName, castNames, crewNames string) error

	// RemoveMovie removes a movie from the search index
	RemoveMovie(ctx context.Context, mediaID string) error

	// RemoveSeries removes a series from the search index
	RemoveSeries(ctx context.Context, seriesMetadataID int64) error
}
