package config

import (
	"fmt"
	"strings"
)

// Config holds all application configuration
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Auth        AuthConfig        `mapstructure:"auth"`
	Job         JobConfig         `mapstructure:"job"`
	Media       MediaConfig       `mapstructure:"media"`
	Transcoding TranscodingConfig `mapstructure:"transcoding"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Logging     LoggingConfig     `mapstructure:"logging"`
	TMDB        TMDBConfig        `mapstructure:"tmdb"`
	Library     LibraryConfig     `mapstructure:"library"`
	WebSocket   WebSocketConfig   `mapstructure:"websocket"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	Username         string `mapstructure:"username"`
	PasswordHash     string `mapstructure:"password_hash"`      // bcrypt hash
	JWTSecret        string `mapstructure:"jwt_secret"`         // Set via AUTH_JWT_SECRET env var
	TokenExpiryHours int    `mapstructure:"token_expiry_hours"` // JWT token expiry in hours
	StreamTokenMins  int    `mapstructure:"stream_token_mins"`  // Stream token expiry in minutes
}

type JobConfig struct {
	WorkerCount                  int `mapstructure:"worker_count"`
	PollingIntervalInSeconds     int `mapstructure:"polling_interval_in_seconds"`
	MaxAttempts                  int `mapstructure:"max_attempts"`
	SchedulerIntervalInSeconds   int `mapstructure:"scheduler_interval_in_seconds"`
	SchedulerBatch               int `mapstructure:"scheduler_batch"`
	BackoffBaseSeconds           int `mapstructure:"backoff_base_seconds"`
	BackoffMaxSeconds            int `mapstructure:"backoff_max_seconds"`
	StuckJobThresholdMinutes     int `mapstructure:"stuck_job_threshold_minutes"`
	StuckJobCheckIntervalMinutes int `mapstructure:"stuck_job_check_interval_minutes"`
}

type ServerConfig struct {
	Host                   string `mapstructure:"host"`
	Port                   int    `mapstructure:"port"`
	ShutdownTimeoutSeconds int    `mapstructure:"shutdown_timeout_seconds"`
}

type MediaConfig struct {
	LibraryPath             string            `mapstructure:"library_path"`
	MediaPath               string            `mapstructure:"media_path"`
	StagingPath             string            `mapstructure:"staging_path"`
	TrashPath               string            `mapstructure:"trash_path"`
	TranscodeOutputPattern  string            `mapstructure:"transcode_output_pattern"`
	SupportedFormats        []string          `mapstructure:"supported_formats"`
	FFProbeTimeoutSeconds   int               `mapstructure:"ffprobe_timeout_seconds"`
	TranscodeTimeoutSeconds int               `mapstructure:"transcode_timeout_seconds"`
	LibraryScan             LibraryScanConfig `mapstructure:"library_scan"`
}

type LibraryScanConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CronSpec string `mapstructure:"cron_spec"`
	Priority int    `mapstructure:"priority"`
}

type TranscodingConfig struct {
	SegmentDuration int              `mapstructure:"segment_duration"`
	QualityProfiles []QualityProfile `mapstructure:"quality_profiles"`
}

type QualityProfile struct {
	Name    string `mapstructure:"name"`
	Enabled bool   `mapstructure:"enabled"` // Whether to create this quality
	// TODO: Check if audio bitrate needs to be set per resolution
	Resolution   string `mapstructure:"resolution"`
	AudioBitrate string `mapstructure:"audio_bitrate"`
	CRF          int    `mapstructure:"crf"`         // Constant Rate Factor (18-26)
	MaxBitrate   string `mapstructure:"max_bitrate"` // Bitrate cap for CRF mode
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
}

