package ports

import (
	"context"
)

// MovieFilterOptions contains filtering and sorting options for movie listings
type MovieFilterOptions struct {
	SortBy    string   // date_added, title, year, rating
	SortOrder string   // asc, desc
	Genres    []string // Filter by genre names (AND logic)
	YearFrom  int      // Filter movies from this year (inclusive)
	YearTo    int      // Filter movies up to this year (inclusive)
	MinRating float64  // Filter movies with rating >= this value
}

// SeriesFilterOptions contains filtering and sorting options for series listings
type SeriesFilterOptions struct {
	SortBy    string   // date_added, name, year, rating
	SortOrder string   // asc, desc
	Genres    []string // Filter by genre names (AND logic)
	YearFrom  int      // Filter series from this year (inclusive)
	YearTo    int      // Filter series up to this year (inclusive)
	MinRating float64  // Filter series with rating >= this value
}

// MovieSummary represents a movie with its metadata for library display
type MovieSummary struct {
	// Media file info
	MediaID          string `json:"media_id"`
	Duration         int    `json:"duration"`
	Resolution       string `json:"resolution"`
	Status           string `json:"status"`            // ready, processing, error
	EnrichmentStatus string `json:"enrichment_status"` // linked, auto_linked, etc.
	CreatedAt        string `json:"created_at"`

	// Transcode status
	TranscodeStatus string `json:"transcode_status"` // none, pending, completed

	// Movie metadata (if linked)
	MovieMetadataID *int64  `json:"movie_metadata_id,omitempty"`
	Title           string  `json:"title"`          // From translation or original_title
	Year            string  `json:"year,omitempty"` // Extracted from release_date
	PosterPath      string  `json:"poster_path,omitempty"`
	BackdropPath    string  `json:"backdrop_path,omitempty"`
	VoteAverage     float64 `json:"vote_average"`
	Genres          string  `json:"genres,omitempty"` // Comma-separated
}

// CreditPerson represents a cast or crew member for display
type CreditPerson struct {
	ID           int64  `json:"id"`
	TMDBPersonID int    `json:"tmdb_person_id"`
	Name         string `json:"name"`
	Character    string `json:"character,omitempty"` // For cast
	Job          string `json:"job,omitempty"`       // For crew
	ProfilePath  string `json:"profile_path,omitempty"`
}

// SimilarMovieItem represents a similar movie for display
type SimilarMovieItem struct {
	TMDBID      int     `json:"tmdb_id"`
	Title       string  `json:"title"`
	PosterPath  string  `json:"poster_path,omitempty"`
	ReleaseDate string  `json:"release_date,omitempty"`
	Year        string  `json:"year,omitempty"`
	VoteAverage float64 `json:"vote_average"`
	InLibrary   bool    `json:"in_library"`
	MediaID     string  `json:"media_id,omitempty"` // Set if InLibrary is true
}

// SimilarSeriesItem represents a similar series for display
type SimilarSeriesItem struct {
	TMDBID           int     `json:"tmdb_id"`
	Name             string  `json:"name"`
	PosterPath       string  `json:"poster_path,omitempty"`
	FirstAirDate     string  `json:"first_air_date,omitempty"`
	Year             string  `json:"year,omitempty"`
	VoteAverage      float64 `json:"vote_average"`
	InLibrary        bool    `json:"in_library"`
	SeriesMetadataID int64   `json:"series_metadata_id,omitempty"` // Set if InLibrary is true
}

// CollectionMovieDisplay represents a movie within a collection for display
type CollectionMovieDisplay struct {
	TMDBID      int     `json:"tmdb_id"`
	Title       string  `json:"title"`
	PosterPath  string  `json:"poster_path,omitempty"`
	ReleaseDate string  `json:"release_date,omitempty"`
	Year        string  `json:"year,omitempty"`
	VoteAverage float64 `json:"vote_average"`
	InLibrary   bool    `json:"in_library"`
	MediaID     string  `json:"media_id,omitempty"` // Set if InLibrary is true
	IsCurrent   bool    `json:"is_current"`         // True if this is the movie being viewed
}

