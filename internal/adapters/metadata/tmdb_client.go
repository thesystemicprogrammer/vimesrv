package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

const (
	tmdbBaseURL      = "https://api.themoviedb.org/3"
	tmdbImageBaseURL = "https://image.tmdb.org/t/p"
)

// TMDBHTTPClient implements ports.TMDBClient using the TMDB REST API
type TMDBHTTPClient struct {
	httpClient  *http.Client
	apiKey      string
	rateLimiter *rateLimiter
}

// rateLimiter implements a sliding window rate limiter
type rateLimiter struct {
	mu             sync.Mutex
	maxRequests    int
	windowDuration time.Duration
	timestamps     []time.Time
}

func newRateLimiter(maxRequests int, windowDuration time.Duration) *rateLimiter {
	return &rateLimiter{
		maxRequests:    maxRequests,
		windowDuration: windowDuration,
		timestamps:     make([]time.Time, 0, maxRequests),
	}
}

// Wait blocks until a request can be made within the rate limit
func (r *rateLimiter) Wait(ctx context.Context) error {
	for {
		r.mu.Lock()

		// Remove expired timestamps
		now := time.Now()
		cutoff := now.Add(-r.windowDuration)
		validIdx := 0
		for i, ts := range r.timestamps {
			if ts.After(cutoff) {
				validIdx = i
				break
			}
			if i == len(r.timestamps)-1 {
				validIdx = len(r.timestamps)
			}
		}
		if validIdx > 0 {
			r.timestamps = r.timestamps[validIdx:]
		}

		// Check if we can make a request
		if len(r.timestamps) < r.maxRequests {
			r.timestamps = append(r.timestamps, now)
			r.mu.Unlock()
			return nil
		}

		// Calculate wait time until oldest request expires
		oldestExpiry := r.timestamps[0].Add(r.windowDuration)
		waitTime := time.Until(oldestExpiry)
		r.mu.Unlock()

		// Wait with context
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Try again
		}
	}
}

// NewTMDBHTTPClient creates a new TMDB HTTP client
func NewTMDBHTTPClient(cfg config.TMDBConfig) *TMDBHTTPClient {
	return &TMDBHTTPClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:      cfg.APIKey,
		rateLimiter: newRateLimiter(cfg.RequestsPer10s, 10*time.Second),
	}
}

// tmdbSearchResponse is the TMDB API response for search endpoints
type tmdbSearchResponse struct {
	Page         int                   `json:"page"`
	TotalPages   int                   `json:"total_pages"`
	TotalResults int                   `json:"total_results"`
	Results      []tmdbSearchResultRaw `json:"results"`
}

// tmdbSearchResultRaw is the raw JSON from TMDB search (handles both movie and TV)
type tmdbSearchResultRaw struct {
	ID               int     `json:"id"`
	MediaType        string  `json:"media_type"`
	Title            string  `json:"title"` // movies
	Name             string  `json:"name"`  // TV
	OriginalTitle    string  `json:"original_title"`
	OriginalName     string  `json:"original_name"`
	Overview         string  `json:"overview"`
	ReleaseDate      string  `json:"release_date"`   // movies
	FirstAirDate     string  `json:"first_air_date"` // TV
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	Popularity       float64 `json:"popularity"`
	GenreIDs         []int   `json:"genre_ids"`
	OriginalLanguage string  `json:"original_language"`
}

func (r tmdbSearchResultRaw) toSearchResult(mediaType string) ports.TMDBSearchResult {
	result := ports.TMDBSearchResult{
		ID:           r.ID,
		MediaType:    mediaType,
		Overview:     r.Overview,
		PosterPath:   r.PosterPath,
		BackdropPath: r.BackdropPath,
		VoteAverage:  r.VoteAverage,
		VoteCount:    r.VoteCount,
		Popularity:   r.Popularity,
		GenreIDs:     r.GenreIDs,
		OriginalLang: r.OriginalLanguage,
	}

	if mediaType == "movie" {
		result.Title = r.Title
		result.OriginalTitle = r.OriginalTitle
		result.ReleaseDate = r.ReleaseDate
	} else {
		result.Title = r.Name
		result.OriginalTitle = r.OriginalName
		result.ReleaseDate = r.FirstAirDate
	}

	// For multi search, media_type comes from the response
	if r.MediaType != "" {
		result.MediaType = r.MediaType
		if r.MediaType == "tv" {
			result.Title = r.Name
			result.OriginalTitle = r.OriginalName
			result.ReleaseDate = r.FirstAirDate
		}
	}

	return result
}

