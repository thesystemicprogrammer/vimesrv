package ports

import "context"

// TMDBGenre represents a genre from TMDB
type TMDBGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TMDBNetwork represents a TV network from TMDB
type TMDBNetwork struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TMDBSearchResult represents a search result from TMDB (movie or TV)
type TMDBSearchResult struct {
	ID            int     `json:"id"`
	MediaType     string  `json:"media_type"` // "movie" or "tv"
	Title         string  `json:"title"`      // title for movies, name for TV
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	ReleaseDate   string  `json:"release_date"` // release_date for movies, first_air_date for TV
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	Popularity    float64 `json:"popularity"`
	GenreIDs      []int   `json:"genre_ids"`
	OriginalLang  string  `json:"original_language"`
}

// TMDBMovieDetails represents detailed movie information from TMDB
type TMDBMovieDetails struct {
	ID            int         `json:"id"`
	IMDbID        string      `json:"imdb_id"`
	Title         string      `json:"title"`
	OriginalTitle string      `json:"original_title"`
	Tagline       string      `json:"tagline"`
	Overview      string      `json:"overview"`
	ReleaseDate   string      `json:"release_date"`
	Runtime       int         `json:"runtime"`
	PosterPath    string      `json:"poster_path"`
	BackdropPath  string      `json:"backdrop_path"`
	Genres        []TMDBGenre `json:"genres"`
	VoteAverage   float64     `json:"vote_average"`
	VoteCount     int         `json:"vote_count"`
	Popularity    float64     `json:"popularity"`
	Status        string      `json:"status"`
	OriginalLang  string      `json:"original_language"`
}

// TMDBSeasonSummary represents basic season info returned with series details
type TMDBSeasonSummary struct {
	ID           int    `json:"id"`
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	AirDate      string `json:"air_date"`
	PosterPath   string `json:"poster_path"`
	EpisodeCount int    `json:"episode_count"`
}

// TMDBSeriesDetails represents detailed TV series information from TMDB
type TMDBSeriesDetails struct {
	ID               int                 `json:"id"`
	Name             string              `json:"name"`
	OriginalName     string              `json:"original_name"`
	Overview         string              `json:"overview"`
	FirstAirDate     string              `json:"first_air_date"`
	LastAirDate      string              `json:"last_air_date"`
	Status           string              `json:"status"`
	PosterPath       string              `json:"poster_path"`
	BackdropPath     string              `json:"backdrop_path"`
	Genres           []TMDBGenre         `json:"genres"`
	Networks         []TMDBNetwork       `json:"networks"`
	VoteAverage      float64             `json:"vote_average"`
	VoteCount        int                 `json:"vote_count"`
	Popularity       float64             `json:"popularity"`
	NumberOfSeasons  int                 `json:"number_of_seasons"`
	NumberOfEpisodes int                 `json:"number_of_episodes"`
	OriginalLang     string              `json:"original_language"`
	Seasons          []TMDBSeasonSummary `json:"seasons"`
}

// TMDBEpisodeSummary represents basic episode info returned with season details
type TMDBEpisodeSummary struct {
	ID            int     `json:"id"`
	EpisodeNumber int     `json:"episode_number"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	AirDate       string  `json:"air_date"`
	StillPath     string  `json:"still_path"`
	Runtime       int     `json:"runtime"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
}

// TMDBSeasonDetails represents detailed season information from TMDB
type TMDBSeasonDetails struct {
	ID           int                  `json:"id"`
	SeasonNumber int                  `json:"season_number"`
	Name         string               `json:"name"`
	Overview     string               `json:"overview"`
	AirDate      string               `json:"air_date"`
	PosterPath   string               `json:"poster_path"`
	Episodes     []TMDBEpisodeSummary `json:"episodes"`
}

// TMDBEpisodeDetails represents detailed episode information from TMDB
type TMDBEpisodeDetails struct {
	ID            int     `json:"id"`
	EpisodeNumber int     `json:"episode_number"`
	SeasonNumber  int     `json:"season_number"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	AirDate       string  `json:"air_date"`
	StillPath     string  `json:"still_path"`
	Runtime       int     `json:"runtime"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
}

// TMDBClient defines the interface for interacting with the TMDB API
type TMDBClient interface {
	// SearchMovie searches for movies by title, optionally filtering by year
	SearchMovie(ctx context.Context, query string, year int, language string) ([]TMDBSearchResult, error)

	// SearchTV searches for TV series by title, optionally filtering by year
	SearchTV(ctx context.Context, query string, year int, language string) ([]TMDBSearchResult, error)

	// SearchMulti searches for both movies and TV series
	SearchMulti(ctx context.Context, query string, language string) ([]TMDBSearchResult, error)

	// GetMovieDetails retrieves detailed movie information
	GetMovieDetails(ctx context.Context, movieID int, language string) (*TMDBMovieDetails, error)

	// GetSeriesDetails retrieves detailed TV series information
	GetSeriesDetails(ctx context.Context, seriesID int, language string) (*TMDBSeriesDetails, error)

	// GetSeasonDetails retrieves detailed season information including episodes
	GetSeasonDetails(ctx context.Context, seriesID, seasonNumber int, language string) (*TMDBSeasonDetails, error)

	// GetEpisodeDetails retrieves detailed episode information
	GetEpisodeDetails(ctx context.Context, seriesID, seasonNumber, episodeNumber int, language string) (*TMDBEpisodeDetails, error)

	// GetImageURL constructs a full image URL from a TMDB image path
	GetImageURL(path string, size string) string
}
