package domain

import "time"

// MetadataCandidate status constants
const (
	CandidateStatusPending  = "pending"
	CandidateStatusSelected = "selected"
	CandidateStatusRejected = "rejected"
)

// MetadataCandidate type constants
const (
	CandidateTypeMovie  = "movie"
	CandidateTypeSeries = "series"
)

// MetadataCandidate stores potential TMDB matches for user selection
type MetadataCandidate struct {
	ID              int64     `json:"id"`
	MediaFileID     string    `json:"media_file_id"`
	CandidateType   string    `json:"candidate_type"` // "movie" or "series"
	TMDBID          int       `json:"tmdb_id"`
	Title           string    `json:"title"`
	ReleaseDate     string    `json:"release_date,omitempty"`
	Overview        string    `json:"overview,omitempty"`
	PosterPath      string    `json:"poster_path,omitempty"` // TMDB URL path, not local
	ConfidenceScore int       `json:"confidence_score"`
	SeasonNumber    *int      `json:"season_number,omitempty"`  // For series: parsed season number
	EpisodeNumber   *int      `json:"episode_number,omitempty"` // For series: parsed episode number
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

// NewMetadataCandidate creates a new MetadataCandidate with default values
func NewMetadataCandidate(mediaFileID string, candidateType string, tmdbID int, title string, confidenceScore int) *MetadataCandidate {
	return &MetadataCandidate{
		MediaFileID:     mediaFileID,
		CandidateType:   candidateType,
		TMDBID:          tmdbID,
		Title:           title,
		ConfidenceScore: confidenceScore,
		Status:          CandidateStatusPending,
		CreatedAt:       time.Now(),
	}
}

// IsMovie returns true if this candidate is for a movie
func (c *MetadataCandidate) IsMovie() bool {
	return c.CandidateType == CandidateTypeMovie
}

// IsSeries returns true if this candidate is for a series
func (c *MetadataCandidate) IsSeries() bool {
	return c.CandidateType == CandidateTypeSeries
}

// IsPending returns true if the candidate status is pending
func (c *MetadataCandidate) IsPending() bool {
	return c.Status == CandidateStatusPending
}

// Select marks this candidate as selected
func (c *MetadataCandidate) Select() {
	c.Status = CandidateStatusSelected
}

// Reject marks this candidate as rejected
func (c *MetadataCandidate) Reject() {
	c.Status = CandidateStatusRejected
}

// SetEpisodeInfo sets the season and episode numbers for series candidates
func (c *MetadataCandidate) SetEpisodeInfo(season, episode int) {
	c.SeasonNumber = &season
	c.EpisodeNumber = &episode
}
