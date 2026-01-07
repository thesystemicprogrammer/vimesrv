package domain

import (
	"database/sql"
	"time"
)

// Favorite represents a user's favorited movie or series
type Favorite struct {
	ID               string
	UserID           string
	MediaType        string // "movie" or "series"
	MovieMetadataID  sql.NullInt64
	SeriesMetadataID sql.NullInt64
	AddedAt          time.Time
}

// FavoriteItem represents an enriched favorite with metadata for display
type FavoriteItem struct {
	Favorite
	// Enriched fields from metadata
	Title        string
	PosterPath   sql.NullString
	BackdropPath sql.NullString
	Year         sql.NullInt64
	Rating       sql.NullFloat64
	Genres       sql.NullString // JSON or comma-separated
	MediaID      sql.NullString // For movies: the media_id to navigate to movie detail
}

// GetMetadataID returns the appropriate metadata ID based on media type
func (f *Favorite) GetMetadataID() int64 {
	if f.MediaType == "movie" && f.MovieMetadataID.Valid {
		return f.MovieMetadataID.Int64
	}
	if f.MediaType == "series" && f.SeriesMetadataID.Valid {
		return f.SeriesMetadataID.Int64
	}
	return 0
}

// IsMovie returns true if this is a movie favorite
func (f *Favorite) IsMovie() bool {
	return f.MediaType == "movie"
}

// IsSeries returns true if this is a series favorite
func (f *Favorite) IsSeries() bool {
	return f.MediaType == "series"
}
