package domain

import "time"

// MovieMetadata stores TMDB movie information (language-independent fields)
type MovieMetadata struct {
	ID            int64     `json:"id"`
	TMDBID        int       `json:"tmdb_id"`
	IMDbID        string    `json:"imdb_id,omitempty"`
	OriginalTitle string    `json:"original_title"`
	ReleaseDate   string    `json:"release_date,omitempty"`
	Runtime       int       `json:"runtime"`
	PosterPath    string    `json:"poster_path,omitempty"`
	BackdropPath  string    `json:"backdrop_path,omitempty"`
	VoteAverage   float64   `json:"vote_average"`
	VoteCount     int       `json:"vote_count"`
	Popularity    float64   `json:"popularity"`
	Status        string    `json:"status,omitempty"`
	OriginalLang  string    `json:"original_language,omitempty"`
	Genres        []string  `json:"genres,omitempty"`
	CollectionID  *int      `json:"collection_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Loaded translations (not persisted directly, joined on query)
	Translations []MovieMetadataTranslation `json:"translations,omitempty"`
}

// MovieMetadataTranslation stores translatable movie content
type MovieMetadataTranslation struct {
	ID              int64     `json:"id"`
	MovieMetadataID int64     `json:"movie_metadata_id"`
	Language        string    `json:"language"`
	Title           string    `json:"title"`
	Tagline         string    `json:"tagline,omitempty"`
	Overview        string    `json:"overview,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NewMovieMetadata creates a new MovieMetadata with default values
func NewMovieMetadata(tmdbID int, originalTitle string) *MovieMetadata {
	now := time.Now()
	return &MovieMetadata{
		TMDBID:        tmdbID,
		OriginalTitle: originalTitle,
		Genres:        []string{},
		Translations:  []MovieMetadataTranslation{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// NewMovieMetadataTranslation creates a new translation entry
func NewMovieMetadataTranslation(movieMetadataID int64, language, title string) *MovieMetadataTranslation {
	now := time.Now()
	return &MovieMetadataTranslation{
		MovieMetadataID: movieMetadataID,
		Language:        language,
		Title:           title,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// GetTranslation returns the translation for the specified language, or nil if not found
func (m *MovieMetadata) GetTranslation(language string) *MovieMetadataTranslation {
	for i := range m.Translations {
		if m.Translations[i].Language == language {
			return &m.Translations[i]
		}
	}
	return nil
}

// GetTitle returns the title for the specified language, falling back to original title
func (m *MovieMetadata) GetTitle(language string) string {
	if t := m.GetTranslation(language); t != nil && t.Title != "" {
		return t.Title
	}
	return m.OriginalTitle
}

// GetOverview returns the overview for the specified language, or empty string if not found
func (m *MovieMetadata) GetOverview(language string) string {
	if t := m.GetTranslation(language); t != nil {
		return t.Overview
	}
	return ""
}

// GetTagline returns the tagline for the specified language, or empty string if not found
func (m *MovieMetadata) GetTagline(language string) string {
	if t := m.GetTranslation(language); t != nil {
		return t.Tagline
	}
	return ""
}

// AddTranslation adds or updates a translation for the movie
func (m *MovieMetadata) AddTranslation(translation MovieMetadataTranslation) {
	for i := range m.Translations {
		if m.Translations[i].Language == translation.Language {
			m.Translations[i] = translation
			return
		}
	}
	m.Translations = append(m.Translations, translation)
}
