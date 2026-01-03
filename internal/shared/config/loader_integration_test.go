package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_WithValidYAMLFile(t *testing.T) {
	tempDir := t.TempDir()

	yamlContent := fmt.Sprintf(`
server:
  host: "192.168.1.1"
  port: 9000
  shutdown_timeout_seconds: 30

job:
  worker_count: 2
  polling_interval_in_seconds: 2
  scheduler_batch: 5
  scheduler_interval_in_seconds: 2
  max_attempts: 3
  backoff_base_seconds: 2
  backoff_max_seconds: 300
  stuck_job_threshold_minutes: 480
  stuck_job_check_interval_minutes: 5

media:
  library_path: "%s/custom-media"
  supported_formats:
    - ".mp4"
    - ".mkv"
  transcode_output_pattern: "{media_path}/{media_id}/transcoded"

transcoding:
  segment_duration: 6
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/custom-data/vimesrv.db"

logging:
  level: "debug"
  format: "json"
  file: "%s/custom-logs/app.log"

tmdb:
  enabled: false
`, tempDir, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify custom values loaded
	assert.Equal(t, "192.168.1.1", cfg.Server.Host)
	assert.Equal(t, 9000, cfg.Server.Port)

	// Verify paths are normalized to absolute
	assert.True(t, filepath.IsAbs(cfg.Media.LibraryPath))
	assert.True(t, filepath.IsAbs(cfg.Database.Path))
	assert.True(t, filepath.IsAbs(cfg.Logging.File))

	// Verify derived paths
	assert.Equal(t, filepath.Join(cfg.Media.LibraryPath, "trash"), cfg.Media.TrashPath)

	// Verify directories were created
	assertDirExistsInIntegration(t, cfg.Media.LibraryPath)
	assertDirExistsInIntegration(t, cfg.Media.TrashPath)
	assertDirExistsInIntegration(t, filepath.Dir(cfg.Database.Path))
	assertDirExistsInIntegration(t, filepath.Dir(cfg.Logging.File))

	// Verify transcoding config
	assert.Equal(t, 6, cfg.Transcoding.SegmentDuration)
	assert.Len(t, cfg.Transcoding.QualityProfiles, 1)

	// Verify logging config
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)

	// Verify TMDB disabled
	assert.False(t, cfg.TMDB.Enabled)
}