type TMDBConfig struct {
	Enabled             bool   `mapstructure:"enabled"`
	APIKey              string `mapstructure:"api_key"`
	Language            string `mapstructure:"language"`
	AutoSearch          bool   `mapstructure:"auto_search"`
	AutoLinkThreshold   int    `mapstructure:"auto_link_threshold"`
	MaxCandidates       int    `mapstructure:"max_candidates"`
	ImageCachePath      string `mapstructure:"image_cache_path"`
	DownloadImages      bool   `mapstructure:"download_images"`
	PosterSize          string `mapstructure:"poster_size"`
	BackdropSize        string `mapstructure:"backdrop_size"`
	ProfileSize         string `mapstructure:"profile_size"`
	RequestsPer10s      int    `mapstructure:"requests_per_10s"`
	MaxCastMembers      int    `mapstructure:"max_cast_members"`
	SimilarContentCount int    `mapstructure:"similar_content_count"`
	CacheTTLHours       int    `mapstructure:"cache_ttl_hours"`
}

// WebSocketConfig holds WebSocket server configuration
type WebSocketConfig struct {
	Enabled             bool `mapstructure:"enabled"`
	PingIntervalSecs    int  `mapstructure:"ping_interval_seconds"`
	PongTimeoutSecs     int  `mapstructure:"pong_timeout_seconds"`
	WriteTimeoutSecs    int  `mapstructure:"write_timeout_seconds"`
	MaxMessageSizeBytes int  `mapstructure:"max_message_size_bytes"`
}

type LibraryConfig struct {
	RecentlyAddedCount int `mapstructure:"recently_added_count"`
}

func (c *Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	if c.Auth.Enabled {
		if err := c.Auth.Validate(); err != nil {
			return fmt.Errorf("auth config: %w", err)
		}
	}

	if err := c.Job.Validate(); err != nil {
		return fmt.Errorf("job config: %w", err)
	}

	if err := c.Media.Validate(); err != nil {
		return fmt.Errorf("media config: %w", err)
	}

	if err := c.Transcoding.Validate(); err != nil {
		return fmt.Errorf("transcoding config: %w", err)
	}

	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}

	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config: %w", err)
	}

	if c.TMDB.Enabled {
		if err := c.TMDB.Validate(); err != nil {
			return fmt.Errorf("tmdb config: %w", err)
		}
	}

	if err := c.Library.Validate(); err != nil {
		return fmt.Errorf("library config: %w", err)
	}

	if c.WebSocket.Enabled {
		if err := c.WebSocket.Validate(); err != nil {
			return fmt.Errorf("websocket config: %w", err)
		}
	}

	return nil
}

func (s *ServerConfig) Validate() error {
	if s.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", s.Port)
	}

	if s.ShutdownTimeoutSeconds < 5 || s.ShutdownTimeoutSeconds > 300 {
		return fmt.Errorf("shutdown_timeout_seconds must be between 5 and 300, got %d", s.ShutdownTimeoutSeconds)
	}

	return nil
}

func (a *AuthConfig) Validate() error {
	if a.Username == "" {
		return fmt.Errorf("username cannot be empty when auth is enabled")
	}

	if a.PasswordHash == "" {
		return fmt.Errorf("password_hash cannot be empty when auth is enabled")
	}

	if a.JWTSecret == "" {
		return fmt.Errorf("jwt_secret cannot be empty when auth is enabled (set via AUTH_JWT_SECRET env var)")
	}

	if len(a.JWTSecret) < 32 {
		return fmt.Errorf("jwt_secret must be at least 32 characters for security")
	}

	if a.TokenExpiryHours < 1 || a.TokenExpiryHours > 720 {
		return fmt.Errorf("token_expiry_hours must be between 1 and 720 (30 days), got %d", a.TokenExpiryHours)
	}

	if a.StreamTokenMins < 5 || a.StreamTokenMins > 1440 {
		return fmt.Errorf("stream_token_mins must be between 5 and 1440 (24 hours), got %d", a.StreamTokenMins)
	}

	return nil
}

