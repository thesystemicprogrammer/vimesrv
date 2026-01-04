package library

import (
	"context"
	"fmt"
	"sort"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SearchLibraryInput contains the input parameters for searching the library
type SearchLibraryInput struct {
	Query    string
	Language string
	Limit    int
}

// SearchResultItem represents a single search result with full details
type SearchResultItem struct {
	Type             string  `json:"type"` // "movie" or "series"
	MediaID          string  `json:"media_id,omitempty"`
	MovieMetadataID  *int64  `json:"movie_metadata_id,omitempty"`
	SeriesMetadataID *int64  `json:"series_metadata_id,omitempty"`
	Title            string  `json:"title"`
	Year             string  `json:"year,omitempty"`
	PosterPath       string  `json:"poster_path,omitempty"`
	VoteAverage      float64 `json:"vote_average"`
	Genres           string  `json:"genres,omitempty"`
	TranscodeStatus  string  `json:"transcode_status,omitempty"`
}

// SearchLibraryOutput contains the search results
type SearchLibraryOutput struct {
	Query   string             `json:"query"`
	Total   int                `json:"total"`
	Results []SearchResultItem `json:"results"`
}

// SearchLibraryUseCase handles library search operations
type SearchLibraryUseCase struct {
	searchRepository  ports.SearchRepository
	libraryRepository ports.LibraryRepository
}

// NewSearchLibraryUseCase creates a new SearchLibraryUseCase instance
func NewSearchLibraryUseCase(
	searchRepository ports.SearchRepository,
	libraryRepository ports.LibraryRepository,
) *SearchLibraryUseCase {
	return &SearchLibraryUseCase{
		searchRepository:  searchRepository,
		libraryRepository: libraryRepository,
	}
}

// Execute searches the library for movies and series matching the query
func (uc *SearchLibraryUseCase) Execute(ctx context.Context, input SearchLibraryInput) (*SearchLibraryOutput, error) {
	// Set defaults
	query := input.Query
	language := input.Language
	limit := input.Limit

	if query == "" {
		return &SearchLibraryOutput{
			Query:   query,
			Total:   0,
			Results: []SearchResultItem{},
		}, nil
	}

	if language == "" {
		language = "en"
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Search both movies and series
	movieResults, err := uc.searchRepository.SearchMovies(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search movies: %w", err)
	}

	seriesResults, err := uc.searchRepository.SearchSeries(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search series: %w", err)
	}

	// Build a combined result list
	var results []SearchResultItem

	// Fetch movie details
	if len(movieResults) > 0 {
		// Get all movies to look up details
		movies, _, err := uc.libraryRepository.ListMovies(ctx, language, 1000, 0, ports.MovieFilterOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch movie details: %w", err)
		}

		// Create lookup map
		moviesByMetadataID := make(map[int64]ports.MovieSummary)
		for _, m := range movies {
			if m.MovieMetadataID != nil {
				moviesByMetadataID[*m.MovieMetadataID] = m
			}
		}

		// Match search results with movie details
		for _, sr := range movieResults {
			if sr.MovieMetadataID != nil {
				if movie, ok := moviesByMetadataID[*sr.MovieMetadataID]; ok {
					results = append(results, SearchResultItem{
						Type:            "movie",
						MediaID:         movie.MediaID,
						MovieMetadataID: sr.MovieMetadataID,
						Title:           movie.Title,
						Year:            movie.Year,
						PosterPath:      movie.PosterPath,
						VoteAverage:     movie.VoteAverage,
						Genres:          movie.Genres,
						TranscodeStatus: movie.TranscodeStatus,
					})
				}
			}
		}
	}

	// Fetch series details
	if len(seriesResults) > 0 {
		// Get all series to look up details
		series, _, err := uc.libraryRepository.ListSeries(ctx, language, false, 1000, 0, ports.SeriesFilterOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch series details: %w", err)
		}

		// Create lookup map
		seriesByID := make(map[int64]ports.SeriesSummary)
		for _, s := range series {
			seriesByID[s.SeriesMetadataID] = s
		}

		// Match search results with series details
		for _, sr := range seriesResults {
			if sr.SeriesMetadataID != nil {
				if s, ok := seriesByID[*sr.SeriesMetadataID]; ok {
					results = append(results, SearchResultItem{
						Type:             "series",
						SeriesMetadataID: sr.SeriesMetadataID,
						Title:            s.Name,
						Year:             s.Year,
						PosterPath:       s.PosterPath,
						VoteAverage:      s.VoteAverage,
						Genres:           s.Genres,
					})
				}
			}
		}
	}

	// Sort by relevance (movies first for now, could be enhanced with ranking)
	sort.Slice(results, func(i, j int) bool {
		// Movies before series
		if results[i].Type != results[j].Type {
			return results[i].Type == "movie"
		}
		// Within same type, sort by title
		return results[i].Title < results[j].Title
	})

	// Limit total results
	if len(results) > limit {
		results = results[:limit]
	}

	return &SearchLibraryOutput{
		Query:   query,
		Total:   len(results),
		Results: results,
	}, nil
}
