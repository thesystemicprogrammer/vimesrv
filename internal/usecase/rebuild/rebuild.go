package rebuild

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// RebuildUseCase orchestrates the database rebuild from a dump file
type RebuildUseCase struct {
	config              *config.Config
	rebuildRepository   ports.RebuildRepository
	transcodeRepository ports.TranscodeRepository
	mediaRepository     ports.MediaRepository
	jobRepository       ports.JobRepository
	filesystem          ports.FileSystemService
	autoLinkMap         map[string]MediaLink // fingerprint -> TMDB link (populated during rebuild)
}

// NewRebuildUseCase creates a new rebuild use case
func NewRebuildUseCase(
	cfg *config.Config,
	rebuildRepository ports.RebuildRepository,
	transcodeRepository ports.TranscodeRepository,
	mediaRepository ports.MediaRepository,
	jobRepository ports.JobRepository,
	filesystem ports.FileSystemService,
) *RebuildUseCase {
	return &RebuildUseCase{
		config:              cfg,
		rebuildRepository:   rebuildRepository,
		transcodeRepository: transcodeRepository,
		mediaRepository:     mediaRepository,
		jobRepository:       jobRepository,
		filesystem:          filesystem,
	}
}

// RebuildResult contains statistics about the rebuild operation
type RebuildResult struct {
	UsersImported       int
	MediaLinksLoaded    int
	FilesScanned        int
	FilesProcessed      int
	FilesLinked         int
	TranscodesRecovered int
	Errors              []RebuildError
}

// RebuildFileName is the name of the rebuild export file
const RebuildFileName = "rebuild.json"

// RebuildDoneFileName is the name after successful rebuild
const RebuildDoneFileName = "rebuild.json.done"

// Execute performs the database rebuild from rebuild.json
func (uc *RebuildUseCase) Execute(ctx context.Context) (*RebuildResult, error) {
	logger.Info().Msg("[rebuild] Starting database rebuild from dump")

	// Step 1: Validate config allows rebuild
	if !uc.config.Rebuild.AllowRebuild {
		return nil, fmt.Errorf("rebuild is disabled in configuration; set rebuild.allow_rebuild: true to enable")
	}

	result := &RebuildResult{}

	// Step 2: Read and parse rebuild.json
	rebuildPath := filepath.Join(uc.config.Media.LibraryPath, RebuildFileName)
	if !uc.filesystem.FileExists(rebuildPath) {
		return nil, fmt.Errorf("rebuild.json not found at %s; run --prepare-rebuild first", rebuildPath)
	}

	jsonData, err := uc.filesystem.ReadFile(rebuildPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rebuild.json: %w", err)
	}

	var data RebuildData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse rebuild.json: %w", err)
	}

	// Validate version
	if data.Version != RebuildDataVersion {
		return nil, fmt.Errorf("unsupported rebuild.json version %d (expected %d)", data.Version, RebuildDataVersion)
	}

	logger.Info().
		Int("users", len(data.Users)).
		Int("media_links", len(data.MediaLinks)).
		Time("created_at", data.CreatedAt).
		Msg("[rebuild] Loaded rebuild.json")

	// Step 3: Clear all tables
	logger.Warn().Msg("[rebuild] Clearing all database tables...")
	if err := uc.rebuildRepository.ClearAllTables(ctx); err != nil {
		return nil, fmt.Errorf("failed to clear database: %w", err)
	}
	logger.Info().Msg("[rebuild] Database cleared successfully")

	// Step 4: Import users
	logger.Info().Msg("[rebuild] Importing users...")
	for _, userData := range data.Users {
		exportedUser := ports.ExportedUser{
			ID:                 userData.ID,
			Username:           userData.Username,
			PasswordHash:       userData.PasswordHash,
			Role:               userData.Role,
			MustChangePassword: userData.MustChangePassword,
			CreatedAt:          userData.CreatedAt.Format(time.RFC3339),
			UpdatedAt:          userData.UpdatedAt.Format(time.RFC3339),
			CreatedBy:          userData.CreatedBy,
		}

		if err := uc.rebuildRepository.ImportUser(ctx, exportedUser); err != nil {
			logger.Error().Err(err).Str("username", userData.Username).Msg("[rebuild] Failed to import user")
			result.Errors = append(result.Errors, RebuildError{
				Operation: "import_user",
				Error:     err.Error(),
			})
			continue
		}
		result.UsersImported++
	}
	logger.Info().Int("count", result.UsersImported).Msg("[rebuild] Users imported")

	// Step 5: Build auto-link map for library scan
	uc.autoLinkMap = make(map[string]MediaLink)
	for _, link := range data.MediaLinks {
		uc.autoLinkMap[link.Fingerprint] = link
	}
	result.MediaLinksLoaded = len(uc.autoLinkMap)
	logger.Info().Int("count", result.MediaLinksLoaded).Msg("[rebuild] Auto-link map populated")

	// Step 6: Write errors file if any occurred
	if len(result.Errors) > 0 {
		uc.writeErrorsFile(result.Errors)
	}

	logger.Info().
		Int("users_imported", result.UsersImported).
		Int("media_links_loaded", result.MediaLinksLoaded).
		Int("errors", len(result.Errors)).
		Msg("[rebuild] Rebuild phase 1 complete. Library scan will complete the process.")

	return result, nil
}

