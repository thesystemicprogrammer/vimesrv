package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func Load(configPath string) (*Config, error) {
	v := viper.New()

	setDefaults(v)

	// Only read config file if a path is explicitly provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		v.SetConfigType("yaml")

		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	v.SetEnvPrefix("VIMESRV")
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

	if err := deriveSubpaths(&cfg); err != nil {
		return nil, fmt.Errorf("failed to derive subpaths: %w", err)
	}

	if err := createDirectories(&cfg); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	return &cfg, nil
}

func LoadWithDefaults() (*Config, error) {
	return Load("")
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "127.0.0.1")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.shutdown_timeout_seconds", 30)

	// Job defaults
	v.SetDefault("job.worker_count", 2)
	v.SetDefault("job.max_attempts", 3)
	v.SetDefault("job.polling_interval_in_seconds", 2)
	v.SetDefault("job.scheduler_interval_in_seconds", 10)
	v.SetDefault("job.scheduler_batch", 3)
	v.SetDefault("job.backoff_base_seconds", 2)
	v.SetDefault("job.backoff_max_seconds", 300)
	v.SetDefault("job.stuck_job_threshold_minutes", 480)
	v.SetDefault("job.stuck_job_check_interval_minutes", 5)

	// Media defaults
	v.SetDefault("media.library_path", "./library")
	v.SetDefault("media.media_path", "./media")
	v.SetDefault("media.staging_path", "./staging")
	v.SetDefault("media.trash_path", "./trash")
	v.SetDefault("media.ffprobe_timeout_seconds", 30)
	v.SetDefault("media.transcode_timeout_seconds", 7200)
	v.SetDefault("media.transcode_output_pattern", "{media_path}/{media_id}/transcoded")
	v.SetDefault("media.supported_formats", []string{
		".mp4", ".mkv", ".avi", ".mov", ".webm", ".flv", ".wmv", ".m4v",
	})

	// Transcoding defaults
	v.SetDefault("transcoding.segment_duration", 4)
	v.SetDefault("transcoding.segment_pattern", "chunk-%03d.m4s")

	// Quality profiles defaults
	v.SetDefault("transcoding.quality_profiles", []map[string]any{
		{
			"name":          "360p",
			"enabled":       true,
			"resolution":    "640x360",
			"crf":           25,
			"max_bitrate":   "900k",
			"audio_bitrate": "96k",
		},
		{
			"name":          "480p",
			"enabled":       true,
			"resolution":    "854x480",
			"crf":           24,
			"max_bitrate":   "1500k",
			"audio_bitrate": "128k",
		},
		{
			"name":          "720p",
			"enabled":       false,
			"resolution":    "1280x720",
			"crf":           23,
			"max_bitrate":   "2800k",
			"audio_bitrate": "128k",
		},
		{
			"name":          "900p",
			"enabled":       false,
			"resolution":    "1600x900",
			"crf":           22,
			"max_bitrate":   "4500k",
			"audio_bitrate": "128k",
		},
		{
			"name":          "1080p",
			"enabled":       false,
			"resolution":    "1920x1080",
			"crf":           21,
			"max_bitrate":   "5500k",
			"audio_bitrate": "192k",
		},
		{
			"name":          "1440p",
			"enabled":       false,
			"resolution":    "2560x1440",
			"crf":           20,
			"max_bitrate":   "9000k",
			"audio_bitrate": "192k",
		},
		{
			"name":          "2160p",
			"enabled":       false,
			"resolution":    "3840x2160",
			"crf":           18,
			"max_bitrate":   "16000k",
			"audio_bitrate": "256k",
		},
	})

	// Database defaults
	v.SetDefault("database.path", "./data/vimesrv.db")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")
	v.SetDefault("logging.file", "")

	// TMDB defaults
	v.SetDefault("tmdb.enabled", false)
	v.SetDefault("tmdb.api_key", "")
	v.SetDefault("tmdb.language", "en-US")
	v.SetDefault("tmdb.auto_search", true)
	v.SetDefault("tmdb.auto_link_threshold", 70)
	v.SetDefault("tmdb.max_candidates", 5)
	v.SetDefault("tmdb.image_cache_path", "./cache/tmdb")
	v.SetDefault("tmdb.download_images", true)
	v.SetDefault("tmdb.poster_size", "w500")
	v.SetDefault("tmdb.backdrop_size", "w1280")
	v.SetDefault("tmdb.requests_per_10s", 35)

	// Library defaults
	v.SetDefault("library.recently_added_count", 20)

	// Rebuild defaults
	v.SetDefault("rebuild.allow_rebuild", false)
	v.SetDefault("rebuild.tmdb_requests_per_10s", 15)
}

