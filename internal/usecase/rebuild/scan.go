package rebuild

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ScanResult contains statistics about the rebuild scan operation
type ScanResult struct {
	FilesScanned   int
	FilesProcessed int
	FilesLinked    int
	FilesSkipped   int
	Errors         []RebuildError
}

// Scanner handles scanning the media library during rebuild
type Scanner struct {
	config          config.MediaConfig
	fileHasher      ports.FileHasher
	ffprobeService  ports.FFProbeService
	filesystem      ports.FileSystemService
	mediaRepository ports.MediaRepository
	linker          *Linker
	autoLinkMap     map[string]AutoLinkData
}

// NewScanner creates a new rebuild scanner
func NewScanner(
	mediaCfg config.MediaConfig,
	fileHasher ports.FileHasher,
	ffprobeService ports.FFProbeService,
	filesystem ports.FileSystemService,
	mediaRepository ports.MediaRepository,
	linker *Linker,
	autoLinkMap map[string]AutoLinkData,
) *Scanner {
	return &Scanner{
		config:          mediaCfg,
		fileHasher:      fileHasher,
		ffprobeService:  ffprobeService,
		filesystem:      filesystem,
		mediaRepository: mediaRepository,
		linker:          linker,
		autoLinkMap:     autoLinkMap,
	}
}

// Scan walks the media_path directory and processes each video file
// For each file: hash -> derive ID -> FFProbe -> create record -> lookup auto-link -> call linker
func (s *Scanner) Scan(ctx context.Context) (*ScanResult, error) {
	result := &ScanResult{}

	// First, count total files for progress logging
	var filesToProcess []string
	err := s.filesystem.WalkDir(s.config.MediaPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		if info.IsDir() {
			// Skip transcoded directories - they contain segment files, not source media
			if info.Name() == "transcoded" {
				return filepath.SkipDir
			}
			return nil
		}
		if s.isSupportedFormat(path) {
			filesToProcess = append(filesToProcess, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	totalFiles := len(filesToProcess)
	logger.Info().Int("total_files", totalFiles).Msg("[rebuild] Starting library scan")

	for i, filePath := range filesToProcess {
		result.FilesScanned++

		// Process the file
		err := s.processFile(ctx, filePath, result)
		if err != nil {
			logger.Error().Err(err).Str("file", filePath).Msg("[rebuild] Failed to process file")
			result.Errors = append(result.Errors, RebuildError{
				Filename:  filepath.Base(filePath),
				Operation: "process_file",
				Error:     err.Error(),
			})
			continue
		}

		// Progress logging every 10 files or at the end
		if (i+1)%10 == 0 || i+1 == totalFiles {
			logger.Info().
				Int("processed", i+1).
				Int("total", totalFiles).
				Int("linked", result.FilesLinked).
				Msg("[rebuild] Scan progress")
		}
	}

	return result, nil
}

// processFile handles a single video file during rebuild
func (s *Scanner) processFile(ctx context.Context, filePath string, result *ScanResult) error {
	// Step 1: Hash the file to get fingerprint
	fingerprint, err := s.fileHasher.HashFile(filePath)
	if err != nil {
		return err
	}

	// Step 2: Derive deterministic ID from fingerprint
	mediaID := domain.DeriveIDFromFingerprint(fingerprint)

	// Step 3: Check if media already exists (skip if so)
	existing, err := s.mediaRepository.Get(ctx, mediaID)
	if err == nil && existing != nil {
		logger.Debug().Str("media_id", mediaID).Msg("[rebuild] Media already exists, skipping")
		result.FilesSkipped++
		return nil
	}

	// Step 4: Get file metadata via FFProbe
	fileInfo, err := s.ffprobeService.ExtractMetadata(filePath)
	if err != nil {
		return err
	}

	// Step 5: Create media file record
	filename := filepath.Base(filePath)
	media := domain.NewMediaFile(mediaID, fingerprint, filePath, filename, filename)
	media.Duration = int(fileInfo.Duration)
	media.Format = fileInfo.Format
	media.VideoCodec = fileInfo.VideoCodec
	media.Resolution = fileInfo.Resolution
	media.Width = fileInfo.Width
	media.Height = fileInfo.Height
	media.Bitrate = fileInfo.Bitrate
	media.AudioTracks = fileInfo.AudioTracks
	media.SubtitleTracks = fileInfo.SubtitleTracks

	// Get file size
	size, err := s.filesystem.GetFileSize(filePath)
	if err == nil {
		media.FileSize = size
	}

	// Step 6: Check auto-link map for pre-existing metadata link
	autoLink, hasAutoLink := s.autoLinkMap[fingerprint]
	if hasAutoLink {
		media.Edition = autoLink.Edition
	}

	// Step 7: Save media file record
	if err := s.mediaRepository.Create(ctx, media); err != nil {
		return err
	}
	result.FilesProcessed++

	// Step 8: If we have auto-link data, call the linker
	if hasAutoLink && s.linker != nil {
		err := s.linker.Link(ctx, media, autoLink)
		if err != nil {
			// Log warning but continue - the file is still in the database
			logger.Warn().
				Err(err).
				Str("media_id", mediaID).
				Str("metadata_type", autoLink.MetadataType).
				Msg("[rebuild] Failed to auto-link, file will need manual enrichment")
			result.Errors = append(result.Errors, RebuildError{
				Fingerprint: fingerprint,
				Filename:    filename,
				Operation:   "auto_link",
				Error:       err.Error(),
			})
		} else {
			result.FilesLinked++
			logger.Debug().
				Str("media_id", mediaID).
				Str("metadata_type", autoLink.MetadataType).
				Msg("[rebuild] Auto-linked successfully")
		}
	}

	return nil
}

// isSupportedFormat checks if the file extension is in the supported formats list
func (s *Scanner) isSupportedFormat(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, format := range s.config.SupportedFormats {
		if ext == format {
			return true
		}
	}
	return false
}
