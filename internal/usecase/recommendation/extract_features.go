package recommendation

import (
	"regexp"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// Feature weight constants for TF-IDF
const (
	GenreWeight    = 3 // Genre tokens appear 3 times
	DirectorWeight = 2 // Director tokens appear 2 times
	ActorWeight    = 1 // Actor tokens appear once
	DecadeWeight   = 1 // Decade tokens appear once
)

// nonAlphanumeric matches all non-alphanumeric characters
var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// ExtractMovieFeatures converts movie data into a weighted feature string for TF-IDF
// The resulting string contains repeated tokens based on their importance:
// - Genres x3: "genre_action genre_action genre_action"
// - Directors x2: "director_christophernolan director_christophernolan"
// - Actors x1: "actor_leonardodicaprio"
// - Decade x1: "decade_2010s"
func ExtractMovieFeatures(data ports.MovieFeatureData) domain.ContentFeatures {
	var tokens []string

	// Add genre tokens (weight: 3)
	for _, genre := range data.Genres {
		token := "genre_" + normalizeToken(genre)
		for i := 0; i < GenreWeight; i++ {
			tokens = append(tokens, token)
		}
	}

	// Add director tokens (weight: 2)
	for _, director := range data.Directors {
		token := "director_" + normalizePersonName(director)
		for i := 0; i < DirectorWeight; i++ {
			tokens = append(tokens, token)
		}
	}

	// Add actor tokens (weight: 1) - top 3 only
	for _, actor := range data.TopCast {
		token := "actor_" + normalizePersonName(actor)
		for i := 0; i < ActorWeight; i++ {
			tokens = append(tokens, token)
		}
	}

	// Add decade token (weight: 1)
	decade := extractDecade(data.ReleaseDate)
	if decade != "" {
		token := "decade_" + decade
		for i := 0; i < DecadeWeight; i++ {
			tokens = append(tokens, token)
		}
	}

	return domain.ContentFeatures{
		ID:          data.ID,
		Type:        "movie",
		FeatureText: strings.Join(tokens, " "),
		Metadata: domain.ContentMetadata{
			Title:        data.OriginalTitle,
			Year:         extractYear(data.ReleaseDate),
			Popularity:   data.Popularity,
			VoteAverage:  data.VoteAverage,
			PosterPath:   data.PosterPath,
			BackdropPath: data.BackdropPath,
			MediaID:      data.MediaID,
		},
	}
}

// ExtractSeriesFeatures converts series data into a weighted feature string for TF-IDF
// Uses the same weighting scheme as movies, with creators instead of directors
func ExtractSeriesFeatures(data ports.SeriesFeatureData) domain.ContentFeatures {
	var tokens []string

	// Add genre tokens (weight: 3)
	for _, genre := range data.Genres {
		token := "genre_" + normalizeToken(genre)
		for i := 0; i < GenreWeight; i++ {
			tokens = append(tokens, token)
		}
	}

	// Add creator tokens (weight: 2) - equivalent to directors for series
	for _, creator := range data.Creators {
		token := "creator_" + normalizePersonName(creator)
		for i := 0; i < DirectorWeight; i++ {
			tokens = append(tokens, token)
		}
	}

	// Add actor tokens (weight: 1) - top 3 only
	for _, actor := range data.TopCast {
		token := "actor_" + normalizePersonName(actor)
		for i := 0; i < ActorWeight; i++ {
			tokens = append(tokens, token)
		}
	}

	// Add decade token (weight: 1)
	decade := extractDecade(data.FirstAirDate)
	if decade != "" {
		token := "decade_" + decade
		for i := 0; i < DecadeWeight; i++ {
			tokens = append(tokens, token)
		}
	}

	return domain.ContentFeatures{
		ID:          data.ID,
		Type:        "series",
		FeatureText: strings.Join(tokens, " "),
		Metadata: domain.ContentMetadata{
			Title:        data.OriginalName,
			Year:         extractYear(data.FirstAirDate),
			Popularity:   data.Popularity,
			VoteAverage:  data.VoteAverage,
			PosterPath:   data.PosterPath,
			BackdropPath: data.BackdropPath,
		},
	}
}

// normalizeToken converts a string to a lowercase alphanumeric token
// e.g., "Science Fiction" -> "sciencefiction"
func normalizeToken(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "")
	return s
}

// normalizePersonName converts a person's name to a searchable token
// e.g., "Robert Downey Jr." -> "robertdowneyjr"
func normalizePersonName(name string) string {
	name = strings.ToLower(name)
	name = nonAlphanumeric.ReplaceAllString(name, "")
	return name
}

// extractDecade extracts the decade from a date string in format "YYYY-MM-DD" or "YYYY"
// e.g., "2012-05-04" -> "2010s", "1999-12-31" -> "1990s"
func extractDecade(dateStr string) string {
	if len(dateStr) < 4 {
		return ""
	}

	yearStr := dateStr[:4]
	if len(yearStr) != 4 {
		return ""
	}

	// Convert year to decade
	// e.g., "2012" -> "2010s"
	decade := yearStr[:3] + "0s"
	return decade
}

// extractYear extracts the year from a date string in format "YYYY-MM-DD" or "YYYY"
// e.g., "2012-05-04" -> "2012"
func extractYear(dateStr string) string {
	if len(dateStr) >= 4 {
		return dateStr[:4]
	}
	return ""
}
