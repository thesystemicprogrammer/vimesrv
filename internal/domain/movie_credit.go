package domain

import "time"

// CreditType distinguishes between cast and crew members
type CreditType string

const (
	CreditTypeCast CreditType = "cast"
	CreditTypeCrew CreditType = "crew"
)

// MovieCredit represents a cast or crew member associated with a movie
type MovieCredit struct {
	ID              int64      `json:"id"`
	MovieMetadataID int64      `json:"movie_metadata_id"`
	CreditType      CreditType `json:"credit_type"`
	TMDBPersonID    int        `json:"tmdb_person_id"`
	Name            string     `json:"name"`
	Character       string     `json:"character,omitempty"`  // For cast
	Job             string     `json:"job,omitempty"`        // For crew (e.g., "Director", "Writer")
	Department      string     `json:"department,omitempty"` // For crew (e.g., "Directing", "Writing")
	ProfilePath     string     `json:"profile_path,omitempty"`
	DisplayOrder    int        `json:"display_order"` // Cast: order from TMDB, Crew: our priority order
	CreatedAt       time.Time  `json:"created_at"`
}

// CrewJobs defines which crew roles we store and their display priority
var CrewJobs = map[string]int{
	"Director":                1,
	"Writer":                  2,
	"Screenplay":              3,
	"Original Music Composer": 4,
	"Producer":                5,
	"Executive Producer":      6,
	"Director of Photography": 7,
}

// MaxCrewPerJob defines how many crew members to store per job type
var MaxCrewPerJob = map[string]int{
	"Director":                10, // Store all directors
	"Writer":                  3,
	"Screenplay":              3,
	"Original Music Composer": 2,
	"Producer":                3,
	"Executive Producer":      3,
	"Director of Photography": 2,
}

// NewMovieCredit creates a new MovieCredit with the current timestamp
func NewMovieCredit(movieMetadataID int64, creditType CreditType, tmdbPersonID int, name string) *MovieCredit {
	return &MovieCredit{
		MovieMetadataID: movieMetadataID,
		CreditType:      creditType,
		TMDBPersonID:    tmdbPersonID,
		Name:            name,
		CreatedAt:       time.Now(),
	}
}

// NewCastCredit creates a new cast credit
func NewCastCredit(movieMetadataID int64, tmdbPersonID int, name, character, profilePath string, order int) *MovieCredit {
	return &MovieCredit{
		MovieMetadataID: movieMetadataID,
		CreditType:      CreditTypeCast,
		TMDBPersonID:    tmdbPersonID,
		Name:            name,
		Character:       character,
		ProfilePath:     profilePath,
		DisplayOrder:    order,
		CreatedAt:       time.Now(),
	}
}

// NewCrewCredit creates a new crew credit
func NewCrewCredit(movieMetadataID int64, tmdbPersonID int, name, job, department, profilePath string) *MovieCredit {
	priority, ok := CrewJobs[job]
	if !ok {
		priority = 100 // Unknown jobs get low priority
	}
	return &MovieCredit{
		MovieMetadataID: movieMetadataID,
		CreditType:      CreditTypeCrew,
		TMDBPersonID:    tmdbPersonID,
		Name:            name,
		Job:             job,
		Department:      department,
		ProfilePath:     profilePath,
		DisplayOrder:    priority,
		CreatedAt:       time.Now(),
	}
}

// IsCast returns true if this is a cast credit
func (c *MovieCredit) IsCast() bool {
	return c.CreditType == CreditTypeCast
}

// IsCrew returns true if this is a crew credit
func (c *MovieCredit) IsCrew() bool {
	return c.CreditType == CreditTypeCrew
}

// IsDirector returns true if this crew member is a director
func (c *MovieCredit) IsDirector() bool {
	return c.CreditType == CreditTypeCrew && c.Job == "Director"
}
