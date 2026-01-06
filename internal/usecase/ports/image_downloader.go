package ports

import "context"

// ImageType constants for organizing downloaded images
const (
	ImageTypeMoviePoster    = "movie_poster"
	ImageTypeMovieBackdrop  = "movie_backdrop"
	ImageTypeSeriesPoster   = "series_poster"
	ImageTypeSeriesBackdrop = "series_backdrop"
	ImageTypeSeasonPoster   = "season_poster"
	ImageTypeEpisodeStill   = "episode_still"
	ImageTypeProfile        = "profile"
)

// ImageDownloader defines the interface for downloading and caching images
type ImageDownloader interface {
	// DownloadImage downloads an image from TMDB and saves it locally
	// tmdbPath is the path returned by TMDB API (e.g., "/1E5baAaEse26fej7uHcjOgEE2t2.jpg")
	// imageType is one of the ImageType constants
	// id is the TMDB ID of the entity (movie, series, etc.)
	// Returns the local file path where the image was saved
	DownloadImage(ctx context.Context, tmdbPath string, imageType string, id int) (string, error)

	// DownloadImageWithLanguage downloads a language-specific image from TMDB
	// The language suffix is appended to the filename (e.g., 123_de.jpg)
	DownloadImageWithLanguage(ctx context.Context, tmdbPath string, imageType string, id int, language string) (string, error)

	// DownloadSeasonImage downloads a season poster
	// seriesID is the TMDB series ID, seasonNumber is the season number
	DownloadSeasonImage(ctx context.Context, tmdbPath string, seriesID int, seasonNumber int) (string, error)

	// DownloadSeasonImageWithLanguage downloads a language-specific season poster
	DownloadSeasonImageWithLanguage(ctx context.Context, tmdbPath string, seriesID int, seasonNumber int, language string) (string, error)

	// DownloadEpisodeImage downloads an episode still image
	// seriesID is the TMDB series ID, seasonNumber and episodeNumber identify the episode
	DownloadEpisodeImage(ctx context.Context, tmdbPath string, seriesID int, seasonNumber int, episodeNumber int) (string, error)

	// DownloadEpisodeImageWithLanguage downloads a language-specific episode still image
	DownloadEpisodeImageWithLanguage(ctx context.Context, tmdbPath string, seriesID int, seasonNumber int, episodeNumber int, language string) (string, error)

	// GetLocalPath returns the local path where an image would be stored
	// This does not download the image, just computes the path
	GetLocalPath(imageType string, id int) string

	// GetLocalPathWithLanguage returns the local path with language suffix
	GetLocalPathWithLanguage(imageType string, id int, language string) string

	// ImageExists checks if an image has already been downloaded
	ImageExists(imageType string, id int) bool

	// ImageExistsWithLanguage checks if a language-specific image has already been downloaded
	ImageExistsWithLanguage(imageType string, id int, language string) bool
}
