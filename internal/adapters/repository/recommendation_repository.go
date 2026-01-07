package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// RecommendationRepository implements ports.RecommendationRepository using SQLite
type RecommendationRepository struct {
	db *sql.DB
}

// NewRecommendationRepository creates a new recommendation repository
func NewRecommendationRepository(db *database.DB) *RecommendationRepository {
	return &RecommendationRepository{db: db.DB}
}

// SaveMovieRecommendations saves pre-computed movie recommendations in batch
func (r *RecommendationRepository) SaveMovieRecommendations(ctx context.Context, sourceID int64, recommendations []domain.MovieRecommendation) error {
	if len(recommendations) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Use INSERT OR REPLACE to handle duplicates
	query := `
		INSERT OR REPLACE INTO movie_recommendations (
			source_movie_metadata_id, recommended_movie_metadata_id,
			similarity_score, rank_order, generated_at
		) VALUES (?, ?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for _, rec := range recommendations {
		_, err := stmt.ExecContext(ctx,
			sourceID,
			rec.RecommendedMovieMetadataID,
			rec.SimilarityScore,
			rec.RankOrder,
			now,
		)
		if err != nil {
			return fmt.Errorf("insert movie recommendation: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetMovieRecommendations retrieves cached movie recommendations for a source movie
func (r *RecommendationRepository) GetMovieRecommendations(ctx context.Context, sourceID int64, limit int) ([]domain.MovieRecommendation, error) {
	query := `
		SELECT id, source_movie_metadata_id, recommended_movie_metadata_id,
			   similarity_score, rank_order, generated_at
		FROM movie_recommendations
		WHERE source_movie_metadata_id = ?
		ORDER BY rank_order
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, sourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("query movie recommendations: %w", err)
	}
	defer rows.Close()

	var recommendations []domain.MovieRecommendation
	for rows.Next() {
		var rec domain.MovieRecommendation
		if err := rows.Scan(
			&rec.ID,
			&rec.SourceMovieMetadataID,
			&rec.RecommendedMovieMetadataID,
			&rec.SimilarityScore,
			&rec.RankOrder,
			&rec.GeneratedAt,
		); err != nil {
			return nil, fmt.Errorf("scan movie recommendation: %w", err)
		}
		recommendations = append(recommendations, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movie recommendations: %w", err)
	}

	if recommendations == nil {
		recommendations = []domain.MovieRecommendation{}
	}

	return recommendations, nil
}

// DeleteAllMovieRecommendations removes all cached movie recommendations
func (r *RecommendationRepository) DeleteAllMovieRecommendations(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM movie_recommendations")
	if err != nil {
		return fmt.Errorf("delete movie recommendations: %w", err)
	}
	return nil
}

// SaveSeriesRecommendations saves pre-computed series recommendations in batch
func (r *RecommendationRepository) SaveSeriesRecommendations(ctx context.Context, sourceID int64, recommendations []domain.SeriesRecommendation) error {
	if len(recommendations) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Use INSERT OR REPLACE to handle duplicates
	query := `
		INSERT OR REPLACE INTO series_recommendations (
			source_series_metadata_id, recommended_series_metadata_id,
			similarity_score, rank_order, generated_at
		) VALUES (?, ?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for _, rec := range recommendations {
		_, err := stmt.ExecContext(ctx,
			sourceID,
			rec.RecommendedSeriesMetadataID,
			rec.SimilarityScore,
			rec.RankOrder,
			now,
		)
		if err != nil {
			return fmt.Errorf("insert series recommendation: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetSeriesRecommendations retrieves cached series recommendations for a source series
func (r *RecommendationRepository) GetSeriesRecommendations(ctx context.Context, sourceID int64, limit int) ([]domain.SeriesRecommendation, error) {
	query := `
		SELECT id, source_series_metadata_id, recommended_series_metadata_id,
			   similarity_score, rank_order, generated_at
		FROM series_recommendations
		WHERE source_series_metadata_id = ?
		ORDER BY rank_order
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, sourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("query series recommendations: %w", err)
	}
	defer rows.Close()

	var recommendations []domain.SeriesRecommendation
	for rows.Next() {
		var rec domain.SeriesRecommendation
		if err := rows.Scan(
			&rec.ID,
			&rec.SourceSeriesMetadataID,
			&rec.RecommendedSeriesMetadataID,
			&rec.SimilarityScore,
			&rec.RankOrder,
			&rec.GeneratedAt,
		); err != nil {
			return nil, fmt.Errorf("scan series recommendation: %w", err)
		}
		recommendations = append(recommendations, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate series recommendations: %w", err)
	}

	if recommendations == nil {
		recommendations = []domain.SeriesRecommendation{}
	}

	return recommendations, nil
}

// DeleteAllSeriesRecommendations removes all cached series recommendations
func (r *RecommendationRepository) DeleteAllSeriesRecommendations(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM series_recommendations")
	if err != nil {
		return fmt.Errorf("delete series recommendations: %w", err)
	}
	return nil
}

// SaveModelMetadata saves or updates recommendation model metadata
func (r *RecommendationRepository) SaveModelMetadata(ctx context.Context, metadata domain.RecommendationModelMetadata) error {
	query := `
		INSERT INTO recommendation_model_metadata (
			model_type, total_items, feature_count, last_built_at, build_duration_ms
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(model_type) DO UPDATE SET
			total_items = excluded.total_items,
			feature_count = excluded.feature_count,
			last_built_at = excluded.last_built_at,
			build_duration_ms = excluded.build_duration_ms
	`

	_, err := r.db.ExecContext(ctx, query,
		metadata.ModelType,
		metadata.TotalItems,
		metadata.FeatureCount,
		metadata.LastBuiltAt,
		metadata.BuildDurationMs,
	)
	if err != nil {
		return fmt.Errorf("save model metadata: %w", err)
	}

	return nil
}

