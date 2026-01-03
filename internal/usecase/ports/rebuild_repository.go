package ports

import (
	"context"
)

// ExportedMediaLink represents a media file linked to TMDB metadata for export
type ExportedMediaLink struct {
	// Fingerprint is the BLAKE2b hash of the media file content
	Fingerprint string

	// MetadataType is either "movie" or "episode"
	MetadataType string

	// For movies: the TMDB movie ID
	TMDBID int

	// For episodes: the TMDB series ID and episode location
	SeriesTMDBID  int
	SeasonNumber  int
	EpisodeNumber int

	// Edition info (e.g., "Director's Cut", "Extended Edition")
	Edition *string
}

// ExportedUser represents a user export for rebuild
type ExportedUser struct {
	ID                 string
	Username           string
	PasswordHash       string
	Role               string
	MustChangePassword bool
	CreatedAt          string // RFC3339 format
	UpdatedAt          string // RFC3339 format
	CreatedBy          *string
}

// RebuildRepository provides database operations for the rebuild feature
type RebuildRepository interface {
	// ExportMediaLinks returns all media files that are linked to metadata
	// with their fingerprints and TMDB IDs for export
	ExportMediaLinks(ctx context.Context) ([]ExportedMediaLink, error)

	// ClearAllTables deletes all data from all tables and runs VACUUM
	ClearAllTables(ctx context.Context) error

	// ImportUser imports a user with the exact ID and password hash from the export
	ImportUser(ctx context.Context, user ExportedUser) error
}
