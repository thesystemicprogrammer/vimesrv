package transcode

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ProcessTranscodeInput represents the input for the ProcessTranscode use case
type ProcessTranscodeInput struct {
	JobID       int64  // ID of the job processing this transcode
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
	jobNotifier   ports.JobNotifier
	config        *config.Config
}

// NewProcessTranscodeUseCase creates a new ProcessTranscodeUseCase
func NewProcessTranscodeUseCase(
	transcodeRepo ports.TranscodeRepository,
	mediaRepo ports.MediaRepository,
	transcoder ports.Transcoder,
	filesystem ports.FileSystemService,
	jobNotifier ports.JobNotifier,
	cfg *config.Config,
) *ProcessTranscodeUseCase {
	return &ProcessTranscodeUseCase{
		transcodeRepo: transcodeRepo,
		mediaRepo:     mediaRepo,
		transcoder:    transcoder,
		filesystem:    filesystem,
		jobNotifier:   jobNotifier,
		config:        cfg,
	}
}

// parseTimeToSeconds converts FFmpeg time format (HH:MM:SS.ms) to seconds
func parseTimeToSeconds(timeStr string) float64 {
	if timeStr == "" {
		return 0
	}

	// Split by colon to get hours, minutes, seconds
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0
	}

	hours, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}

	minutes, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0
	}

	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0
	}

	return hours*3600 + minutes*60 + seconds
}

// makeProgressCallback creates a logging callback for transcode progress
func (uc *ProcessTranscodeUseCase) makeProgressCallback(
	jobID int64,
	transcodeID, mediaID, trackType, filename string,
	totalDurationSecs int,
) ports.ProgressCallback {
	lastLogTime := time.Time{} // Zero time ensures first progress is logged
	interval := time.Duration(uc.config.Transcoding.ProgressLogIntervalSecs) * time.Second

	return func(progress ports.TranscodeProgress) {
		// Skip the final 100% callback (completion is logged separately)
		if progress.Percentage == 100 {
			return
		}

		// Throttle based on configured interval
		if time.Since(lastLogTime) < interval {
			return
		}

		// Calculate percentage from time
		currentSecs := parseTimeToSeconds(progress.Time)
		percent := 0.0
		if totalDurationSecs > 0 {
			percent = (currentSecs * 100) / float64(totalDurationSecs)
			if percent > 99 {
				percent = 99
			}
		}

		logger.Debug().
			Str("transcode_id", transcodeID).
			Str("media_id", mediaID).
			Str("filename", filename).
			Str("track_type", trackType).
			Str("time", progress.Time).
			Float64("percent", percent).
			Str("speed", progress.Speed).
			Msg("Transcode progress")

		// Broadcast progress via WebSocket
		uc.jobNotifier.NotifyJobProgress(jobID, "transcode_video", ports.JobProgress{
			Frame:      progress.Frame,
			FPS:        progress.FPS,
			Bitrate:    progress.Bitrate,
			Time:       progress.Time,
			Speed:      progress.Speed,
			Percentage: percent,
			Message:    fmt.Sprintf("Transcoding %s - %s", trackType, filename),
		})

		lastLogTime = time.Now()
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

	// Build output path
	outputPath := uc.buildOutputPath(media, transcode)

	// Mark transcode as processing with the output path
	if err := uc.transcodeRepo.MarkProcessing(ctx, transcode.ID, outputPath); err != nil {
		return nil, fmt.Errorf("failed to mark transcode as processing: %w", err)
	}

	// Refresh the transcode to get updated status
	transcode, err = uc.transcodeRepo.Get(ctx, input.TranscodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh transcode: %w", err)
	}

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
		transcodeErr = uc.transcodeVideo(ctx, input.JobID, media, transcode, outputPath)
	case domain.TrackTypeAudio:
		transcodeErr = uc.transcodeAudio(ctx, input.JobID, media, transcode, outputPath)
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
func (uc *ProcessTranscodeUseCase) transcodeVideo(ctx context.Context, jobID int64, media *domain.MediaFile, transcode *domain.Transcode, outputPath string) error {
	var width, height, crf, maxBitrate int

	// Handle "original" quality - keep original resolution, use settings from highest enabled profile
	if transcode.Quality == "original" {
		// Don't set width/height - FFmpeg will keep the original resolution
		// This also ensures rotation metadata is handled correctly without dimension distortion
		width = 0
		height = 0

		// Find highest enabled quality profile for CRF/bitrate settings
		var baseProfile *config.QualityProfile
		for i := len(uc.config.Transcoding.QualityProfiles) - 1; i >= 0; i-- {
			if uc.config.Transcoding.QualityProfiles[i].Enabled {
				baseProfile = &uc.config.Transcoding.QualityProfiles[i]
				break
			}
		}
		if baseProfile == nil {
			return fmt.Errorf("no enabled quality profile found for original quality settings")
		}

		crf = baseProfile.CRF
		maxBitrateStr := strings.TrimSuffix(baseProfile.MaxBitrate, "k")
		var err error
		maxBitrate, err = strconv.Atoi(maxBitrateStr)
		if err != nil {
			return fmt.Errorf("invalid max bitrate in base profile: %w", err)
		}

		logger.Info().
			Str("transcode_id", transcode.ID).
			Int("crf", crf).
			Int("max_bitrate", maxBitrate).
			Str("base_profile", baseProfile.Name).
			Msg("Using original resolution (no scaling) with settings from highest enabled profile")
	} else {
		// Find quality profile by name
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
		var err error
		width, err = strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid width in resolution: %w", err)
		}
		height, err = strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid height in resolution: %w", err)
		}

		crf = quality.CRF

		// Parse max bitrate (remove 'k' suffix)
		maxBitrateStr := strings.TrimSuffix(quality.MaxBitrate, "k")
		maxBitrate, err = strconv.Atoi(maxBitrateStr)
		if err != nil {
			return fmt.Errorf("invalid max bitrate: %w", err)
		}
	}

	// Determine video encoder - use config or default to libx264
	videoEncoder := uc.config.Transcoding.VideoEncoder
	if videoEncoder == "" {
		videoEncoder = "libx264"
	}

	// Determine preset - use config or default to medium
	preset := uc.config.Transcoding.EncoderPreset
	if preset == "" {
		preset = "medium"
	}

	// Build transcode options
	opts := ports.TranscodeOptions{
		InputPath:       media.FilePath,
		OutputPath:      outputPath,
		Width:           width,
		Height:          height,
		VideoCodec:      videoEncoder,
		CRF:             crf,
		MaxBitrate:      maxBitrate,
		Preset:          preset,
		FFmpegInputArgs: uc.config.Transcoding.FFmpegInputArgs,
		ScaleFilter:     uc.config.Transcoding.ScaleFilter,
		SegmentTime:     uc.config.Transcoding.SegmentDuration,
		TrackType:       "video",
	}

	// Execute transcoding with progress callback
	callback := uc.makeProgressCallback(jobID, transcode.ID, media.ID, string(transcode.TrackType), media.Filename, media.Duration)
	return uc.transcoder.TranscodeVideo(ctx, opts, callback)
}

// transcodeAudio handles audio transcoding
func (uc *ProcessTranscodeUseCase) transcodeAudio(ctx context.Context, jobID int64, media *domain.MediaFile, transcode *domain.Transcode, outputPath string) error {
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

	// Execute transcoding with progress callback
	callback := uc.makeProgressCallback(jobID, transcode.ID, media.ID, opts.TrackType, media.Filename, media.Duration)
	return uc.transcoder.TranscodeAudio(ctx, opts, callback)
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
