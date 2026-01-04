package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/transcode"
)

// ScanResult contains statistics about the scan operation
type ScanResult struct {
	Processed  int // Total files processed
	Imported   int // Successfully imported
	Skipped    int // Skipped (invalid or unsupported)
	Duplicates int // Duplicate files (already in library)
	Failed     int // Failed to process
}

// ScanLibraryInput contains parameters for library scanning with progress reporting
type ScanLibraryInput struct {
	JobID int64 // Job ID for progress notifications (0 = no notifications)
}

// fileInfo holds information about a file to be processed
type fileInfo struct {
	path string
	size int64
}

// EnrichMetadataJobPayload is the payload for enrich_metadata jobs
type EnrichMetadataJobPayload struct {
	MediaID  string `json:"media_id"`
	Filename string `json:"filename"`
}

type ScanLibraryUseCase struct {
	config                     config.MediaConfig
	tmdbConfig                 config.TMDBConfig
	fileHasher                 ports.FileHasher
	ffprobeService             ports.FFProbeService
	fileSystemService          ports.FileSystemService
	mediaRepository            ports.MediaRepository
	enqueueJobUseCase          *job.EnqueueJobUseCase
	jobNotifier                ports.JobNotifier
	createTranscodeJobsUseCase *transcode.CreateTranscodeJobsUseCase
}

func NewScanLibraryUseCase(
	config config.MediaConfig,
	fileHasher ports.FileHasher,
	ffprobeService ports.FFProbeService,
	fileSystemService ports.FileSystemService,
	mediaRepository ports.MediaRepository,
	createTranscodeJobsUseCase *transcode.CreateTranscodeJobsUseCase,
	jobNotifier ports.JobNotifier,
) *ScanLibraryUseCase {
	return &ScanLibraryUseCase{
		config:                     config,
		fileHasher:                 fileHasher,
		ffprobeService:             ffprobeService,
		fileSystemService:          fileSystemService,
		mediaRepository:            mediaRepository,
		createTranscodeJobsUseCase: createTranscodeJobsUseCase,
		jobNotifier:                jobNotifier,
	}
}

// WithEnrichment adds enrichment job capability to the use case
func (uc *ScanLibraryUseCase) WithEnrichment(tmdbConfig config.TMDBConfig, enqueueJobUseCase *job.EnqueueJobUseCase) *ScanLibraryUseCase {
	uc.tmdbConfig = tmdbConfig
	uc.enqueueJobUseCase = enqueueJobUseCase
	return uc
}

// Execute scans the staging directory and imports video files into the library.
// This is a backward-compatible method that calls ExecuteWithProgress without job context.
func (uc *ScanLibraryUseCase) Execute(ctx context.Context) error {
	return uc.ExecuteWithProgress(ctx, ScanLibraryInput{JobID: 0})
}

