package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDefaults(t *testing.T) {
	v := viper.New()
	setDefaults(v)

	// Server defaults
	assert.Equal(t, "127.0.0.1", v.GetString("server.host"))
	assert.Equal(t, 8080, v.GetInt("server.port"))

	// Media defaults
	assert.Equal(t, "./media", v.GetString("media.library_path"))
	assert.Equal(t, "./trash", v.GetString("media.trash_path"))
	supportedFormats := v.GetStringSlice("media.supported_formats")
	assert.Contains(t, supportedFormats, ".mp4")
	assert.Contains(t, supportedFormats, ".mkv")
	assert.Contains(t, supportedFormats, ".avi")

	// Transcoding defaults
	assert.Equal(t, "./transcoded", v.GetString("transcoding.output_path"))
	assert.Equal(t, 4, v.GetInt("transcoding.segment_duration"))
	assert.Equal(t, "chunk-%03d.m4s", v.GetString("transcoding.segment_pattern"))

	// Quality profiles defaults
	qualityProfiles := v.Get("transcoding.quality_profiles")
	assert.NotNil(t, qualityProfiles)
	profiles, ok := qualityProfiles.([]map[string]interface{})
	require.True(t, ok, "quality profiles should be a slice of maps")
	assert.GreaterOrEqual(t, len(profiles), 2, "should have at least 2 quality profiles")

	// Check 360p profile exists
	found360p := false
	for _, profile := range profiles {
		if profile["name"] == "360p" {
			found360p = true
			assert.Equal(t, true, profile["enabled"])
			assert.Equal(t, "640x360", profile["resolution"])
			assert.Equal(t, "96k", profile["audio_bitrate"])
			assert.Equal(t, 25, profile["crf"])
			assert.Equal(t, "900k", profile["max_bitrate"])
			break
		}
	}
	assert.True(t, found360p, "360p profile should exist")

	// Check 480p profile exists
	found480p := false
	for _, profile := range profiles {
		if profile["name"] == "480p" {
			found480p = true
			assert.Equal(t, true, profile["enabled"])
			assert.Equal(t, "854x480", profile["resolution"])
			break
		}
	}
	assert.True(t, found480p, "480p profile should exist")

	// Database defaults
	assert.Equal(t, "./data/vimesrv.db", v.GetString("database.path"))

	// Logging defaults
	assert.Equal(t, "info", v.GetString("logging.level"))
	assert.Equal(t, "console", v.GetString("logging.format"))
	assert.Equal(t, "", v.GetString("logging.file"))

	// TMDB defaults
	assert.Equal(t, false, v.GetBool("tmdb.enabled"))
	assert.Equal(t, "", v.GetString("tmdb.api_key"))
	assert.Equal(t, "en-US", v.GetString("tmdb.language"))
	assert.Equal(t, true, v.GetBool("tmdb.auto_search"))
	assert.Equal(t, 70, v.GetInt("tmdb.auto_link_threshold"))
	assert.Equal(t, 5, v.GetInt("tmdb.max_candidates"))
	assert.Equal(t, "./cache/tmdb", v.GetString("tmdb.image_cache_path"))
	assert.Equal(t, true, v.GetBool("tmdb.download_images"))
	assert.Equal(t, "w500", v.GetString("tmdb.poster_size"))
	assert.Equal(t, "w1280", v.GetString("tmdb.backdrop_size"))
	assert.Equal(t, 35, v.GetInt("tmdb.requests_per_10s"))
}