// bindEnvVars binds environment variables to viper keys
func bindEnvVars(v *viper.Viper) {
	// Server
	v.BindEnv("server.host", "SERVER_HOST")
	v.BindEnv("server.port", "SERVER_PORT")

	// Media
	v.BindEnv("media.library_path", "MEDIA_LIBRARY_PATH")
	v.BindEnv("media.trash_path", "MEDIA_TRASH_PATH")
	v.BindEnv("media.ffprobe_timeout_seconds", "MEDIA_FFPROBE_TIMEOUT_SECONDS")
	v.BindEnv("media.transcode_timeout_seconds", "MEDIA_TRANSCODE_TIMEOUT_SECONDS")

	// Transcoding
	v.BindEnv("transcoding.segment_duration", "TRANSCODING_SEGMENT_DURATION")

	// Database
	v.BindEnv("database.path", "DATABASE_PATH")

	// Logging
	v.BindEnv("logging.level", "LOG_LEVEL")
	v.BindEnv("logging.format", "LOG_FORMAT")
	v.BindEnv("logging.file", "LOG_FILE")

	// TMDB
	v.BindEnv("tmdb.enabled", "TMDB_ENABLED")
	v.BindEnv("tmdb.api_key", "TMDB_API_KEY")
	v.BindEnv("tmdb.language", "TMDB_LANGUAGE")
	v.BindEnv("tmdb.auto_search", "TMDB_AUTO_SEARCH")
	v.BindEnv("tmdb.auto_link_threshold", "TMDB_AUTO_LINK_THRESHOLD")
	v.BindEnv("tmdb.max_candidates", "TMDB_MAX_CANDIDATES")
	v.BindEnv("tmdb.image_cache_path", "TMDB_IMAGE_CACHE_PATH")
	v.BindEnv("tmdb.download_images", "TMDB_DOWNLOAD_IMAGES")
	v.BindEnv("tmdb.poster_size", "TMDB_POSTER_SIZE")
	v.BindEnv("tmdb.backdrop_size", "TMDB_BACKDROP_SIZE")
	v.BindEnv("tmdb.requests_per_10s", "TMDB_REQUESTS_PER_10S")

	// Library
	v.BindEnv("library.recently_added_count", "LIBRARY_RECENTLY_ADDED_COUNT")
}

func normalizePathsToAbsolute(cfg *Config) error {
	var err error

	// Normalize media library path
	if cfg.Media.LibraryPath != "" {
		cfg.Media.LibraryPath, err = filepath.Abs(cfg.Media.LibraryPath)
		if err != nil {
			return fmt.Errorf("failed to normalize media library path: %w", err)
		}
	}

	// Note: Media.TrashPath is derived in deriveSubpaths()

	// Normalize database path
	if cfg.Database.Path != "" {
		cfg.Database.Path, err = filepath.Abs(cfg.Database.Path)
		if err != nil {
			return fmt.Errorf("failed to normalize database path: %w", err)
		}
	}

	// Normalize log file path if specified
	if cfg.Logging.File != "" {
		cfg.Logging.File, err = filepath.Abs(cfg.Logging.File)
		if err != nil {
			return fmt.Errorf("failed to normalize log file path: %w", err)
		}
	}

	return nil
}

// deriveSubpaths sets paths that are derived from other config values
func deriveSubpaths(cfg *Config) error {
	// Derive media path as subdirectory of library path
	cfg.Media.MediaPath = filepath.Join(cfg.Media.LibraryPath, cfg.Media.MediaPath)

	// Derive staging path as subdirectory of library path
	cfg.Media.StagingPath = filepath.Join(cfg.Media.LibraryPath, cfg.Media.StagingPath)

	// Derive trash path as subdirectory of library path
	cfg.Media.TrashPath = filepath.Join(cfg.Media.LibraryPath, cfg.Media.TrashPath)

	return nil
}

// createDirectories creates necessary directories if they don't exist
func createDirectories(cfg *Config) error {
	// List of directories to create
	dirs := []string{
		cfg.Media.LibraryPath,
		cfg.Media.MediaPath,
		cfg.Media.StagingPath,
		cfg.Media.TrashPath,
	}

	// Also create subdirectories in trash
	trashSubdirs := []string{
		filepath.Join(cfg.Media.TrashPath, "original"),
		filepath.Join(cfg.Media.TrashPath, "transcoded"),
	}
	dirs = append(dirs, trashSubdirs...)

	// Create database directory (parent of db file)
	dbDir := filepath.Dir(cfg.Database.Path)
	if dbDir != "" && dbDir != "." {
		dirs = append(dirs, dbDir)
	}

	// Create log file directory if log file is specified
	if cfg.Logging.File != "" {
		logDir := filepath.Dir(cfg.Logging.File)
		if logDir != "" && logDir != "." {
			dirs = append(dirs, logDir)
		}
	}

	// Create all directories
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
