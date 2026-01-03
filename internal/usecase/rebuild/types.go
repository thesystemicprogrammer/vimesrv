package rebuild

import "time"

// RebuildDataVersion is the current version of the rebuild data format
const RebuildDataVersion = 1

// RebuildData is the JSON structure for rebuild.json
// This file is created by --prepare-rebuild and consumed by --rebuild-from-dump
type RebuildData struct {
	Version    int         `json:"version"`
	CreatedAt  time.Time   `json:"created_at"`
	Users      []UserData  `json:"users"`
	MediaLinks []MediaLink `json:"media_links"`
}

// UserData represents a user export for rebuild
type UserData struct {
	ID                 string    `json:"id"`
	Username           string    `json:"username"`
	PasswordHash       string    `json:"password_hash"`
	Role               string    `json:"role"`
	MustChangePassword bool      `json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	CreatedBy          *string   `json:"created_by,omitempty"`
}

// MediaLink represents the link between a media file and its TMDB metadata
type MediaLink struct {
	// Fingerprint is the BLAKE2b hash of the media file content
	// This is the stable identifier used to match files after rebuild
	Fingerprint string `json:"fingerprint"`

	// MetadataType is either "movie" or "episode"
	MetadataType string `json:"metadata_type"`

	// For movies: the TMDB movie ID
	TMDBID int `json:"tmdb_id,omitempty"`

	// For episodes: the TMDB series ID and episode location
	SeriesTMDBID  int `json:"series_tmdb_id,omitempty"`
	SeasonNumber  int `json:"season_number,omitempty"`
	EpisodeNumber int `json:"episode_number,omitempty"`

	// Edition info (e.g., "Director's Cut", "Extended Edition")
	Edition *string `json:"edition,omitempty"`
}

// RebuildError represents an error that occurred during rebuild
type RebuildError struct {
	Fingerprint string `json:"fingerprint,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Operation   string `json:"operation"`
	Error       string `json:"error"`
}

// RebuildErrors is the JSON structure for rebuild-errors.json
type RebuildErrors struct {
	CreatedAt time.Time      `json:"created_at"`
	Errors    []RebuildError `json:"errors"`
}

// AutoLinkData is the in-memory representation of a MediaLink for quick lookup during rebuild.
// It's keyed by fingerprint in a map for O(1) access when processing each media file.
type AutoLinkData struct {
	// MetadataType is either "movie" or "episode"
	MetadataType string

	// For movies: the TMDB movie ID
	TMDBID int

	// For episodes: the TMDB series ID and episode location
	SeriesTMDBID  int
	SeasonNumber  int
	EpisodeNumber int

	// Edition info (e.g., "Director's Cut", "Extended Edition")
	Edition string
}

// ToAutoLinkData converts a MediaLink to an AutoLinkData struct
func (m *MediaLink) ToAutoLinkData() AutoLinkData {
	edition := ""
	if m.Edition != nil {
		edition = *m.Edition
	}
	return AutoLinkData{
		MetadataType:  m.MetadataType,
		TMDBID:        m.TMDBID,
		SeriesTMDBID:  m.SeriesTMDBID,
		SeasonNumber:  m.SeasonNumber,
		EpisodeNumber: m.EpisodeNumber,
		Edition:       edition,
	}
}

// BuildAutoLinkMap builds a fingerprint-to-AutoLinkData map for quick lookup during rebuild
func BuildAutoLinkMap(data *RebuildData) map[string]AutoLinkData {
	result := make(map[string]AutoLinkData, len(data.MediaLinks))
	for _, link := range data.MediaLinks {
		result[link.Fingerprint] = link.ToAutoLinkData()
	}
	return result
}