// MovieCollectionInfo represents collection information for a movie
type MovieCollectionInfo struct {
	CollectionID int                      `json:"collection_id"`
	Name         string                   `json:"name"`
	Overview     string                   `json:"overview,omitempty"`
	PosterPath   string                   `json:"poster_path,omitempty"`
	BackdropPath string                   `json:"backdrop_path,omitempty"`
	Movies       []CollectionMovieDisplay `json:"movies"`
	Position     int                      `json:"position"` // 1-based position of current movie
	TotalMovies  int                      `json:"total_movies"`
}

// MovieDetail represents a movie with full details including credits and certifications
type MovieDetail struct {
	MovieSummary

	// Extended metadata
	OriginalTitle string `json:"original_title,omitempty"`
	Tagline       string `json:"tagline,omitempty"`
	Overview      string `json:"overview,omitempty"`
	ReleaseDate   string `json:"release_date,omitempty"`
	Runtime       int    `json:"runtime,omitempty"`
	MovieStatus   string `json:"movie_status,omitempty"` // Released, Post Production, etc.
	IMDbID        string `json:"imdb_id,omitempty"`
	TMDBID        int    `json:"tmdb_id,omitempty"`
	CollectionID  *int   `json:"-"` // Used internally to fetch collection, not exposed to API

	// Certification for user's locale
	Certification string `json:"certification,omitempty"`

	// Credits
	Cast      []CreditPerson `json:"cast,omitempty"`
	Directors []CreditPerson `json:"directors,omitempty"`
	Crew      []CreditPerson `json:"crew,omitempty"` // Other notable crew

	// Similar movies
	SimilarMovies []SimilarMovieItem `json:"similar_movies,omitempty"`

	// Collection info (if movie belongs to a collection)
	Collection *MovieCollectionInfo `json:"collection,omitempty"`

	// Audio and subtitle languages
	AudioLanguages    []string `json:"audio_languages,omitempty"`
	SubtitleLanguages []string `json:"subtitle_languages,omitempty"`
}

// SeriesSummary represents a series with summary info for library display
type SeriesSummary struct {
	SeriesMetadataID  int64   `json:"series_metadata_id"`
	TMDBID            int     `json:"tmdb_id"`
	Name              string  `json:"name"`           // From translation or original_name
	Year              string  `json:"year,omitempty"` // Extracted from first_air_date
	PosterPath        string  `json:"poster_path,omitempty"`
	BackdropPath      string  `json:"backdrop_path,omitempty"`
	VoteAverage       float64 `json:"vote_average"`
	Genres            string  `json:"genres,omitempty"` // Comma-separated
	NumberOfSeasons   int     `json:"number_of_seasons"`
	NumberOfEpisodes  int     `json:"number_of_episodes"`
	AvailableEpisodes int     `json:"available_episodes"` // Count of linked media files
}

// EpisodeSummary represents an episode with its metadata and media file info
type EpisodeSummary struct {
	// Media file info (if available)
	MediaID         *string `json:"media_id,omitempty"`
	Duration        int     `json:"duration"`
	Status          string  `json:"status,omitempty"`
	TranscodeStatus string  `json:"transcode_status,omitempty"`

	// Episode metadata
	EpisodeMetadataID int64   `json:"episode_metadata_id"`
	SeasonNumber      int     `json:"season_number"`
	EpisodeNumber     int     `json:"episode_number"`
	Name              string  `json:"name"`
	Overview          string  `json:"overview,omitempty"`
	AirDate           string  `json:"air_date,omitempty"`
	StillPath         string  `json:"still_path,omitempty"`
	VoteAverage       float64 `json:"vote_average"`

	// Audio and subtitle languages
	AudioLanguages    []string `json:"audio_languages,omitempty"`
	SubtitleLanguages []string `json:"subtitle_languages,omitempty"`
}

