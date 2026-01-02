package ports

// ParsedFilename contains structured information extracted from a video filename
type ParsedFilename struct {
	// Title is the extracted title (cleaned of metadata)
	Title string

	// Year is the release year if found (0 if not found)
	Year int

	// SeasonNumber is the season number for series (0 if not found or not a series)
	SeasonNumber int

	// EpisodeNumber is the episode number for series (0 if not found or not a series)
	EpisodeNumber int

	// IsSeries indicates whether the filename matches a TV series pattern
	IsSeries bool

	// Quality is the video quality if detected (e.g., "1080p", "720p", "4K")
	Quality string

	// Source is the video source if detected (e.g., "BluRay", "WEB-DL", "HDTV")
	Source string

	// Edition is the video edition if detected (e.g., "Director's Cut", "Extended", "Theatrical")
	Edition string

	// CleanTitle is the title suitable for TMDB search (lowercase, normalized)
	CleanTitle string
}

// FilenameParser defines the interface for parsing video filenames
type FilenameParser interface {
	// Parse extracts structured information from a video filename
	Parse(filename string) *ParsedFilename
}
