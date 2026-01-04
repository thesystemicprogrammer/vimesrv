package transcode

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
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
	Language      string `json:"language,omitempty"`
	ChannelLayout string `json:"channel_layout,omitempty"`
}

// createTranscodeJob creates a job for a transcode record
func (uc *CreateTranscodeJobsUseCase) createTranscodeJob(ctx context.Context, transcodeID string) error {
	_, err := uc.enqueueJobUseCase.Execute(ctx, job.EnqueueJobInput{
		Type:     shared.JobTypeTranscodeVideo,
		Payload:  TranscodeJobPayload{TranscodeID: transcodeID},
		Priority: shared.JobPriorityTranscode,
	})
	if err != nil {
		return fmt.Errorf("failed to enqueue transcode job: %w", err)
	}

	return nil
}

// createVideoTranscodeJobs creates transcode jobs for each video quality profile.
// Returns the number of jobs created.
func (uc *CreateTranscodeJobsUseCase) createVideoTranscodeJobs(ctx context.Context, media *domain.MediaFile, qualities []config.QualityProfile) (int, error) {
	var videoJobs int

	for _, quality := range qualities {
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
		if err := uc.createTranscodeJob(ctx, transcodeID); err != nil {
			return 0, fmt.Errorf("failed to create video transcode job: %w", err)
		}

		videoJobs++
	}

	return videoJobs, nil
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
			Type: shared.JobTypeTranscodeVideo, // Audio uses same job type
			Payload: TranscodeJobPayload{
				TranscodeID:   transcodeID,
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
			return 0, fmt.Errorf("failed to save subtitle stream %d: %w", subtitleInfo.StreamIndex, err)
		}
	}

	var subtitleJobs int

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
			return 0, fmt.Errorf("failed to create subtitle transcode record: %w", err)
		}

		// Create job for this transcode
		if err := uc.createTranscodeJob(ctx, transcodeID); err != nil {
			return 0, fmt.Errorf("failed to create subtitle transcode job: %w", err)
		}

		subtitleJobs++
	}

	return subtitleJobs, nil
}
