package config

import (
	"fmt"
	"strings"
)

// Config holds all application configuration
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Media       MediaConfig       `mapstructure:"media"`
	Transcoding TranscodingConfig `mapstructure:"transcoding"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Logging     LoggingConfig     `mapstructure:"logging"`
	TMDB        TMDBConfig        `mapstructure:"tmdb"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// MediaConfig holds media library configuration
type MediaConfig struct {
	LibraryPath      string   `mapstructure:"library_path"`
	TrashPath        string   `mapstructure:"trash_path"`
	SupportedFormats []string `mapstructure:"supported_formats"`
}

// TranscodingConfig holds transcoding configuration
type TranscodingConfig struct {
	OutputPath      string           `mapstructure:"output_path"`
	SegmentDuration int              `mapstructure:"segment_duration"`
	QualityProfiles []QualityProfile `mapstructure:"quality_profiles"`
}

// QualityProfile represents a transcoding quality profile
type QualityProfile struct {
	Name    string `mapstructure:"name"`
	Enabled bool   `mapstructure:"enabled"` // Whether to create this quality
	// TODO: Check if audio bitrate needs to be set per resolution
	Resolution   string `mapstructure:"resolution"`
	AudioBitrate string `mapstructure:"audio_bitrate"`
	CRF          int    `mapstructure:"crf"`         // Constant Rate Factor (18-26)
	MaxBitrate   string `mapstructure:"max_bitrate"` // Bitrate cap for CRF mode
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
}

