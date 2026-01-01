package transcode

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
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
	jobRepo            ports.JobRepository
	audioStreamRepo    ports.AudioStreamRepository
	subtitleStreamRepo ports.SubtitleStreamRepository
	ffprobe            ports.FFProbeService
	config             *config.Config
}

// NewCreateTranscodeJobsUseCase creates a new CreateTranscodeJobsUseCase
func NewCreateTranscodeJobsUseCase(
	mediaRepo ports.MediaRepository,
	transcodeRepo ports.TranscodeRepository,
	jobRepo ports.JobRepository,
	audioStreamRepo ports.AudioStreamRepository,
	subtitleStreamRepo ports.SubtitleStreamRepository,
	ffprobe ports.FFProbeService,
	cfg *config.Config,
) *CreateTranscodeJobsUseCase {
	return &CreateTranscodeJobsUseCase{
		mediaRepo:          mediaRepo,
		transcodeRepo:      transcodeRepo,
		jobRepo:            jobRepo,
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

	var videoJobs, audioJobs, subtitleJobs int

	// Create video transcode jobs (one per quality profile)
	for _, quality := range enabledQualities {
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
			return nil, fmt.Errorf("failed to create video transcode record: %w", err)
		}

		// Create job for this transcode
		if err := uc.createTranscodeJob(ctx, transcodeID); err != nil {
			return nil, fmt.Errorf("failed to create video transcode job: %w", err)
		}

		videoJobs++
	}

	// Detect audio streams
	audioStreams, err := uc.ffprobe.GetAudioStreams(media.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect audio streams: %w", err)
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
			return nil, fmt.Errorf("failed to save audio stream %d: %w", audioInfo.StreamIndex, err)
		}
	}

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
			return nil, fmt.Errorf("failed to create audio transcode record: %w", err)
		}

		// Create job for this transcode
		if err := uc.createTranscodeJob(ctx, transcodeID); err != nil {
			return nil, fmt.Errorf("failed to create audio transcode job: %w", err)
		}

		audioJobs++
	}

	// Detect subtitle streams
	subtitleStreams, err := uc.ffprobe.GetSubtitleStreams(media.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect subtitle streams: %w", err)
	}

	// Save subtitle streams to database
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
			return nil, fmt.Errorf("failed to save subtitle stream %d: %w", subtitleInfo.StreamIndex, err)
		}
	}

	// Create subtitle transcode jobs (one per subtitle stream)
	for _, subtitleStream := range subtitleStreams {
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
			return nil, fmt.Errorf("failed to create subtitle transcode record: %w", err)
		}

		// Create job for this transcode
		if err := uc.createTranscodeJob(ctx, transcodeID); err != nil {
			return nil, fmt.Errorf("failed to create subtitle transcode job: %w", err)
		}

		subtitleJobs++
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

// createTranscodeJob creates a job for a transcode record
func (uc *CreateTranscodeJobsUseCase) createTranscodeJob(ctx context.Context, transcodeID string) error {
	// Create job payload
	payload := struct {
		TranscodeID string `json:"transcode_id"`
	}{
		TranscodeID: transcodeID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal job payload: %w", err)
	}

	// Create job with proper MaxAttempts from config
	job := &domain.Job{
		Type:        shared.JobTypeTranscodeVideo,
		Payload:     payloadBytes,
		Status:      "queued",
		Priority:    5, // Lower priority than library scan (0)
		RunAt:       time.Now(),
		MaxAttempts: uc.config.Job.MaxAttempts, // Set from config
	}

	_, err = uc.jobRepo.Enqueue(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to enqueue transcode job: %w", err)
	}

	return nil
}
