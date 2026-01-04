package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// Create a minimal valid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "worker.yaml")

	configContent := `
server:
  url: "http://localhost:8080"
  auth_token: "test-token-at-least-16-chars"
worker:
  name: "test-worker"
media:
  media_path: "/tmp/test/media"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check defaults are applied
	assert.Equal(t, 5, cfg.Worker.PollIntervalSeconds)
	assert.Equal(t, 30, cfg.Worker.HeartbeatIntervalSeconds)
	assert.Equal(t, 2, cfg.Worker.Concurrency)
	assert.Equal(t, 5, cfg.Worker.ProgressIntervalSeconds)
	assert.Equal(t, "ffmpeg", cfg.Transcoding.FFmpegPath)
	assert.Equal(t, "ffprobe", cfg.Transcoding.FFprobePath)
	assert.Equal(t, 7200, cfg.Transcoding.TimeoutSeconds)
	assert.Equal(t, 4, cfg.Transcoding.SegmentDuration)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "console", cfg.Logging.Format)
}

func TestLoad_CustomValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "worker.yaml")

	configContent := `
server:
  url: "https://server.example.com:9000"
  auth_token: "custom-token-at-least-16-chars"
worker:
  id: "custom-worker-id"
  name: "custom-worker"
  poll_interval_seconds: 10
  heartbeat_interval_seconds: 60
  concurrency: 4
  progress_interval_seconds: 10
media:
  media_path: "/mnt/nfs/media"
transcoding:
  ffmpeg_path: "/usr/local/bin/ffmpeg"
  ffprobe_path: "/usr/local/bin/ffprobe"
  timeout_seconds: 3600
  segment_duration: 6
