// Package config provides configuration loading for the transcoding worker binary.
package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all worker configuration
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Worker      WorkerSettings    `mapstructure:"worker"`
	Media       MediaConfig       `mapstructure:"media"`
	Transcoding TranscodingConfig `mapstructure:"transcoding"`
	Logging     LoggingConfig     `mapstructure:"logging"`
}

// ServerConfig contains settings for connecting to the main vimesrv server
type ServerConfig struct {
	// URL is the base URL of the main vimesrv server (e.g., "http://localhost:8080")
	URL string `mapstructure:"url"`

	// AuthToken is the shared secret for worker authentication (must match server's worker.auth_token)
	AuthToken string `mapstructure:"auth_token"`
}

// WorkerSettings contains worker-specific settings
type WorkerSettings struct {
	// ID is the unique identifier for this worker (auto-generated UUID if empty)
	ID string `mapstructure:"id"`

	// Name is a human-readable name for this worker (for logging and display)
	Name string `mapstructure:"name"`

	// PollIntervalSeconds is how often to poll for new jobs
	PollIntervalSeconds int `mapstructure:"poll_interval_seconds"`

	// HeartbeatIntervalSeconds is how often to send heartbeat when idle
	HeartbeatIntervalSeconds int `mapstructure:"heartbeat_interval_seconds"`

	// Concurrency is the maximum number of concurrent transcode jobs
	Concurrency int `mapstructure:"concurrency"`

	// ProgressIntervalSeconds is how often to report progress during transcoding
	ProgressIntervalSeconds int `mapstructure:"progress_interval_seconds"`
}

// MediaConfig contains paths to media files
type MediaConfig struct {
	// MediaPath is the path to the media library (must match server's media_path via shared storage)
	MediaPath string `mapstructure:"media_path"`
}