// TMDBConfig holds TMDB API configuration
type TMDBConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	APIKey            string `mapstructure:"api_key"`
	Language          string `mapstructure:"language"`
	AutoSearch        bool   `mapstructure:"auto_search"`
	AutoLinkThreshold int    `mapstructure:"auto_link_threshold"`
	MaxCandidates     int    `mapstructure:"max_candidates"`
	ImageCachePath    string `mapstructure:"image_cache_path"`
	DownloadImages    bool   `mapstructure:"download_images"`
	PosterSize        string `mapstructure:"poster_size"`
	BackdropSize      string `mapstructure:"backdrop_size"`
	RequestsPer10s    int    `mapstructure:"requests_per_10s"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	// Validate media config
	if err := c.Media.Validate(); err != nil {
		return fmt.Errorf("media config: %w", err)
	}

	// Validate transcoding config
	if err := c.Transcoding.Validate(); err != nil {
		return fmt.Errorf("transcoding config: %w", err)
	}

	// Validate database config
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}

	// Validate logging config
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config: %w", err)
	}

	// Validate TMDB config (only if enabled)
	if c.TMDB.Enabled {
		if err := c.TMDB.Validate(); err != nil {
			return fmt.Errorf("tmdb config: %w", err)
		}
	}

	return nil
}

// Validate validates server configuration
func (s *ServerConfig) Validate() error {
	if s.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", s.Port)
	}

	return nil
}

// Validate validates media configuration
func (m *MediaConfig) Validate() error {
	if m.LibraryPath == "" {
		return fmt.Errorf("library_path cannot be empty")
	}

	if m.TrashPath == "" {
		return fmt.Errorf("trash_path cannot be empty")
	}

	if len(m.SupportedFormats) == 0 {
		return fmt.Errorf("supported_formats cannot be empty")
	}

	// Validate that all formats start with a dot
	for _, format := range m.SupportedFormats {
		if !strings.HasPrefix(format, ".") {
			return fmt.Errorf("format must start with a dot, got: %s", format)
		}
	}

	return nil
}

// Validate validates transcoding configuration
func (t *TranscodingConfig) Validate() error {
	if t.OutputPath == "" {
		return fmt.Errorf("output_path cannot be empty")
	}

	if t.SegmentDuration < 1 {
		return fmt.Errorf("segment_duration must be at least 1 second, got %d", t.SegmentDuration)
	}

	if len(t.QualityProfiles) == 0 {
		return fmt.Errorf("qualities cannot be empty, at least one quality must be defined")
	}

	// Validate each quality
	for i, quality := range t.QualityProfiles {
		if err := quality.Validate(); err != nil {
			return fmt.Errorf("quality[%d]: %w", i, err)
		}
	}

	// Check at least one quality is enabled
	hasEnabled := false
	for _, q := range t.QualityProfiles {
		if q.Enabled {
			hasEnabled = true
			break
		}
	}
	if !hasEnabled {
		return fmt.Errorf("at least one quality must be enabled")
	}

	return nil
}

// Validate validates quality configuration
func (q *QualityProfile) Validate() error {
	if q.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if q.Resolution == "" {
		return fmt.Errorf("resolution cannot be empty")
	}

	if q.AudioBitrate == "" {
		return fmt.Errorf("audio_bitrate cannot be empty")
	}

	// CRF validation (must be in range 18-26)
	if q.CRF < 18 || q.CRF > 26 {
		return fmt.Errorf("crf must be between 18 and 26 for quality %s, got %d", q.Name, q.CRF)
	}

	// MaxBitrate required for CRF mode
	if q.MaxBitrate == "" {
		return fmt.Errorf("max_bitrate cannot be empty for quality %s (required for CRF mode)", q.Name)
	}

	return nil
}

// Validate validates database configuration
func (d *DatabaseConfig) Validate() error {
	if d.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	return nil
}

// Validate validates logging configuration
func (l *LoggingConfig) Validate() error {
	if l.Level == "" {
		return fmt.Errorf("level cannot be empty")
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[l.Level] {
		return fmt.Errorf("level must be one of: debug, info, warn, error; got: %s", l.Level)
	}

	if l.Format == "" {
		return fmt.Errorf("format cannot be empty")
	}

	validFormats := map[string]bool{
		"console": true,
		"json":    true,
	}

	if !validFormats[l.Format] {
		return fmt.Errorf("format must be one of: console, json; got: %s", l.Format)
	}

	return nil
}

// Validate validates TMDB configuration
func (t *TMDBConfig) Validate() error {
	if t.APIKey == "" {
		return fmt.Errorf("api_key cannot be empty when TMDB is enabled")
	}

	if t.Language == "" {
		return fmt.Errorf("language cannot be empty")
	}

	if t.AutoLinkThreshold < 0 || t.AutoLinkThreshold > 100 {
		return fmt.Errorf("auto_link_threshold must be between 0 and 100, got %d", t.AutoLinkThreshold)
	}

	if t.MaxCandidates < 1 {
		return fmt.Errorf("max_candidates must be at least 1, got %d", t.MaxCandidates)
	}

	if t.ImageCachePath == "" {
		return fmt.Errorf("image_cache_path cannot be empty")
	}

	if t.PosterSize == "" {
		return fmt.Errorf("poster_size cannot be empty")
	}

	if t.BackdropSize == "" {
		return fmt.Errorf("backdrop_size cannot be empty")
	}

	// Validate poster size
	validPosterSizes := map[string]bool{
		"w92": true, "w154": true, "w185": true, "w342": true, "w500": true, "w780": true, "original": true,
	}
	if !validPosterSizes[t.PosterSize] {
		return fmt.Errorf("invalid poster_size: %s (must be one of: w92, w154, w185, w342, w500, w780, original)", t.PosterSize)
	}

	// Validate backdrop size
	validBackdropSizes := map[string]bool{
		"w300": true, "w780": true, "w1280": true, "original": true,
	}
	if !validBackdropSizes[t.BackdropSize] {
		return fmt.Errorf("invalid backdrop_size: %s (must be one of: w300, w780, w1280, original)", t.BackdropSize)
	}

	if t.RequestsPer10s < 1 || t.RequestsPer10s > 40 {
		return fmt.Errorf("requests_per_10s must be between 1 and 40 (TMDB limit), got %d", t.RequestsPer10s)
	}

	return nil
}

// Address returns the server address in host:port format
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
