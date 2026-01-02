package metadata

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// HTTPImageDownloader implements ports.ImageDownloader using HTTP
type HTTPImageDownloader struct {
	httpClient   *http.Client
	basePath     string
	posterSize   string
	backdropSize string
	tmdbClient   ports.TMDBClient
}

// NewHTTPImageDownloader creates a new HTTP image downloader
func NewHTTPImageDownloader(cfg config.TMDBConfig, tmdbClient ports.TMDBClient) *HTTPImageDownloader {
	return &HTTPImageDownloader{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		basePath:     cfg.ImageCachePath,
		posterSize:   cfg.PosterSize,
		backdropSize: cfg.BackdropSize,
		tmdbClient:   tmdbClient,
	}
}

// DownloadImage downloads an image from TMDB and saves it locally
func (d *HTTPImageDownloader) DownloadImage(ctx context.Context, tmdbPath string, imageType string, id int) (string, error) {
	if tmdbPath == "" {
		return "", nil
	}

	// Determine size based on image type
	size := d.getSizeForType(imageType)

	// Get the full URL
	imageURL := d.tmdbClient.GetImageURL(tmdbPath, size)
	if imageURL == "" {
		return "", nil
	}

	// Determine local path
	localPath := d.GetLocalPath(imageType, id)

	// Download and save
	if err := d.downloadToPath(ctx, imageURL, localPath); err != nil {
		return "", fmt.Errorf("download image: %w", err)
	}

	return localPath, nil
}

// DownloadSeasonImage downloads a season poster
func (d *HTTPImageDownloader) DownloadSeasonImage(ctx context.Context, tmdbPath string, seriesID int, seasonNumber int) (string, error) {
	if tmdbPath == "" {
		return "", nil
	}

	imageURL := d.tmdbClient.GetImageURL(tmdbPath, d.posterSize)
	if imageURL == "" {
		return "", nil
	}

	// Create unique local path for season
	localPath := d.getSeasonPath(seriesID, seasonNumber)

	if err := d.downloadToPath(ctx, imageURL, localPath); err != nil {
		return "", fmt.Errorf("download season image: %w", err)
	}

	return localPath, nil
}

// DownloadEpisodeImage downloads an episode still image
func (d *HTTPImageDownloader) DownloadEpisodeImage(ctx context.Context, tmdbPath string, seriesID int, seasonNumber int, episodeNumber int) (string, error) {
	if tmdbPath == "" {
		return "", nil
	}

	// Episodes use backdrop size for stills
	imageURL := d.tmdbClient.GetImageURL(tmdbPath, d.backdropSize)
	if imageURL == "" {
		return "", nil
	}

	// Create unique local path for episode
	localPath := d.getEpisodePath(seriesID, seasonNumber, episodeNumber)

	if err := d.downloadToPath(ctx, imageURL, localPath); err != nil {
		return "", fmt.Errorf("download episode image: %w", err)
	}

	return localPath, nil
}

// GetLocalPath returns the local path where an image would be stored
func (d *HTTPImageDownloader) GetLocalPath(imageType string, id int) string {
	ext := ".jpg"
	return filepath.Join(d.basePath, imageType, fmt.Sprintf("%d%s", id, ext))
}

// ImageExists checks if an image has already been downloaded
func (d *HTTPImageDownloader) ImageExists(imageType string, id int) bool {
	path := d.GetLocalPath(imageType, id)
	_, err := os.Stat(path)
	return err == nil
}

// getSizeForType returns the appropriate size for an image type
func (d *HTTPImageDownloader) getSizeForType(imageType string) string {
	switch imageType {
	case ports.ImageTypeMoviePoster, ports.ImageTypeSeriesPoster, ports.ImageTypeSeasonPoster:
		return d.posterSize
	case ports.ImageTypeMovieBackdrop, ports.ImageTypeSeriesBackdrop, ports.ImageTypeEpisodeStill:
		return d.backdropSize
	default:
		return d.posterSize
	}
}

// getSeasonPath returns the local path for a season image
func (d *HTTPImageDownloader) getSeasonPath(seriesID, seasonNumber int) string {
	ext := ".jpg"
	filename := fmt.Sprintf("%d_s%02d%s", seriesID, seasonNumber, ext)
	return filepath.Join(d.basePath, ports.ImageTypeSeasonPoster, filename)
}

// getEpisodePath returns the local path for an episode image
func (d *HTTPImageDownloader) getEpisodePath(seriesID, seasonNumber, episodeNumber int) string {
	ext := ".jpg"
	filename := fmt.Sprintf("%d_s%02de%02d%s", seriesID, seasonNumber, episodeNumber, ext)
	return filepath.Join(d.basePath, ports.ImageTypeEpisodeStill, filename)
}

// downloadToPath downloads a URL to a local file path
func (d *HTTPImageDownloader) downloadToPath(ctx context.Context, imageURL, localPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Execute request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create temporary file
	tempPath := localPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() {
		file.Close()
		// Clean up temp file on error
		if _, err := os.Stat(tempPath); err == nil {
			os.Remove(tempPath)
		}
	}()

	// Copy content
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	// Close before rename
	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, localPath); err != nil {
		return fmt.Errorf("rename file: %w", err)
	}

	return nil
}

// Ensure HTTPImageDownloader implements ImageDownloader
var _ ports.ImageDownloader = (*HTTPImageDownloader)(nil)