func (j *JobConfig) Validate() error {
	if j.WorkerCount < 1 || j.WorkerCount > 10 {
		return fmt.Errorf("worker count must be between 1 and 10, got %d", j.WorkerCount)
	}

	if j.PollingIntervalInSeconds < 1 || j.PollingIntervalInSeconds > 60 {
		return fmt.Errorf("polling interval must be between 1 and 60, got %d", j.PollingIntervalInSeconds)
	}

	if j.MaxAttempts < 1 || j.MaxAttempts > 100 {
		return fmt.Errorf("max attempts must be between 1 and 100, got %d", j.MaxAttempts)
	}

	if j.SchedulerIntervalInSeconds < 1 || j.SchedulerIntervalInSeconds > 3600 {
		return fmt.Errorf("max scheduler interval must be between 1 and 3600, got %d", j.SchedulerIntervalInSeconds)
	}

	if j.SchedulerBatch < 1 || j.SchedulerBatch > 100 {
		return fmt.Errorf("max scheduler batch must be between 1 and 100, got %d", j.SchedulerBatch)
	}

	if j.BackoffBaseSeconds < 1 || j.BackoffBaseSeconds > 60 {
		return fmt.Errorf("backoff base seconds must be between 1 and 60, got %d", j.BackoffBaseSeconds)
	}

	if j.BackoffMaxSeconds < j.BackoffBaseSeconds || j.BackoffMaxSeconds > 3600 {
		return fmt.Errorf("backoff max seconds must be between backoff_base_seconds and 3600, got %d", j.BackoffMaxSeconds)
	}

	if j.StuckJobThresholdMinutes < 30 || j.StuckJobThresholdMinutes > 10080 {
		return fmt.Errorf("stuck job threshold must be between 30 and 10080 minutes (7 days), got %d", j.StuckJobThresholdMinutes)
	}

	if j.StuckJobCheckIntervalMinutes < 1 || j.StuckJobCheckIntervalMinutes > 1440 {
		return fmt.Errorf("stuck job check interval must be between 1 and 1440 minutes (24 hours), got %d", j.StuckJobCheckIntervalMinutes)
	}

	return nil
}

func (m *MediaConfig) Validate() error {
	if m.LibraryPath == "" {
		return fmt.Errorf("library_path cannot be empty")
	}

	if m.MediaPath == "" {
		return fmt.Errorf("media_path cannot be empty")
	}

	if m.StagingPath == "" {
		return fmt.Errorf("staging_path cannot be empty")
	}

	if m.TrashPath == "" {
		return fmt.Errorf("trash_path cannot be empty")
	}

	if m.TranscodeOutputPattern == "" {
		return fmt.Errorf("transcode_output_pattern cannot be empty")
	}

	if len(m.SupportedFormats) == 0 {
		return fmt.Errorf("supported_formats cannot be empty")
	}

	for _, format := range m.SupportedFormats {
		if !strings.HasPrefix(format, ".") {
			return fmt.Errorf("format must start with a dot, got: %s", format)
		}
	}

	if m.FFProbeTimeoutSeconds < 1 || m.FFProbeTimeoutSeconds > 300 {
		return fmt.Errorf("ffprobe_timeout_seconds must be between 1 and 300, got %d", m.FFProbeTimeoutSeconds)
	}

	if m.TranscodeTimeoutSeconds < 60 || m.TranscodeTimeoutSeconds > 36000 {
		return fmt.Errorf("transcode_timeout_seconds must be between 60 and 36000 (10 hours), got %d", m.TranscodeTimeoutSeconds)
	}

	if err := m.LibraryScan.Validate(); err != nil {
		return fmt.Errorf("library_scan: %w", err)
	}

	return nil
}

func (l *LibraryScanConfig) Validate() error {
	if l.Enabled && l.CronSpec == "" {
		return fmt.Errorf("cron_spec cannot be empty when library scan is enabled")
	}

	return nil
}

