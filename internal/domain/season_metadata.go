package domain

import "time"

// SeasonMetadata stores TMDB season information (language-independent fields)
type SeasonMetadata struct {
	ID           int64     `json:"id"`
	SeriesID     int64     `json:"series_id"`
	TMDBID       int       `json:"tmdb_id"`
	SeasonNumber int       `json:"season_number"`
	AirDate      string    `json:"air_date,omitempty"`
	PosterPath   string    `json:"poster_path,omitempty"`
	EpisodeCount int       `json:"episode_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Loaded translations (not persisted directly, joined on query)
	Translations []SeasonMetadataTranslation `json:"translations,omitempty"`
	// Loaded episodes (not persisted directly, joined on query)
	Episodes []EpisodeMetadata `json:"episodes,omitempty"`
}

// SeasonMetadataTranslation stores translatable season content
type SeasonMetadataTranslation struct {
	ID               int64     `json:"id"`
	SeasonMetadataID int64     `json:"season_metadata_id"`
	Language         string    `json:"language"`
	Name             string    `json:"name"`
	Overview         string    `json:"overview,omitempty"`
	PosterPath       string    `json:"poster_path,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NewSeasonMetadata creates a new SeasonMetadata with default values
func NewSeasonMetadata(seriesID int64, tmdbID int, seasonNumber int) *SeasonMetadata {
	now := time.Now()
	return &SeasonMetadata{
		SeriesID:     seriesID,
		TMDBID:       tmdbID,
		SeasonNumber: seasonNumber,
		Translations: []SeasonMetadataTranslation{},
		Episodes:     []EpisodeMetadata{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// NewSeasonMetadataTranslation creates a new translation entry
func NewSeasonMetadataTranslation(seasonMetadataID int64, language, name string) *SeasonMetadataTranslation {
	now := time.Now()
	return &SeasonMetadataTranslation{
		SeasonMetadataID: seasonMetadataID,
		Language:         language,
		Name:             name,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// GetTranslation returns the translation for the specified language, or nil if not found
func (s *SeasonMetadata) GetTranslation(language string) *SeasonMetadataTranslation {
	for i := range s.Translations {
		if s.Translations[i].Language == language {
			return &s.Translations[i]
		}
	}
	return nil
}

// GetName returns the name for the specified language, or a default name
func (s *SeasonMetadata) GetName(language string) string {
	if t := s.GetTranslation(language); t != nil && t.Name != "" {
		return t.Name
	}
	// Default season name if no translation available
	return ""
}

// GetOverview returns the overview for the specified language, or empty string if not found
func (s *SeasonMetadata) GetOverview(language string) string {
	if t := s.GetTranslation(language); t != nil {
		return t.Overview
	}
	return ""
}

// GetPosterPath returns the poster path for the specified language, falling back to base poster
func (s *SeasonMetadata) GetPosterPath(language string) string {
	if t := s.GetTranslation(language); t != nil && t.PosterPath != "" {
		return t.PosterPath
	}
	return s.PosterPath
}

// AddTranslation adds or updates a translation for the season
func (s *SeasonMetadata) AddTranslation(translation SeasonMetadataTranslation) {
	for i := range s.Translations {
		if s.Translations[i].Language == translation.Language {
			s.Translations[i] = translation
			return
		}
	}
	s.Translations = append(s.Translations, translation)
}

// GetEpisode returns the episode by number, or nil if not found
func (s *SeasonMetadata) GetEpisode(episodeNumber int) *EpisodeMetadata {
	for i := range s.Episodes {
		if s.Episodes[i].EpisodeNumber == episodeNumber {
			return &s.Episodes[i]
		}
	}
	return nil
}
