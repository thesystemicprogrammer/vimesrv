package domain

import "time"

// EpisodeMetadata stores TMDB episode information (language-independent fields)
type EpisodeMetadata struct {
	ID            int64     `json:"id"`
	SeasonID      int64     `json:"season_id"`
	TMDBID        int       `json:"tmdb_id"`
	EpisodeNumber int       `json:"episode_number"`
	AirDate       string    `json:"air_date,omitempty"`
	StillPath     string    `json:"still_path,omitempty"`
	Runtime       int       `json:"runtime"`
	VoteAverage   float64   `json:"vote_average"`
	VoteCount     int       `json:"vote_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Loaded translations (not persisted directly, joined on query)
	Translations []EpisodeMetadataTranslation `json:"translations,omitempty"`
}

// EpisodeMetadataTranslation stores translatable episode content
type EpisodeMetadataTranslation struct {
	ID                int64     `json:"id"`
	EpisodeMetadataID int64     `json:"episode_metadata_id"`
	Language          string    `json:"language"`
	Name              string    `json:"name"`
	Overview          string    `json:"overview,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// NewEpisodeMetadata creates a new EpisodeMetadata with default values
func NewEpisodeMetadata(seasonID int64, tmdbID int, episodeNumber int) *EpisodeMetadata {
	now := time.Now()
	return &EpisodeMetadata{
		SeasonID:      seasonID,
		TMDBID:        tmdbID,
		EpisodeNumber: episodeNumber,
		Translations:  []EpisodeMetadataTranslation{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// NewEpisodeMetadataTranslation creates a new translation entry
func NewEpisodeMetadataTranslation(episodeMetadataID int64, language, name string) *EpisodeMetadataTranslation {
	now := time.Now()
	return &EpisodeMetadataTranslation{
		EpisodeMetadataID: episodeMetadataID,
		Language:          language,
		Name:              name,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// GetTranslation returns the translation for the specified language, or nil if not found
func (e *EpisodeMetadata) GetTranslation(language string) *EpisodeMetadataTranslation {
	for i := range e.Translations {
		if e.Translations[i].Language == language {
			return &e.Translations[i]
		}
	}
	return nil
}

// GetName returns the name for the specified language, or empty string if not found
func (e *EpisodeMetadata) GetName(language string) string {
	if t := e.GetTranslation(language); t != nil && t.Name != "" {
		return t.Name
	}
	return ""
}

// GetOverview returns the overview for the specified language, or empty string if not found
func (e *EpisodeMetadata) GetOverview(language string) string {
	if t := e.GetTranslation(language); t != nil {
		return t.Overview
	}
	return ""
}

// AddTranslation adds or updates a translation for the episode
func (e *EpisodeMetadata) AddTranslation(translation EpisodeMetadataTranslation) {
	for i := range e.Translations {
		if e.Translations[i].Language == translation.Language {
			e.Translations[i] = translation
			return
		}
	}
	e.Translations = append(e.Translations, translation)
}