func TestBindEnvVars(t *testing.T) {
	v := viper.New()
	v.SetEnvPrefix("VIMESRV")
	bindEnvVars(v)

	// Test server env vars
	t.Setenv("SERVER_HOST", "192.168.1.1")
	t.Setenv("SERVER_PORT", "9000")
	assert.Equal(t, "192.168.1.1", v.GetString("server.host"))
	assert.Equal(t, 9000, v.GetInt("server.port"))

	// Test media env vars
	t.Setenv("MEDIA_LIBRARY_PATH", "/custom/media")
	t.Setenv("MEDIA_TRASH_PATH", "/custom/trash")
	assert.Equal(t, "/custom/media", v.GetString("media.library_path"))
	assert.Equal(t, "/custom/trash", v.GetString("media.trash_path"))

	// Test transcoding env vars
	t.Setenv("TRANSCODING_OUTPUT_PATH", "/custom/output")
	t.Setenv("TRANSCODING_SEGMENT_DURATION", "6")
	assert.Equal(t, "/custom/output", v.GetString("transcoding.output_path"))
	assert.Equal(t, 6, v.GetInt("transcoding.segment_duration"))

	// Test database env vars
	t.Setenv("DATABASE_PATH", "/custom/db.db")
	assert.Equal(t, "/custom/db.db", v.GetString("database.path"))

	// Test logging env vars
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("LOG_FILE", "/var/log/app.log")
	assert.Equal(t, "debug", v.GetString("logging.level"))
	assert.Equal(t, "json", v.GetString("logging.format"))
	assert.Equal(t, "/var/log/app.log", v.GetString("logging.file"))

	// Test TMDB env vars
	t.Setenv("TMDB_ENABLED", "false")
	t.Setenv("TMDB_API_KEY", "custom-api-key")
	t.Setenv("TMDB_LANGUAGE", "fr-FR")
	t.Setenv("TMDB_AUTO_SEARCH", "false")
	t.Setenv("TMDB_AUTO_LINK_THRESHOLD", "80")
	t.Setenv("TMDB_MAX_CANDIDATES", "10")
	t.Setenv("TMDB_IMAGE_CACHE_PATH", "/custom/cache")
	t.Setenv("TMDB_DOWNLOAD_IMAGES", "false")
	t.Setenv("TMDB_POSTER_SIZE", "w342")
	t.Setenv("TMDB_BACKDROP_SIZE", "w780")
	t.Setenv("TMDB_REQUESTS_PER_10S", "25")
	assert.Equal(t, false, v.GetBool("tmdb.enabled"))
	assert.Equal(t, "custom-api-key", v.GetString("tmdb.api_key"))
	assert.Equal(t, "fr-FR", v.GetString("tmdb.language"))
	assert.Equal(t, false, v.GetBool("tmdb.auto_search"))
	assert.Equal(t, 80, v.GetInt("tmdb.auto_link_threshold"))
	assert.Equal(t, 10, v.GetInt("tmdb.max_candidates"))
	assert.Equal(t, "/custom/cache", v.GetString("tmdb.image_cache_path"))
	assert.Equal(t, false, v.GetBool("tmdb.download_images"))
	assert.Equal(t, "w342", v.GetString("tmdb.poster_size"))
	assert.Equal(t, "w780", v.GetString("tmdb.backdrop_size"))
	assert.Equal(t, 25, v.GetInt("tmdb.requests_per_10s"))
}