// GetModelMetadata retrieves metadata for a recommendation model
func (r *RecommendationRepository) GetModelMetadata(ctx context.Context, modelType string) (*domain.RecommendationModelMetadata, error) {
	query := `
		SELECT id, model_type, total_items, feature_count, last_built_at, build_duration_ms
		FROM recommendation_model_metadata
		WHERE model_type = ?
	`

	var metadata domain.RecommendationModelMetadata
	var featureCount sql.NullInt64
	var buildDurationMs sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, modelType).Scan(
		&metadata.ID,
		&metadata.ModelType,
		&metadata.TotalItems,
		&featureCount,
		&metadata.LastBuiltAt,
		&buildDurationMs,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get model metadata: %w", err)
	}

	if featureCount.Valid {
		metadata.FeatureCount = int(featureCount.Int64)
	}
	if buildDurationMs.Valid {
		metadata.BuildDurationMs = int(buildDurationMs.Int64)
	}

	return &metadata, nil
}

// Ensure RecommendationRepository implements ports.RecommendationRepository
var _ ports.RecommendationRepository = (*RecommendationRepository)(nil)

// UserWatchDataRepository implements ports.UserWatchDataRepository using SQLite
type UserWatchDataRepository struct {
	db *sql.DB
}

// NewUserWatchDataRepository creates a new user watch data repository
func NewUserWatchDataRepository(db *database.DB) *UserWatchDataRepository {
	return &UserWatchDataRepository{db: db.DB}
}

// GetUserWatchData retrieves watch history and favorites for recommendation scoring
func (r *UserWatchDataRepository) GetUserWatchData(ctx context.Context, userID string) (*ports.UserWatchData, error) {
	data := &ports.UserWatchData{
		CompletedMovies:         []int64{},
		WatchedEpisodesBySeries: make(map[int64]int),
		FavoritedMovies:         []int64{},
		FavoritedSeries:         []int64{},
	}

	// 1. Get completed movies
	completedMoviesQuery := `
		SELECT DISTINCT mm.id
		FROM watch_progress wp
		JOIN media_files mf ON wp.media_id = mf.id
		JOIN movie_metadata mm ON mf.movie_metadata_id = mm.id
		WHERE wp.user_id = ? 
		  AND wp.completed = 1 
		  AND mf.metadata_type = 'movie'
		  AND mf.movie_metadata_id IS NOT NULL
	`
	rows, err := r.db.QueryContext(ctx, completedMoviesQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("query completed movies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var movieID int64
		if err := rows.Scan(&movieID); err != nil {
			return nil, fmt.Errorf("scan completed movie: %w", err)
		}
		data.CompletedMovies = append(data.CompletedMovies, movieID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate completed movies: %w", err)
	}

	// 2. Get watched episode counts per series
	watchedEpisodesQuery := `
		SELECT sm.id, COUNT(*) as episode_count
		FROM watch_progress wp
		JOIN media_files mf ON wp.media_id = mf.id
		JOIN episode_metadata em ON mf.episode_metadata_id = em.id
		JOIN season_metadata ssm ON em.season_id = ssm.id
		JOIN series_metadata sm ON ssm.series_id = sm.id
		WHERE wp.user_id = ? 
		  AND wp.completed = 1
		  AND mf.metadata_type = 'episode'
		  AND mf.episode_metadata_id IS NOT NULL
		GROUP BY sm.id
	`
	rows2, err := r.db.QueryContext(ctx, watchedEpisodesQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("query watched episodes: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var seriesID int64
		var count int
		if err := rows2.Scan(&seriesID, &count); err != nil {
			return nil, fmt.Errorf("scan watched episode count: %w", err)
		}
		data.WatchedEpisodesBySeries[seriesID] = count
	}
	if err := rows2.Err(); err != nil {
		return nil, fmt.Errorf("iterate watched episodes: %w", err)
	}

	// 3. Get favorited movies
	favoritedMoviesQuery := `
		SELECT movie_metadata_id
		FROM favorites
		WHERE user_id = ? AND media_type = 'movie' AND movie_metadata_id IS NOT NULL
	`
	rows3, err := r.db.QueryContext(ctx, favoritedMoviesQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("query favorited movies: %w", err)
	}
	defer rows3.Close()

	for rows3.Next() {
		var movieID int64
		if err := rows3.Scan(&movieID); err != nil {
			return nil, fmt.Errorf("scan favorited movie: %w", err)
		}
		data.FavoritedMovies = append(data.FavoritedMovies, movieID)
	}
	if err := rows3.Err(); err != nil {
		return nil, fmt.Errorf("iterate favorited movies: %w", err)
	}

	// 4. Get favorited series
	favoritedSeriesQuery := `
		SELECT series_metadata_id
		FROM favorites
		WHERE user_id = ? AND media_type = 'series' AND series_metadata_id IS NOT NULL
	`
	rows4, err := r.db.QueryContext(ctx, favoritedSeriesQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("query favorited series: %w", err)
	}
	defer rows4.Close()

	for rows4.Next() {
		var seriesID int64
		if err := rows4.Scan(&seriesID); err != nil {
			return nil, fmt.Errorf("scan favorited series: %w", err)
		}
		data.FavoritedSeries = append(data.FavoritedSeries, seriesID)
	}
	if err := rows4.Err(); err != nil {
		return nil, fmt.Errorf("iterate favorited series: %w", err)
	}

	return data, nil
}

// Ensure UserWatchDataRepository implements ports.UserWatchDataRepository
var _ ports.UserWatchDataRepository = (*UserWatchDataRepository)(nil)
