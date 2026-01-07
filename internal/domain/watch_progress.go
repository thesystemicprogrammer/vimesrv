package domain

import (
	"database/sql"
	"time"
)

// WatchProgress tracks a user's playback position and completion status for media
type WatchProgress struct {
	ID                string
	UserID            string
	MediaID           sql.NullString
	EpisodeMetadataID sql.NullInt64
	PositionSeconds   int
	DurationSeconds   int
	ProgressPercent   float64
	LastWatchedAt     time.Time
	Completed         bool
	ManuallyRemoved   bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ContinueWatchingItem represents an enriched watch progress item for the Continue Watching section
type ContinueWatchingItem struct {
	WatchProgress
	// Enriched fields from media/metadata
	Title        string
	PosterPath   sql.NullString
	BackdropPath sql.NullString
	MediaType    string // "movie" or "episode"
	Year         sql.NullInt64
	Resolution   sql.NullString
	// For episodes
	SeriesName       sql.NullString
	SeriesMetadataID sql.NullInt64
	SeasonNumber     sql.NullInt64
	EpisodeNumber    sql.NullInt64
	EpisodeName      sql.NullString
	// Collection support (for "Continue the Series")
	IsCollectionNext bool // True if this is a "next in collection" suggestion
	CollectionID     sql.NullInt64
	CollectionName   sql.NullString
}

// IsCompleted returns true if the video has been watched past the completion threshold (95%)
func (wp *WatchProgress) IsCompleted() bool {
	return wp.ProgressPercent >= 95.0
}

// ShouldShowInContinueWatching returns true if this item should appear in Continue Watching
func (wp *WatchProgress) ShouldShowInContinueWatching() bool {
	return !wp.Completed && !wp.ManuallyRemoved
}