// TranscodingConfig contains FFmpeg-related settings
type TranscodingConfig struct {
	// FFmpegPath is the path to the ffmpeg binary
	FFmpegPath string `mapstructure:"ffmpeg_path"`

	// FFprobePath is the path to the ffprobe binary
	FFprobePath string `mapstructure:"ffprobe_path"`

	// TimeoutSeconds is the maximum time for a single transcode job
	TimeoutSeconds int `mapstructure:"timeout_seconds"`

	// SegmentDuration is the segment duration in seconds (must match server)
	SegmentDuration int `mapstructure:"segment_duration"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string `mapstructure:"level"`

	// Format is the log format (console or json)
	Format string `mapstructure:"format"`
}

// Load loads worker configuration from the specified file path
func Load(configPath string) (*Config, error) {
	v := viper.New()

	setDefaults(v)

	// Read config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		v.SetConfigType("yaml")

		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Environment variable support with VIMESRV_WORKER prefix
	v.SetEnvPrefix("VIMESRV_WORKER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	bindEnvVars(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	if err := normalizePathsToAbsolute(&cfg); err != nil {
		return nil, fmt.Errorf("failed to normalize paths: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.url", "http://localhost:8080")
	v.SetDefault("server.auth_token", "")

	// Worker defaults
	v.SetDefault("worker.id", "")
	v.SetDefault("worker.name", "worker-1")
	v.SetDefault("worker.poll_interval_seconds", 5)
	v.SetDefault("worker.heartbeat_interval_seconds", 30)
	v.SetDefault("worker.concurrency", 2)
	v.SetDefault("worker.progress_interval_seconds", 5)

	// Media defaults
	v.SetDefault("media.media_path", "/mnt/nfs/media")

	// Transcoding defaults
	v.SetDefault("transcoding.ffmpeg_path", "ffmpeg")
	v.SetDefault("transcoding.ffprobe_path", "ffprobe")
	v.SetDefault("transcoding.timeout_seconds", 7200)
	v.SetDefault("transcoding.segment_duration", 4)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")
}

func bindEnvVars(v *viper.Viper) {
	// Server
	v.BindEnv("server.url", "VIMESRV_WORKER_SERVER_URL")
	v.BindEnv("server.auth_token", "VIMESRV_WORKER_AUTH_TOKEN")

	// Worker
	v.BindEnv("worker.id", "VIMESRV_WORKER_ID")
	v.BindEnv("worker.name", "VIMESRV_WORKER_NAME")
	v.BindEnv("worker.poll_interval_seconds", "VIMESRV_WORKER_POLL_INTERVAL")
	v.BindEnv("worker.heartbeat_interval_seconds", "VIMESRV_WORKER_HEARTBEAT_INTERVAL")
	v.BindEnv("worker.concurrency", "VIMESRV_WORKER_CONCURRENCY")
	v.BindEnv("worker.progress_interval_seconds", "VIMESRV_WORKER_PROGRESS_INTERVAL")

	// Media
	v.BindEnv("media.media_path", "VIMESRV_WORKER_MEDIA_PATH")

	// Transcoding
	v.BindEnv("transcoding.ffmpeg_path", "VIMESRV_WORKER_FFMPEG_PATH")
	v.BindEnv("transcoding.ffprobe_path", "VIMESRV_WORKER_FFPROBE_PATH")
	v.BindEnv("transcoding.timeout_seconds", "VIMESRV_WORKER_TRANSCODE_TIMEOUT")
	v.BindEnv("transcoding.segment_duration", "VIMESRV_WORKER_SEGMENT_DURATION")

	// Logging
	v.BindEnv("logging.level", "VIMESRV_WORKER_LOG_LEVEL")
	v.BindEnv("logging.format", "VIMESRV_WORKER_LOG_FORMAT")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	if err := c.Worker.Validate(); err != nil {
		return fmt.Errorf("worker config: %w", err)
	}

	if err := c.Media.Validate(); err != nil {
		return fmt.Errorf("media config: %w", err)
	}

	if err := c.Transcoding.Validate(); err != nil {
		return fmt.Errorf("transcoding config: %w", err)
	}

	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config: %w", err)
	}

	return nil
}

func (s *ServerConfig) Validate() error {
	if s.URL == "" {
		return fmt.Errorf("url cannot be empty")
	}

	if !strings.HasPrefix(s.URL, "http://") && !strings.HasPrefix(s.URL, "https://") {
		return fmt.Errorf("url must start with http:// or https://, got: %s", s.URL)
	}

	if s.AuthToken == "" {
		return fmt.Errorf("auth_token cannot be empty")
	}

	if len(s.AuthToken) < 16 {
		return fmt.Errorf("auth_token must be at least 16 characters for security")
	}

	return nil
}

func (w *WorkerSettings) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if w.PollIntervalSeconds < 1 || w.PollIntervalSeconds > 300 {
		return fmt.Errorf("poll_interval_seconds must be between 1 and 300, got %d", w.PollIntervalSeconds)
	}

	if w.HeartbeatIntervalSeconds < 5 || w.HeartbeatIntervalSeconds > 300 {
		return fmt.Errorf("heartbeat_interval_seconds must be between 5 and 300, got %d", w.HeartbeatIntervalSeconds)
	}

	if w.Concurrency < 1 || w.Concurrency > 16 {
		return fmt.Errorf("concurrency must be between 1 and 16, got %d", w.Concurrency)
	}

	if w.ProgressIntervalSeconds < 1 || w.ProgressIntervalSeconds > 60 {
		return fmt.Errorf("progress_interval_seconds must be between 1 and 60, got %d", w.ProgressIntervalSeconds)
	}

	return nil
}

func (m *MediaConfig) Validate() error {
	if m.MediaPath == "" {
		return fmt.Errorf("media_path cannot be empty")
	}

	return nil
}

func (t *TranscodingConfig) Validate() error {
	if t.FFmpegPath == "" {
		return fmt.Errorf("ffmpeg_path cannot be empty")
	}

	if t.FFprobePath == "" {
		return fmt.Errorf("ffprobe_path cannot be empty")
	}

	if t.TimeoutSeconds < 60 || t.TimeoutSeconds > 36000 {
		return fmt.Errorf("timeout_seconds must be between 60 and 36000 (10 hours), got %d", t.TimeoutSeconds)
	}

	if t.SegmentDuration < 1 || t.SegmentDuration > 30 {
		return fmt.Errorf("segment_duration must be between 1 and 30, got %d", t.SegmentDuration)
	}

	return nil
}

func (l *LoggingConfig) Validate() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[l.Level] {
		return fmt.Errorf("level must be one of: debug, info, warn, error; got: %s", l.Level)
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

func normalizePathsToAbsolute(cfg *Config) error {
	var err error

	// Normalize media path to absolute
	if cfg.Media.MediaPath != "" {
		cfg.Media.MediaPath, err = filepath.Abs(cfg.Media.MediaPath)
		if err != nil {
			return fmt.Errorf("failed to normalize media path: %w", err)
		}
	}

	return nil
}
