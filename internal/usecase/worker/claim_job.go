package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/worker"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// TranscodeJobPayload is the payload structure for transcode_video jobs
type TranscodeJobPayload struct {
	TranscodeID string `json:"transcode_id"`
}

// WorkerJob contains all information a worker needs to process a transcode job
type WorkerJob struct {
	JobID            int64                  `json:"job_id"`
	TranscodeID      string                 `json:"transcode_id"`
	TrackType        string                 `json:"track_type"`
	TrackIndex       int                    `json:"track_index"`
	Quality          string                 `json:"quality,omitempty"`
	InputPath        string                 `json:"input_path"`
	OutputPath       string                 `json:"output_path"`
	MediaDuration    float64                `json:"media_duration"`
	TranscodeOptions WorkerTranscodeOptions `json:"transcode_options"`
}

// WorkerTranscodeOptions contains FFmpeg parameters for transcoding
type WorkerTranscodeOptions struct {
	// Video options
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	VideoCodec string `json:"video_codec,omitempty"`
	CRF        int    `json:"crf,omitempty"`
	MaxBitrate int    `json:"max_bitrate,omitempty"`
	Preset     string `json:"preset,omitempty"`

	// Audio options
	AudioCodec   string `json:"audio_codec,omitempty"`
	AudioBitrate int    `json:"audio_bitrate,omitempty"`

	// Segmentation
	SegmentTime int `json:"segment_time"`
}

// ClaimJobForWorkerUseCase handles claiming a transcode job for a distributed worker
type ClaimJobForWorkerUseCase struct {
	jobRepo          ports.JobRepository
	transcodeRepo    ports.TranscodeRepository
	mediaRepo        ports.MediaRepository
	workerRegistry   *worker.Registry
	workerConfigRepo ports.WorkerConfigRepository
	jobNotifier      ports.JobNotifier
	backoff          ports.BackoffStrategy
	config           *config.Config
}

// NewClaimJobForWorkerUseCase creates a new ClaimJobForWorkerUseCase
func NewClaimJobForWorkerUseCase(
	jobRepo ports.JobRepository,
	transcodeRepo ports.TranscodeRepository,
	mediaRepo ports.MediaRepository,
	workerRegistry *worker.Registry,
	workerConfigRepo ports.WorkerConfigRepository,
	jobNotifier ports.JobNotifier,
	backoff ports.BackoffStrategy,
	cfg *config.Config,
) *ClaimJobForWorkerUseCase {
	return &ClaimJobForWorkerUseCase{
		jobRepo:          jobRepo,
		transcodeRepo:    transcodeRepo,
		mediaRepo:        mediaRepo,
		workerRegistry:   workerRegistry,
		workerConfigRepo: workerConfigRepo,
		jobNotifier:      jobNotifier,
		backoff:          backoff,
		config:           cfg,
	}
}

// Execute claims the next available transcode job for the worker
// Returns nil if no jobs are available
func (uc *ClaimJobForWorkerUseCase) Execute(ctx context.Context, workerID string) (*WorkerJob, error) {
	// 1. Verify worker is registered and touch its timestamp
	if !uc.workerRegistry.Touch(workerID) {
		return nil, fmt.Errorf("worker not registered: %s", workerID)
	}

	// 2. Get worker config to determine allowed job types
	workerConfig, err := uc.workerConfigRepo.Get(ctx, workerID)
	if err != nil {
		logger.Warn().Err(err).Str("worker_id", workerID).Msg("failed to get worker config, using defaults")
		// Default: allow all types if config not found
		workerConfig = nil
	}

	// 3. Build list of allowed job types based on worker config
	allowedTypes := uc.buildAllowedJobTypes(workerConfig)
	if len(allowedTypes) == 0 {
		logger.Debug().Str("worker_id", workerID).Msg("worker not configured to accept any job types")
		return nil, nil // Worker can't process any job types
	}

	// 4. Claim next transcode job atomically with type filtering
	job, err := uc.jobRepo.ClaimNextTranscodeJobWithTypes(ctx, workerID, allowedTypes)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, nil // No jobs available
	}

	// 3. Parse job payload to get transcode ID
	var payload TranscodeJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		// Fail job with proper retry logic
		uc.failClaimedJob(ctx, job, "invalid payload: "+err.Error())
		return nil, fmt.Errorf("invalid job payload: %w", err)
	}

	// 4. Fetch transcode record
	transcode, err := uc.transcodeRepo.Get(ctx, payload.TranscodeID)
	if err != nil {
		uc.failClaimedJob(ctx, job, "transcode not found: "+err.Error())
		return nil, fmt.Errorf("transcode not found: %w", err)
	}

	// 5. Fetch media file
	media, err := uc.mediaRepo.Get(ctx, transcode.MediaID)
	if err != nil {
		uc.failClaimedJob(ctx, job, "media not found: "+err.Error())
		return nil, fmt.Errorf("media not found: %w", err)
	}

	// 6. Build transcode options based on track type
	opts, err := uc.buildTranscodeOptions(transcode)
	if err != nil {
		uc.failClaimedJob(ctx, job, "failed to build options: "+err.Error())
		return nil, err
	}

	// 7. Build output paths - relative for worker, absolute for database
	relativeOutputPath := uc.buildOutputPath(media, transcode)
	absoluteOutputPath := filepath.Join(uc.config.Media.MediaPath, relativeOutputPath)

	// 8. Mark transcode as processing and store the absolute output path
	if err := uc.transcodeRepo.MarkProcessing(ctx, transcode.ID, absoluteOutputPath); err != nil {
		logger.Warn().Err(err).Str("transcode_id", transcode.ID).Msg("failed to mark transcode as processing")
	}

	// 9. Increment worker's active job count
	uc.workerRegistry.IncrementActiveJobs(workerID)

	// 10. Notify job started
	uc.jobNotifier.NotifyJobStarted(job)

	logger.Info().
		Int64("job_id", job.ID).
		Str("transcode_id", transcode.ID).
		Str("worker_id", workerID).
		Str("track_type", string(transcode.TrackType)).
		Str("quality", transcode.Quality).
		Msg("Worker claimed transcode job")

	// 11. Return complete WorkerJob with relative paths
	// Workers will prepend their own media_path to these relative paths
	return &WorkerJob{
		JobID:            job.ID,
		TranscodeID:      transcode.ID,
		TrackType:        string(transcode.TrackType),
		TrackIndex:       transcode.TrackIndex,
		Quality:          transcode.Quality,
		InputPath:        uc.toRelativePath(media.FilePath),
		OutputPath:       relativeOutputPath,
		MediaDuration:    float64(media.Duration),
		TranscodeOptions: opts,
	}, nil
}