// GetAutoLinkMap returns the fingerprint -> TMDB link map for use during library scan
// This should be called after Execute() to get the map for auto-linking
func (uc *RebuildUseCase) GetAutoLinkMap() map[string]MediaLink {
	return uc.autoLinkMap
}

// ExecuteFullRebuild performs a complete database rebuild including:
// 1. Clear database and import users (Execute)
// 2. Scan media library and auto-link files
// 3. Recover transcode records
// 4. Rename rebuild.json to rebuild.json.done
func (uc *RebuildUseCase) ExecuteFullRebuild(ctx context.Context, scanner *Scanner) (*RebuildResult, error) {
	// Step 1: Run the base rebuild (clear DB, import users, build auto-link map)
	result, err := uc.Execute(ctx)
	if err != nil {
		return nil, err
	}

	// Step 2: Scan the media library
	logger.Info().Msg("[rebuild] Starting media library scan...")
	scanResult, err := scanner.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("library scan failed: %w", err)
	}

	result.FilesScanned = scanResult.FilesScanned
	result.FilesProcessed = scanResult.FilesProcessed
	result.FilesLinked = scanResult.FilesLinked
	result.Errors = append(result.Errors, scanResult.Errors...)

	logger.Info().
		Int("scanned", result.FilesScanned).
		Int("processed", result.FilesProcessed).
		Int("linked", result.FilesLinked).
		Msg("[rebuild] Library scan complete")

	// Step 3: Recover transcode records
	transcodesRecovered, err := uc.RecoverTranscodes(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("[rebuild] Transcode recovery failed, continuing...")
		result.Errors = append(result.Errors, RebuildError{
			Operation: "recover_transcodes",
			Error:     err.Error(),
		})
	} else {
		result.TranscodesRecovered = transcodesRecovered
	}

	// Step 4: Write final errors file if any
	if len(result.Errors) > 0 {
		uc.writeErrorsFile(result.Errors)
	}

	// Step 5: Rename rebuild.json to rebuild.json.done
	rebuildPath := filepath.Join(uc.config.Media.LibraryPath, RebuildFileName)
	donePath := filepath.Join(uc.config.Media.LibraryPath, RebuildDoneFileName)
	if err := uc.filesystem.Rename(rebuildPath, donePath); err != nil {
		logger.Warn().Err(err).Msg("[rebuild] Failed to rename rebuild.json to .done")
	} else {
		logger.Info().Str("path", donePath).Msg("[rebuild] Renamed rebuild.json to rebuild.json.done")
	}

	logger.Info().
		Int("users_imported", result.UsersImported).
		Int("files_processed", result.FilesProcessed).
		Int("files_linked", result.FilesLinked).
		Int("transcodes_recovered", result.TranscodesRecovered).
		Int("errors", len(result.Errors)).
		Msg("[rebuild] Database rebuild complete")

	return result, nil
}

// RecoverTranscodes scans media directories and recreates transcode records
// for files that already exist on disk. This should be called after library scan completes.
func (uc *RebuildUseCase) RecoverTranscodes(ctx context.Context) (int, error) {
	logger.Info().Msg("[rebuild] Starting transcode recovery...")

	mediaPath := uc.config.Media.MediaPath
	recovered := 0

	// List all media directories
	entries, err := uc.filesystem.ReadDir(mediaPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read media directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		mediaID := entry.Name()
		transcodedDir := filepath.Join(mediaPath, mediaID, "transcoded")

		if !uc.filesystem.FileExists(transcodedDir) {
			continue
		}

		// Check if media exists in database (should exist after library scan)
		media, err := uc.mediaRepository.Get(ctx, mediaID)
		if err != nil || media == nil {
			logger.Warn().Str("media_id", mediaID).Msg("[rebuild] Transcode directory found but media not in database")
			continue
		}

		// Recover transcodes for this media
		count, err := uc.recoverTranscodesForMedia(ctx, mediaID, transcodedDir)
		if err != nil {
			logger.Error().Err(err).Str("media_id", mediaID).Msg("[rebuild] Failed to recover transcodes")
			continue
		}

		recovered += count
	}

	logger.Info().Int("recovered", recovered).Msg("[rebuild] Transcode recovery complete")
	return recovered, nil
}

