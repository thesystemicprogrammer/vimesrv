package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

type WatchProgressRepository struct {
	db *sql.DB
}

func NewWatchProgressRepository(db *database.DB) *WatchProgressRepository {
	return &WatchProgressRepository{db: db.DB}
}

func (r *WatchProgressRepository) scanWatchProgressRow(s scanner) (*domain.WatchProgress, error) {
	var wp domain.WatchProgress
	var completed, manuallyRemoved int

	err := s.Scan(
		&wp.ID,
		&wp.UserID,
		&wp.MediaID,
		&wp.EpisodeMetadataID,
		&wp.PositionSeconds,
		&wp.DurationSeconds,
		&wp.ProgressPercent,
		&wp.LastWatchedAt,
		&completed,
		&manuallyRemoved,
		&wp.CreatedAt,
		&wp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	wp.Completed = completed == 1
	wp.ManuallyRemoved = manuallyRemoved == 1

	return &wp, nil
}

func (r *WatchProgressRepository) scanContinueWatchingRow(s scanner) (*domain.ContinueWatchingItem, error) {
	var item domain.ContinueWatchingItem
	var completed, manuallyRemoved, isCollectionNext int

	err := s.Scan(
		&item.ID,
		&item.UserID,
		&item.MediaID,
		&item.EpisodeMetadataID,
		&item.PositionSeconds,
		&item.DurationSeconds,
		&item.ProgressPercent,
		&item.LastWatchedAt,
		&completed,
		&manuallyRemoved,
		&item.CreatedAt,
		&item.UpdatedAt,
		// Enriched fields
		&item.Title,
		&item.PosterPath,
		&item.BackdropPath,
		&item.MediaType,
		&item.Year,
		&item.Resolution,
		&item.SeriesName,
		&item.SeriesMetadataID,
		&item.SeasonNumber,
		&item.EpisodeNumber,
		&item.EpisodeName,
		&isCollectionNext,
		&item.CollectionID,
		&item.CollectionName,
	)
	if err != nil {
		return nil, err
	}

	item.Completed = completed == 1
	item.ManuallyRemoved = manuallyRemoved == 1
	item.IsCollectionNext = isCollectionNext == 1

	return &item, nil
}

// SaveProgress creates or updates watch progress for a user
func (r *WatchProgressRepository) SaveProgress(ctx context.Context, progress *domain.WatchProgress) error {
	// Generate ID if new
	if progress.ID == "" {
		progress.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	progress.UpdatedAt = now

	// Determine completion status
	progress.Completed = progress.ProgressPercent >= 95.0

	completedInt := 0
	if progress.Completed {
		completedInt = 1
	}
	manuallyRemovedInt := 0
	if progress.ManuallyRemoved {
		manuallyRemovedInt = 1
	}

	// Use different UPSERT queries depending on whether episode_metadata_id is NULL
	// This is necessary because we have partial unique indexes
	var query string
	var err error

	if !progress.EpisodeMetadataID.Valid {
		// For movies (episode_metadata_id IS NULL)
		// Use the idx_watch_progress_unique_movie index
		query = `
			INSERT INTO watch_progress (
				id, user_id, media_id, episode_metadata_id, 
				position_seconds, duration_seconds, progress_percent,
				last_watched_at, completed, manually_removed, 
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(user_id, media_id) WHERE episode_metadata_id IS NULL DO UPDATE SET
				position_seconds = excluded.position_seconds,
				duration_seconds = excluded.duration_seconds,
				progress_percent = excluded.progress_percent,
				last_watched_at = excluded.last_watched_at,
				completed = excluded.completed,
				updated_at = excluded.updated_at
		`
		_, err = r.db.ExecContext(ctx, query,
			progress.ID,
			progress.UserID,
			progress.MediaID,
			progress.EpisodeMetadataID,
			progress.PositionSeconds,
			progress.DurationSeconds,
			progress.ProgressPercent,
			progress.LastWatchedAt,
			completedInt,
			manuallyRemovedInt,
			now, // created_at
			now, // updated_at
		)
	} else {
		// For episodes (episode_metadata_id IS NOT NULL)
		// Use the idx_watch_progress_unique_episode index
		query = `
			INSERT INTO watch_progress (
				id, user_id, media_id, episode_metadata_id, 
				position_seconds, duration_seconds, progress_percent,
				last_watched_at, completed, manually_removed, 
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(user_id, media_id, episode_metadata_id) WHERE episode_metadata_id IS NOT NULL DO UPDATE SET
				position_seconds = excluded.position_seconds,
				duration_seconds = excluded.duration_seconds,
				progress_percent = excluded.progress_percent,
				last_watched_at = excluded.last_watched_at,
				completed = excluded.completed,
				updated_at = excluded.updated_at
		`
		_, err = r.db.ExecContext(ctx, query,
			progress.ID,
			progress.UserID,
			progress.MediaID,
			progress.EpisodeMetadataID,
			progress.PositionSeconds,
			progress.DurationSeconds,
			progress.ProgressPercent,
			progress.LastWatchedAt,
			completedInt,
			manuallyRemovedInt,
			now, // created_at
			now, // updated_at
		)
	}

	if err != nil {
		logger.Error().Err(err).Str("user_id", progress.UserID).Msg("failed to save watch progress")
		return err
	}

	return nil
}

// GetProgress retrieves watch progress for a specific media item
func (r *WatchProgressRepository) GetProgress(ctx context.Context, userID, mediaID string, episodeID *int64) (*domain.WatchProgress, error) {
	const query = `
		SELECT id, user_id, media_id, episode_metadata_id,
			position_seconds, duration_seconds, progress_percent,
			last_watched_at, completed, manually_removed,
			created_at, updated_at
		FROM watch_progress
		WHERE user_id = ? AND media_id = ? AND 
			(episode_metadata_id = ? OR (episode_metadata_id IS NULL AND ? IS NULL))
	`

	row := r.db.QueryRowContext(ctx, query, userID, mediaID, episodeID, episodeID)
	progress, err := r.scanWatchProgressRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Str("media_id", mediaID).Msg("failed to get watch progress")
		return nil, err
	}

	return progress, nil
}

// GetContinueWatching retrieves in-progress items for Continue Watching section
func (r *WatchProgressRepository) GetContinueWatching(ctx context.Context, userID string, limit int) ([]domain.ContinueWatchingItem, error) {
	const query = `
		SELECT 
			wp.id, wp.user_id, wp.media_id, wp.episode_metadata_id,
			wp.position_seconds, wp.duration_seconds, wp.progress_percent,
			wp.last_watched_at, wp.completed, wp.manually_removed,
			wp.created_at, wp.updated_at,
			-- Enriched fields for movies
			CASE 
				WHEN mf.metadata_type = 'movie' THEN COALESCE(mmt.title, mm.original_title)
				WHEN mf.metadata_type = 'episode' THEN COALESCE(emt.name, 'Episode ' || em.episode_number)
				ELSE mf.title
			END as title,
			CASE 
				WHEN mf.metadata_type = 'movie' THEN COALESCE(mmt.poster_path, mm.poster_path)
				WHEN mf.metadata_type = 'episode' THEN COALESCE(emt.still_path, em.still_path, sm.poster_path, series_meta.poster_path)
				ELSE NULL
			END as poster_path,
			CASE 
				WHEN mf.metadata_type = 'movie' THEN COALESCE(mmt.backdrop_path, mm.backdrop_path)
				WHEN mf.metadata_type = 'episode' THEN COALESCE(series_mt.backdrop_path, series_meta.backdrop_path)
				ELSE NULL
			END as backdrop_path,
			mf.metadata_type as media_type,
			CASE 
				WHEN mf.metadata_type = 'movie' THEN CAST(substr(mm.release_date, 1, 4) AS INTEGER)
				ELSE NULL
			END as year,
			mf.resolution,
			-- Episode-specific fields
			CASE 
				WHEN mf.metadata_type = 'episode' THEN COALESCE(series_mt.name, series_meta.original_name)
				ELSE NULL
			END as series_name,
			CASE 
				WHEN mf.metadata_type = 'episode' THEN series_meta.id
				ELSE NULL
			END as series_metadata_id,
			seas_meta.season_number,
			em.episode_number,
			COALESCE(emt.name, 'Episode ' || em.episode_number) as episode_name,
			0 as is_collection_next,
			NULL as collection_id,
			NULL as collection_name
		FROM watch_progress wp
		JOIN media_files mf ON wp.media_id = mf.id
		LEFT JOIN movie_metadata mm ON mf.movie_metadata_id = mm.id
		LEFT JOIN movie_metadata_translations mmt ON mm.id = mmt.movie_metadata_id AND mmt.language = 'en'
		LEFT JOIN episode_metadata em ON mf.episode_metadata_id = em.id
		LEFT JOIN episode_metadata_translations emt ON em.id = emt.episode_metadata_id AND emt.language = 'en'
		LEFT JOIN season_metadata seas_meta ON em.season_id = seas_meta.id
		LEFT JOIN season_metadata_translations sm ON seas_meta.id = sm.season_metadata_id AND sm.language = 'en'
		LEFT JOIN series_metadata series_meta ON seas_meta.series_id = series_meta.id
		LEFT JOIN series_metadata_translations series_mt ON series_meta.id = series_mt.series_metadata_id AND series_mt.language = 'en'
		WHERE wp.user_id = ? AND wp.completed = 0 AND wp.manually_removed = 0
		ORDER BY wp.last_watched_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("failed to get continue watching items")
		return nil, err
	}
	defer rows.Close()

	var items []domain.ContinueWatchingItem
	for rows.Next() {
		item, err := r.scanContinueWatchingRow(rows)
		if err != nil {
			logger.Error().Err(err).Msg("failed to scan continue watching row")
			return nil, err
		}
		items = append(items, *item)
	}

	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("error iterating continue watching rows")
		return nil, err
	}

	return items, nil
}

// RemoveFromContinueWatching marks an item as manually removed
func (r *WatchProgressRepository) RemoveFromContinueWatching(ctx context.Context, userID, mediaID string, episodeID *int64) error {
	const query = `
		UPDATE watch_progress
		SET manually_removed = 1, updated_at = ?
		WHERE user_id = ? AND media_id = ? AND 
			(episode_metadata_id = ? OR (episode_metadata_id IS NULL AND ? IS NULL))
	`

	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, query, now, userID, mediaID, episodeID, episodeID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Str("media_id", mediaID).Msg("failed to remove from continue watching")
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logger.Warn().Str("user_id", userID).Str("media_id", mediaID).Msg("no watch progress found to remove")
	}

	return nil
}

// GetWatchHistory retrieves completed watch history for a user
func (r *WatchProgressRepository) GetWatchHistory(ctx context.Context, userID string, page, perPage int) ([]domain.ContinueWatchingItem, int, error) {
	offset := (page - 1) * perPage

	// Count total items
	const countQuery = `
		SELECT COUNT(*) FROM watch_progress
		WHERE user_id = ? AND completed = 1
	`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("failed to count watch history")
		return nil, 0, err
	}

	// Get items (reuse continue watching query structure but with completed = 1)
	const query = `
		SELECT 
			wp.id, wp.user_id, wp.media_id, wp.episode_metadata_id,
			wp.position_seconds, wp.duration_seconds, wp.progress_percent,
			wp.last_watched_at, wp.completed, wp.manually_removed,
			wp.created_at, wp.updated_at,
			CASE 
				WHEN mf.metadata_type = 'movie' THEN COALESCE(mmt.title, mm.original_title)
				WHEN mf.metadata_type = 'episode' THEN COALESCE(emt.name, 'Episode ' || em.episode_number)
				ELSE mf.title
			END as title,
			CASE 
				WHEN mf.metadata_type = 'movie' THEN COALESCE(mmt.poster_path, mm.poster_path)
				WHEN mf.metadata_type = 'episode' THEN COALESCE(emt.still_path, em.still_path, sm.poster_path, series_meta.poster_path)
				ELSE NULL
			END as poster_path,
			CASE 
				WHEN mf.metadata_type = 'movie' THEN COALESCE(mmt.backdrop_path, mm.backdrop_path)
				WHEN mf.metadata_type = 'episode' THEN COALESCE(series_mt.backdrop_path, series_meta.backdrop_path)
				ELSE NULL
			END as backdrop_path,
			mf.metadata_type as media_type,
			CASE 
				WHEN mf.metadata_type = 'movie' THEN CAST(substr(mm.release_date, 1, 4) AS INTEGER)
				ELSE NULL
			END as year,
			mf.resolution,
			CASE 
				WHEN mf.metadata_type = 'episode' THEN COALESCE(series_mt.name, series_meta.original_name)
				ELSE NULL
			END as series_name,
			CASE 
				WHEN mf.metadata_type = 'episode' THEN series_meta.id
				ELSE NULL
			END as series_metadata_id,
			seas_meta.season_number,
			em.episode_number,
			COALESCE(emt.name, 'Episode ' || em.episode_number) as episode_name,
			0 as is_collection_next,
			NULL as collection_id,
			NULL as collection_name
		FROM watch_progress wp
		JOIN media_files mf ON wp.media_id = mf.id
		LEFT JOIN movie_metadata mm ON mf.movie_metadata_id = mm.id
		LEFT JOIN movie_metadata_translations mmt ON mm.id = mmt.movie_metadata_id AND mmt.language = 'en'
		LEFT JOIN episode_metadata em ON mf.episode_metadata_id = em.id
		LEFT JOIN episode_metadata_translations emt ON em.id = emt.episode_metadata_id AND emt.language = 'en'
		LEFT JOIN season_metadata seas_meta ON em.season_id = seas_meta.id
		LEFT JOIN season_metadata_translations sm ON seas_meta.id = sm.season_metadata_id AND sm.language = 'en'
		LEFT JOIN series_metadata series_meta ON seas_meta.series_id = series_meta.id
		LEFT JOIN series_metadata_translations series_mt ON series_meta.id = series_mt.series_metadata_id AND series_mt.language = 'en'
		WHERE wp.user_id = ? AND wp.completed = 1
		ORDER BY wp.last_watched_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, userID, perPage, offset)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("failed to get watch history")
		return nil, 0, err
	}
	defer rows.Close()

	var items []domain.ContinueWatchingItem
	for rows.Next() {
		item, err := r.scanContinueWatchingRow(rows)
		if err != nil {
			logger.Error().Err(err).Msg("failed to scan watch history row")
			return nil, 0, err
		}
		items = append(items, *item)
	}

	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("error iterating watch history rows")
		return nil, 0, err
	}

	return items, total, nil
}

// DeleteHistory hard deletes all watch progress for a user
func (r *WatchProgressRepository) DeleteHistory(ctx context.Context, userID string) error {
	const query = `DELETE FROM watch_progress WHERE user_id = ?`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("failed to delete watch history")
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	logger.Info().Str("user_id", userID).Int64("rows_deleted", rowsAffected).Msg("deleted watch history")

	return nil
}

// MarkAsCompleted explicitly marks an item as completed
func (r *WatchProgressRepository) MarkAsCompleted(ctx context.Context, userID, mediaID string, episodeID *int64) error {
	const query = `
		UPDATE watch_progress
		SET completed = 1, progress_percent = 100, updated_at = ?
		WHERE user_id = ? AND media_id = ? AND 
			(episode_metadata_id = ? OR (episode_metadata_id IS NULL AND ? IS NULL))
	`

	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, query, now, userID, mediaID, episodeID, episodeID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Str("media_id", mediaID).Msg("failed to mark as completed")
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no watch progress found to mark as completed")
	}

	return nil
}
