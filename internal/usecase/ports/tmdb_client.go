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

// TMDBCastMember represents a cast member from TMDB credits
type TMDBCastMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character"`
	ProfilePath string `json:"profile_path"`
	Order       int    `json:"order"`
}

// TMDBCrewMember represents a crew member from TMDB credits
type TMDBCrewMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Job         string `json:"job"`
	Department  string `json:"department"`
	ProfilePath string `json:"profile_path"`
}

// TMDBMovieCredits represents the full credits response from TMDB
type TMDBMovieCredits struct {
	ID   int              `json:"id"`
	Cast []TMDBCastMember `json:"cast"`
	Crew []TMDBCrewMember `json:"crew"`
}

// TMDBReleaseDate represents a release date entry within a country
type TMDBReleaseDate struct {
	Certification string `json:"certification"`
	Type          int    `json:"type"` // 1=Premiere, 2=Theatrical (limited), 3=Theatrical, 4=Digital, 5=Physical, 6=TV
	ReleaseDate   string `json:"release_date"`
}

// TMDBReleaseDateCountry represents release dates for a specific country
type TMDBReleaseDateCountry struct {
	ISO3166_1    string            `json:"iso_3166_1"`
	ReleaseDates []TMDBReleaseDate `json:"release_dates"`
}

// TMDBReleaseDatesResponse represents the release dates API response
type TMDBReleaseDatesResponse struct {
	ID      int                      `json:"id"`
	Results []TMDBReleaseDateCountry `json:"results"`
}

// TMDBSimilarMovie represents a similar movie from TMDB
type TMDBSimilarMovie struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	PosterPath    string  `json:"poster_path"`
	ReleaseDate   string  `json:"release_date"`
	VoteAverage   float64 `json:"vote_average"`
	Overview      string  `json:"overview"`
}

// TMDBSimilarMoviesResponse represents the similar movies API response
type TMDBSimilarMoviesResponse struct {
	Page         int                `json:"page"`
	TotalPages   int                `json:"total_pages"`
	TotalResults int                `json:"total_results"`
	Results      []TMDBSimilarMovie `json:"results"`
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

	// GetMovieCredits retrieves cast and crew for a movie
	GetMovieCredits(ctx context.Context, movieID int) (*TMDBMovieCredits, error)

	// GetMovieReleaseDates retrieves release dates and certifications for a movie
	GetMovieReleaseDates(ctx context.Context, movieID int) (*TMDBReleaseDatesResponse, error)

	// GetSimilarMovies retrieves similar movies for a given movie
	GetSimilarMovies(ctx context.Context, movieID int, language string) (*TMDBSimilarMoviesResponse, error)
}
