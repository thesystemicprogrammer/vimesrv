package domain

import "time"

// SeriesCredit represents a cast or crew member associated with a TV series (aggregate across all episodes)
type SeriesCredit struct {
	ID                int64      `json:"id"`
	SeriesMetadataID  int64      `json:"series_metadata_id"`
	CreditType        CreditType `json:"credit_type"`
	TMDBPersonID      int        `json:"tmdb_person_id"`
	Name              string     `json:"name"`
	Roles             string     `json:"roles,omitempty"`      // JSON array of {character, episode_count} for cast
	Jobs              string     `json:"jobs,omitempty"`       // JSON array of {job, episode_count} for crew
	Department        string     `json:"department,omitempty"` // For crew (e.g., "Directing", "Writing")
	ProfilePath       string     `json:"profile_path,omitempty"`
	TotalEpisodeCount int        `json:"total_episode_count"`
	DisplayOrder      int        `json:"display_order"`
	CreatedAt         time.Time  `json:"created_at"`
}

// SeriesRole represents a character played by a cast member
type SeriesRole struct {
	Character    string `json:"character"`
	EpisodeCount int    `json:"episode_count"`
}

// SeriesJob represents a job performed by a crew member
type SeriesJob struct {
	Job          string `json:"job"`
	EpisodeCount int    `json:"episode_count"`
}

// NewSeriesCastCredit creates a new series cast credit
func NewSeriesCastCredit(seriesMetadataID int64, tmdbPersonID int, name, roles, profilePath string, totalEpisodeCount, order int) *SeriesCredit {
	return &SeriesCredit{
		SeriesMetadataID:  seriesMetadataID,
		CreditType:        CreditTypeCast,
		TMDBPersonID:      tmdbPersonID,
		Name:              name,
		Roles:             roles,
		ProfilePath:       profilePath,
		TotalEpisodeCount: totalEpisodeCount,
		DisplayOrder:      order,
		CreatedAt:         time.Now(),
	}
}

// NewSeriesCrewCredit creates a new series crew credit
func NewSeriesCrewCredit(seriesMetadataID int64, tmdbPersonID int, name, jobs, department, profilePath string, totalEpisodeCount, order int) *SeriesCredit {
	return &SeriesCredit{
		SeriesMetadataID:  seriesMetadataID,
		CreditType:        CreditTypeCrew,
		TMDBPersonID:      tmdbPersonID,
		Name:              name,
		Jobs:              jobs,
		Department:        department,
		ProfilePath:       profilePath,
		TotalEpisodeCount: totalEpisodeCount,
		DisplayOrder:      order,
		CreatedAt:         time.Now(),
	}
}

// IsCast returns true if this is a cast credit
func (c *SeriesCredit) IsCast() bool {
	return c.CreditType == CreditTypeCast
}

// IsCrew returns true if this is a crew credit
func (c *SeriesCredit) IsCrew() bool {
	return c.CreditType == CreditTypeCrew
}