// doRequest performs an HTTP request with rate limiting
func (c *TMDBHTTPClient) doRequest(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	// Build URL
	u, err := url.Parse(tmdbBaseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	// Add API key
	if params == nil {
		params = url.Values{}
	}
	params.Set("api_key", c.apiKey)
	u.RawQuery = params.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// SearchMovie searches for movies by title
func (c *TMDBHTTPClient) SearchMovie(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	if language != "" {
		params.Set("language", language)
	}
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}

	body, err := c.doRequest(ctx, "/search/movie", params)
	if err != nil {
		return nil, err
	}

	var resp tmdbSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	results := make([]ports.TMDBSearchResult, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = r.toSearchResult("movie")
	}

	return results, nil
}

// SearchTV searches for TV series by title
func (c *TMDBHTTPClient) SearchTV(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	if language != "" {
		params.Set("language", language)
	}
	if year > 0 {
		params.Set("first_air_date_year", strconv.Itoa(year))
	}

	body, err := c.doRequest(ctx, "/search/tv", params)
	if err != nil {
		return nil, err
	}

	var resp tmdbSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	results := make([]ports.TMDBSearchResult, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = r.toSearchResult("tv")
	}

	return results, nil
}

// SearchMulti searches for both movies and TV series
func (c *TMDBHTTPClient) SearchMulti(ctx context.Context, query string, language string) ([]ports.TMDBSearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	if language != "" {
		params.Set("language", language)
	}

	body, err := c.doRequest(ctx, "/search/multi", params)
	if err != nil {
		return nil, err
	}

	var resp tmdbSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Filter to only movies and TV
	results := make([]ports.TMDBSearchResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		if r.MediaType == "movie" || r.MediaType == "tv" {
			results = append(results, r.toSearchResult(r.MediaType))
		}
	}

	return results, nil
}

// GetMovieDetails retrieves detailed movie information
func (c *TMDBHTTPClient) GetMovieDetails(ctx context.Context, movieID int, language string) (*ports.TMDBMovieDetails, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	body, err := c.doRequest(ctx, fmt.Sprintf("/movie/%d", movieID), params)
	if err != nil {
		return nil, err
	}

	var details ports.TMDBMovieDetails
	if err := json.Unmarshal(body, &details); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &details, nil
}

// GetSeriesDetails retrieves detailed TV series information
func (c *TMDBHTTPClient) GetSeriesDetails(ctx context.Context, seriesID int, language string) (*ports.TMDBSeriesDetails, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	body, err := c.doRequest(ctx, fmt.Sprintf("/tv/%d", seriesID), params)
	if err != nil {
		return nil, err
	}

	var details ports.TMDBSeriesDetails
	if err := json.Unmarshal(body, &details); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &details, nil
}

// GetSeasonDetails retrieves detailed season information
func (c *TMDBHTTPClient) GetSeasonDetails(ctx context.Context, seriesID, seasonNumber int, language string) (*ports.TMDBSeasonDetails, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	body, err := c.doRequest(ctx, fmt.Sprintf("/tv/%d/season/%d", seriesID, seasonNumber), params)
	if err != nil {
		return nil, err
	}

	var details ports.TMDBSeasonDetails
	if err := json.Unmarshal(body, &details); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &details, nil
}

// GetEpisodeDetails retrieves detailed episode information
func (c *TMDBHTTPClient) GetEpisodeDetails(ctx context.Context, seriesID, seasonNumber, episodeNumber int, language string) (*ports.TMDBEpisodeDetails, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	body, err := c.doRequest(ctx, fmt.Sprintf("/tv/%d/season/%d/episode/%d", seriesID, seasonNumber, episodeNumber), params)
	if err != nil {
		return nil, err
	}

	var details ports.TMDBEpisodeDetails
	if err := json.Unmarshal(body, &details); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &details, nil
}

// GetImageURL constructs a full image URL from a TMDB image path
func (c *TMDBHTTPClient) GetImageURL(path string, size string) string {
	if path == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s%s", tmdbImageBaseURL, size, path)
}

// GetMovieCredits retrieves cast and crew for a movie
func (c *TMDBHTTPClient) GetMovieCredits(ctx context.Context, movieID int) (*ports.TMDBMovieCredits, error) {
	body, err := c.doRequest(ctx, fmt.Sprintf("/movie/%d/credits", movieID), nil)
	if err != nil {
		return nil, err
	}

	var credits ports.TMDBMovieCredits
	if err := json.Unmarshal(body, &credits); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &credits, nil
}

// GetMovieReleaseDates retrieves release dates and certifications for a movie
func (c *TMDBHTTPClient) GetMovieReleaseDates(ctx context.Context, movieID int) (*ports.TMDBReleaseDatesResponse, error) {
	body, err := c.doRequest(ctx, fmt.Sprintf("/movie/%d/release_dates", movieID), nil)
	if err != nil {
		return nil, err
	}

	var response ports.TMDBReleaseDatesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &response, nil
}

// GetSimilarMovies retrieves similar movies for a given movie
func (c *TMDBHTTPClient) GetSimilarMovies(ctx context.Context, movieID int, language string) (*ports.TMDBSimilarMoviesResponse, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	body, err := c.doRequest(ctx, fmt.Sprintf("/movie/%d/similar", movieID), params)
	if err != nil {
		return nil, err
	}

	var response ports.TMDBSimilarMoviesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &response, nil
}

// GetSimilarSeries retrieves similar series for a given TV series
func (c *TMDBHTTPClient) GetSimilarSeries(ctx context.Context, seriesID int, language string) (*ports.TMDBSimilarSeriesResponse, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	body, err := c.doRequest(ctx, fmt.Sprintf("/tv/%d/similar", seriesID), params)
	if err != nil {
		return nil, err
	}

	var response ports.TMDBSimilarSeriesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &response, nil
}

// GetCollectionDetails retrieves collection details including all movies
func (c *TMDBHTTPClient) GetCollectionDetails(ctx context.Context, collectionID int, language string) (*ports.TMDBCollectionDetails, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	body, err := c.doRequest(ctx, fmt.Sprintf("/collection/%d", collectionID), params)
	if err != nil {
		return nil, err
	}

	var details ports.TMDBCollectionDetails
	if err := json.Unmarshal(body, &details); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &details, nil
}

// GetCollectionTranslations retrieves translations for a collection
func (c *TMDBHTTPClient) GetCollectionTranslations(ctx context.Context, collectionID int) (*ports.TMDBCollectionTranslationsResponse, error) {
	body, err := c.doRequest(ctx, fmt.Sprintf("/collection/%d/translations", collectionID), nil)
	if err != nil {
		return nil, err
	}

	var response ports.TMDBCollectionTranslationsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &response, nil
}

// GetMovieTranslations retrieves all translations for a movie
func (c *TMDBHTTPClient) GetMovieTranslations(ctx context.Context, movieID int) (*ports.TMDBMovieTranslationsResponse, error) {
	body, err := c.doRequest(ctx, fmt.Sprintf("/movie/%d/translations", movieID), nil)
	if err != nil {
		return nil, err
	}

	var response ports.TMDBMovieTranslationsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &response, nil
}

// GetSeriesTranslations retrieves all translations for a TV series
func (c *TMDBHTTPClient) GetSeriesTranslations(ctx context.Context, seriesID int) (*ports.TMDBSeriesTranslationsResponse, error) {
	body, err := c.doRequest(ctx, fmt.Sprintf("/tv/%d/translations", seriesID), nil)
	if err != nil {
		return nil, err
	}

	var response ports.TMDBSeriesTranslationsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &response, nil
}

// Ensure TMDBHTTPClient implements TMDBClient
var _ ports.TMDBClient = (*TMDBHTTPClient)(nil)
