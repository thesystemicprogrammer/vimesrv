package domain

import (
	"fmt"
	"time"
)

// Media file status constants
const (
	MediaStatusProcessing = "processing"
	MediaStatusReady      = "ready"
	MediaStatusError      = "error"
)

// Enrichment status constants
const (
	EnrichmentStatusPending         = "pending"          // Not yet processed
	EnrichmentStatusAutoLinked      = "auto_linked"      // High confidence match, automatically linked
	EnrichmentStatusCandidatesFound = "candidates_found" // Multiple candidates, awaiting user selection
	EnrichmentStatusManualRequired  = "manual_required"  // No candidates found, manual search needed
	EnrichmentStatusLinked          = "linked"           // User confirmed metadata link
	EnrichmentStatusSkipped         = "skipped"          // User chose to skip enrichment
)

// Metadata type constants
const (
	MetadataTypeNone    = ""        // Not yet determined
	MetadataTypeMovie   = "movie"   // Linked to a movie
	MetadataTypeEpisode = "episode" // Linked to a TV episode
)

// MediaFile represents a video file in the media library
type MediaFile struct {
	ID                string    `json:"id"`
	Fingerprint       string    `json:"fingerprint"`
	FilePath          string    `json:"file_path"`
	OriginalFilename  string    `json:"original_filename"`
	Filename          string    `json:"filename"`
	Title             string    `json:"title,omitempty"`
	Duration          int       `json:"duration"`
	FileSize          int64     `json:"file_size"`
	Format            string    `json:"format,omitempty"`
	VideoCodec        string    `json:"video_codec,omitempty"`
	AudioCodecs       []string  `json:"audio_codecs,omitempty"`
	Resolution        string    `json:"resolution,omitempty"`
	Width             int       `json:"width"`
	Height            int       `json:"height"`
	Bitrate           int       `json:"bitrate"`
	AudioTracks       int       `json:"audio_tracks"`
	SubtitleTracks    int       `json:"subtitle_tracks"`
	SubtitleLanguages []string  `json:"subtitle_languages,omitempty"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	ScannedAt         time.Time `json:"scanned_at"`

	// Metadata enrichment fields
	EnrichmentStatus  string `json:"enrichment_status"`             // pending, auto_linked, candidates_found, manual_required, linked, skipped
	MetadataType      string `json:"metadata_type,omitempty"`       // "", "movie", "episode"
	MovieMetadataID   *int64 `json:"movie_metadata_id,omitempty"`   // FK to movie_metadata (nullable)
	EpisodeMetadataID *int64 `json:"episode_metadata_id,omitempty"` // FK to episode_metadata (nullable)
	Edition           string `json:"edition,omitempty"`             // "Theatrical", "Director's Cut", "Extended", etc.
}

// NewMediaFile creates a new MediaFile with default values
func NewMediaFile(id, fingerprint, filePath, originalFilename, filename string) *MediaFile {
	now := time.Now()
	return &MediaFile{
		ID:                id,
		Fingerprint:       fingerprint,
		FilePath:          filePath,
		OriginalFilename:  originalFilename,
		Filename:          filename,
		Status:            MediaStatusReady,
		EnrichmentStatus:  EnrichmentStatusPending,
		MetadataType:      MetadataTypeNone,
		CreatedAt:         now,
		UpdatedAt:         now,
		ScannedAt:         now,
		AudioCodecs:       []string{},
		SubtitleLanguages: []string{},
	}
}

// IsValid checks if the media file status is valid
func (m *MediaFile) IsValid() bool {
	return m.Status == MediaStatusProcessing ||
		m.Status == MediaStatusReady ||
		m.Status == MediaStatusError
}

// SetProcessing sets the status to processing
func (m *MediaFile) SetProcessing() {
	m.Status = MediaStatusProcessing
	m.UpdatedAt = time.Now()
}

// SetReady sets the status to ready
func (m *MediaFile) SetReady() {
	m.Status = MediaStatusReady
	m.UpdatedAt = time.Now()
}

// SetError sets the status to error
func (m *MediaFile) SetError() {
	m.Status = MediaStatusError
	m.UpdatedAt = time.Now()
}

// Enrichment status methods

// SetEnrichmentAutoLinked marks the media as auto-linked with high confidence
func (m *MediaFile) SetEnrichmentAutoLinked() {
	m.EnrichmentStatus = EnrichmentStatusAutoLinked
	m.UpdatedAt = time.Now()
}

// SetEnrichmentCandidatesFound marks the media as having candidates awaiting selection
func (m *MediaFile) SetEnrichmentCandidatesFound() {
	m.EnrichmentStatus = EnrichmentStatusCandidatesFound
	m.UpdatedAt = time.Now()
}

// SetEnrichmentManualRequired marks the media as requiring manual search
func (m *MediaFile) SetEnrichmentManualRequired() {
	m.EnrichmentStatus = EnrichmentStatusManualRequired
	m.UpdatedAt = time.Now()
}

// SetEnrichmentLinked marks the media as having user-confirmed metadata
func (m *MediaFile) SetEnrichmentLinked() {
	m.EnrichmentStatus = EnrichmentStatusLinked
	m.UpdatedAt = time.Now()
}

// SetEnrichmentSkipped marks the media as skipped by user
func (m *MediaFile) SetEnrichmentSkipped() {
	m.EnrichmentStatus = EnrichmentStatusSkipped
	m.UpdatedAt = time.Now()
}

// LinkToMovie links this media file to a movie metadata record
func (m *MediaFile) LinkToMovie(movieMetadataID int64) {
	m.MetadataType = MetadataTypeMovie
	m.MovieMetadataID = &movieMetadataID
	m.EpisodeMetadataID = nil
	m.UpdatedAt = time.Now()
}

// LinkToEpisode links this media file to an episode metadata record
func (m *MediaFile) LinkToEpisode(episodeMetadataID int64) {
	m.MetadataType = MetadataTypeEpisode
	m.EpisodeMetadataID = &episodeMetadataID
	m.MovieMetadataID = nil
	m.UpdatedAt = time.Now()
}

// ClearMetadataLink removes any metadata association
func (m *MediaFile) ClearMetadataLink() {
	m.MetadataType = MetadataTypeNone
	m.MovieMetadataID = nil
	m.EpisodeMetadataID = nil
	m.EnrichmentStatus = EnrichmentStatusPending
	m.UpdatedAt = time.Now()
}

// IsLinkedToMovie returns true if this media file is linked to a movie
func (m *MediaFile) IsLinkedToMovie() bool {
	return m.MetadataType == MetadataTypeMovie && m.MovieMetadataID != nil
}

// IsLinkedToEpisode returns true if this media file is linked to an episode
func (m *MediaFile) IsLinkedToEpisode() bool {
	return m.MetadataType == MetadataTypeEpisode && m.EpisodeMetadataID != nil
}

// HasMetadata returns true if this media file has any metadata linked
func (m *MediaFile) HasMetadata() bool {
	return m.IsLinkedToMovie() || m.IsLinkedToEpisode()
}

// NeedsEnrichment returns true if this media file needs metadata enrichment
func (m *MediaFile) NeedsEnrichment() bool {
	return m.EnrichmentStatus == EnrichmentStatusPending
}

// AwaitingUserSelection returns true if this media file has candidates waiting for user selection
func (m *MediaFile) AwaitingUserSelection() bool {
	return m.EnrichmentStatus == EnrichmentStatusCandidatesFound
}

// DeriveIDFromFingerprint creates a deterministic, UUID-formatted ID from a fingerprint.
// The fingerprint is a 128-char hex string (BLAKE2b-512).
// Returns a 36-char string formatted as: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
// This ensures the same file content always produces the same media ID,
// enabling database rebuilds while preserving transcode paths and API URLs.
func DeriveIDFromFingerprint(fingerprint string) string {
	if len(fingerprint) < 32 {
		// Fallback for short fingerprints (shouldn't happen in practice)
		return fingerprint
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		fingerprint[0:8],
		fingerprint[8:12],
		fingerprint[12:16],
		fingerprint[16:20],
		fingerprint[20:32],
	)
}