// SeasonSummary represents a season with its episodes
type SeasonSummary struct {
	SeasonMetadataID int64            `json:"season_metadata_id"`
	SeasonNumber     int              `json:"season_number"`
	Name             string           `json:"name"`
	Overview         string           `json:"overview,omitempty"`
	PosterPath       string           `json:"poster_path,omitempty"`
	AirDate          string           `json:"air_date,omitempty"`
	EpisodeCount     int              `json:"episode_count"`
	Episodes         []EpisodeSummary `json:"episodes,omitempty"`
}

// SeriesDetail represents a full series with seasons and episodes for detail view
type SeriesDetail struct {
	SeriesSummary
	Overview string          `json:"overview,omitempty"`
	Seasons  []SeasonSummary `json:"seasons,omitempty"`

	// Similar series
	SimilarSeries []SimilarSeriesItem `json:"similar_series,omitempty"`
}

// UnmatchedMediaSummary represents a media file without metadata
type UnmatchedMediaSummary struct {
	MediaID          string `json:"media_id"`
	Filename         string `json:"filename"`
	Title            string `json:"title"`
	Duration         int    `json:"duration"`
	Resolution       string `json:"resolution"`
	EnrichmentStatus string `json:"enrichment_status"`
	CreatedAt        string `json:"created_at"`
}

// RecentlyAddedItem represents a recently added item (movie or season) for display
// Movies are returned as individual items, episodes are grouped by season
type RecentlyAddedItem struct {
	// Type discriminator: "movie" or "season"
	Type string `json:"type"`

	// Common fields
	Title        string  `json:"title"`
	Year         string  `json:"year,omitempty"`
	PosterPath   string  `json:"poster_path,omitempty"`
	BackdropPath string  `json:"backdrop_path,omitempty"`
	VoteAverage  float64 `json:"vote_average"`
	CreatedAt    string  `json:"created_at"` // Most recent addition time

	// Movie-specific fields (when Type == "movie")
	MediaID         string `json:"media_id,omitempty"`
	MovieMetadataID *int64 `json:"movie_metadata_id,omitempty"`
	TranscodeStatus string `json:"transcode_status,omitempty"`

	// Season-specific fields (when Type == "season")
	SeriesMetadataID *int64 `json:"series_metadata_id,omitempty"`
	SeasonNumber     *int   `json:"season_number,omitempty"`
	EpisodeCount     int    `json:"episode_count,omitempty"` // Number of episodes added
}

// LibraryRepository provides library-focused queries that join media files with metadata
type LibraryRepository interface {
	// ListMovies returns movies with their metadata
	// language is used to select the appropriate translation (e.g., "en", "de")
	// filterOpts contains sorting and filtering options
	ListMovies(ctx context.Context, language string, limit, offset int, filterOpts MovieFilterOptions) ([]MovieSummary, int, error)

	// GetMovie returns a single movie with full details
	GetMovie(ctx context.Context, mediaID string, language string) (*MovieSummary, error)

	// GetMovieDetail returns a movie with full details including credits and certification
	GetMovieDetail(ctx context.Context, mediaID string, language string, maxCast int) (*MovieDetail, error)

	// ListSeries returns series with summary info
	// Only returns series that have at least one linked episode (or all if includeEmpty is true)
	// filterOpts contains sorting and filtering options
	ListSeries(ctx context.Context, language string, includeEmpty bool, limit, offset int, filterOpts SeriesFilterOptions) ([]SeriesSummary, int, error)

	// GetSeriesDetail returns a series with all seasons and episodes
	GetSeriesDetail(ctx context.Context, seriesID int64, language string) (*SeriesDetail, error)

	// ListRecentlyAdded returns the most recently added media (movies and seasons)
	// Movies are returned as individual items, episodes are grouped by season
	// Returns items sorted by most recent addition descending
	ListRecentlyAdded(ctx context.Context, language string, limit int) ([]RecentlyAddedItem, error)

	// ListUnmatched returns media files that don't have metadata linked
	ListUnmatched(ctx context.Context, limit, offset int) ([]UnmatchedMediaSummary, int, error)

	// ListMovieGenres returns all unique genres from movies in the library
	ListMovieGenres(ctx context.Context) ([]string, error)

	// ListSeriesGenres returns all unique genres from series in the library
	ListSeriesGenres(ctx context.Context) ([]string, error)
}
