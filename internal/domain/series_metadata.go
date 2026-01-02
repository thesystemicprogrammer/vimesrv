package domain

import "time"

// SeriesMetadata stores TMDB TV series information (language-independent fields)
type SeriesMetadata struct {
	ID               int64     `json:"id"`
	TMDBID           int       `json:"tmdb_id"`
	OriginalName     string    `json:"original_name"`
	FirstAirDate     string    `json:"first_air_date,omitempty"`
	LastAirDate      string    `json:"last_air_date,omitempty"`
	Status           string    `json:"status,omitempty"`
	PosterPath       string    `json:"poster_path,omitempty"`
	BackdropPath     string    `json:"backdrop_path,omitempty"`
	Genres           []string  `json:"genres,omitempty"`
	Networks         []string  `json:"networks,omitempty"`
	VoteAverage      float64   `json:"vote_average"`
	VoteCount        int       `json:"vote_count"`
	Popularity       float64   `json:"popularity"`
	NumberOfSeasons  int       `json:"number_of_seasons"`
	NumberOfEpisodes int       `json:"number_of_episodes"`
	OriginalLang     string    `json:"original_language,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Loaded translations (not persisted directly, joined on query)
	Translations []SeriesMetadataTranslation `json:"translations,omitempty"`
	// Loaded seasons (not persisted directly, joined on query)
	Seasons []SeasonMetadata `json:"seasons,omitempty"`
}

// SeriesMetadataTranslation stores translatable series content
type SeriesMetadataTranslation struct {
	ID               int64     `json:"id"`
	SeriesMetadataID int64     `json:"series_metadata_id"`
	Language         string    `json:"language"`
	Name             string    `json:"name"`
	Overview         string    `json:"overview,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NewSeriesMetadata creates a new SeriesMetadata with default values
func NewSeriesMetadata(tmdbID int, originalName string) *SeriesMetadata {
	now := time.Now()
	return &SeriesMetadata{
		TMDBID:       tmdbID,
		OriginalName: originalName,
		Genres:       []string{},
		Networks:     []string{},
		Translations: []SeriesMetadataTranslation{},
		Seasons:      []SeasonMetadata{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// NewSeriesMetadataTranslation creates a new translation entry
func NewSeriesMetadataTranslation(seriesMetadataID int64, language, name string) *SeriesMetadataTranslation {
	now := time.Now()
	return &SeriesMetadataTranslation{
		SeriesMetadataID: seriesMetadataID,
		Language:         language,
		Name:             name,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// GetTranslation returns the translation for the specified language, or nil if not found
func (s *SeriesMetadata) GetTranslation(language string) *SeriesMetadataTranslation {
	for i := range s.Translations {
		if s.Translations[i].Language == language {
			return &s.Translations[i]
		}
	}
	return nil
}

// GetName returns the name for the specified language, falling back to original name
func (s *SeriesMetadata) GetName(language string) string {
	if t := s.GetTranslation(language); t != nil && t.Name != "" {
		return t.Name
	}
	return s.OriginalName
}

// GetOverview returns the overview for the specified language, or empty string if not found
func (s *SeriesMetadata) GetOverview(language string) string {
	if t := s.GetTranslation(language); t != nil {
		return t.Overview
	}
	return ""
}

// AddTranslation adds or updates a translation for the series
func (s *SeriesMetadata) AddTranslation(translation SeriesMetadataTranslation) {
	for i := range s.Translations {
		if s.Translations[i].Language == translation.Language {
			s.Translations[i] = translation
			return
		}
	}
	s.Translations = append(s.Translations, translation)
}

// GetSeason returns the season by number, or nil if not found
func (s *SeriesMetadata) GetSeason(seasonNumber int) *SeasonMetadata {
	for i := range s.Seasons {
		if s.Seasons[i].SeasonNumber == seasonNumber {
			return &s.Seasons[i]
		}
	}
	return nil
}
