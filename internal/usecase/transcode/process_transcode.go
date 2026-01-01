package transcode

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ProcessTranscodeInput represents the input for the ProcessTranscode use case
type ProcessTranscodeInput struct {
	TranscodeID string // ID of the transcode record
}

// ProcessTranscodeOutput represents the output of the ProcessTranscode use case
type ProcessTranscodeOutput struct {
	TranscodeID  string
	OutputPath   string
	SegmentCount int
	Success      bool
}

// ProcessTranscodeUseCase executes a single transcode job
type ProcessTranscodeUseCase struct {
	transcodeRepo ports.TranscodeRepository
	mediaRepo     ports.MediaRepository
	transcoder    ports.Transcoder
	filesystem    ports.FileSystemService
	config        *config.Config
}

// NewProcessTranscodeUseCase creates a new ProcessTranscodeUseCase
func NewProcessTranscodeUseCase(
	transcodeRepo ports.TranscodeRepository,
	mediaRepo ports.MediaRepository,
	transcoder ports.Transcoder,
	filesystem ports.FileSystemService,
	cfg *config.Config,
) *ProcessTranscodeUseCase {
	return &ProcessTranscodeUseCase{
		transcodeRepo: transcodeRepo,
		mediaRepo:     mediaRepo,
		transcoder:    transcoder,
		filesystem:    filesystem,
		config:        cfg,
	}
}

// Execute processes a single transcode job
func (uc *ProcessTranscodeUseCase) Execute(ctx context.Context, input ProcessTranscodeInput) (*ProcessTranscodeOutput, error) {
	// Get transcode record
	transcode, err := uc.transcodeRepo.Get(ctx, input.TranscodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to find transcode: %w", err)
	}

	// Get media file
	media, err := uc.mediaRepo.Get(ctx, transcode.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to find media: %w", err)
	}

	// Mark transcode as processing
	if err := uc.transcodeRepo.MarkProcessing(ctx, transcode.ID); err != nil {
		return nil, fmt.Errorf("failed to mark transcode as processing: %w", err)
	}

	// Refresh the transcode to get updated status
	transcode, err = uc.transcodeRepo.Get(ctx, input.TranscodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh transcode: %w", err)
	}

	// Build output path
	outputPath := uc.buildOutputPath(media, transcode)

	// Log transcode start with details
	logEvent := logger.Info().
		Str("transcode_id", transcode.ID).
		Str("media_id", media.ID).
		Str("track_type", string(transcode.TrackType)).
		Int("track_index", transcode.TrackIndex).
		Str("output_path", outputPath)

	if transcode.TrackType == domain.TrackTypeVideo {
		logEvent = logEvent.Str("quality", transcode.Quality)
	}

	logEvent.Msg("Starting transcode job")

	// Execute transcoding based on track type
	var transcodeErr error
	switch transcode.TrackType {
	case domain.TrackTypeVideo:
		transcodeErr = uc.transcodeVideo(ctx, media, transcode, outputPath)
	case domain.TrackTypeAudio:
		transcodeErr = uc.transcodeAudio(ctx, media, transcode, outputPath)
	case domain.TrackTypeSubtitle:
		transcodeErr = uc.transcodeSubtitle(ctx, media, transcode, outputPath)
	default:
		transcodeErr = fmt.Errorf("unknown track type: %s", transcode.TrackType)
	}

	// Update transcode status
	if transcodeErr != nil {
		logger.Error().
			Err(transcodeErr).
			Str("transcode_id", transcode.ID).
			Str("media_id", media.ID).
			Str("track_type", string(transcode.TrackType)).
			Msg("Transcode job failed")

		if err := uc.transcodeRepo.MarkFailed(ctx, transcode.ID); err != nil {
			return nil, fmt.Errorf("failed to mark transcode as failed: %w", err)
		}
		return nil, transcodeErr
	}

	// Count segments (for video/audio)
	segmentCount := 0
	if transcode.TrackType == domain.TrackTypeVideo || transcode.TrackType == domain.TrackTypeAudio {
		segments, err := uc.transcoder.ProbeSegmentDurations(ctx, outputPath)
		if err == nil {
			segmentCount = len(segments)
			// Save segment timings
			if saveErr := uc.saveSegmentTimings(outputPath, segments); saveErr != nil {
				// Log but don't fail the transcode
				fmt.Printf("Warning: failed to save segment timings: %v\n", saveErr)
			}
		}
	}

	// Mark transcode as completed
	if err := uc.transcodeRepo.MarkCompleted(ctx, transcode.ID, outputPath); err != nil {
		return nil, fmt.Errorf("failed to mark transcode as completed: %w", err)
	}

	// Log successful completion
	logger.Info().
		Str("transcode_id", transcode.ID).
		Str("media_id", media.ID).
		Str("track_type", string(transcode.TrackType)).
		Int("segment_count", segmentCount).
		Str("output_path", outputPath).
		Msg("Transcode job completed successfully")

	return &ProcessTranscodeOutput{
		TranscodeID:  transcode.ID,
		OutputPath:   outputPath,
		SegmentCount: segmentCount,
		Success:      true,
	}, nil
}