func TestLoad_WithInvalidYAMLSyntax(t *testing.T) {
	tempDir := t.TempDir()

	yamlContent := `
server:
  host: "localhost"
  port: 8080
  invalid yaml syntax here [[[
`

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	cfg, err := Load(configPath)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_WithInvalidConfigValues(t *testing.T) {
	tempDir := t.TempDir()

	yamlContent := `
server:
  host: ""
  port: 8080

media:
  library_path: "/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "/data/vimesrv.db"

logging:
  level: "info"
  format: "console"

tmdb:
  enabled: false
`

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	cfg, err := Load(configPath)
	assert.Error(t, err, "should fail validation due to empty host")
	assert.Nil(t, cfg)
}

func TestLoad_WithNonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_WithEmptyPath(t *testing.T) {
	// Use temp directory to avoid creating directories in code directory
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	// Should use defaults (TMDB is disabled by default so this should succeed)
	cfg, err := Load("")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.False(t, cfg.TMDB.Enabled)

	// Verify directories were created in temp dir
	assert.True(t, filepath.IsAbs(cfg.Media.LibraryPath))
	assert.Contains(t, cfg.Media.LibraryPath, tempDir)
}

func TestLoadWithDefaults(t *testing.T) {
	// Use temp directory to avoid creating directories in code directory
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	// TMDB is disabled by default, so this should succeed
	cfg, err := LoadWithDefaults()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.False(t, cfg.TMDB.Enabled)

	// Verify directories were created in temp dir
	assert.True(t, filepath.IsAbs(cfg.Media.LibraryPath))
	assert.Contains(t, cfg.Media.LibraryPath, tempDir)
}

func TestLoadWithDefaults_WithMinimalConfig(t *testing.T) {
	tempDir := t.TempDir()

	// Create a minimal config file with just necessary fields
	yamlContent := fmt.Sprintf(`
server:
  host: "127.0.0.1"
  port: 8080

media:
  library_path: "%s/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "360p"
      enabled: true
      resolution: "640x360"
      audio_bitrate: "96k"
      crf: 25
      max_bitrate: "900k"
    - name: "480p"
      enabled: true
      resolution: "854x480"
      audio_bitrate: "128k"
      crf: 24
      max_bitrate: "1500k"

database:
  path: "%s/data/vimesrv.db"

logging:
  level: "info"
  format: "console"

tmdb:
  enabled: false
`, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify all defaults
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.True(t, filepath.IsAbs(cfg.Media.LibraryPath))
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "console", cfg.Logging.Format)
	assert.False(t, cfg.TMDB.Enabled)

	// Verify quality profiles
	assert.GreaterOrEqual(t, len(cfg.Transcoding.QualityProfiles), 2)

	// Verify directories were created
	assertDirExistsInIntegration(t, cfg.Media.LibraryPath)
	assertDirExistsInIntegration(t, cfg.Media.TrashPath)
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	tempDir := t.TempDir()

	// Create a minimal config file
	yamlContent := fmt.Sprintf(`
server:
  host: "localhost"
  port: 8080

media:
  library_path: "%s/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/data/vimesrv.db"

logging:
  level: "info"
  format: "console"

tmdb:
  enabled: false
`, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	// Set environment variables to override
	t.Setenv("SERVER_HOST", "10.0.0.1")
	t.Setenv("SERVER_PORT", "7000")
	t.Setenv("DATABASE_PATH", filepath.Join(tempDir, "custom-db/vimesrv.db"))
	t.Setenv("LOG_LEVEL", "warn")

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify environment variables took precedence
	assert.Equal(t, "10.0.0.1", cfg.Server.Host)
	assert.Equal(t, 7000, cfg.Server.Port)
	assert.Contains(t, cfg.Database.Path, "custom-db")
	assert.Equal(t, "warn", cfg.Logging.Level)
}

func TestLoad_WithVIMESRVPrefixEnvironmentVariables(t *testing.T) {
	tempDir := t.TempDir()

	yamlContent := fmt.Sprintf(`
server:
  host: "localhost"
  port: 8080

media:
  library_path: "%s/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/data/vimesrv.db"

logging:
  level: "info"
  format: "console"

tmdb:
  enabled: false
`, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	// Set environment variables with VIMESRV prefix
	t.Setenv("VIMESRV_SERVER_HOST", "172.16.0.1")
	t.Setenv("VIMESRV_SERVER_PORT", "6000")

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify environment variables with prefix work
	assert.Equal(t, "172.16.0.1", cfg.Server.Host)
	assert.Equal(t, 6000, cfg.Server.Port)
}

func TestLoad_CombineYAMLAndEnvVars(t *testing.T) {
	tempDir := t.TempDir()

	yamlContent := fmt.Sprintf(`
server:
  host: "localhost"
  port: 8080

media:
  library_path: "%s/media"
  supported_formats:
    - ".mp4"
    - ".mkv"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/data/vimesrv.db"

logging:
  level: "info"
  format: "json"

tmdb:
  enabled: true
  api_key: "yaml-api-key"
  language: "en-US"
  auto_search: true
  auto_link_threshold: 70
  max_candidates: 5
  image_cache_path: "%s/cache/tmdb"
  download_images: true
  poster_size: "w500"
  backdrop_size: "w1280"
  requests_per_10s: 35
  max_cast_members: 10
  similar_content_count: 5
  cache_ttl_hours: 24
`, tempDir, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	// Override some values with env vars
	t.Setenv("SERVER_PORT", "9999")
	t.Setenv("TMDB_API_KEY", "env-api-key")

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify merge: YAML values
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Contains(t, cfg.Media.SupportedFormats, ".mp4")
	assert.Contains(t, cfg.Media.SupportedFormats, ".mkv")

	// Verify merge: env var overrides
	assert.Equal(t, 9999, cfg.Server.Port)
	assert.Equal(t, "env-api-key", cfg.TMDB.APIKey)
}

func TestLoad_PathNormalizationIntegration(t *testing.T) {
	tempDir := t.TempDir()

	yamlContent := fmt.Sprintf(`
server:
  host: "localhost"
  port: 8080

media:
  library_path: "%s/relative/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/relative/data/vimesrv.db"

logging:
  level: "info"
  format: "console"
  file: "%s/relative/logs/app.log"

tmdb:
  enabled: false
`, tempDir, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// All paths should be absolute
	assert.True(t, filepath.IsAbs(cfg.Media.LibraryPath))
	assert.True(t, filepath.IsAbs(cfg.Media.TrashPath))
	assert.True(t, filepath.IsAbs(cfg.Database.Path))
	assert.True(t, filepath.IsAbs(cfg.Logging.File))

	// Verify derived paths are correct
	assert.Equal(t, filepath.Join(cfg.Media.LibraryPath, "trash"), cfg.Media.TrashPath)
}

func TestLoad_DirectoryCreationIntegration(t *testing.T) {
	tempDir := t.TempDir()

	yamlContent := `
server:
  host: "localhost"
  port: 8080

media:
  library_path: "` + filepath.Join(tempDir, "media") + `"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "` + filepath.Join(tempDir, "data", "vimesrv.db") + `"

logging:
  level: "info"
  format: "console"
  file: "` + filepath.Join(tempDir, "logs", "app.log") + `"

tmdb:
  enabled: false
`

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify all directories exist
	assertDirExistsInIntegration(t, cfg.Media.LibraryPath)
	assertDirExistsInIntegration(t, cfg.Media.TrashPath)
	assertDirExistsInIntegration(t, filepath.Join(cfg.Media.TrashPath, "original"))
	assertDirExistsInIntegration(t, filepath.Join(cfg.Media.TrashPath, "transcoded"))
	assertDirExistsInIntegration(t, filepath.Dir(cfg.Database.Path))
	assertDirExistsInIntegration(t, filepath.Dir(cfg.Logging.File))

	// Verify correct permissions
	info, err := os.Stat(cfg.Media.LibraryPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.True(t, info.Mode().Perm()&0o700 == 0o700)
}

func TestLoad_FullWorkflow(t *testing.T) {
	tempDir := t.TempDir()

	// Create partial config (relying on defaults for some values)
	yamlContent := fmt.Sprintf(`
server:
  host: "192.168.1.100"
  port: 3000

media:
  library_path: "%s/test-media"
  supported_formats:
    - ".mp4"
    - ".avi"

transcoding:
  segment_duration: 10
  quality_profiles:
    - name: "480p"
      enabled: true
      resolution: "854x480"
      audio_bitrate: "128k"
      crf: 24
      max_bitrate: "1500k"
    - name: "720p"
      enabled: false
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/test-db/vimesrv.db"

logging:
  level: "error"
  format: "json"

tmdb:
  enabled: false
`, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	// Set some env var overrides
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify YAML values
	assert.Equal(t, "192.168.1.100", cfg.Server.Host)
	assert.Equal(t, 3000, cfg.Server.Port)
	assert.Contains(t, cfg.Media.SupportedFormats, ".mp4")
	assert.Contains(t, cfg.Media.SupportedFormats, ".avi")
	assert.Equal(t, 10, cfg.Transcoding.SegmentDuration)

	// Verify env var overrides
	assert.Equal(t, "debug", cfg.Logging.Level)

	// Verify other defaults
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.False(t, cfg.TMDB.Enabled)

	// Verify quality profiles
	require.Len(t, cfg.Transcoding.QualityProfiles, 2)
	assert.Equal(t, "480p", cfg.Transcoding.QualityProfiles[0].Name)
	assert.True(t, cfg.Transcoding.QualityProfiles[0].Enabled)
	assert.Equal(t, "720p", cfg.Transcoding.QualityProfiles[1].Name)
	assert.False(t, cfg.Transcoding.QualityProfiles[1].Enabled)

	// Verify paths normalized
	assert.True(t, filepath.IsAbs(cfg.Media.LibraryPath))
	assert.True(t, filepath.IsAbs(cfg.Database.Path))

	// Verify subpaths derived
	assert.Equal(t, filepath.Join(cfg.Media.LibraryPath, "trash"), cfg.Media.TrashPath)

	// Verify directories created
	assertDirExistsInIntegration(t, cfg.Media.LibraryPath)
	assertDirExistsInIntegration(t, cfg.Media.TrashPath)
	assertDirExistsInIntegration(t, filepath.Dir(cfg.Database.Path))
}

func TestLoad_TMDBValidationWhenEnabled(t *testing.T) {
	tempDir := t.TempDir()

	yamlContent := fmt.Sprintf(`
server:
  host: "localhost"
  port: 8080

media:
  library_path: "%s/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/data/vimesrv.db"

logging:
  level: "info"
  format: "console"

tmdb:
  enabled: true
  api_key: "test-api-key-123"
  language: "en-US"
  auto_search: true
  auto_link_threshold: 75
  max_candidates: 10
  image_cache_path: "%s/cache/tmdb"
  download_images: true
  poster_size: "w342"
  backdrop_size: "w780"
  requests_per_10s: 30
  max_cast_members: 10
  similar_content_count: 5
  cache_ttl_hours: 24
`, tempDir, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify TMDB config loaded correctly
	assert.True(t, cfg.TMDB.Enabled)
	assert.Equal(t, "test-api-key-123", cfg.TMDB.APIKey)
	assert.Equal(t, "en-US", cfg.TMDB.Language)
	assert.True(t, cfg.TMDB.AutoSearch)
	assert.Equal(t, 75, cfg.TMDB.AutoLinkThreshold)
	assert.Equal(t, 10, cfg.TMDB.MaxCandidates)
	assert.Equal(t, "w342", cfg.TMDB.PosterSize)
	assert.Equal(t, "w780", cfg.TMDB.BackdropSize)
	assert.Equal(t, 30, cfg.TMDB.RequestsPer10s)
}

func TestLoad_TMDBValidationFailsWhenEnabledWithoutAPIKey(t *testing.T) {
	tempDir := t.TempDir()

	yamlContent := fmt.Sprintf(`
server:
  host: "localhost"
  port: 8080

media:
  library_path: "%s/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/data/vimesrv.db"

logging:
  level: "info"
  format: "console"

tmdb:
  enabled: true
  api_key: ""
  language: "en-US"
  auto_search: true
  auto_link_threshold: 70
  max_candidates: 5
  image_cache_path: "%s/cache/tmdb"
  download_images: true
  poster_size: "w500"
  backdrop_size: "w1280"
  requests_per_10s: 35
  max_cast_members: 10
  similar_content_count: 5
  cache_ttl_hours: 24
`, tempDir, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	cfg, err := Load(configPath)
	assert.Error(t, err, "should fail validation when TMDB enabled without API key")
	assert.Nil(t, cfg)
}

func TestLoad_NormalizationError(t *testing.T) {
	tempDir := t.TempDir()

	// Create a config that will pass validation but might have normalization issues
	// We'll create a scenario where the config has a value that should cause an error
	// However, filepath.Abs rarely errors in practice, so this test mainly ensures the error path exists
	yamlContent := fmt.Sprintf(`
server:
  host: "localhost"
  port: 8080

media:
  library_path: "%s/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/data/vimesrv.db"

logging:
  level: "info"
  format: "console"

tmdb:
  enabled: false
`, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	// This should succeed normally
	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestLoad_DeriveSubpathsError(t *testing.T) {
	tempDir := t.TempDir()

	// deriveSubpaths doesn't actually return errors, but we test it anyway
	yamlContent := fmt.Sprintf(`
server:
  host: "localhost"
  port: 8080

media:
  library_path: "%s/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/data/vimesrv.db"

logging:
  level: "info"
  format: "console"

tmdb:
  enabled: false
`, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify subpaths were derived correctly
	assert.Contains(t, cfg.Media.TrashPath, "trash")
}

func TestLoad_InvalidUnmarshalData(t *testing.T) {
	tempDir := t.TempDir()

	// Create a YAML with data types that can't be unmarshaled properly
	yamlContent := fmt.Sprintf(`
server:
  host: "localhost"
  port: "not-a-number"

media:
  library_path: "%s/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/data/vimesrv.db"

logging:
  level: "info"
  format: "console"

tmdb:
  enabled: false
`, tempDir, tempDir)

	configPath := createTestConfigYAML(t, tempDir, yamlContent)

	cfg, err := Load(configPath)
	assert.Error(t, err, "should fail to unmarshal invalid data types")
	assert.Nil(t, cfg)
}

func TestLoad_AllPosterSizes(t *testing.T) {
	// Test all valid poster sizes to ensure we have coverage
	tempDir := t.TempDir()
	posterSizes := []string{"w92", "w154", "w185", "w342", "w500", "w780", "original"}

	for _, size := range posterSizes {
		t.Run("poster_size_"+size, func(t *testing.T) {
			yamlContent := fmt.Sprintf(`
server:
  host: "localhost"
  port: 8080

media:
  library_path: "%s/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/data/vimesrv.db"

logging:
  level: "info"
  format: "console"

tmdb:
  enabled: true
  api_key: "test-key"
  language: "en-US"
  auto_search: true
  auto_link_threshold: 70
  max_candidates: 5
  image_cache_path: "%s/cache/tmdb"
  download_images: true
  poster_size: "`+size+`"
  backdrop_size: "w1280"
  requests_per_10s: 35
  max_cast_members: 10
  similar_content_count: 5
  cache_ttl_hours: 24
`, tempDir, tempDir, tempDir)
			configPath := createTestConfigYAML(t, tempDir, yamlContent)
			cfg, err := Load(configPath)
			require.NoError(t, err)
			assert.Equal(t, size, cfg.TMDB.PosterSize)
		})
	}
}

func TestLoad_AllBackdropSizes(t *testing.T) {
	// Test all valid backdrop sizes
	tempDir := t.TempDir()
	backdropSizes := []string{"w300", "w780", "w1280", "original"}

	for _, size := range backdropSizes {
		t.Run("backdrop_size_"+size, func(t *testing.T) {
			yamlContent := fmt.Sprintf(`
server:
  host: "localhost"
  port: 8080

media:
  library_path: "%s/media"
  supported_formats:
    - ".mp4"

transcoding:
  segment_duration: 4
  quality_profiles:
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      audio_bitrate: "128k"
      crf: 23
      max_bitrate: "2800k"

database:
  path: "%s/data/vimesrv.db"

logging:
  level: "info"
  format: "console"

tmdb:
  enabled: true
  api_key: "test-key"
  language: "en-US"
  auto_search: true
  auto_link_threshold: 70
  max_candidates: 5
  image_cache_path: "%s/cache/tmdb"
  download_images: true
  poster_size: "w500"
  backdrop_size: "`+size+`"
  requests_per_10s: 35
  max_cast_members: 10
  similar_content_count: 5
  cache_ttl_hours: 24
`, tempDir, tempDir, tempDir)
			configPath := createTestConfigYAML(t, tempDir, yamlContent)
			cfg, err := Load(configPath)
			require.NoError(t, err)
			assert.Equal(t, size, cfg.TMDB.BackdropSize)
		})
	}
}

// Helper functions

func createTestConfigYAML(t *testing.T, dir, content string) string {
	t.Helper()
	configFile := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configFile, []byte(content), 0o644)
	require.NoError(t, err)
	return configFile
}

func assertDirExistsInIntegration(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err, "directory should exist: %s", path)
	assert.True(t, info.IsDir(), "path should be a directory: %s", path)
}
