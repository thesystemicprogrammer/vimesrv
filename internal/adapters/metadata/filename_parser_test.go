package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegexFilenameParser_Parse(t *testing.T) {
	parser := NewRegexFilenameParser()

	tests := []struct {
		name           string
		filename       string
		expectedTitle  string
		expectedYear   int
		expectedSeason int
		expectedEp     int
		expectedSeries bool
		expectedQual   string
		expectedSrc    string
		expectedEd     string
	}{
		// Movie patterns
		{
			name:          "Simple movie with year",
			filename:      "The Matrix (1999).mkv",
			expectedTitle: "The Matrix",
			expectedYear:  1999,
		},
		{
			name:          "Movie with dots and year",
			filename:      "The.Matrix.1999.BluRay.1080p.x264.mkv",
			expectedTitle: "The Matrix",
			expectedYear:  1999,
			expectedQual:  "1080p",
			expectedSrc:   "BluRay",
		},
		{
			name:          "Movie with underscores",
			filename:      "The_Matrix_1999_1080p_WEB-DL.mkv",
			expectedTitle: "The Matrix",
			expectedYear:  1999,
			expectedQual:  "1080p",
			expectedSrc:   "WEB-DL",
		},
		{
			name:          "Movie with Director's Cut",
			filename:      "Blade.Runner.1982.Directors.Cut.BluRay.1080p.mkv",
			expectedTitle: "Blade Runner",
			expectedYear:  1982,
			expectedQual:  "1080p",
			expectedSrc:   "BluRay",
			expectedEd:    "Director's Cut",
		},
		{
			name:          "Movie with Extended Edition",
			filename:      "The.Lord.of.the.Rings.2001.Extended.Edition.2160p.UHD.mkv",
			expectedTitle: "The Lord of the Rings",
			expectedYear:  2001,
			expectedQual:  "2160p",
			expectedEd:    "Extended Edition",
		},
		{
			name:          "Movie with release group",
			filename:      "Inception.2010.1080p.BluRay.x264-SPARKS.mkv",
			expectedTitle: "Inception",
			expectedYear:  2010,
			expectedQual:  "1080p",
			expectedSrc:   "BluRay",
		},
		{
			name:          "Movie with 4K indicator",
			filename:      "Dune.2021.4K.WEB-DL.mkv",
			expectedTitle: "Dune",
			expectedYear:  2021,
			expectedQual:  "2160p",
			expectedSrc:   "WEB-DL",
		},
		{
			name:          "Movie with brackets for year",
			filename:      "Pulp Fiction [1994] 720p.mkv",
			expectedTitle: "Pulp Fiction",
			expectedYear:  1994,
			expectedQual:  "720p",
		},

		// Series patterns - S01E02 format
		{
			name:           "Series with S01E02",
			filename:       "Breaking.Bad.S01E02.1080p.BluRay.mkv",
			expectedTitle:  "Breaking Bad",
			expectedSeason: 1,
			expectedEp:     2,
			expectedSeries: true,
			expectedQual:   "1080p",
			expectedSrc:    "BluRay",
		},
		{
			name:           "Series with s01e02 lowercase",
			filename:       "game.of.thrones.s08e06.1080p.web-dl.mkv",
			expectedTitle:  "game of thrones",
			expectedSeason: 8,
			expectedEp:     6,
			expectedSeries: true,
			expectedQual:   "1080p",
			expectedSrc:    "WEB-DL",
		},
		{
			name:           "Series with S1E2 short format",
			filename:       "The.Office.S1E2.720p.mkv",
			expectedTitle:  "The Office",
			expectedSeason: 1,
			expectedEp:     2,
			expectedSeries: true,
			expectedQual:   "720p",
		},

		// Series patterns - 1x02 format
		{
			name:           "Series with 1x02",
			filename:       "Friends.1x02.The.One.With.Stuff.mkv",
			expectedTitle:  "Friends",
			expectedSeason: 1,
			expectedEp:     2,
			expectedSeries: true,
		},
		{
			name:           "Series with 01x02",
			filename:       "The.Simpsons.01x02.720p.mkv",
			expectedTitle:  "The Simpsons",
			expectedSeason: 1,
			expectedEp:     2,
			expectedSeries: true,
			expectedQual:   "720p",
		},

		// Series patterns - Season X Episode Y format
		{
			name:           "Series with Season Episode format",
			filename:       "Stranger Things Season 2 Episode 5 1080p.mkv",
			expectedTitle:  "Stranger Things",
			expectedSeason: 2,
			expectedEp:     5,
			expectedSeries: true,
			expectedQual:   "1080p",
		},
		{
			name:           "Series with dots in Season Episode",
			filename:       "The.Mandalorian.Season.1.Episode.3.2160p.mkv",
			expectedTitle:  "The Mandalorian",
			expectedSeason: 1,
			expectedEp:     3,
			expectedSeries: true,
			expectedQual:   "2160p",
		},

		// Series patterns - Episode only
		{
			name:           "Series with Episode only",
			filename:       "Some.Show.E05.720p.mkv",
			expectedTitle:  "Some Show",
			expectedSeason: 1, // Assumes season 1
			expectedEp:     5,
			expectedSeries: true,
			expectedQual:   "720p",
		},

		// Edge cases
		{
			name:          "Movie with numbers in title",
			filename:      "2001.A.Space.Odyssey.1968.1080p.BluRay.mkv",
			expectedTitle: "2001 A Space Odyssey",
			expectedYear:  1968,
			expectedQual:  "1080p",
			expectedSrc:   "BluRay",
		},
		{
			name:          "Movie with ampersand",
			filename:      "Fast.&.Furious.2009.720p.mkv",
			expectedTitle: "Fast & Furious",
			expectedYear:  2009,
			expectedQual:  "720p",
		},
		{
			name:          "Movie no year",
			filename:      "Some.Random.Movie.1080p.BluRay.mkv",
			expectedTitle: "Some Random Movie",
			expectedYear:  0,
			expectedQual:  "1080p",
			expectedSrc:   "BluRay",
		},
		{
			name:          "Remastered edition",
			filename:      "E.T.1982.Remastered.1080p.BluRay.mkv",
			expectedTitle: "E T",
			expectedYear:  1982,
			expectedQual:  "1080p",
			expectedSrc:   "BluRay",
			expectedEd:    "Remastered",
		},
		{
			name:          "IMAX edition",
			filename:      "Oppenheimer.2023.IMAX.2160p.WEB-DL.mkv",
			expectedTitle: "Oppenheimer",
			expectedYear:  2023,
			expectedQual:  "2160p",
			expectedSrc:   "WEB-DL",
			expectedEd:    "IMAX",
		},
		{
			name:          "Movie with year at end of filename (no trailing separator)",
			filename:      "avatar_fire_and_ash_2025.mp4",
			expectedTitle: "avatar fire and ash",
			expectedYear:  2025,
		},
		{
			name:          "Movie with year at end using dots",
			filename:      "Some.Movie.Title.2024.mp4",
			expectedTitle: "Some Movie Title",
			expectedYear:  2024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.filename)

			assert.Equal(t, tt.expectedTitle, result.Title, "Title mismatch")
			assert.Equal(t, tt.expectedYear, result.Year, "Year mismatch")
			assert.Equal(t, tt.expectedSeason, result.SeasonNumber, "Season mismatch")
			assert.Equal(t, tt.expectedEp, result.EpisodeNumber, "Episode mismatch")
			assert.Equal(t, tt.expectedSeries, result.IsSeries, "IsSeries mismatch")
			assert.Equal(t, tt.expectedQual, result.Quality, "Quality mismatch")
			assert.Equal(t, tt.expectedSrc, result.Source, "Source mismatch")
			assert.Equal(t, tt.expectedEd, result.Edition, "Edition mismatch")
		})
	}
}

