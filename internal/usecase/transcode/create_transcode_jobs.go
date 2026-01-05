package transcode

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// CreateTranscodeJobsInput represents the input for creating transcode jobs
type CreateTranscodeJobsInput struct {
	MediaID string // ID of the media file to transcode
}

// CreateTranscodeJobsOutput represents the output after creating transcode jobs
type CreateTranscodeJobsOutput struct {
	MediaID      string
	TotalJobs    int
	VideoJobs    int
	AudioJobs    int
	SubtitleJobs int
}

// CreateTranscodeJobsUseCase creates transcode jobs for a media file
type CreateTranscodeJobsUseCase struct {
	mediaRepo          ports.MediaRepository
	transcodeRepo      ports.TranscodeRepository
	enqueueJobUseCase  *job.EnqueueJobUseCase
	audioStreamRepo    ports.AudioStreamRepository
	subtitleStreamRepo ports.SubtitleStreamRepository
	ffprobe            ports.FFProbeService
	config             *config.Config
}

// NewCreateTranscodeJobsUseCase creates a new CreateTranscodeJobsUseCase
func NewCreateTranscodeJobsUseCase(
	mediaRepo ports.MediaRepository,
	transcodeRepo ports.TranscodeRepository,
	enqueueJobUseCase *job.EnqueueJobUseCase,
	audioStreamRepo ports.AudioStreamRepository,
	subtitleStreamRepo ports.SubtitleStreamRepository,
	ffprobe ports.FFProbeService,
	cfg *config.Config,
) *CreateTranscodeJobsUseCase {
	return &CreateTranscodeJobsUseCase{
		mediaRepo:          mediaRepo,
		transcodeRepo:      transcodeRepo,
		enqueueJobUseCase:  enqueueJobUseCase,
		audioStreamRepo:    audioStreamRepo,
		subtitleStreamRepo: subtitleStreamRepo,
		ffprobe:            ffprobe,
		config:             cfg,
	}
}

// Execute creates all necessary transcode jobs for a media file
func (uc *CreateTranscodeJobsUseCase) Execute(ctx context.Context, input CreateTranscodeJobsInput) (*CreateTranscodeJobsOutput, error) {
	// Get media file
	media, err := uc.mediaRepo.Get(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	// Get enabled quality profiles
	var enabledQualities []config.QualityProfile
	for _, q := range uc.config.Transcoding.QualityProfiles {
		if q.Enabled {
			enabledQualities = append(enabledQualities, q)
		}
	}

	if len(enabledQualities) == 0 {
		return nil, fmt.Errorf("no enabled quality profiles found")
	}

	// Create video transcode jobs (one per quality profile)
	videoJobs, err := uc.createVideoTranscodeJobs(ctx, media, enabledQualities)
	if err != nil {
		return nil, err
	}

	// Create audio transcode jobs (one per audio stream, shared across all qualities)
	audioJobs, err := uc.createAudioTranscodeJobs(ctx, media)
	if err != nil {
		return nil, err
	}

	// Create subtitle transcode jobs (one per subtitle stream)
	subtitleJobs, err := uc.createSubtitleTranscodeJobs(ctx, media)
	if err != nil {
		return nil, err
	}

	totalJobs := videoJobs + audioJobs + subtitleJobs

	return &CreateTranscodeJobsOutput{
		MediaID:      media.ID,
		TotalJobs:    totalJobs,
		VideoJobs:    videoJobs,
		AudioJobs:    audioJobs,
		SubtitleJobs: subtitleJobs,
	}, nil
}

// TranscodeJobPayload is the payload for transcode jobs
type TranscodeJobPayload struct {
	TranscodeID   string `json:"transcode_id"`
	Filename      string `json:"filename"`
	Language      string `json:"language,omitempty"`
	ChannelLayout string `json:"channel_layout,omitempty"`
}

// parseResolutionHeight extracts height from a resolution string like "1280x720"
func parseResolutionHeight(resolution string) (int, error) {
	parts := strings.Split(resolution, "x")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid resolution format: %s", resolution)
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid height in resolution %s: %w", resolution, err)
	}
	return height, nil
}

