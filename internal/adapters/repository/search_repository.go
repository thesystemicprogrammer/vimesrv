package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SearchRepository implements full-text search using SQLite FTS5
type SearchRepository struct {
	db *sql.DB
}

// NewSearchRepository creates a new SearchRepository instance
func NewSearchRepository(db *database.DB) ports.SearchRepository {
	return &SearchRepository{
		db: db.DB,
	}
}

// sanitizeFTS5Query prepares a query string for FTS5
// It escapes special characters and adds prefix matching
func sanitizeFTS5Query(query string) string {
	// Trim whitespace
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	// Split into words
	words := strings.Fields(query)
	var sanitized []string

	for _, word := range words {
		// Remove FTS5 special characters
		word = strings.ReplaceAll(word, "\"", "")
		word = strings.ReplaceAll(word, "'", "")
		word = strings.ReplaceAll(word, "*", "")
		word = strings.ReplaceAll(word, "-", " ")
		word = strings.ReplaceAll(word, ":", "")
		word = strings.ReplaceAll(word, "(", "")
		word = strings.ReplaceAll(word, ")", "")
		word = strings.ReplaceAll(word, "^", "")
		word = strings.TrimSpace(word)

		if word != "" {
			// Add prefix matching for better search experience
			sanitized = append(sanitized, "\""+word+"\"*")
		}
	}

	// Join with AND logic (all words must match)
	return strings.Join(sanitized, " AND ")
}

// SearchMovies searches the movie FTS index
func (r *SearchRepository) SearchMovies(ctx context.Context, query string, limit int) ([]ports.SearchResult, error) {
	ftsQuery := sanitizeFTS5Query(query)
	if ftsQuery == "" {
		return []ports.SearchResult{}, nil
	}

	// Search with ranking by relevance
	// JOIN with media_files to get media_id since FTS table may have empty/stale media_id
	sqlQuery := `
		SELECT 
			mf.id,
			ms.movie_metadata_id,
			bm25(movie_search) as rank
		FROM movie_search ms
		JOIN media_files mf ON mf.movie_metadata_id = ms.movie_metadata_id
		WHERE movie_search MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, sqlQuery, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search movies: %w", err)
	}
	defer rows.Close()

	var results []ports.SearchResult
	for rows.Next() {
		var result ports.SearchResult
		var movieMetadataID int64
		result.Type = "movie"

		err := rows.Scan(
			&result.MediaID,
			&movieMetadataID,
			&result.Rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan movie search result: %w", err)
		}

		result.MovieMetadataID = &movieMetadataID
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating movie search results: %w", err)
	}

	return results, nil
}

// SearchSeries searches the series FTS index
func (r *SearchRepository) SearchSeries(ctx context.Context, query string, limit int) ([]ports.SearchResult, error) {
	ftsQuery := sanitizeFTS5Query(query)
	if ftsQuery == "" {
		return []ports.SearchResult{}, nil
	}

	// Search with ranking by relevance
	sqlQuery := `
		SELECT 
			ss.series_metadata_id,
			bm25(series_search) as rank
		FROM series_search ss
		WHERE series_search MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, sqlQuery, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search series: %w", err)
	}
	defer rows.Close()

	var results []ports.SearchResult
	for rows.Next() {
		var result ports.SearchResult
		var seriesMetadataID int64
		result.Type = "series"

		err := rows.Scan(
			&seriesMetadataID,
			&result.Rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan series search result: %w", err)
		}

		result.SeriesMetadataID = &seriesMetadataID
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating series search results: %w", err)
	}

	return results, nil
}

// IndexMovie adds or updates a movie in the search index
// Note: media_id is stored for compatibility but the search query JOINs with media_files
// to get the current media_id, so it's not critical if media_id is empty here.
func (r *SearchRepository) IndexMovie(ctx context.Context, mediaID string, movieMetadataID int64, title, originalTitle, castNames, crewNames string) error {
	// Delete existing entry by movie_metadata_id (more reliable than media_id)
	_, err := r.db.ExecContext(ctx, "DELETE FROM movie_search WHERE movie_metadata_id = ?", movieMetadataID)
	if err != nil {
		return fmt.Errorf("failed to remove existing movie from search index: %w", err)
	}

	// Insert new entry
	_, err = r.db.ExecContext(ctx,
		"INSERT INTO movie_search(media_id, movie_metadata_id, title, original_title, cast_names, crew_names) VALUES (?, ?, ?, ?, ?, ?)",
		mediaID, movieMetadataID, title, originalTitle, castNames, crewNames)
	if err != nil {
		return fmt.Errorf("failed to index movie: %w", err)
	}

	return nil
}

// IndexSeries adds or updates a series in the search index
func (r *SearchRepository) IndexSeries(ctx context.Context, seriesMetadataID int64, name, originalName, castNames, crewNames string) error {
	// First try to delete existing entry
	_, err := r.db.ExecContext(ctx, "DELETE FROM series_search WHERE series_metadata_id = ?", seriesMetadataID)
	if err != nil {
		return fmt.Errorf("failed to remove existing series from search index: %w", err)
	}

	// Insert new entry
	_, err = r.db.ExecContext(ctx,
		"INSERT INTO series_search(series_metadata_id, name, original_name, cast_names, crew_names) VALUES (?, ?, ?, ?, ?)",
		seriesMetadataID, name, originalName, castNames, crewNames)
	if err != nil {
		return fmt.Errorf("failed to index series: %w", err)
	}

	return nil
}

// RemoveMovie removes a movie from the search index
func (r *SearchRepository) RemoveMovie(ctx context.Context, mediaID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM movie_search WHERE media_id = ?", mediaID)
	if err != nil {
		return fmt.Errorf("failed to remove movie from search index: %w", err)
	}
	return nil
}

// RemoveSeries removes a series from the search index
func (r *SearchRepository) RemoveSeries(ctx context.Context, seriesMetadataID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM series_search WHERE series_metadata_id = ?", seriesMetadataID)
	if err != nil {
		return fmt.Errorf("failed to remove series from search index: %w", err)
	}
	return nil
}