// buildTranscodeOptions builds the transcode options based on track type
func (uc *ClaimJobForWorkerUseCase) buildTranscodeOptions(transcode *domain.Transcode) (WorkerTranscodeOptions, error) {
	opts := WorkerTranscodeOptions{
		SegmentTime: uc.config.Transcoding.SegmentDuration,
	}

	switch transcode.TrackType {
	case domain.TrackTypeVideo:
		// Handle "original" quality - keep original resolution, use settings from highest enabled profile
		if transcode.Quality == "original" {
			// Don't set width/height - FFmpeg will keep the original resolution
			// This also ensures rotation metadata is handled correctly without dimension distortion
			opts.Width = 0
			opts.Height = 0

			// Find highest enabled quality profile for CRF/bitrate settings
			var baseProfile *config.QualityProfile
			for i := len(uc.config.Transcoding.QualityProfiles) - 1; i >= 0; i-- {
				if uc.config.Transcoding.QualityProfiles[i].Enabled {
					baseProfile = &uc.config.Transcoding.QualityProfiles[i]
					break
				}
			}
			if baseProfile == nil {
				return opts, fmt.Errorf("no enabled quality profile found for original quality settings")
			}

			// Parse max bitrate from base profile
			maxBitrateStr := strings.TrimSuffix(baseProfile.MaxBitrate, "k")
			maxBitrate, err := strconv.Atoi(maxBitrateStr)
			if err != nil {
				return opts, fmt.Errorf("invalid max bitrate in base profile: %w", err)
			}

			opts.VideoCodec = "libx264"
			opts.CRF = baseProfile.CRF
			opts.MaxBitrate = maxBitrate
			opts.Preset = "medium"

			logger.Info().
				Str("transcode_id", transcode.ID).
				Int("crf", opts.CRF).
				Int("max_bitrate", opts.MaxBitrate).
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
				return opts, fmt.Errorf("quality profile not found: %s", transcode.Quality)
			}

			// Parse resolution
			parts := strings.Split(quality.Resolution, "x")
			if len(parts) != 2 {
				return opts, fmt.Errorf("invalid resolution format: %s", quality.Resolution)
			}
			width, err := strconv.Atoi(parts[0])
			if err != nil {
				return opts, fmt.Errorf("invalid width in resolution: %w", err)
			}
			height, err := strconv.Atoi(parts[1])
			if err != nil {
				return opts, fmt.Errorf("invalid height in resolution: %w", err)
			}

			// Parse max bitrate (remove 'k' suffix)
			maxBitrateStr := strings.TrimSuffix(quality.MaxBitrate, "k")
			maxBitrate, err := strconv.Atoi(maxBitrateStr)
			if err != nil {
				return opts, fmt.Errorf("invalid max bitrate: %w", err)
			}

			opts.Width = width
			opts.Height = height
			opts.VideoCodec = "libx264"
			opts.CRF = quality.CRF
			opts.MaxBitrate = maxBitrate
			opts.Preset = "medium"
		}

	case domain.TrackTypeAudio:
		// Use audio bitrate from first enabled quality profile
		var audioBitrateStr string
		for _, q := range uc.config.Transcoding.QualityProfiles {
			if q.Enabled {
				audioBitrateStr = q.AudioBitrate
				break
			}
		}
		if audioBitrateStr == "" {
			return opts, fmt.Errorf("no enabled quality profile found")
		}

		audioBitrateStr = strings.TrimSuffix(audioBitrateStr, "k")
		audioBitrate, err := strconv.Atoi(audioBitrateStr)
		if err != nil {
			return opts, fmt.Errorf("invalid audio bitrate: %w", err)
		}

		opts.AudioCodec = "aac"
		opts.AudioBitrate = audioBitrate

	case domain.TrackTypeSubtitle:
		// No special options needed for subtitles
	}

	return opts, nil
}