// createVideoTranscodeJobs creates transcode jobs for each video quality profile
// that doesn't exceed the source video's resolution.
// If all profiles exceed the source resolution, creates a single "original" quality job.
// Returns the number of jobs created.
func (uc *CreateTranscodeJobsUseCase) createVideoTranscodeJobs(ctx context.Context, media *domain.MediaFile, qualities []config.QualityProfile) (int, error) {
	var videoJobs int

	// Filter quality profiles to only those that don't exceed source resolution
	// If source height is unknown (0), skip filtering and create all profiles
	var filteredQualities []config.QualityProfile
	if media.Height > 0 {
		for _, quality := range qualities {
			targetHeight, err := parseResolutionHeight(quality.Resolution)
			if err != nil {
				logger.Warn().
					Str("media_id", media.ID).
					Str("quality", quality.Name).
					Str("resolution", quality.Resolution).
					Err(err).
					Msg("Failed to parse quality profile resolution, skipping")
				continue
			}

			if targetHeight <= media.Height {
				filteredQualities = append(filteredQualities, quality)
			} else {
				logger.Info().
					Str("media_id", media.ID).
					Str("quality", quality.Name).
					Int("target_height", targetHeight).
					Int("source_height", media.Height).
					Msg("Skipping quality profile: target resolution exceeds source")
			}
		}
	} else {
		// Source height unknown, use all profiles
		filteredQualities = qualities
	}

	// If no profiles remain after filtering, create an "original" quality transcode
	if len(filteredQualities) == 0 && media.Height > 0 {
		logger.Info().
			Str("media_id", media.ID).
			Int("source_height", media.Height).
			Msg("All quality profiles exceed source resolution, creating original quality transcode")

		return uc.createOriginalQualityTranscode(ctx, media)
	}

	// Create transcode jobs for each filtered quality profile
	for _, quality := range filteredQualities {
		transcodeID := fmt.Sprintf("%s-video-%s", media.ID, quality.Name)

		// Create transcode record
		transcode := domain.NewTranscode(
			transcodeID,
			media.ID,
			quality.Name,
			domain.TrackTypeVideo,
			0, // Video track index is always 0
		)

		if err := uc.transcodeRepo.Create(ctx, transcode); err != nil {
			return 0, fmt.Errorf("failed to create video transcode record: %w", err)
		}

		// Create job for this transcode
		_, err := uc.enqueueJobUseCase.Execute(ctx, job.EnqueueJobInput{
			Type: shared.JobTypeTranscodeVideo,
			Payload: TranscodeJobPayload{
				TranscodeID: transcodeID,
				Filename:    media.Filename,
			},
			Priority: shared.JobPriorityTranscode,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to create video transcode job: %w", err)
		}

		videoJobs++
	}

	return videoJobs, nil
}

// createOriginalQualityTranscode creates a single transcode job at the original resolution.
// This is used when all configured quality profiles exceed the source video's resolution.
func (uc *CreateTranscodeJobsUseCase) createOriginalQualityTranscode(ctx context.Context, media *domain.MediaFile) (int, error) {
	transcodeID := fmt.Sprintf("%s-video-original", media.ID)

	// Create transcode record with "original" quality
	transcode := domain.NewTranscode(
		transcodeID,
		media.ID,
		"original",
		domain.TrackTypeVideo,
		0, // Video track index is always 0
	)

	if err := uc.transcodeRepo.Create(ctx, transcode); err != nil {
		return 0, fmt.Errorf("failed to create original video transcode record: %w", err)
	}

	// Create job for this transcode
	_, err := uc.enqueueJobUseCase.Execute(ctx, job.EnqueueJobInput{
		Type: shared.JobTypeTranscodeVideo,
		Payload: TranscodeJobPayload{
			TranscodeID: transcodeID,
			Filename:    media.Filename,
		},
		Priority: shared.JobPriorityTranscode,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create original video transcode job: %w", err)
	}

	return 1, nil
}

// createAudioTranscodeJobs detects audio streams, saves them to the database,
// and creates transcode jobs for each stream. Returns the number of jobs created.
func (uc *CreateTranscodeJobsUseCase) createAudioTranscodeJobs(ctx context.Context, media *domain.MediaFile) (int, error) {
	// Detect audio streams
	audioStreams, err := uc.ffprobe.GetAudioStreams(media.FilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to detect audio streams: %w", err)
	}

	// Save audio streams to database
	for _, audioInfo := range audioStreams {
		audioStream := domain.NewAudioStream(
			media.ID,
			audioInfo.StreamIndex,
			audioInfo.Codec,
			audioInfo.Language,
			audioInfo.Channels,
			audioInfo.ChannelLayout,
			audioInfo.SampleRate,
			audioInfo.Title,
		)
		if err := uc.audioStreamRepo.Create(ctx, audioStream); err != nil {
			return 0, fmt.Errorf("failed to save audio stream %d: %w", audioInfo.StreamIndex, err)
		}
	}

	var audioJobs int

	// Create audio transcode jobs (one per audio stream, shared across all qualities)
	for _, audioStream := range audioStreams {
		transcodeID := fmt.Sprintf("%s-audio-%d", media.ID, audioStream.StreamIndex)

		// Create transcode record (quality empty for shared audio)
		transcode := domain.NewTranscode(
			transcodeID,
			media.ID,
			"", // Empty quality means shared across all video qualities
			domain.TrackTypeAudio,
			audioStream.StreamIndex,
		)

		if err := uc.transcodeRepo.Create(ctx, transcode); err != nil {
			return 0, fmt.Errorf("failed to create audio transcode record: %w", err)
		}

		// Create job with extended payload including language and channel layout
		_, err = uc.enqueueJobUseCase.Execute(ctx, job.EnqueueJobInput{
			Type: shared.JobTypeTranscodeAudio,
			Payload: TranscodeJobPayload{
				TranscodeID:   transcodeID,
				Filename:      media.Filename,
				Language:      audioStream.Language,
				ChannelLayout: audioStream.ChannelLayout,
			},
			Priority: shared.JobPriorityTranscode,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to enqueue audio transcode job: %w", err)
		}

		audioJobs++
	}

	return audioJobs, nil
}

// createSubtitleTranscodeJobs detects subtitle streams, saves them to the database,
// and creates transcode jobs for each stream. Returns the number of jobs created.
func (uc *CreateTranscodeJobsUseCase) createSubtitleTranscodeJobs(ctx context.Context, media *domain.MediaFile) (int, error) {
	// Detect subtitle streams
	subtitleStreams, err := uc.ffprobe.GetSubtitleStreams(media.FilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to detect subtitle streams: %w", err)
	}

	// Save subtitle streams to database and create transcode jobs for text-based subtitles
	var subtitleJobs int

	for _, subtitleInfo := range subtitleStreams {
		subtitleStream := domain.NewSubtitleStream(
			media.ID,
			subtitleInfo.StreamIndex,
			subtitleInfo.Codec,
			subtitleInfo.Language,
			subtitleInfo.Title,
			subtitleInfo.Forced,
		)
		if err := uc.subtitleStreamRepo.Create(ctx, subtitleStream); err != nil {
			return 0, fmt.Errorf("failed to save subtitle stream %d: %w", subtitleInfo.StreamIndex, err)
		}

		// Skip bitmap-based subtitles (PGS, DVD, DVB) as they cannot be converted to WebVTT
		if !subtitleStream.IsTextBased() {
			logger.Info().
				Str("media_id", media.ID).
				Int("stream_index", subtitleStream.StreamIndex).
				Str("codec", subtitleStream.Codec).
				Str("language", subtitleStream.Language).
				Msg("Skipping bitmap-based subtitle: cannot convert to WebVTT without OCR")
			continue
		}

		transcodeID := fmt.Sprintf("%s-subtitle-%d", media.ID, subtitleStream.StreamIndex)

		// Create transcode record (quality empty for subtitles)
		transcode := domain.NewTranscode(
			transcodeID,
			media.ID,
			"", // Empty quality for subtitles
			domain.TrackTypeSubtitle,
			subtitleStream.StreamIndex,
		)

		if err := uc.transcodeRepo.Create(ctx, transcode); err != nil {
			return 0, fmt.Errorf("failed to create subtitle transcode record: %w", err)
		}

		// Create job for this transcode with language
		_, err = uc.enqueueJobUseCase.Execute(ctx, job.EnqueueJobInput{
			Type: shared.JobTypeTranscodeSubtitle,
			Payload: TranscodeJobPayload{
				TranscodeID: transcodeID,
				Filename:    media.Filename,
				Language:    subtitleStream.Language,
			},
			Priority: shared.JobPriorityTranscode,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to create subtitle transcode job: %w", err)
		}

		subtitleJobs++
	}

	return subtitleJobs, nil
}
