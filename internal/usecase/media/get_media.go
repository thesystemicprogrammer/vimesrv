package media

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetMediaInput contains the input parameters for getting media
type GetMediaInput struct {
	MediaID string
}

// GetMediaOutput contains the retrieved media and related data
type GetMediaOutput struct {
	Media           *domain.MediaFile
	Transcodes      []*domain.Transcode
	AudioStreams    []*domain.AudioStream
	SubtitleStreams []*domain.SubtitleStream
}

// GetMediaUseCase retrieves a media file with all its related transcode jobs and streams
type GetMediaUseCase struct {
	mediaRepository    ports.MediaRepository
	transcodeRepo      ports.TranscodeRepository
	audioStreamRepo    ports.AudioStreamRepository
	subtitleStreamRepo ports.SubtitleStreamRepository
}

// NewGetMediaUseCase creates a new GetMediaUseCase instance
func NewGetMediaUseCase(
	mediaRepository ports.MediaRepository,
	transcodeRepo ports.TranscodeRepository,
	audioStreamRepo ports.AudioStreamRepository,
	subtitleStreamRepo ports.SubtitleStreamRepository,
) *GetMediaUseCase {
	return &GetMediaUseCase{
		mediaRepository:    mediaRepository,
		transcodeRepo:      transcodeRepo,
		audioStreamRepo:    audioStreamRepo,
		subtitleStreamRepo: subtitleStreamRepo,
	}
}

// Execute retrieves a media file and all related data
func (uc *GetMediaUseCase) Execute(ctx context.Context, input GetMediaInput) (*GetMediaOutput, error) {
	if input.MediaID == "" {
		return nil, fmt.Errorf("media ID is required")
	}

	// Get the media file
	media, err := uc.mediaRepository.Get(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}
	if media == nil {
		return nil, fmt.Errorf("media not found: %s", input.MediaID)
	}

	// Get all transcode jobs for this media
	transcodes, err := uc.transcodeRepo.GetByMediaID(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transcodes: %w", err)
	}

	// Get all audio streams for this media
	audioStreams, err := uc.audioStreamRepo.GetByMediaID(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get audio streams: %w", err)
	}

	// Get all subtitle streams for this media
	subtitleStreams, err := uc.subtitleStreamRepo.GetByMediaID(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtitle streams: %w", err)
	}

	return &GetMediaOutput{
		Media:           media,
		Transcodes:      transcodes,
		AudioStreams:    audioStreams,
		SubtitleStreams: subtitleStreams,
	}, nil
}