func TestRegexFilenameParser_CleanTitle(t *testing.T) {
	parser := NewRegexFilenameParser()

	tests := []struct {
		name          string
		filename      string
		expectedClean string
	}{
		{
			name:          "Simple title",
			filename:      "The Matrix (1999).mkv",
			expectedClean: "the matrix",
		},
		{
			name:          "Title with special chars",
			filename:      "Spider-Man: Homecoming (2017).mkv",
			expectedClean: "spiderman homecoming",
		},
		{
			name:          "Title with ampersand",
			filename:      "Fast & Furious (2009).mkv",
			expectedClean: "fast and furious",
		},
		{
			name:          "Title with dots",
			filename:      "The.Lord.of.the.Rings.2001.1080p.mkv",
			expectedClean: "the lord of the rings",
		},
		{
			name:          "Title with numbers",
			filename:      "2 Fast 2 Furious (2003).mkv",
			expectedClean: "2 fast 2 furious",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.filename)
			assert.Equal(t, tt.expectedClean, result.CleanTitle)
		})
	}
}

func TestRegexFilenameParser_SourceNormalization(t *testing.T) {
	parser := NewRegexFilenameParser()

	tests := []struct {
		filename       string
		expectedSource string
	}{
		{"Movie.BluRay.mkv", "BluRay"},
		{"Movie.Blu-Ray.mkv", "BluRay"},
		{"Movie.BDRip.mkv", "BluRay"},
		{"Movie.BRRip.mkv", "BluRay"},
		{"Movie.WEB-DL.mkv", "WEB-DL"},
		{"Movie.WEBDL.mkv", "WEB-DL"},
		{"Movie.WEBRip.mkv", "WEBRip"},
		{"Movie.WEB.mkv", "WEBRip"},
		{"Movie.DVDRip.mkv", "DVD"},
		{"Movie.DVD.mkv", "DVD"},
		{"Movie.HDTV.mkv", "HDTV"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := parser.Parse(tt.filename)
			assert.Equal(t, tt.expectedSource, result.Source)
		})
	}
}

func TestRegexFilenameParser_QualityNormalization(t *testing.T) {
	parser := NewRegexFilenameParser()

	tests := []struct {
		filename        string
		expectedQuality string
	}{
		{"Movie.2160p.mkv", "2160p"},
		{"Movie.4K.mkv", "2160p"},
		{"Movie.UHD.mkv", "2160p"},
		{"Movie.1080p.mkv", "1080p"},
		{"Movie.720p.mkv", "720p"},
		{"Movie.480p.mkv", "480p"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := parser.Parse(tt.filename)
			assert.Equal(t, tt.expectedQuality, result.Quality)
		})
	}
}