func (t *TranscodingConfig) Validate() error {
	if t.SegmentDuration < 1 {
		return fmt.Errorf("segment_duration must be at least 1 second, got %d", t.SegmentDuration)
	}

	if len(t.QualityProfiles) == 0 {
		return fmt.Errorf("qualities cannot be empty, at least one quality must be defined")
	}

	for i, quality := range t.QualityProfiles {
		if err := quality.Validate(); err != nil {
			return fmt.Errorf("quality[%d]: %w", i, err)
		}
	}

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

	if q.CRF < 18 || q.CRF > 26 {
		return fmt.Errorf("crf must be between 18 and 26 for quality %s, got %d", q.Name, q.CRF)
	}

	if q.MaxBitrate == "" {
		return fmt.Errorf("max_bitrate cannot be empty for quality %s (required for CRF mode)", q.Name)
	}

	return nil
}

func (d *DatabaseConfig) Validate() error {
	if d.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	return nil
}

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

	validPosterSizes := map[string]bool{
		"w92": true, "w154": true, "w185": true, "w342": true, "w500": true, "w780": true, "original": true,
	}
	if !validPosterSizes[t.PosterSize] {
		return fmt.Errorf("invalid poster_size: %s (must be one of: w92, w154, w185, w342, w500, w780, original)", t.PosterSize)
	}

	validBackdropSizes := map[string]bool{
		"w300": true, "w780": true, "w1280": true, "original": true,
	}
	if !validBackdropSizes[t.BackdropSize] {
		return fmt.Errorf("invalid backdrop_size: %s (must be one of: w300, w780, w1280, original)", t.BackdropSize)
	}

	// ProfileSize is optional - only validate if set
	if t.ProfileSize != "" {
		validProfileSizes := map[string]bool{
			"w45": true, "w185": true, "h632": true, "original": true,
		}
		if !validProfileSizes[t.ProfileSize] {
			return fmt.Errorf("invalid profile_size: %s (must be one of: w45, w185, h632, original)", t.ProfileSize)
		}
	}

	if t.RequestsPer10s < 1 || t.RequestsPer10s > 40 {
		return fmt.Errorf("requests_per_10s must be between 1 and 40 (TMDB limit), got %d", t.RequestsPer10s)
	}

	if t.MaxCastMembers < 1 || t.MaxCastMembers > 50 {
		return fmt.Errorf("max_cast_members must be between 1 and 50, got %d", t.MaxCastMembers)
	}

	if t.SimilarContentCount < 2 || t.SimilarContentCount > 12 {
		return fmt.Errorf("similar_content_count must be between 2 and 12, got %d", t.SimilarContentCount)
	}

	if t.CacheTTLHours < 1 || t.CacheTTLHours > 8760 {
		return fmt.Errorf("cache_ttl_hours must be between 1 and 8760 (1 year), got %d", t.CacheTTLHours)
	}

	return nil
}

func (l *LibraryConfig) Validate() error {
	if l.RecentlyAddedCount < 1 || l.RecentlyAddedCount > 100 {
		return fmt.Errorf("recently_added_count must be between 1 and 100, got %d", l.RecentlyAddedCount)
	}

	return nil
}

func (w *WebSocketConfig) Validate() error {
	if w.PingIntervalSecs < 10 || w.PingIntervalSecs > 120 {
		return fmt.Errorf("ping_interval_seconds must be between 10 and 120, got %d", w.PingIntervalSecs)
	}

	if w.PongTimeoutSecs < 5 || w.PongTimeoutSecs > 60 {
		return fmt.Errorf("pong_timeout_seconds must be between 5 and 60, got %d", w.PongTimeoutSecs)
	}

	if w.WriteTimeoutSecs < 5 || w.WriteTimeoutSecs > 60 {
		return fmt.Errorf("write_timeout_seconds must be between 5 and 60, got %d", w.WriteTimeoutSecs)
	}

	if w.MaxMessageSizeBytes < 256 || w.MaxMessageSizeBytes > 65536 {
		return fmt.Errorf("max_message_size_bytes must be between 256 and 65536, got %d", w.MaxMessageSizeBytes)
	}

	return nil
}

func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
