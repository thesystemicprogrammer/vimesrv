package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ScanResult contains statistics about the scan operation
type ScanResult struct {
	Processed  int // Total files processed
	Imported   int // Successfully imported
	Skipped    int // Skipped (invalid or unsupported)
	Duplicates int // Duplicate files (already in library)
	Failed     int // Failed to process
}

type ScanLibraryUseCase struct {
	config            config.MediaConfig
	fileHasher        ports.FileHasher
	ffprobeService    ports.FFProbeService
	fileSystemService ports.FileSystemService
	mediaRepository   ports.MediaRepository
}

func NewScanLibraryUseCase(
	config config.MediaConfig,
	fileHasher ports.FileHasher,
	ffprobeService ports.FFProbeService,
	fileSystemService ports.FileSystemService,
	mediaRepository ports.MediaRepository,
) *ScanLibraryUseCase {
	return &ScanLibraryUseCase{
		config:            config,
		fileHasher:        fileHasher,
		ffprobeService:    ffprobeService,
		fileSystemService: fileSystemService,
		mediaRepository:   mediaRepository,
	}
}

// Execute scans the staging directory and imports video files into the library
func (uc *ScanLibraryUseCase) Execute(ctx context.Context) error {
	logger.Info().Str("staging_path", uc.config.StagingPath).Msg("Starting library scan")

	// Validate staging path exists
	if !uc.fileSystemService.FileExists(uc.config.StagingPath) {
		return fmt.Errorf("staging path does not exist: %s", uc.config.StagingPath)
	}

	result := &ScanResult{}

	// Walk staging directory and process files
	err := uc.fileSystemService.WalkDir(uc.config.StagingPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Warn().Err(err).Str("path", path).Msg("Error accessing path")
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Process the file
		if err := uc.processFile(ctx, path, result); err != nil {
			logger.Error().Err(err).Str("path", path).Msg("Failed to process file")
			result.Failed++
		}

		result.Processed++
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk staging directory: %w", err)
	}

	// Clean up empty directories in staging
	if err := uc.fileSystemService.RemoveEmptyDirs(uc.config.StagingPath); err != nil {
		logger.Warn().Err(err).Msg("Failed to remove empty directories")
	}

	logger.Info().
		Int("processed", result.Processed).
		Int("imported", result.Imported).
		Int("skipped", result.Skipped).
		Int("duplicates", result.Duplicates).
		Int("failed", result.Failed).
		Msg("Library scan completed")

	return nil
}

// processFile processes a single file from the staging directory
func (uc *ScanLibraryUseCase) processFile(ctx context.Context, filePath string, result *ScanResult) error {
	logger.Debug().Str("file", filePath).Msg("Processing file")

	// Check if file has supported extension
	if !uc.isSupportedFormat(filePath) {
		logger.Debug().Str("file", filePath).Msg("Skipping unsupported file format")
		result.Skipped++
		return nil
	}

	// Validate video with ffprobe
	valid, err := uc.ffprobeService.ValidateVideo(filePath)
	if err != nil {
		logger.Warn().Err(err).Str("file", filePath).Msg("Failed to validate video")
		result.Skipped++
		uc.deleteFile(filePath)
		return nil
	}

	if !valid {
		logger.Warn().Str("file", filePath).Msg("Invalid video file, skipping")
		result.Skipped++
		uc.deleteFile(filePath)
		return nil
	}

	// Generate fingerprint
	fingerprint, err := uc.fileHasher.HashFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to generate fingerprint: %w", err)
	}

	logger.Debug().Str("file", filePath).Str("fingerprint", fingerprint).Msg("Generated fingerprint")

	// Check for duplicate
	exists, err := uc.mediaRepository.ExistsByFingerprint(ctx, fingerprint)
	if err != nil {
		return fmt.Errorf("failed to check for duplicate: %w", err)
	}

	if exists {
		logger.Warn().
			Str("file", filePath).
			Str("fingerprint", fingerprint).
			Msg("Duplicate file detected, skipping")
		result.Duplicates++
		uc.deleteFile(filePath)
		return nil
	}

	// Extract metadata
	metadata, err := uc.ffprobeService.ExtractMetadata(filePath)
	if err != nil {
		return fmt.Errorf("failed to extract metadata: %w", err)
	}

	// Get original filename
	originalFilename := filepath.Base(filePath)

	// Generate UUID for the media file
	id := uuid.New().String()

	// Create target directory: {media_path}/{uuid}/
	targetDir := filepath.Join(uc.config.MediaPath, id)
	if err := uc.fileSystemService.CreateDir(targetDir); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Target file path
	targetPath := filepath.Join(targetDir, originalFilename)

	// Copy file to library
	logger.Info().
		Str("src", filePath).
		Str("dst", targetPath).
		Msg("Copying file to library")

	if err := uc.fileSystemService.CopyFile(filePath, targetPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Create media file record
	mediaFile := domain.NewMediaFile(
		id,
		fingerprint,
		targetPath,
		originalFilename,
		originalFilename,
	)

	// Populate metadata
	mediaFile.Duration = metadata.Duration
	mediaFile.FileSize = metadata.FileSize
	mediaFile.Format = metadata.Format
	mediaFile.VideoCodec = metadata.VideoCodec
	mediaFile.AudioCodecs = metadata.AudioCodecs
	mediaFile.Resolution = metadata.Resolution
	mediaFile.Width = metadata.Width
	mediaFile.Height = metadata.Height
	mediaFile.Bitrate = metadata.Bitrate
	mediaFile.AudioTracks = metadata.AudioTracks
	mediaFile.SubtitleTracks = metadata.SubtitleTracks
	mediaFile.SubtitleLanguages = metadata.SubtitleLanguages

	// Insert into database
	if err := uc.mediaRepository.Create(ctx, mediaFile); err != nil {
		// Rollback: delete copied file
		logger.Error().Err(err).Str("file", targetPath).Msg("Failed to insert into database, rolling back")
		if deleteErr := uc.fileSystemService.DeleteFile(targetPath); deleteErr != nil {
			logger.Error().Err(deleteErr).Str("file", targetPath).Msg("Failed to delete file during rollback")
		}
		return fmt.Errorf("failed to insert into database: %w", err)
	}

	// Delete from staging
	if err := uc.fileSystemService.DeleteFile(filePath); err != nil {
		logger.Warn().Err(err).Str("file", filePath).Msg("Failed to delete file from staging")
		// Don't return error, file was imported successfully
	}

	logger.Info().
		Str("file", originalFilename).
		Str("fingerprint", fingerprint).
		Str("target", targetPath).
		Msg("File imported successfully")

	result.Imported++
	return nil
}

// isSupportedFormat checks if the file extension is in the supported formats list
func (uc *ScanLibraryUseCase) isSupportedFormat(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, supportedExt := range uc.config.SupportedFormats {
		if ext == strings.ToLower(supportedExt) {
			return true
		}
	}
	return false
}

// deleteFile safely deletes a file from staging, logging errors but not failing
func (uc *ScanLibraryUseCase) deleteFile(filePath string) {
	if err := uc.fileSystemService.DeleteFile(filePath); err != nil {
		logger.Warn().Err(err).Str("file", filePath).Msg("Failed to delete file from staging")
	} else {
		logger.Debug().Str("file", filePath).Msg("Deleted file from staging")
	}
}