// ExecuteWithProgress scans the staging directory and imports video files into the library.
// When JobID > 0 and jobNotifier is set, it sends progress notifications via WebSocket.
func (uc *ScanLibraryUseCase) ExecuteWithProgress(ctx context.Context, input ScanLibraryInput) error {
	logger.Info().Str("staging_path", uc.config.StagingPath).Msg("Starting library scan")

	// Validate staging path exists
	if !uc.fileSystemService.FileExists(uc.config.StagingPath) {
		return fmt.Errorf("staging path does not exist: %s", uc.config.StagingPath)
	}

	// Collect all supported files first to know total count and size
	files, totalSize := uc.collectSupportedFiles()
	totalFiles := len(files)

	if totalFiles == 0 {
		logger.Info().Msg("No files to process in staging directory")
		return nil
	}

	logger.Info().
		Int("total_files", totalFiles).
		Int64("total_size_bytes", totalSize).
		Msg("Collected files for processing")

	result := &ScanResult{}
	var processedSize int64

	// Process each file with progress tracking
	for i, file := range files {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fileIndex := i + 1
		if err := uc.processFileWithProgress(ctx, file, result, fileIndex, totalFiles, &processedSize, totalSize, input.JobID); err != nil {
			logger.Error().Err(err).Str("path", file.path).Msg("Failed to process file")
			result.Failed++
		}
		result.Processed++
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

// collectSupportedFiles walks the staging directory and returns all supported files with their sizes
func (uc *ScanLibraryUseCase) collectSupportedFiles() ([]fileInfo, int64) {
	var files []fileInfo
	var totalSize int64

	_ = uc.fileSystemService.WalkDir(uc.config.StagingPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Warn().Err(err).Str("path", path).Msg("Error accessing path during collection")
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only include supported formats
		if uc.isSupportedFormat(path) {
			size := info.Size()
			files = append(files, fileInfo{path: path, size: size})
			totalSize += size
		}

		return nil
	})

	return files, totalSize
}

// processFileWithProgress processes a single file with progress reporting
func (uc *ScanLibraryUseCase) processFileWithProgress(
	ctx context.Context,
	file fileInfo,
	result *ScanResult,
	fileIndex, totalFiles int,
	processedSize *int64,
	totalSize int64,
	jobID int64,
) error {
	filePath := file.path
	filename := filepath.Base(filePath)

	logger.Debug().Str("file", filePath).Msg("Processing file")

	// Report analyzing phase (indeterminate)
	uc.notifyProgress(jobID, "Analyzing", fileIndex, totalFiles, filename, -1)

	// Validate video with ffprobe
	valid, err := uc.ffprobeService.ValidateVideo(filePath)
	if err != nil {
		logger.Warn().Err(err).Str("file", filePath).Msg("Failed to validate video")
		result.Skipped++
		*processedSize += file.size
		uc.deleteFile(filePath)
		return nil
	}

	if !valid {
		logger.Warn().Str("file", filePath).Msg("Invalid video file, skipping")
		result.Skipped++
		*processedSize += file.size
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
		*processedSize += file.size
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

	// Generate deterministic ID from fingerprint
	id := domain.DeriveIDFromFingerprint(fingerprint)

	// Create target directory: {media_path}/{uuid}/
	targetDir := filepath.Join(uc.config.MediaPath, id)
	if err := uc.fileSystemService.CreateDir(targetDir); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Target file path
	targetPath := filepath.Join(targetDir, originalFilename)

	// Copy file to library with progress callback
	logger.Info().
		Str("src", filePath).
		Str("dst", targetPath).
		Msg("Copying file to library")

	// Create progress callback for large file copy
	var copyCallback ports.CopyProgressCallback
	if jobID > 0 && uc.jobNotifier != nil {
		startProcessedSize := *processedSize
		copyCallback = func(written, total int64, filePercent float64) {
			// Calculate overall percentage across all files
			overallPercent := float64(startProcessedSize+written) / float64(totalSize) * 100
			uc.notifyProgress(jobID, "Copying", fileIndex, totalFiles, filename, overallPercent)
		}
	}

	if err := uc.fileSystemService.CopyFileWithProgress(filePath, targetPath, copyCallback); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Update processed size after successful copy
	*processedSize += file.size

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

	// Create transcode jobs for the imported media file
	if uc.createTranscodeJobsUseCase != nil {
		logger.Info().Str("media_id", id).Msg("Creating transcode jobs for imported media")
		transcodeOutput, err := uc.createTranscodeJobsUseCase.Execute(ctx, transcode.CreateTranscodeJobsInput{
			MediaID: id,
		})
		if err != nil {
			logger.Error().Err(err).Str("media_id", id).Msg("Failed to create transcode jobs")
			// Don't fail the import - just log the error
		} else {
			logger.Info().
				Str("media_id", id).
				Int("total_jobs", transcodeOutput.TotalJobs).
				Int("video_jobs", transcodeOutput.VideoJobs).
				Int("audio_jobs", transcodeOutput.AudioJobs).
				Int("subtitle_jobs", transcodeOutput.SubtitleJobs).
				Msg("Transcode jobs created successfully")
		}
	}

	// Enqueue metadata enrichment job if TMDB is enabled
	if uc.tmdbConfig.Enabled && uc.tmdbConfig.AutoSearch && uc.enqueueJobUseCase != nil {
		if err := uc.enqueueEnrichmentJob(ctx, id, originalFilename); err != nil {
			logger.Error().Err(err).Str("media_id", id).Msg("Failed to enqueue enrichment job")
			// Don't fail the import - just log the error
		} else {
			logger.Info().Str("media_id", id).Msg("Enrichment job enqueued")
		}
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

// notifyProgress sends a progress notification if jobNotifier is configured
// percentage < 0 indicates indeterminate progress (analyzing phase)
func (uc *ScanLibraryUseCase) notifyProgress(jobID int64, action string, fileIndex, totalFiles int, filename string, percentage float64) {
	if jobID <= 0 || uc.jobNotifier == nil {
		return
	}

	var msg string
	if totalFiles == 1 {
		msg = fmt.Sprintf("%s: %s", action, filename)
	} else {
		msg = fmt.Sprintf("%s file %d/%d: %s", action, fileIndex, totalFiles, filename)
	}

	progress := ports.JobProgress{
		Message: msg,
	}

	// Only set percentage if >= 0 (determinate progress)
	if percentage >= 0 {
		progress.Percentage = percentage
	}

	uc.jobNotifier.NotifyJobProgress(jobID, shared.JobTypeScanLibrary, progress)
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

// enqueueEnrichmentJob enqueues a metadata enrichment job for the given media file
func (uc *ScanLibraryUseCase) enqueueEnrichmentJob(ctx context.Context, mediaID string, filename string) error {
	payload := EnrichMetadataJobPayload{
		MediaID:  mediaID,
		Filename: filename,
	}

	_, err := uc.enqueueJobUseCase.Execute(ctx, job.EnqueueJobInput{
		Type:        shared.JobTypeEnrichMetadata,
		Payload:     payload,
		Priority:    shared.JobPriorityEnrichMetadata,
		MaxAttempts: 3, // Enrichment jobs can retry a few times
	})
	if err != nil {
		return fmt.Errorf("failed to enqueue enrichment job: %w", err)
	}

	return nil
}