// transcodeVideo handles video transcoding
func (uc *ProcessTranscodeUseCase) transcodeVideo(ctx context.Context, media *domain.MediaFile, transcode *domain.Transcode, outputPath string) error {
	// Find quality profile
	var quality *config.QualityProfile
	for _, q := range uc.config.Transcoding.QualityProfiles {
		if q.Name == transcode.Quality {
			quality = &q
			break
		}
	}
	if quality == nil {
		return fmt.Errorf("quality profile not found: %s", transcode.Quality)
	}

	// Parse resolution
	parts := strings.Split(quality.Resolution, "x")
	if len(parts) != 2 {
		return fmt.Errorf("invalid resolution format: %s", quality.Resolution)
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid width in resolution: %w", err)
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid height in resolution: %w", err)
	}

	// Parse max bitrate (remove 'k' suffix)
	maxBitrateStr := strings.TrimSuffix(quality.MaxBitrate, "k")
	maxBitrate, err := strconv.Atoi(maxBitrateStr)
	if err != nil {
		return fmt.Errorf("invalid max bitrate: %w", err)
	}

	// Build transcode options
	opts := ports.TranscodeOptions{
		InputPath:   media.FilePath,
		OutputPath:  outputPath,
		Width:       width,
		Height:      height,
		VideoCodec:  "libx264",
		CRF:         quality.CRF,
		MaxBitrate:  maxBitrate,
		Preset:      "medium",
		SegmentTime: uc.config.Transcoding.SegmentDuration,
		TrackType:   "video",
	}

	// Execute transcoding with progress callback
	return uc.transcoder.TranscodeVideo(ctx, opts, nil)
}

// transcodeAudio handles audio transcoding
func (uc *ProcessTranscodeUseCase) transcodeAudio(ctx context.Context, media *domain.MediaFile, transcode *domain.Transcode, outputPath string) error {
	// Parse audio bitrate from quality profile (use first enabled profile)
	var audioBitrateStr string
	for _, q := range uc.config.Transcoding.QualityProfiles {
		if q.Enabled {
			audioBitrateStr = q.AudioBitrate
			break
		}
	}
	if audioBitrateStr == "" {
		return fmt.Errorf("no enabled quality profile found")
	}

	// Parse audio bitrate
	audioBitrateStr = strings.TrimSuffix(audioBitrateStr, "k")
	audioBitrate, err := strconv.Atoi(audioBitrateStr)
	if err != nil {
		return fmt.Errorf("invalid audio bitrate: %w", err)
	}

	// Build transcode options
	opts := ports.TranscodeOptions{
		InputPath:         media.FilePath,
		OutputPath:        outputPath,
		SourceStreamIndex: transcode.TrackIndex,
		AudioCodec:        "aac",
		AudioBitrate:      audioBitrate,
		AudioChannels:     0, // Preserve source channels
		SegmentTime:       uc.config.Transcoding.SegmentDuration,
		TrackType:         fmt.Sprintf("audio-%d", transcode.TrackIndex),
	}

	// Execute transcoding
	return uc.transcoder.TranscodeAudio(ctx, opts, nil)
}

// transcodeSubtitle handles subtitle extraction
func (uc *ProcessTranscodeUseCase) transcodeSubtitle(ctx context.Context, media *domain.MediaFile, transcode *domain.Transcode, outputPath string) error {
	// Build transcode options
	opts := ports.TranscodeOptions{
		InputPath:         media.FilePath,
		OutputPath:        outputPath, // For subtitles, this is the .vtt file path
		SourceStreamIndex: transcode.TrackIndex,
		TrackType:         fmt.Sprintf("subtitle-%d", transcode.TrackIndex),
	}

	// Execute subtitle extraction
	return uc.transcoder.ExtractSubtitle(ctx, opts)
}

// buildOutputPath constructs the output path for transcode output
func (uc *ProcessTranscodeUseCase) buildOutputPath(media *domain.MediaFile, transcode *domain.Transcode) string {
	// Get base path from config pattern
	pattern := uc.config.Media.TranscodeOutputPattern
	basePath := strings.ReplaceAll(pattern, "{media_path}", uc.config.Media.MediaPath)
	basePath = strings.ReplaceAll(basePath, "{media_id}", media.ID)

	// Build track-specific path
	switch transcode.TrackType {
	case domain.TrackTypeVideo:
		// video/{quality}/video
		return filepath.Join(basePath, transcode.Quality, "video")
	case domain.TrackTypeAudio:
		// audio-{index}
		return filepath.Join(basePath, fmt.Sprintf("audio-%d", transcode.TrackIndex))
	case domain.TrackTypeSubtitle:
		// subtitle-{index}.vtt
		return filepath.Join(basePath, fmt.Sprintf("subtitle-%d.vtt", transcode.TrackIndex))
	default:
		return basePath
	}
}

// saveSegmentTimings saves segment timing data to JSON
func (uc *ProcessTranscodeUseCase) saveSegmentTimings(outputPath string, segments []ports.SegmentInfo) error {
	data := struct {
		Segments []ports.SegmentInfo `json:"segments"`
	}{
		Segments: segments,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal segment timings: %w", err)
	}

	timingFilePath := filepath.Join(outputPath, "segments.json")
	return uc.filesystem.WriteFile(timingFilePath, jsonData)
}