logging:
  level: "debug"
  format: "json"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "https://server.example.com:9000", cfg.Server.URL)
	assert.Equal(t, "custom-token-at-least-16-chars", cfg.Server.AuthToken)
	assert.Equal(t, "custom-worker-id", cfg.Worker.ID)
	assert.Equal(t, "custom-worker", cfg.Worker.Name)
	assert.Equal(t, 10, cfg.Worker.PollIntervalSeconds)
	assert.Equal(t, 60, cfg.Worker.HeartbeatIntervalSeconds)
	assert.Equal(t, 4, cfg.Worker.Concurrency)
	assert.Equal(t, 10, cfg.Worker.ProgressIntervalSeconds)
	assert.Equal(t, "/usr/local/bin/ffmpeg", cfg.Transcoding.FFmpegPath)
	assert.Equal(t, "/usr/local/bin/ffprobe", cfg.Transcoding.FFprobePath)
	assert.Equal(t, 3600, cfg.Transcoding.TimeoutSeconds)
	assert.Equal(t, 6, cfg.Transcoding.SegmentDuration)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    ServerConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: ServerConfig{
				URL:       "http://localhost:8080",
				AuthToken: "token-at-least-16-chars",
			},
			expectErr: false,
		},
		{
			name: "valid https config",
			config: ServerConfig{
				URL:       "https://server.example.com",
				AuthToken: "token-at-least-16-chars",
			},
			expectErr: false,
		},
		{
			name: "empty url",
			config: ServerConfig{
				URL:       "",
				AuthToken: "token-at-least-16-chars",
			},
			expectErr: true,
			errMsg:    "url cannot be empty",
		},
		{
			name: "invalid url scheme",
			config: ServerConfig{
				URL:       "ftp://server.example.com",
				AuthToken: "token-at-least-16-chars",
			},
			expectErr: true,
			errMsg:    "url must start with http:// or https://",
		},
		{
			name: "empty auth token",
			config: ServerConfig{
				URL:       "http://localhost:8080",
				AuthToken: "",
			},
			expectErr: true,
			errMsg:    "auth_token cannot be empty",
		},
		{
			name: "short auth token",
			config: ServerConfig{
				URL:       "http://localhost:8080",
				AuthToken: "short",
			},
			expectErr: true,
			errMsg:    "auth_token must be at least 16 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWorkerSettings_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    WorkerSettings
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: WorkerSettings{
				Name:                     "test-worker",
				PollIntervalSeconds:      5,
				HeartbeatIntervalSeconds: 30,
				Concurrency:              2,
				ProgressIntervalSeconds:  5,
			},
			expectErr: false,
		},
		{
			name: "empty name",
			config: WorkerSettings{
				Name:                     "",
				PollIntervalSeconds:      5,
				HeartbeatIntervalSeconds: 30,
				Concurrency:              2,
				ProgressIntervalSeconds:  5,
			},
			expectErr: true,
			errMsg:    "name cannot be empty",
		},
		{
			name: "poll interval too low",
			config: WorkerSettings{
				Name:                     "test",
				PollIntervalSeconds:      0,
				HeartbeatIntervalSeconds: 30,
				Concurrency:              2,
				ProgressIntervalSeconds:  5,
			},
			expectErr: true,
			errMsg:    "poll_interval_seconds must be between 1 and 300",
		},
		{
			name: "poll interval too high",
			config: WorkerSettings{
				Name:                     "test",
				PollIntervalSeconds:      400,
				HeartbeatIntervalSeconds: 30,
				Concurrency:              2,
				ProgressIntervalSeconds:  5,
			},
			expectErr: true,
			errMsg:    "poll_interval_seconds must be between 1 and 300",
		},
		{
			name: "heartbeat interval too low",
			config: WorkerSettings{
				Name:                     "test",
				PollIntervalSeconds:      5,
				HeartbeatIntervalSeconds: 2,
				Concurrency:              2,
				ProgressIntervalSeconds:  5,
			},
			expectErr: true,
			errMsg:    "heartbeat_interval_seconds must be between 5 and 300",
		},
		{
			name: "concurrency too low",
			config: WorkerSettings{
				Name:                     "test",
				PollIntervalSeconds:      5,
				HeartbeatIntervalSeconds: 30,
				Concurrency:              0,
				ProgressIntervalSeconds:  5,
			},
			expectErr: true,
			errMsg:    "concurrency must be between 1 and 16",
		},
		{
			name: "concurrency too high",
			config: WorkerSettings{
				Name:                     "test",
				PollIntervalSeconds:      5,
				HeartbeatIntervalSeconds: 30,
				Concurrency:              20,
				ProgressIntervalSeconds:  5,
			},
			expectErr: true,
			errMsg:    "concurrency must be between 1 and 16",
		},
		{
			name: "progress interval too low",
			config: WorkerSettings{
				Name:                     "test",
				PollIntervalSeconds:      5,
				HeartbeatIntervalSeconds: 30,
				Concurrency:              2,
				ProgressIntervalSeconds:  0,
			},
			expectErr: true,
			errMsg:    "progress_interval_seconds must be between 1 and 60",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMediaConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    MediaConfig
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid config",
			config:    MediaConfig{MediaPath: "/mnt/media"},
			expectErr: false,
		},
		{
			name:      "empty path",
			config:    MediaConfig{MediaPath: ""},
			expectErr: true,
			errMsg:    "media_path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTranscodingConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    TranscodingConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: TranscodingConfig{
				FFmpegPath:      "ffmpeg",
				FFprobePath:     "ffprobe",
				TimeoutSeconds:  7200,
				SegmentDuration: 4,
			},
			expectErr: false,
		},
		{
			name: "empty ffmpeg path",
			config: TranscodingConfig{
				FFmpegPath:      "",
				FFprobePath:     "ffprobe",
				TimeoutSeconds:  7200,
				SegmentDuration: 4,
			},
			expectErr: true,
			errMsg:    "ffmpeg_path cannot be empty",
		},
		{
			name: "empty ffprobe path",
			config: TranscodingConfig{
				FFmpegPath:      "ffmpeg",
				FFprobePath:     "",
				TimeoutSeconds:  7200,
				SegmentDuration: 4,
			},
			expectErr: true,
			errMsg:    "ffprobe_path cannot be empty",
		},
		{
			name: "timeout too low",
			config: TranscodingConfig{
				FFmpegPath:      "ffmpeg",
				FFprobePath:     "ffprobe",
				TimeoutSeconds:  30,
				SegmentDuration: 4,
			},
			expectErr: true,
			errMsg:    "timeout_seconds must be between 60 and 36000",
		},
		{
			name: "segment duration too low",
			config: TranscodingConfig{
				FFmpegPath:      "ffmpeg",
				FFprobePath:     "ffprobe",
				TimeoutSeconds:  7200,
				SegmentDuration: 0,
			},
			expectErr: true,
			errMsg:    "segment_duration must be between 1 and 30",
		},
		{
			name: "segment duration too high",
			config: TranscodingConfig{
				FFmpegPath:      "ffmpeg",
				FFprobePath:     "ffprobe",
				TimeoutSeconds:  7200,
				SegmentDuration: 60,
			},
			expectErr: true,
			errMsg:    "segment_duration must be between 1 and 30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoggingConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    LoggingConfig
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid console",
			config:    LoggingConfig{Level: "info", Format: "console"},
			expectErr: false,
		},
		{
			name:      "valid json",
			config:    LoggingConfig{Level: "debug", Format: "json"},
			expectErr: false,
		},
		{
			name:      "all log levels",
			config:    LoggingConfig{Level: "warn", Format: "console"},
			expectErr: false,
		},
		{
			name:      "invalid level",
			config:    LoggingConfig{Level: "invalid", Format: "console"},
			expectErr: true,
			errMsg:    "level must be one of: debug, info, warn, error",
		},
		{
			name:      "invalid format",
			config:    LoggingConfig{Level: "info", Format: "xml"},
			expectErr: true,
			errMsg:    "format must be one of: console, json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "worker.yaml")

	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	require.Error(t, err)
}

func TestLoad_ValidationFails(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "worker.yaml")

	// Missing required auth_token
	configContent := `
server:
  url: "http://localhost:8080"
  auth_token: ""
worker:
  name: "test"
media:
  media_path: "/tmp"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config validation failed")
}
