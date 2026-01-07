package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// FeatureExtractionRepository provides data for building recommendation models
type FeatureExtractionRepository struct {
	db *sql.DB
}

// NewFeatureExtractionRepository creates a new feature extraction repository
func NewFeatureExtractionRepository(db *database.DB) *FeatureExtractionRepository {
	return &FeatureExtractionRepository{db: db.DB}
}

// GetMoviesWithFeatures retrieves all movies in the library with their credits for feature extraction
func (r *FeatureExtractionRepository) GetMoviesWithFeatures(ctx context.Context) ([]ports.MovieFeatureData, error) {
	// Query movies that exist in the library (have media_files)
	// Join with credits to get directors and top cast
	query := `
		SELECT 
			mm.id,
			mm.original_title,
			mm.genres,
			mm.release_date,
			mm.popularity,
			mm.vote_average,
			mm.poster_path,
			mm.backdrop_path,
			mf.id as media_id,
			COALESCE(
				(SELECT GROUP_CONCAT(name, '|') 
				 FROM movie_credits 
				 WHERE movie_metadata_id = mm.id 
				   AND credit_type = 'crew' 
				   AND job = 'Director'
				 ORDER BY display_order),
				''
			) as directors,
			COALESCE(
				(SELECT GROUP_CONCAT(name, '|') 
				 FROM (
					SELECT name 
					FROM movie_credits 
					WHERE movie_metadata_id = mm.id 
					  AND credit_type = 'cast'
					ORDER BY display_order 
					LIMIT 3
				 )),
				''
			) as top_cast
		FROM movie_metadata mm
		JOIN media_files mf ON mf.movie_metadata_id = mm.id
		WHERE mf.metadata_type = 'movie' AND mf.movie_metadata_id IS NOT NULL
		ORDER BY mm.id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query movies with features: %w", err)
	}
	defer rows.Close()

	var movies []ports.MovieFeatureData
	for rows.Next() {
		var m ports.MovieFeatureData
		var genresJSON string
		var posterPath, backdropPath, mediaID sql.NullString
		var directors, topCast string

		if err := rows.Scan(
			&m.ID,
			&m.OriginalTitle,
			&genresJSON,
			&m.ReleaseDate,
			&m.Popularity,
			&m.VoteAverage,
			&posterPath,
			&backdropPath,
			&mediaID,
			&directors,
			&topCast,
		); err != nil {
			return nil, fmt.Errorf("scan movie feature: %w", err)
		}

		// Parse genres JSON
		if err := json.Unmarshal([]byte(genresJSON), &m.Genres); err != nil {
			m.Genres = []string{}
		}

		// Parse nullable strings
		if posterPath.Valid {
			m.PosterPath = posterPath.String
		}
		if backdropPath.Valid {
			m.BackdropPath = backdropPath.String
		}
		if mediaID.Valid {
			m.MediaID = mediaID.String
		}

		// Parse pipe-delimited directors and cast
		if directors != "" {
			m.Directors = splitAndTrim(directors, "|")
		} else {
			m.Directors = []string{}
		}
		if topCast != "" {
			m.TopCast = splitAndTrim(topCast, "|")
		} else {
			m.TopCast = []string{}
		}

		movies = append(movies, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movie features: %w", err)
	}

	if movies == nil {
		movies = []ports.MovieFeatureData{}
	}

	return movies, nil
}

// GetSeriesWithFeatures retrieves all series in the library with their credits for feature extraction
func (r *FeatureExtractionRepository) GetSeriesWithFeatures(ctx context.Context) ([]ports.SeriesFeatureData, error) {
	// Query series that have episodes in the library
	// Join with credits to get creators and top cast
	// Note: Series credits have Jobs as JSON, we need to extract creators
	query := `
		SELECT DISTINCT
			sm.id,
			sm.original_name,
			sm.genres,
			sm.first_air_date,
			sm.popularity,
			sm.vote_average,
			sm.poster_path,
			sm.backdrop_path,
			COALESCE(
				(SELECT GROUP_CONCAT(name, '|') 
				 FROM series_credits 
				 WHERE series_metadata_id = sm.id 
				   AND credit_type = 'crew' 
				   AND (jobs LIKE '%Creator%' OR jobs LIKE '%Executive Producer%')
				 ORDER BY display_order
				 LIMIT 3),
				''
			) as creators,
			COALESCE(
				(SELECT GROUP_CONCAT(name, '|') 
				 FROM (
					SELECT name 
					FROM series_credits 
					WHERE series_metadata_id = sm.id 
					  AND credit_type = 'cast'
					ORDER BY total_episode_count DESC, display_order
					LIMIT 3
				 )),
				''
			) as top_cast
		FROM series_metadata sm
		WHERE EXISTS (
			SELECT 1 
			FROM season_metadata ssm
			JOIN episode_metadata em ON ssm.id = em.season_id
			JOIN media_files mf ON mf.episode_metadata_id = em.id
			WHERE ssm.series_id = sm.id
		)
		ORDER BY sm.id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query series with features: %w", err)
	}
	defer rows.Close()

	var seriesList []ports.SeriesFeatureData
	for rows.Next() {
		var s ports.SeriesFeatureData
		var genresJSON string
		var posterPath, backdropPath sql.NullString
		var creators, topCast string

		if err := rows.Scan(
			&s.ID,
			&s.OriginalName,
			&genresJSON,
			&s.FirstAirDate,
			&s.Popularity,
			&s.VoteAverage,
			&posterPath,
			&backdropPath,
			&creators,
			&topCast,
		); err != nil {
			return nil, fmt.Errorf("scan series feature: %w", err)
		}

		// Parse genres JSON
		if err := json.Unmarshal([]byte(genresJSON), &s.Genres); err != nil {
			s.Genres = []string{}
		}

		// Parse nullable strings
		if posterPath.Valid {
			s.PosterPath = posterPath.String
		}
		if backdropPath.Valid {
			s.BackdropPath = backdropPath.String
		}

		// Parse pipe-delimited creators and cast
		if creators != "" {
			s.Creators = splitAndTrim(creators, "|")
		} else {
			s.Creators = []string{}
		}
		if topCast != "" {
			s.TopCast = splitAndTrim(topCast, "|")
		} else {
			s.TopCast = []string{}
		}

		seriesList = append(seriesList, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate series features: %w", err)
	}

	if seriesList == nil {
		seriesList = []ports.SeriesFeatureData{}
	}

	return seriesList, nil
}

// splitAndTrim splits a string by delimiter and trims whitespace from each element
func splitAndTrim(s, delimiter string) []string {
	parts := strings.Split(s, delimiter)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Ensure FeatureExtractionRepository implements ports.FeatureExtractionRepository
var _ ports.FeatureExtractionRepository = (*FeatureExtractionRepository)(nil)
