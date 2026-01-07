package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

type FavoriteRepository struct {
	db *sql.DB
}

func NewFavoriteRepository(db *database.DB) *FavoriteRepository {
	return &FavoriteRepository{db: db.DB}
}

func (r *FavoriteRepository) scanFavoriteRow(s scanner) (*domain.Favorite, error) {
	var fav domain.Favorite

	err := s.Scan(
		&fav.ID,
		&fav.UserID,
		&fav.MediaType,
		&fav.MovieMetadataID,
		&fav.SeriesMetadataID,
		&fav.AddedAt,
	)
	if err != nil {
		return nil, err
	}

	return &fav, nil
}

func (r *FavoriteRepository) scanFavoriteItemRow(s scanner) (*domain.FavoriteItem, error) {
	var item domain.FavoriteItem

	err := s.Scan(
		&item.ID,
		&item.UserID,
		&item.MediaType,
		&item.MovieMetadataID,
		&item.SeriesMetadataID,
		&item.AddedAt,
		&item.Title,
		&item.PosterPath,
		&item.BackdropPath,
		&item.Year,
		&item.Rating,
		&item.Genres,
		&item.MediaID,
	)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// AddFavorite adds a movie or series to favorites
func (r *FavoriteRepository) AddFavorite(ctx context.Context, favorite *domain.Favorite) error {
	if favorite.ID == "" {
		favorite.ID = uuid.New().String()
	}

	favorite.AddedAt = time.Now().UTC()

	const query = `
		INSERT INTO favorites (id, user_id, media_type, movie_metadata_id, series_metadata_id, added_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		favorite.ID,
		favorite.UserID,
		favorite.MediaType,
		favorite.MovieMetadataID,
		favorite.SeriesMetadataID,
		favorite.AddedAt,
	)
	if err != nil {
		logger.Error().Err(err).Str("user_id", favorite.UserID).Str("media_type", favorite.MediaType).Msg("failed to add favorite")
		return err
	}

	return nil
}

// RemoveFavorite removes a movie or series from favorites
func (r *FavoriteRepository) RemoveFavorite(ctx context.Context, userID string, mediaType string, metadataID int64) error {
	var query string
	if mediaType == "movie" {
		query = `DELETE FROM favorites WHERE user_id = ? AND media_type = 'movie' AND movie_metadata_id = ?`
	} else {
		query = `DELETE FROM favorites WHERE user_id = ? AND media_type = 'series' AND series_metadata_id = ?`
	}

	result, err := r.db.ExecContext(ctx, query, userID, metadataID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Str("media_type", mediaType).Int64("metadata_id", metadataID).Msg("failed to remove favorite")
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logger.Warn().Str("user_id", userID).Str("media_type", mediaType).Int64("metadata_id", metadataID).Msg("no favorite found to remove")
	}

	return nil
}

// GetUserFavorites retrieves all favorited items for a user with enriched metadata
func (r *FavoriteRepository) GetUserFavorites(ctx context.Context, userID string, limit int) ([]domain.FavoriteItem, error) {
	const query = `
		SELECT 
			f.id, f.user_id, f.media_type, f.movie_metadata_id, f.series_metadata_id, f.added_at,
			CASE 
				WHEN f.media_type = 'movie' THEN COALESCE(mmt.title, mm.original_title)
				WHEN f.media_type = 'series' THEN COALESCE(smt.name, sm.original_name)
			END as title,
			CASE 
				WHEN f.media_type = 'movie' THEN COALESCE(mmt.poster_path, mm.poster_path)
				WHEN f.media_type = 'series' THEN COALESCE(smt.poster_path, sm.poster_path)
			END as poster_path,
			CASE 
				WHEN f.media_type = 'movie' THEN COALESCE(mmt.backdrop_path, mm.backdrop_path)
				WHEN f.media_type = 'series' THEN COALESCE(smt.backdrop_path, sm.backdrop_path)
			END as backdrop_path,
			CASE 
				WHEN f.media_type = 'movie' THEN CAST(substr(mm.release_date, 1, 4) AS INTEGER)
				WHEN f.media_type = 'series' THEN CAST(substr(sm.first_air_date, 1, 4) AS INTEGER)
			END as year,
			CASE 
				WHEN f.media_type = 'movie' THEN mm.vote_average
				WHEN f.media_type = 'series' THEN sm.vote_average
			END as rating,
			CASE 
				WHEN f.media_type = 'movie' THEN mm.genres
				WHEN f.media_type = 'series' THEN sm.genres
			END as genres,
			CASE 
				WHEN f.media_type = 'movie' THEN mf.id
				ELSE NULL
			END as media_id
		FROM favorites f
		LEFT JOIN movie_metadata mm ON f.movie_metadata_id = mm.id
		LEFT JOIN movie_metadata_translations mmt ON mm.id = mmt.movie_metadata_id AND mmt.language = 'en'
		LEFT JOIN series_metadata sm ON f.series_metadata_id = sm.id
		LEFT JOIN series_metadata_translations smt ON sm.id = smt.series_metadata_id AND smt.language = 'en'
		LEFT JOIN media_files mf ON f.movie_metadata_id = mf.movie_metadata_id
		WHERE f.user_id = ?
		ORDER BY f.added_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("failed to get user favorites")
		return nil, err
	}
	defer rows.Close()

	var items []domain.FavoriteItem
	for rows.Next() {
		item, err := r.scanFavoriteItemRow(rows)
		if err != nil {
			logger.Error().Err(err).Msg("failed to scan favorite item row")
			return nil, err
		}
		items = append(items, *item)
	}

	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("error iterating favorite rows")
		return nil, err
	}

	return items, nil
}

// IsFavorited checks if a movie or series is favorited by a user
func (r *FavoriteRepository) IsFavorited(ctx context.Context, userID string, mediaType string, metadataID int64) (bool, error) {
	var query string
	if mediaType == "movie" {
		query = `SELECT 1 FROM favorites WHERE user_id = ? AND media_type = 'movie' AND movie_metadata_id = ? LIMIT 1`
	} else {
		query = `SELECT 1 FROM favorites WHERE user_id = ? AND media_type = 'series' AND series_metadata_id = ? LIMIT 1`
	}

	var exists int
	err := r.db.QueryRowContext(ctx, query, userID, metadataID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Str("media_type", mediaType).Int64("metadata_id", metadataID).Msg("failed to check if favorited")
		return false, err
	}

	return true, nil
}