// recoverTranscodesForMedia scans a single media's transcoded directory and creates records
func (uc *RebuildUseCase) recoverTranscodesForMedia(ctx context.Context, mediaID, transcodedDir string) (int, error) {
	recovered := 0

	// Scan for video quality directories (360p/, 720p/, etc.)
	entries, err := uc.filesystem.ReadDir(transcodedDir)
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check if this is a video quality directory (e.g., "360p", "720p")
		if isVideoQualityDir(name) {
			if uc.hasValidTranscode(filepath.Join(transcodedDir, name)) {
				transcode := uc.createTranscodeRecord(mediaID, name, domain.TrackTypeVideo, 0, filepath.Join(transcodedDir, name))
				if err := uc.transcodeRepository.Create(ctx, transcode); err != nil {
					logger.Warn().Err(err).Str("media_id", mediaID).Str("quality", name).Msg("[rebuild] Failed to create video transcode record")
				} else {
					recovered++
				}
			}
		}

		// Check if this is an audio directory (e.g., "audio-0", "audio-1")
		if idx, ok := parseAudioDir(name); ok {
			if uc.hasValidTranscode(filepath.Join(transcodedDir, name)) {
				transcode := uc.createTranscodeRecord(mediaID, "", domain.TrackTypeAudio, idx, filepath.Join(transcodedDir, name))
				if err := uc.transcodeRepository.Create(ctx, transcode); err != nil {
					logger.Warn().Err(err).Str("media_id", mediaID).Int("track", idx).Msg("[rebuild] Failed to create audio transcode record")
				} else {
					recovered++
				}
			}
		}
	}

	// Scan for subtitle files (subtitle-0.vtt, subtitle-1.vtt, etc.)
	files, err := uc.filesystem.ListFiles(transcodedDir, "*.vtt")
	if err == nil {
		for _, f := range files {
			if idx, ok := parseSubtitleFile(filepath.Base(f)); ok {
				transcode := uc.createTranscodeRecord(mediaID, "", domain.TrackTypeSubtitle, idx, f)
				if err := uc.transcodeRepository.Create(ctx, transcode); err != nil {
					logger.Warn().Err(err).Str("media_id", mediaID).Int("track", idx).Msg("[rebuild] Failed to create subtitle transcode record")
				} else {
					recovered++
				}
			}
		}
	}

	return recovered, nil
}

// hasValidTranscode checks if a transcode directory contains valid output files
func (uc *RebuildUseCase) hasValidTranscode(dir string) bool {
	// Check for init.mp4 or segment files
	initPath := filepath.Join(dir, "init.mp4")
	if uc.filesystem.FileExists(initPath) {
		return true
	}

	// Check for any .m4s segment files
	segments, err := uc.filesystem.ListFiles(dir, "*.m4s")
	if err == nil && len(segments) > 0 {
		return true
	}

	return false
}

// createTranscodeRecord creates a completed transcode record
func (uc *RebuildUseCase) createTranscodeRecord(mediaID, quality string, trackType domain.TrackType, trackIndex int, outputPath string) *domain.Transcode {
	id := fmt.Sprintf("%s-%s-%s-%d", mediaID, trackType, quality, trackIndex)
	transcode := domain.NewTranscode(id, mediaID, quality, trackType, trackIndex)
	transcode.MarkCompleted(outputPath)
	return transcode
}

// writeErrorsFile writes any errors that occurred during rebuild to a file
func (uc *RebuildUseCase) writeErrorsFile(errors []RebuildError) {
	errorsData := RebuildErrors{
		CreatedAt: time.Now().UTC(),
		Errors:    errors,
	}

	jsonData, err := json.MarshalIndent(errorsData, "", "  ")
	if err != nil {
		logger.Error().Err(err).Msg("[rebuild] Failed to marshal errors file")
		return
	}

	errorsPath := filepath.Join(uc.config.Media.LibraryPath, "rebuild-errors.json")
	if err := uc.filesystem.WriteFile(errorsPath, jsonData); err != nil {
		logger.Error().Err(err).Msg("[rebuild] Failed to write errors file")
	} else {
		logger.Warn().Str("path", errorsPath).Int("count", len(errors)).Msg("[rebuild] Errors written to file")
	}
}

// isVideoQualityDir checks if a directory name is a video quality (e.g., "360p", "720p")
func isVideoQualityDir(name string) bool {
	validQualities := []string{"360p", "480p", "720p", "1080p", "1440p", "2160p"}
	for _, q := range validQualities {
		if name == q {
			return true
		}
	}
	return false
}

// parseAudioDir extracts the track index from an audio directory name (e.g., "audio-0" -> 0)
func parseAudioDir(name string) (int, bool) {
	var idx int
	if _, err := fmt.Sscanf(name, "audio-%d", &idx); err == nil {
		return idx, true
	}
	return 0, false
}

// parseSubtitleFile extracts the track index from a subtitle filename (e.g., "subtitle-0.vtt" -> 0)
func parseSubtitleFile(name string) (int, bool) {
	var idx int
	if _, err := fmt.Sscanf(name, "subtitle-%d.vtt", &idx); err == nil {
		return idx, true
	}
	return 0, false
}