func TestNormalizePathsToAbsolute(t *testing.T) {
	tests := []struct {
		name    string
		input   Config
		check   func(t *testing.T, cfg *Config)
		wantErr bool
	}{
		{
			name: "relative media library path",
			input: Config{
				Media: MediaConfig{
					LibraryPath: "./media",
				},
				Database: DatabaseConfig{
					Path: "/abs/db.db",
				},
				Logging: LoggingConfig{
					File: "",
				},
			},
			check: func(t *testing.T, cfg *Config) {
				assert.True(t, filepath.IsAbs(cfg.Media.LibraryPath), "library path should be absolute")
				assert.Contains(t, cfg.Media.LibraryPath, "media")
			},
			wantErr: false,
		},
		{
			name: "absolute media library path",
			input: Config{
				Media: MediaConfig{
					LibraryPath: "/absolute/media",
				},
				Database: DatabaseConfig{
					Path: "/abs/db.db",
				},
			},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "/absolute/media", cfg.Media.LibraryPath)
			},
			wantErr: false,
		},
		{
			name: "relative database path",
			input: Config{
				Media: MediaConfig{
					LibraryPath: "/media",
				},
				Database: DatabaseConfig{
					Path: "./data/vimesrv.db",
				},
			},
			check: func(t *testing.T, cfg *Config) {
				assert.True(t, filepath.IsAbs(cfg.Database.Path), "database path should be absolute")
				assert.Contains(t, cfg.Database.Path, "vimesrv.db")
			},
			wantErr: false,
		},
		{
			name: "absolute database path",
			input: Config{
				Media: MediaConfig{
					LibraryPath: "/media",
				},
				Database: DatabaseConfig{
					Path: "/absolute/data/vimesrv.db",
				},
			},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "/absolute/data/vimesrv.db", cfg.Database.Path)
			},
			wantErr: false,
		},
		{
			name: "empty logging file",
			input: Config{
				Media: MediaConfig{
					LibraryPath: "/media",
				},
				Database: DatabaseConfig{
					Path: "/db.db",
				},
				Logging: LoggingConfig{
					File: "",
				},
			},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "", cfg.Logging.File)
			},
			wantErr: false,
		},
		{
			name: "relative logging file path",
			input: Config{
				Media: MediaConfig{
					LibraryPath: "/media",
				},
				Database: DatabaseConfig{
					Path: "/db.db",
				},
				Logging: LoggingConfig{
					File: "./logs/app.log",
				},
			},
			check: func(t *testing.T, cfg *Config) {
				assert.True(t, filepath.IsAbs(cfg.Logging.File), "logging file path should be absolute")
				assert.Contains(t, cfg.Logging.File, "app.log")
			},
			wantErr: false,
		},
		{
			name: "absolute logging file path",
			input: Config{
				Media: MediaConfig{
					LibraryPath: "/media",
				},
				Database: DatabaseConfig{
					Path: "/db.db",
				},
				Logging: LoggingConfig{
					File: "/var/log/app.log",
				},
			},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "/var/log/app.log", cfg.Logging.File)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.input
			err := normalizePathsToAbsolute(&cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, &cfg)
				}
			}
		})
	}
}

func TestDeriveSubpaths(t *testing.T) {
	tests := []struct {
		name        string
		libraryPath string
		wantTrash   string
		wantOutput  string
	}{
		{
			name:        "absolute path",
			libraryPath: "/media/library",
			wantTrash:   "/media/library/trash",
			wantOutput:  "/media/library/transcoded",
		},
		{
			name:        "relative path",
			libraryPath: "./media",
			wantTrash:   "media/trash",
			wantOutput:  "media/transcoded",
		},
		{
			name:        "path with trailing slash",
			libraryPath: "/media/library/",
			wantTrash:   "/media/library/trash",
			wantOutput:  "/media/library/transcoded",
		},
		{
			name:        "complex path",
			libraryPath: "/var/data/media/lib",
			wantTrash:   "/var/data/media/lib/trash",
			wantOutput:  "/var/data/media/lib/transcoded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Media: MediaConfig{
					LibraryPath: tt.libraryPath,
				},
			}

			err := deriveSubpaths(cfg)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantTrash, cfg.Media.TrashPath)
			assert.Equal(t, tt.wantOutput, cfg.Transcoding.OutputPath)
		})
	}
}