// toRelativePath converts an absolute file path to a path relative to the server's media_path.
// Workers will prepend their own media_path to resolve the full path on their filesystem.
func (uc *ClaimJobForWorkerUseCase) toRelativePath(absolutePath string) string {
	mediaPath := uc.config.Media.MediaPath
	if strings.HasPrefix(absolutePath, mediaPath) {
		rel := strings.TrimPrefix(absolutePath, mediaPath)
		return strings.TrimPrefix(rel, "/")
	}
	// Path doesn't start with media_path, return as-is (shouldn't happen in practice)
	return absolutePath
}

// buildOutputPath constructs the output path for transcode output.
// Returns a relative path that workers will prepend their media_path to.
func (uc *ClaimJobForWorkerUseCase) buildOutputPath(media *domain.MediaFile, transcode *domain.Transcode) string {
	// Get relative base path from config pattern (replace {media_path} with empty, {media_id} with actual ID)
	pattern := uc.config.Media.TranscodeOutputPattern
	// Remove {media_path} prefix to get relative path
	basePath := strings.ReplaceAll(pattern, "{media_path}/", "")
	basePath = strings.ReplaceAll(basePath, "{media_path}", "")
	basePath = strings.ReplaceAll(basePath, "{media_id}", media.ID)

	// Build track-specific path
	switch transcode.TrackType {
	case domain.TrackTypeVideo:
		return filepath.Join(basePath, transcode.Quality, "video")
	case domain.TrackTypeAudio:
		return filepath.Join(basePath, fmt.Sprintf("audio-%d", transcode.TrackIndex))
	case domain.TrackTypeSubtitle:
		return filepath.Join(basePath, fmt.Sprintf("subtitle-%d.vtt", transcode.TrackIndex))
	default:
		return basePath
	}
}

// failClaimedJob handles failure of a job that was already claimed.
// It respects max attempts and uses proper backoff, unlike a simple Reschedule.
func (uc *ClaimJobForWorkerUseCase) failClaimedJob(ctx context.Context, job *domain.Job, errMsg string) {
	// Check if we should retry or mark as dead
	if job.Attempts < job.MaxAttempts {
		// Reschedule with backoff
		delay := uc.backoff.NextDelay(job.Attempts)
		nextRun := time.Now().Add(delay)

		if err := uc.jobRepo.Reschedule(ctx, job.ID, nextRun, errMsg); err != nil {
			logger.Error().Err(err).Int64("job_id", job.ID).Msg("failed to reschedule job during claim failure")
			return
		}

		logger.Warn().
			Int64("job_id", job.ID).
			Int("attempt", job.Attempts).
			Int("max_attempts", job.MaxAttempts).
			Str("error", errMsg).
			Time("next_run", nextRun).
			Msg("Job failed during claim, rescheduled for retry")
	} else {
		// Max attempts reached, mark as dead
		if err := uc.jobRepo.MarkDead(ctx, job.ID, errMsg); err != nil {
			logger.Error().Err(err).Int64("job_id", job.ID).Msg("failed to mark job as dead during claim failure")
			return
		}

		// Try to mark transcode as failed
		var payload TranscodeJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err == nil && payload.TranscodeID != "" {
			if err := uc.transcodeRepo.MarkFailed(ctx, payload.TranscodeID); err != nil {
				logger.Warn().Err(err).Str("transcode_id", payload.TranscodeID).Msg("failed to mark transcode as failed")
			}
		}

		logger.Error().
			Int64("job_id", job.ID).
			Int("attempts", job.Attempts).
			Str("error", errMsg).
			Msg("Job failed permanently during claim")

		uc.jobNotifier.NotifyJobFailed(job, errMsg)
	}
}

// buildAllowedJobTypes returns the list of job types a worker is allowed to claim
// based on its configuration. Subtitles are always allowed for any worker that
// can accept either video or audio.
func (uc *ClaimJobForWorkerUseCase) buildAllowedJobTypes(cfg *domain.WorkerConfig) []string {
	// If no config, allow all types (backward compatibility)
	if cfg == nil {
		return []string{"transcode_video", "transcode_audio", "transcode_subtitle"}
	}

	var types []string

	if cfg.AcceptsVideo {
		types = append(types, "transcode_video")
	}
	if cfg.AcceptsAudio {
		types = append(types, "transcode_audio")
	}
	// Subtitles are lightweight and can be processed by any worker
	// that accepts either video or audio
	if cfg.AcceptsVideo || cfg.AcceptsAudio {
		types = append(types, "transcode_subtitle")
	}

	return types
}
