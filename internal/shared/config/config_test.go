package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ServerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ServerConfig{
				Host:                   "localhost",
				Port:                   8080,
				ShutdownTimeoutSeconds: 30,
			},
			wantErr: false,
		},
		{
			name: "valid config with IP",
			config: ServerConfig{
				Host:                   "127.0.0.1",
				Port:                   3000,
				ShutdownTimeoutSeconds: 30,
			},
			wantErr: false,
		},
		{
			name: "valid config with port 1",
			config: ServerConfig{
				Host:                   "0.0.0.0",
				Port:                   1,
				ShutdownTimeoutSeconds: 30,
			},
			wantErr: false,
		},
		{
			name: "valid config with port 65535",
			config: ServerConfig{
				Host:                   "localhost",
				Port:                   65535,
				ShutdownTimeoutSeconds: 30,
			},
			wantErr: false,
		},
		{
			name: "empty host",
			config: ServerConfig{
				Host: "",
				Port: 8080,
			},
			wantErr: true,
		},
		{
			name: "port zero",
			config: ServerConfig{
				Host: "localhost",
				Port: 0,
			},
			wantErr: true,
		},
		{
			name: "negative port",
			config: ServerConfig{
				Host: "localhost",
				Port: -1,
			},
			wantErr: true,
		},
		{
			name: "port too large",
			config: ServerConfig{
				Host: "localhost",
				Port: 65536,
			},
			wantErr: true,
		},
		{
			name: "port way too large",
			config: ServerConfig{
				Host: "localhost",
				Port: 70000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServerConfig_Address(t *testing.T) {
	tests := []struct {
		name     string
		config   ServerConfig
		wantAddr string
	}{
		{
			name: "standard localhost",
			config: ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
			wantAddr: "localhost:8080",
		},
		{
			name: "IP address",
			config: ServerConfig{
				Host: "127.0.0.1",
				Port: 3000,
			},
			wantAddr: "127.0.0.1:3000",
		},
		{
			name: "all interfaces",
			config: ServerConfig{
				Host: "0.0.0.0",
				Port: 9000,
			},
			wantAddr: "0.0.0.0:9000",
		},
		{
			name: "port 80",
			config: ServerConfig{
				Host: "example.com",
				Port: 80,
			},
			wantAddr: "example.com:80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := tt.config.Address()
			assert.Equal(t, tt.wantAddr, addr)
		})
	}
}

func TestMediaConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  MediaConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: MediaConfig{
				LibraryPath:           "/media/library",
				MediaPath:             "/media/library/media",
				StagingPath:           "/media/library/staging",
				TrashPath:             "/media/trash",
				SupportedFormats:      []string{".mp4", ".mkv", ".avi"},
				FFProbeTimeoutSeconds: 30,
				LibraryScan: LibraryScanConfig{
					Enabled:  true,
					CronSpec: "0 * * * * *",
					Priority: 0,
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with many formats",
			config: MediaConfig{
				LibraryPath:           "/path/to/media",
				MediaPath:             "/path/to/media/media",
				StagingPath:           "/path/to/media/staging",
				TrashPath:             "/path/to/trash",
				SupportedFormats:      []string{".mp4", ".mkv", ".avi", ".mov", ".webm", ".flv"},
				FFProbeTimeoutSeconds: 30,
				LibraryScan: LibraryScanConfig{
					Enabled:  false,
					CronSpec: "",
					Priority: 0,
				},
			},
			wantErr: false,
		},
		{
			name: "empty library_path",
			config: MediaConfig{
				LibraryPath:      "",
				TrashPath:        "/media/trash",
				SupportedFormats: []string{".mp4"},
			},
			wantErr: true,
		},
		{
			name: "empty trash_path",
			config: MediaConfig{
				LibraryPath:      "/media/library",
				TrashPath:        "",
				SupportedFormats: []string{".mp4"},
			},
			wantErr: true,
		},
		{
			name: "empty supported_formats",
			config: MediaConfig{
				LibraryPath:      "/media/library",
				TrashPath:        "/media/trash",
				SupportedFormats: []string{},
			},
			wantErr: true,
		},
		{
			name: "format without dot",
			config: MediaConfig{
				LibraryPath:      "/media/library",
				TrashPath:        "/media/trash",
				SupportedFormats: []string{"mp4"},
			},
			wantErr: true,
		},
		{
			name: "multiple formats with one invalid",
			config: MediaConfig{
				LibraryPath:      "/media/library",
				TrashPath:        "/media/trash",
				SupportedFormats: []string{".mp4", "mkv", ".avi"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLibraryScanConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  LibraryScanConfig
		wantErr bool
	}{
		{
			name: "valid config enabled with cron spec",
			config: LibraryScanConfig{
				Enabled:  true,
				CronSpec: "0 * * * * *",
				Priority: 0,
			},
			wantErr: false,
		},
		{
			name: "valid config disabled with empty cron spec",
			config: LibraryScanConfig{
				Enabled:  false,
				CronSpec: "",
				Priority: 0,
			},
			wantErr: false,
		},
		{
			name: "valid config disabled with cron spec",
			config: LibraryScanConfig{
				Enabled:  false,
				CronSpec: "0 * * * * *",
				Priority: 5,
			},
			wantErr: false,
		},
		{
			name: "enabled with empty cron spec should fail",
			config: LibraryScanConfig{
				Enabled:  true,
				CronSpec: "",
				Priority: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQualityProfile_Validate(t *testing.T) {
	tests := []struct {
		name    string
		profile QualityProfile
		wantErr bool
	}{
		{
			name: "valid profile",
			profile: QualityProfile{
				Name:         "720p",
				Enabled:      true,
				Resolution:   "1280x720",
				AudioBitrate: "128k",
				CRF:          23,
				MaxBitrate:   "2800k",
			},
			wantErr: false,
		},
		{
			name: "valid profile with CRF 18",
			profile: QualityProfile{
				Name:         "1080p",
				Enabled:      true,
				Resolution:   "1920x1080",
				AudioBitrate: "192k",
				CRF:          18,
				MaxBitrate:   "5000k",
			},
			wantErr: false,
		},
		{
			name: "valid profile with CRF 26",
			profile: QualityProfile{
				Name:         "360p",
				Enabled:      true,
				Resolution:   "640x360",
				AudioBitrate: "96k",
				CRF:          26,
				MaxBitrate:   "900k",
			},
			wantErr: false,
		},
		{
			name: "valid profile disabled",
			profile: QualityProfile{
				Name:         "4K",
				Enabled:      false,
				Resolution:   "3840x2160",
				AudioBitrate: "256k",
				CRF:          20,
				MaxBitrate:   "16000k",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			profile: QualityProfile{
				Name:         "",
				Enabled:      true,
				Resolution:   "1280x720",
				AudioBitrate: "128k",
				CRF:          23,
				MaxBitrate:   "2800k",
			},
			wantErr: true,
		},
		{
			name: "empty resolution",
			profile: QualityProfile{
				Name:         "720p",
				Enabled:      true,
				Resolution:   "",
				AudioBitrate: "128k",
				CRF:          23,
				MaxBitrate:   "2800k",
			},
			wantErr: true,
		},
		{
			name: "empty audio_bitrate",
			profile: QualityProfile{
				Name:         "720p",
				Enabled:      true,
				Resolution:   "1280x720",
				AudioBitrate: "",
				CRF:          23,
				MaxBitrate:   "2800k",
			},
			wantErr: true,
		},
		{
			name: "CRF too low",
			profile: QualityProfile{
				Name:         "720p",
				Enabled:      true,
				Resolution:   "1280x720",
				AudioBitrate: "128k",
				CRF:          17,
				MaxBitrate:   "2800k",
			},
			wantErr: true,
		},
		{
			name: "CRF too high",
			profile: QualityProfile{
				Name:         "720p",
				Enabled:      true,
				Resolution:   "1280x720",
				AudioBitrate: "128k",
				CRF:          27,
				MaxBitrate:   "2800k",
			},
			wantErr: true,
		},
		{
			name: "empty max_bitrate",
			profile: QualityProfile{
				Name:         "720p",
				Enabled:      true,
				Resolution:   "1280x720",
				AudioBitrate: "128k",
				CRF:          23,
				MaxBitrate:   "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTranscodingConfig_Validate(t *testing.T) {
	validQuality := QualityProfile{
		Name:         "720p",
		Enabled:      true,
		Resolution:   "1280x720",
		AudioBitrate: "128k",
		CRF:          23,
		MaxBitrate:   "2800k",
	}

	disabledQuality := QualityProfile{
		Name:         "1080p",
		Enabled:      false,
		Resolution:   "1920x1080",
		AudioBitrate: "192k",
		CRF:          21,
		MaxBitrate:   "5500k",
	}

	invalidQuality := QualityProfile{
		Name:         "",
		Enabled:      true,
		Resolution:   "1280x720",
		AudioBitrate: "128k",
		CRF:          23,
		MaxBitrate:   "2800k",
	}

	tests := []struct {
		name    string
		config  TranscodingConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: TranscodingConfig{
				SegmentDuration: 4,
				QualityProfiles: []QualityProfile{validQuality},
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple qualities",
			config: TranscodingConfig{
				SegmentDuration: 6,
				QualityProfiles: []QualityProfile{validQuality, disabledQuality},
			},
			wantErr: false,
		},
		{
			name: "segment_duration zero",
			config: TranscodingConfig{
				SegmentDuration: 0,
				QualityProfiles: []QualityProfile{validQuality},
			},
			wantErr: true,
		},
		{
			name: "segment_duration negative",
			config: TranscodingConfig{
				SegmentDuration: -1,
				QualityProfiles: []QualityProfile{validQuality},
			},
			wantErr: true,
		},
		{
			name: "empty qualities",
			config: TranscodingConfig{
				SegmentDuration: 4,
				QualityProfiles: []QualityProfile{},
			},
			wantErr: true,
		},
		{
			name: "all qualities disabled",
			config: TranscodingConfig{
				SegmentDuration: 4,
				QualityProfiles: []QualityProfile{disabledQuality},
			},
			wantErr: true,
		},
		{
			name: "invalid quality profile",
			config: TranscodingConfig{
				SegmentDuration: 4,
				QualityProfiles: []QualityProfile{invalidQuality},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDatabaseConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DatabaseConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: DatabaseConfig{
				Path: "/data/vimesrv.db",
			},
			wantErr: false,
		},
		{
			name: "valid config with relative path",
			config: DatabaseConfig{
				Path: "./vimesrv.db",
			},
			wantErr: false,
		},
		{
			name: "empty path",
			config: DatabaseConfig{
				Path: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoggingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  LoggingConfig
		wantErr bool
	}{
		{
			name: "valid config debug console",
			config: LoggingConfig{
				Level:  "debug",
				Format: "console",
				File:   "",
			},
			wantErr: false,
		},
		{
			name: "valid config info json",
			config: LoggingConfig{
				Level:  "info",
				Format: "json",
				File:   "/var/log/vimesrv.log",
			},
			wantErr: false,
		},
		{
			name: "valid config warn console",
			config: LoggingConfig{
				Level:  "warn",
				Format: "console",
				File:   "",
			},
			wantErr: false,
		},
		{
			name: "valid config error json",
			config: LoggingConfig{
				Level:  "error",
				Format: "json",
				File:   "",
			},
			wantErr: false,
		},
		{
			name: "empty level",
			config: LoggingConfig{
				Level:  "",
				Format: "console",
				File:   "",
			},
			wantErr: true,
		},
		{
			name: "invalid level trace",
			config: LoggingConfig{
				Level:  "trace",
				Format: "console",
				File:   "",
			},
			wantErr: true,
		},
		{
			name: "invalid level critical",
			config: LoggingConfig{
				Level:  "critical",
				Format: "console",
				File:   "",
			},
			wantErr: true,
		},
		{
			name: "invalid level fatal",
			config: LoggingConfig{
				Level:  "fatal",
				Format: "console",
				File:   "",
			},
			wantErr: true,
		},
		{
			name: "empty format",
			config: LoggingConfig{
				Level:  "info",
				Format: "",
				File:   "",
			},
			wantErr: true,
		},
		{
			name: "invalid format text",
			config: LoggingConfig{
				Level:  "info",
				Format: "text",
				File:   "",
			},
			wantErr: true,
		},
		{
			name: "invalid format xml",
			config: LoggingConfig{
				Level:  "info",
				Format: "xml",
				File:   "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTMDBConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  TMDBConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "w1280",
				RequestsPer10s:    35,
			},
			wantErr: false,
		},
		{
			name: "valid config with threshold 0",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        false,
				AutoLinkThreshold: 0,
				MaxCandidates:     10,
				ImageCachePath:    "/cache",
				DownloadImages:    false,
				PosterSize:        "w342",
				BackdropSize:      "w780",
				RequestsPer10s:    40,
			},
			wantErr: false,
		},
		{
			name: "valid config with threshold 100",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "fr-FR",
				AutoSearch:        true,
				AutoLinkThreshold: 100,
				MaxCandidates:     1,
				ImageCachePath:    "/cache",
				DownloadImages:    true,
				PosterSize:        "original",
				BackdropSize:      "original",
				RequestsPer10s:    1,
			},
			wantErr: false,
		},
		{
			name: "valid config with all poster sizes",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w92",
				BackdropSize:      "w300",
				RequestsPer10s:    35,
			},
			wantErr: false,
		},
		{
			name: "empty api_key",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "w1280",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "empty language",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "w1280",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "auto_link_threshold negative",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: -1,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "w1280",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "auto_link_threshold too high",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 101,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "w1280",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "max_candidates zero",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     0,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "w1280",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "empty image_cache_path",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "w1280",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "empty poster_size",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "",
				BackdropSize:      "w1280",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "invalid poster_size",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w100",
				BackdropSize:      "w1280",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "invalid poster_size large",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "large",
				BackdropSize:      "w1280",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "empty backdrop_size",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "invalid backdrop_size",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "w500",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "invalid backdrop_size xlarge",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "xlarge",
				RequestsPer10s:    35,
			},
			wantErr: true,
		},
		{
			name: "requests_per_10s zero",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "w1280",
				RequestsPer10s:    0,
			},
			wantErr: true,
		},
		{
			name: "requests_per_10s too high",
			config: TMDBConfig{
				Enabled:           true,
				APIKey:            "test-api-key",
				Language:          "en-US",
				AutoSearch:        true,
				AutoLinkThreshold: 70,
				MaxCandidates:     5,
				ImageCachePath:    "/cache/tmdb",
				DownloadImages:    true,
				PosterSize:        "w500",
				BackdropSize:      "w1280",
				RequestsPer10s:    41,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	validJobConfig := JobConfig{
		WorkerCount:                  2,
		PollingIntervalInSeconds:     2,
		MaxAttempts:                  3,
		SchedulerIntervalInSeconds:   2,
		SchedulerBatch:               5,
		BackoffBaseSeconds:           2,
		BackoffMaxSeconds:            300,
		StuckJobThresholdMinutes:     480,
		StuckJobCheckIntervalMinutes: 5,
	}

	validServerConfig := ServerConfig{
		Host:                   "localhost",
		Port:                   8080,
		ShutdownTimeoutSeconds: 30,
	}

	invalidServerConfig := ServerConfig{
		Host:                   "",
		Port:                   8080,
		ShutdownTimeoutSeconds: 30,
	}

	validMediaConfig := MediaConfig{
		LibraryPath:           "/media/library",
		MediaPath:             "/media/library/media",
		StagingPath:           "/media/library/staging",
		TrashPath:             "/media/trash",
		SupportedFormats:      []string{".mp4", ".mkv"},
		FFProbeTimeoutSeconds: 30,
		LibraryScan: LibraryScanConfig{
			Enabled:  true,
			CronSpec: "0 * * * * *",
			Priority: 0,
		},
	}

	invalidMediaConfig := MediaConfig{
		LibraryPath:      "",
		MediaPath:        "/media/media",
		StagingPath:      "/media/staging",
		TrashPath:        "/media/trash",
		SupportedFormats: []string{".mp4"},
	}

	validQuality := QualityProfile{
		Name:         "720p",
		Enabled:      true,
		Resolution:   "1280x720",
		AudioBitrate: "128k",
		CRF:          23,
		MaxBitrate:   "2800k",
	}

	validTranscodingConfig := TranscodingConfig{
		SegmentDuration: 4,
		QualityProfiles: []QualityProfile{validQuality},
	}

	invalidTranscodingConfig := TranscodingConfig{
		SegmentDuration: 0,
		QualityProfiles: []QualityProfile{validQuality},
	}

	validDatabaseConfig := DatabaseConfig{
		Path: "/data/vimesrv.db",
	}

	invalidDatabaseConfig := DatabaseConfig{
		Path: "",
	}

	validLoggingConfig := LoggingConfig{
		Level:  "info",
		Format: "console",
	}

	invalidLoggingConfig := LoggingConfig{
		Level:  "invalid",
		Format: "console",
	}

	validTMDBConfig := TMDBConfig{
		Enabled:           true,
		APIKey:            "test-api-key",
		Language:          "en-US",
		AutoSearch:        true,
		AutoLinkThreshold: 70,
		MaxCandidates:     5,
		ImageCachePath:    "/cache/tmdb",
		DownloadImages:    true,
		PosterSize:        "w500",
		BackdropSize:      "w1280",
		RequestsPer10s:    35,
	}

	invalidTMDBConfig := TMDBConfig{
		Enabled:           true,
		APIKey:            "",
		Language:          "en-US",
		AutoSearch:        true,
		AutoLinkThreshold: 70,
		MaxCandidates:     5,
		ImageCachePath:    "/cache/tmdb",
		DownloadImages:    true,
		PosterSize:        "w500",
		BackdropSize:      "w1280",
		RequestsPer10s:    35,
	}

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid complete config",
			config: Config{
				Server:      validServerConfig,
				Job:         validJobConfig,
				Media:       validMediaConfig,
				Transcoding: validTranscodingConfig,
				Database:    validDatabaseConfig,
				Logging:     validLoggingConfig,
				TMDB:        validTMDBConfig,
			},
			wantErr: false,
		},
		{
			name: "valid config with TMDB disabled",
			config: Config{
				Server:      validServerConfig,
				Job:         validJobConfig,
				Media:       validMediaConfig,
				Transcoding: validTranscodingConfig,
				Database:    validDatabaseConfig,
				Logging:     validLoggingConfig,
				TMDB: TMDBConfig{
					Enabled: false,
					APIKey:  "", // Should not validate when disabled
				},
			},
			wantErr: false,
		},
		{
			name: "invalid server config",
			config: Config{
				Server:      invalidServerConfig,
				Job:         validJobConfig,
				Media:       validMediaConfig,
				Transcoding: validTranscodingConfig,
				Database:    validDatabaseConfig,
				Logging:     validLoggingConfig,
				TMDB:        validTMDBConfig,
			},
			wantErr: true,
		},
		{
			name: "invalid media config",
			config: Config{
				Server:      validServerConfig,
				Job:         validJobConfig,
				Media:       invalidMediaConfig,
				Transcoding: validTranscodingConfig,
				Database:    validDatabaseConfig,
				Logging:     validLoggingConfig,
				TMDB:        validTMDBConfig,
			},
			wantErr: true,
		},
		{
			name: "invalid transcoding config",
			config: Config{
				Server:      validServerConfig,
				Job:         validJobConfig,
				Media:       validMediaConfig,
				Transcoding: invalidTranscodingConfig,
				Database:    validDatabaseConfig,
				Logging:     validLoggingConfig,
				TMDB:        validTMDBConfig,
			},
			wantErr: true,
		},
		{
			name: "invalid database config",
			config: Config{
				Server:      validServerConfig,
				Job:         validJobConfig,
				Media:       validMediaConfig,
				Transcoding: validTranscodingConfig,
				Database:    invalidDatabaseConfig,
				Logging:     validLoggingConfig,
				TMDB:        validTMDBConfig,
			},
			wantErr: true,
		},
		{
			name: "invalid logging config",
			config: Config{
				Server:      validServerConfig,
				Job:         validJobConfig,
				Media:       validMediaConfig,
				Transcoding: validTranscodingConfig,
				Database:    validDatabaseConfig,
				Logging:     invalidLoggingConfig,
				TMDB:        validTMDBConfig,
			},
			wantErr: true,
		},
		{
			name: "invalid TMDB config when enabled",
			config: Config{
				Server:      validServerConfig,
				Job:         validJobConfig,
				Media:       validMediaConfig,
				Transcoding: validTranscodingConfig,
				Database:    validDatabaseConfig,
				Logging:     validLoggingConfig,
				TMDB:        invalidTMDBConfig,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