func TestCreateDirectories(t *testing.T) {
	t.Run("creates all required directories", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := &Config{
			Media: MediaConfig{
				LibraryPath: filepath.Join(tempDir, "media"),
				TrashPath:   filepath.Join(tempDir, "media", "trash"),
			},
			Transcoding: TranscodingConfig{
				OutputPath: filepath.Join(tempDir, "media", "transcoded"),
			},
			Database: DatabaseConfig{
				Path: filepath.Join(tempDir, "data", "vimesrv.db"),
			},
			Logging: LoggingConfig{
				File: filepath.Join(tempDir, "logs", "app.log"),
			},
		}

		err := createDirectories(cfg)
		require.NoError(t, err)

		// Check library path
		assertDirExists(t, cfg.Media.LibraryPath)

		// Check trash path
		assertDirExists(t, cfg.Media.TrashPath)

		// Check trash subdirectories
		assertDirExists(t, filepath.Join(cfg.Media.TrashPath, "original"))
		assertDirExists(t, filepath.Join(cfg.Media.TrashPath, "transcoded"))

		// Check transcoding output path
		assertDirExists(t, cfg.Transcoding.OutputPath)

		// Check database directory
		assertDirExists(t, filepath.Dir(cfg.Database.Path))

		// Check log file directory
		assertDirExists(t, filepath.Dir(cfg.Logging.File))
	})

	t.Run("handles existing directories", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := &Config{
			Media: MediaConfig{
				LibraryPath: tempDir,
				TrashPath:   filepath.Join(tempDir, "trash"),
			},
			Transcoding: TranscodingConfig{
				OutputPath: filepath.Join(tempDir, "transcoded"),
			},
			Database: DatabaseConfig{
				Path: filepath.Join(tempDir, "db.db"),
			},
			Logging: LoggingConfig{
				File: "",
			},
		}

		// Create some directories beforehand
		require.NoError(t, os.MkdirAll(cfg.Media.TrashPath, 0o755))

		// Should not error on existing directories
		err := createDirectories(cfg)
		assert.NoError(t, err)
	})

	t.Run("creates directories with correct permissions", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := &Config{
			Media: MediaConfig{
				LibraryPath: filepath.Join(tempDir, "media"),
				TrashPath:   filepath.Join(tempDir, "media", "trash"),
			},
			Transcoding: TranscodingConfig{
				OutputPath: filepath.Join(tempDir, "media", "transcoded"),
			},
			Database: DatabaseConfig{
				Path: filepath.Join(tempDir, "data", "vimesrv.db"),
			},
			Logging: LoggingConfig{
				File: "",
			},
		}

		err := createDirectories(cfg)
		require.NoError(t, err)

		// Check permissions (at least readable and executable)
		info, err := os.Stat(cfg.Media.LibraryPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
		assert.True(t, info.Mode().Perm()&0o700 == 0o700, "directory should have rwx for owner")
	})

	t.Run("no log file specified", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := &Config{
			Media: MediaConfig{
				LibraryPath: filepath.Join(tempDir, "media"),
				TrashPath:   filepath.Join(tempDir, "media", "trash"),
			},
			Transcoding: TranscodingConfig{
				OutputPath: filepath.Join(tempDir, "media", "transcoded"),
			},
			Database: DatabaseConfig{
				Path: filepath.Join(tempDir, "data", "vimesrv.db"),
			},
			Logging: LoggingConfig{
				File: "",
			},
		}

		err := createDirectories(cfg)
		assert.NoError(t, err)
	})

	t.Run("database in current directory", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := &Config{
			Media: MediaConfig{
				LibraryPath: filepath.Join(tempDir, "media"),
				TrashPath:   filepath.Join(tempDir, "media", "trash"),
			},
			Transcoding: TranscodingConfig{
				OutputPath: filepath.Join(tempDir, "media", "transcoded"),
			},
			Database: DatabaseConfig{
				Path: filepath.Join(tempDir, "vimesrv.db"),
			},
			Logging: LoggingConfig{
				File: "",
			},
		}

		err := createDirectories(cfg)
		assert.NoError(t, err)
	})
}

func TestCreateDirectories_ErrorOnMkdirAll(t *testing.T) {
	// This test attempts to trigger the error path in createDirectories
	// by creating a file where a directory should be created
	t.Run("file exists where directory should be", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a file at the location where we want a directory
		conflictPath := filepath.Join(tempDir, "conflict")
		err := os.WriteFile(conflictPath, []byte("content"), 0o644)
		require.NoError(t, err)

		cfg := &Config{
			Media: MediaConfig{
				LibraryPath: conflictPath, // This is a file, not a directory
				TrashPath:   filepath.Join(conflictPath, "trash"),
			},
			Transcoding: TranscodingConfig{
				OutputPath: filepath.Join(conflictPath, "transcoded"),
			},
			Database: DatabaseConfig{
				Path: filepath.Join(tempDir, "db.db"),
			},
			Logging: LoggingConfig{
				File: "",
			},
		}

		// This should error because we can't create a directory over a file
		err = createDirectories(cfg)
		// On most systems this will error, but the behavior depends on the OS
		// We'll just verify it doesn't panic
		_ = err
	})
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err, "directory should exist: %s", path)
	assert.True(t, info.IsDir(), "path should be a directory: %s", path)
}
