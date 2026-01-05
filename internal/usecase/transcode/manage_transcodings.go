package transcode

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// QualityProfileInfo represents a quality profile with its availability status
type QualityProfileInfo struct {
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	Resolution   string `json:"resolution"`
	HasTranscode bool   `json:"has_transcode"`
	IsQualified  bool   `json:"is_qualified"` // True if profile resolution <= source resolution
}

// AudioStreamInfo represents an audio stream with its transcode status
type AudioStreamInfo struct {
	Index        int    `json:"index"`
	Language     string `json:"language"`
	Title        string `json:"title"`
	Channels     int    `json:"channels"`
	HasTranscode bool   `json:"has_transcode"`
}

// SubtitleStreamInfo represents a subtitle stream with its transcode status
type SubtitleStreamInfo struct {
	Index        int    `json:"index"`
	Language     string `json:"language"`
	Title        string `json:"title"`
	HasTranscode bool   `json:"has_transcode"`
}

// TranscodeInfo represents a transcode record with status
type TranscodeInfo struct {
	ID         string `json:"id"`
	Quality    string `json:"quality,omitempty"`
	TrackIndex int    `json:"track_index"`
	Language   string `json:"language,omitempty"`
	Title      string `json:"title,omitempty"`
	Channels   int    `json:"channels,omitempty"`
	Status     string `json:"status"`
}

// MediaTranscodingDetails contains all transcoding info for a media file
type MediaTranscodingDetails struct {
	MediaID         string               `json:"media_id"`
	Title           string               `json:"title"`
	Filename        string               `json:"filename"`
	Resolution      string               `json:"resolution"`
	Height          int                  `json:"height"`
	VideoTranscodes []TranscodeInfo      `json:"video_transcodes"`
	AudioTranscodes []TranscodeInfo      `json:"audio_transcodes"`
	SubtitleTrans   []TranscodeInfo      `json:"subtitle_transcodes"`
	AudioStreams    []AudioStreamInfo    `json:"available_audio_streams"`
	SubtitleStreams []SubtitleStreamInfo `json:"available_subtitle_streams"`
	QualityProfiles []QualityProfileInfo `json:"qualified_quality_profiles"`
}

// GetTranscodingDetailsInput is the input for GetTranscodingDetailsUseCase
type GetTranscodingDetailsInput struct {
	MediaID string
}

// GetTranscodingDetailsUseCase retrieves transcoding details for a media file
type GetTranscodingDetailsUseCase struct {
	mediaRepo          ports.MediaRepository
	transcodeRepo      ports.TranscodeRepository
	audioStreamRepo    ports.AudioStreamRepository
	subtitleStreamRepo ports.SubtitleStreamRepository
	config             *config.Config
}

// NewGetTranscodingDetailsUseCase creates a new GetTranscodingDetailsUseCase
func NewGetTranscodingDetailsUseCase(
	mediaRepo ports.MediaRepository,
	transcodeRepo ports.TranscodeRepository,
	audioStreamRepo ports.AudioStreamRepository,
	subtitleStreamRepo ports.SubtitleStreamRepository,
	cfg *config.Config,
) *GetTranscodingDetailsUseCase {
	return &GetTranscodingDetailsUseCase{
		mediaRepo:          mediaRepo,
		transcodeRepo:      transcodeRepo,
		audioStreamRepo:    audioStreamRepo,
		subtitleStreamRepo: subtitleStreamRepo,
		config:             cfg,
	}
}

// Execute retrieves transcoding details for a media file
func (uc *GetTranscodingDetailsUseCase) Execute(ctx context.Context, input GetTranscodingDetailsInput) (*MediaTranscodingDetails, error) {
	// Get media file
	media, err := uc.mediaRepo.Get(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	// Get transcodes
	transcodes, err := uc.transcodeRepo.GetByMediaID(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transcodes: %w", err)
	}

	// Get audio streams
	audioStreams, err := uc.audioStreamRepo.GetByMediaID(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get audio streams: %w", err)
	}

	// Get subtitle streams
	subtitleStreams, err := uc.subtitleStreamRepo.GetByMediaID(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtitle streams: %w", err)
	}

	// Build transcode maps for quick lookup
	audioTranscodeMap := make(map[int]*domain.Transcode)
	subtitleTranscodeMap := make(map[int]*domain.Transcode)
	videoTranscodeMap := make(map[string]*domain.Transcode)

	for _, t := range transcodes {
		switch t.TrackType {
		case domain.TrackTypeAudio:
			audioTranscodeMap[t.TrackIndex] = t
		case domain.TrackTypeSubtitle:
			subtitleTranscodeMap[t.TrackIndex] = t
		case domain.TrackTypeVideo:
			videoTranscodeMap[t.Quality] = t
		}
	}

	// Build video transcodes list
	var videoTranscodes []TranscodeInfo
	for _, t := range transcodes {
		if t.TrackType == domain.TrackTypeVideo {
			videoTranscodes = append(videoTranscodes, TranscodeInfo{
				ID:      t.ID,
				Quality: t.Quality,
				Status:  string(t.Status),
			})
		}
	}

	// Build audio transcodes list with stream info
	var audioTranscodes []TranscodeInfo
	for _, t := range transcodes {
		if t.TrackType == domain.TrackTypeAudio {
			info := TranscodeInfo{
				ID:         t.ID,
				TrackIndex: t.TrackIndex,
				Status:     string(t.Status),
			}
			// Add stream info if available
			for _, as := range audioStreams {
				if as.StreamIndex == t.TrackIndex {
					info.Language = as.Language
					info.Title = as.Title
					info.Channels = as.Channels
					break
				}
			}
			audioTranscodes = append(audioTranscodes, info)
		}
	}

	// Build subtitle transcodes list with stream info
	var subtitleTrans []TranscodeInfo
	for _, t := range transcodes {
		if t.TrackType == domain.TrackTypeSubtitle {
			info := TranscodeInfo{
				ID:         t.ID,
				TrackIndex: t.TrackIndex,
				Status:     string(t.Status),
			}
			// Add stream info if available
			for _, ss := range subtitleStreams {
				if ss.StreamIndex == t.TrackIndex {
					info.Language = ss.Language
					info.Title = ss.Title
					break
				}
			}
			subtitleTrans = append(subtitleTrans, info)
		}
	}

	// Build available audio streams info
	var audioStreamInfos []AudioStreamInfo
	for _, as := range audioStreams {
		_, hasTranscode := audioTranscodeMap[as.StreamIndex]
		audioStreamInfos = append(audioStreamInfos, AudioStreamInfo{
			Index:        as.StreamIndex,
			Language:     as.Language,
			Title:        as.Title,
			Channels:     as.Channels,
			HasTranscode: hasTranscode,
		})
	}

	// Build available subtitle streams info
	var subtitleStreamInfos []SubtitleStreamInfo
	for _, ss := range subtitleStreams {
		_, hasTranscode := subtitleTranscodeMap[ss.StreamIndex]
		subtitleStreamInfos = append(subtitleStreamInfos, SubtitleStreamInfo{
			Index:        ss.StreamIndex,
			Language:     ss.Language,
			Title:        ss.Title,
			HasTranscode: hasTranscode,
		})
	}

	// Build quality profiles info (qualified = resolution <= source)
	var qualityProfiles []QualityProfileInfo
	for _, qp := range uc.config.Transcoding.QualityProfiles {
		_, hasTranscode := videoTranscodeMap[qp.Name]

		// Check if profile is qualified (resolution <= source)
		isQualified := true
		if media.Height > 0 {
			targetHeight, err := parseResolutionHeight(qp.Resolution)
			if err == nil {
				isQualified = targetHeight <= media.Height
			}
		}

		qualityProfiles = append(qualityProfiles, QualityProfileInfo{
			Name:         qp.Name,
			Enabled:      qp.Enabled,
			Resolution:   qp.Resolution,
			HasTranscode: hasTranscode,
			IsQualified:  isQualified,
		})
	}

	title := media.Title
	if title == "" {
		title = media.Filename
	}

	return &MediaTranscodingDetails{
		MediaID:         media.ID,
		Title:           title,
		Filename:        media.Filename,
		Resolution:      media.Resolution,
		Height:          media.Height,
		VideoTranscodes: videoTranscodes,
		AudioTranscodes: audioTranscodes,
		SubtitleTrans:   subtitleTrans,
		AudioStreams:    audioStreamInfos,
		SubtitleStreams: subtitleStreamInfos,
		QualityProfiles: qualityProfiles,
	}, nil
}

// AddTranscodingInput is the input for AddTranscodingUseCase
type AddTranscodingInput struct {
	MediaID    string
	Type       string // "video", "audio", "subtitle"
	Quality    string // For video: quality profile name (e.g., "720p")
	TrackIndex int    // For audio/subtitle: stream index
}

// AddTranscodingOutput is the output of AddTranscodingUseCase
type AddTranscodingOutput struct {
	TranscodeID string `json:"transcode_id"`
	JobEnqueued bool   `json:"job_enqueued"`
}

// AddTranscodingUseCase creates a new transcode and enqueues a job
type AddTranscodingUseCase struct {
	mediaRepo          ports.MediaRepository
	transcodeRepo      ports.TranscodeRepository
	audioStreamRepo    ports.AudioStreamRepository
	subtitleStreamRepo ports.SubtitleStreamRepository
	enqueueJobUseCase  *job.EnqueueJobUseCase
	config             *config.Config
}

// NewAddTranscodingUseCase creates a new AddTranscodingUseCase
func NewAddTranscodingUseCase(
	mediaRepo ports.MediaRepository,
	transcodeRepo ports.TranscodeRepository,
	audioStreamRepo ports.AudioStreamRepository,
	subtitleStreamRepo ports.SubtitleStreamRepository,
	enqueueJobUseCase *job.EnqueueJobUseCase,
	cfg *config.Config,
) *AddTranscodingUseCase {
	return &AddTranscodingUseCase{
		mediaRepo:          mediaRepo,
		transcodeRepo:      transcodeRepo,
		audioStreamRepo:    audioStreamRepo,
		subtitleStreamRepo: subtitleStreamRepo,
		enqueueJobUseCase:  enqueueJobUseCase,
		config:             cfg,
	}
}

// Execute creates a new transcode and enqueues a job
func (uc *AddTranscodingUseCase) Execute(ctx context.Context, input AddTranscodingInput) (*AddTranscodingOutput, error) {
	// Get media file
	media, err := uc.mediaRepo.Get(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	var transcodeID string
	var jobType string
	var payload TranscodeJobPayload

	switch input.Type {
	case "video":
		// Validate quality profile exists
		var found bool
		for _, qp := range uc.config.Transcoding.QualityProfiles {
			if qp.Name == input.Quality {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("quality profile %q not found", input.Quality)
		}

		transcodeID = fmt.Sprintf("%s-video-%s", media.ID, input.Quality)
		jobType = shared.JobTypeTranscodeVideo
		payload = TranscodeJobPayload{
			TranscodeID: transcodeID,
			Filename:    media.Filename,
		}

		// Create transcode record
		transcode := domain.NewTranscode(
			transcodeID,
			media.ID,
			input.Quality,
			domain.TrackTypeVideo,
			0,
		)
		if err := uc.transcodeRepo.Create(ctx, transcode); err != nil {
			return nil, fmt.Errorf("failed to create video transcode record: %w", err)
		}

	case "audio":
		// Get audio stream info
		audioStreams, err := uc.audioStreamRepo.GetByMediaID(ctx, input.MediaID)
		if err != nil {
			return nil, fmt.Errorf("failed to get audio streams: %w", err)
		}

		var audioStream *domain.AudioStream
		for _, as := range audioStreams {
			if as.StreamIndex == input.TrackIndex {
				audioStream = as
				break
			}
		}
		if audioStream == nil {
			return nil, fmt.Errorf("audio stream %d not found", input.TrackIndex)
		}

		transcodeID = fmt.Sprintf("%s-audio-%d", media.ID, input.TrackIndex)
		jobType = shared.JobTypeTranscodeAudio
		payload = TranscodeJobPayload{
			TranscodeID:   transcodeID,
			Filename:      media.Filename,
			Language:      audioStream.Language,
			ChannelLayout: audioStream.ChannelLayout,
		}

		// Create transcode record
		transcode := domain.NewTranscode(
			transcodeID,
			media.ID,
			"",
			domain.TrackTypeAudio,
			input.TrackIndex,
		)
		if err := uc.transcodeRepo.Create(ctx, transcode); err != nil {
			return nil, fmt.Errorf("failed to create audio transcode record: %w", err)
		}

	case "subtitle":
		// Get subtitle stream info
		subtitleStreams, err := uc.subtitleStreamRepo.GetByMediaID(ctx, input.MediaID)
		if err != nil {
			return nil, fmt.Errorf("failed to get subtitle streams: %w", err)
		}

		var subtitleStream *domain.SubtitleStream
		for _, ss := range subtitleStreams {
			if ss.StreamIndex == input.TrackIndex {
				subtitleStream = ss
				break
			}
		}
		if subtitleStream == nil {
			return nil, fmt.Errorf("subtitle stream %d not found", input.TrackIndex)
		}

		// Check if subtitle is text-based (can be converted to WebVTT)
		if !subtitleStream.IsTextBased() {
			return nil, fmt.Errorf("subtitle stream %d uses bitmap-based codec '%s' which cannot be converted to WebVTT without OCR", input.TrackIndex, subtitleStream.Codec)
		}

		transcodeID = fmt.Sprintf("%s-subtitle-%d", media.ID, input.TrackIndex)
		jobType = shared.JobTypeTranscodeSubtitle
		payload = TranscodeJobPayload{
			TranscodeID: transcodeID,
			Filename:    media.Filename,
			Language:    subtitleStream.Language,
		}

		// Create transcode record
		transcode := domain.NewTranscode(
			transcodeID,
			media.ID,
			"",
			domain.TrackTypeSubtitle,
			input.TrackIndex,
		)
		if err := uc.transcodeRepo.Create(ctx, transcode); err != nil {
			return nil, fmt.Errorf("failed to create subtitle transcode record: %w", err)
		}

	default:
		return nil, fmt.Errorf("invalid transcode type: %s", input.Type)
	}

	// Enqueue job
	_, err = uc.enqueueJobUseCase.Execute(ctx, job.EnqueueJobInput{
		Type:     jobType,
		Payload:  payload,
		Priority: shared.JobPriorityTranscode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue transcode job: %w", err)
	}

	logger.Info().
		Str("transcode_id", transcodeID).
		Str("type", input.Type).
		Str("media_id", media.ID).
		Msg("Transcode job created and enqueued")

	return &AddTranscodingOutput{
		TranscodeID: transcodeID,
		JobEnqueued: true,
	}, nil
}

// RecreateTranscodingInput is the input for RecreateTranscodingUseCase
type RecreateTranscodingInput struct {
	TranscodeID string
}

// RecreateTranscodingOutput is the output of RecreateTranscodingUseCase
type RecreateTranscodingOutput struct {
	TranscodeID string `json:"transcode_id"`
	JobEnqueued bool   `json:"job_enqueued"`
}

// RecreateTranscodingUseCase deletes output files, resets status, and enqueues a new job
type RecreateTranscodingUseCase struct {
	mediaRepo          ports.MediaRepository
	transcodeRepo      ports.TranscodeRepository
	audioStreamRepo    ports.AudioStreamRepository
	subtitleStreamRepo ports.SubtitleStreamRepository
	fileSystem         ports.FileSystemService
	enqueueJobUseCase  *job.EnqueueJobUseCase
	config             *config.Config
}

// NewRecreateTranscodingUseCase creates a new RecreateTranscodingUseCase
func NewRecreateTranscodingUseCase(
	mediaRepo ports.MediaRepository,
	transcodeRepo ports.TranscodeRepository,
	audioStreamRepo ports.AudioStreamRepository,
	subtitleStreamRepo ports.SubtitleStreamRepository,
	fileSystem ports.FileSystemService,
	enqueueJobUseCase *job.EnqueueJobUseCase,
	cfg *config.Config,
) *RecreateTranscodingUseCase {
	return &RecreateTranscodingUseCase{
		mediaRepo:          mediaRepo,
		transcodeRepo:      transcodeRepo,
		audioStreamRepo:    audioStreamRepo,
		subtitleStreamRepo: subtitleStreamRepo,
		fileSystem:         fileSystem,
		enqueueJobUseCase:  enqueueJobUseCase,
		config:             cfg,
	}
}

// Execute deletes output files, resets status, and enqueues a new job
func (uc *RecreateTranscodingUseCase) Execute(ctx context.Context, input RecreateTranscodingInput) (*RecreateTranscodingOutput, error) {
	// Get transcode
	transcode, err := uc.transcodeRepo.Get(ctx, input.TranscodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transcode: %w", err)
	}

	// Check if processing
	if transcode.IsProcessing() {
		return nil, fmt.Errorf("cannot recreate transcode that is currently processing")
	}

	// Get media file
	media, err := uc.mediaRepo.Get(ctx, transcode.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	// Delete output files if they exist
	if transcode.OutputPath != "" && uc.fileSystem.FileExists(transcode.OutputPath) {
		// For video/audio, output path is a directory containing segments
		// For subtitles, output path is a single .vtt file
		if transcode.TrackType == domain.TrackTypeSubtitle {
			if err := uc.fileSystem.DeleteFile(transcode.OutputPath); err != nil {
				logger.Warn().
					Str("transcode_id", transcode.ID).
					Str("path", transcode.OutputPath).
					Err(err).
					Msg("Failed to delete subtitle file")
			}
		} else {
			if err := uc.fileSystem.RemoveDir(transcode.OutputPath); err != nil {
				logger.Warn().
					Str("transcode_id", transcode.ID).
					Str("path", transcode.OutputPath).
					Err(err).
					Msg("Failed to delete transcode output directory")
			}
		}
	}

	// Reset transcode status to pending
	if err := uc.transcodeRepo.UpdateStatus(ctx, transcode.ID, domain.TranscodePending); err != nil {
		return nil, fmt.Errorf("failed to reset transcode status: %w", err)
	}

	// Build job payload
	var jobType string
	var payload TranscodeJobPayload

	switch transcode.TrackType {
	case domain.TrackTypeVideo:
		jobType = shared.JobTypeTranscodeVideo
		payload = TranscodeJobPayload{
			TranscodeID: transcode.ID,
			Filename:    media.Filename,
		}

	case domain.TrackTypeAudio:
		jobType = shared.JobTypeTranscodeAudio
		// Get audio stream info for language and channel layout
		audioStreams, err := uc.audioStreamRepo.GetByMediaID(ctx, transcode.MediaID)
		if err != nil {
			return nil, fmt.Errorf("failed to get audio streams: %w", err)
		}
		var language, channelLayout string
		for _, as := range audioStreams {
			if as.StreamIndex == transcode.TrackIndex {
				language = as.Language
				channelLayout = as.ChannelLayout
				break
			}
		}
		payload = TranscodeJobPayload{
			TranscodeID:   transcode.ID,
			Filename:      media.Filename,
			Language:      language,
			ChannelLayout: channelLayout,
		}

	case domain.TrackTypeSubtitle:
		jobType = shared.JobTypeTranscodeSubtitle
		// Get subtitle stream info for language and codec validation
		subtitleStreams, err := uc.subtitleStreamRepo.GetByMediaID(ctx, transcode.MediaID)
		if err != nil {
			return nil, fmt.Errorf("failed to get subtitle streams: %w", err)
		}
		var subtitleStream *domain.SubtitleStream
		for _, ss := range subtitleStreams {
			if ss.StreamIndex == transcode.TrackIndex {
				subtitleStream = ss
				break
			}
		}
		if subtitleStream == nil {
			return nil, fmt.Errorf("subtitle stream %d not found", transcode.TrackIndex)
		}

		// Check if subtitle is text-based (can be converted to WebVTT)
		if !subtitleStream.IsTextBased() {
			return nil, fmt.Errorf("subtitle stream %d uses bitmap-based codec '%s' which cannot be converted to WebVTT without OCR", transcode.TrackIndex, subtitleStream.Codec)
		}

		payload = TranscodeJobPayload{
			TranscodeID: transcode.ID,
			Filename:    media.Filename,
			Language:    subtitleStream.Language,
		}
	}

	// Enqueue job
	_, err = uc.enqueueJobUseCase.Execute(ctx, job.EnqueueJobInput{
		Type:     jobType,
		Payload:  payload,
		Priority: shared.JobPriorityTranscode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue transcode job: %w", err)
	}

	logger.Info().
		Str("transcode_id", transcode.ID).
		Str("type", string(transcode.TrackType)).
		Str("media_id", media.ID).
		Msg("Transcode recreated and job enqueued")

	return &RecreateTranscodingOutput{
		TranscodeID: transcode.ID,
		JobEnqueued: true,
	}, nil
}

// DeleteTranscodingInput is the input for DeleteTranscodingUseCase
type DeleteTranscodingInput struct {
	TranscodeID string
}

// DeleteTranscodingOutput is the output of DeleteTranscodingUseCase
type DeleteTranscodingOutput struct {
	Deleted bool `json:"deleted"`
}

// DeleteTranscodingUseCase deletes a transcode and its output files
type DeleteTranscodingUseCase struct {
	transcodeRepo ports.TranscodeRepository
	fileSystem    ports.FileSystemService
	config        *config.Config
}

// NewDeleteTranscodingUseCase creates a new DeleteTranscodingUseCase
func NewDeleteTranscodingUseCase(
	transcodeRepo ports.TranscodeRepository,
	fileSystem ports.FileSystemService,
	cfg *config.Config,
) *DeleteTranscodingUseCase {
	return &DeleteTranscodingUseCase{
		transcodeRepo: transcodeRepo,
		fileSystem:    fileSystem,
		config:        cfg,
	}
}

// Execute deletes a transcode and its output files
func (uc *DeleteTranscodingUseCase) Execute(ctx context.Context, input DeleteTranscodingInput) (*DeleteTranscodingOutput, error) {
	// Get transcode
	transcode, err := uc.transcodeRepo.Get(ctx, input.TranscodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transcode: %w", err)
	}

	// Check if processing
	if transcode.IsProcessing() {
		return nil, fmt.Errorf("cannot delete transcode that is currently processing")
	}

	// Delete output files if they exist
	if transcode.OutputPath != "" && uc.fileSystem.FileExists(transcode.OutputPath) {
		// For video/audio, output path is a directory containing segments
		// For subtitles, output path is a single .vtt file
		if transcode.TrackType == domain.TrackTypeSubtitle {
			if err := uc.fileSystem.DeleteFile(transcode.OutputPath); err != nil {
				logger.Warn().
					Str("transcode_id", transcode.ID).
					Str("path", transcode.OutputPath).
					Err(err).
					Msg("Failed to delete subtitle file")
			}
		} else {
			if err := uc.fileSystem.RemoveDir(transcode.OutputPath); err != nil {
				logger.Warn().
					Str("transcode_id", transcode.ID).
					Str("path", transcode.OutputPath).
					Err(err).
					Msg("Failed to delete transcode output directory")
			}
		}
	}

	// Delete transcode record
	if err := uc.transcodeRepo.Delete(ctx, transcode.ID); err != nil {
		return nil, fmt.Errorf("failed to delete transcode record: %w", err)
	}

	logger.Info().
		Str("transcode_id", transcode.ID).
		Str("type", string(transcode.TrackType)).
		Msg("Transcode deleted")

	return &DeleteTranscodingOutput{
		Deleted: true,
	}, nil
}

// SearchMediaInput is the input for SearchMediaForTranscodingsUseCase
type SearchMediaInput struct {
	Query string
	Limit int
}

// MediaSearchResult is a lightweight media result for the admin search
type MediaSearchResult struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Filename   string `json:"filename"`
	Resolution string `json:"resolution"`
}

// SearchMediaOutput is the output of SearchMediaForTranscodingsUseCase
type SearchMediaOutput struct {
	Results []MediaSearchResult `json:"results"`
	Count   int                 `json:"count"`
}

// SearchMediaForTranscodingsUseCase searches media files for the transcoding admin
type SearchMediaForTranscodingsUseCase struct {
	mediaRepo ports.MediaRepository
}

// NewSearchMediaForTranscodingsUseCase creates a new SearchMediaForTranscodingsUseCase
func NewSearchMediaForTranscodingsUseCase(
	mediaRepo ports.MediaRepository,
) *SearchMediaForTranscodingsUseCase {
	return &SearchMediaForTranscodingsUseCase{
		mediaRepo: mediaRepo,
	}
}

// Execute searches media files by title, filename, or ID
func (uc *SearchMediaForTranscodingsUseCase) Execute(ctx context.Context, input SearchMediaInput) (*SearchMediaOutput, error) {
	if input.Limit <= 0 {
		input.Limit = 20
	}

	query := strings.TrimSpace(input.Query)
	if query == "" {
		return &SearchMediaOutput{
			Results: []MediaSearchResult{},
			Count:   0,
		}, nil
	}

	// Check if query looks like a UUID (exact match on ID)
	if isUUID(query) {
		media, err := uc.mediaRepo.Get(ctx, query)
		if err != nil {
			// Not found, return empty
			return &SearchMediaOutput{
				Results: []MediaSearchResult{},
				Count:   0,
			}, nil
		}

		title := media.Title
		if title == "" {
			title = media.Filename
		}

		return &SearchMediaOutput{
			Results: []MediaSearchResult{
				{
					ID:         media.ID,
					Title:      title,
					Filename:   media.Filename,
					Resolution: media.Resolution,
				},
			},
			Count: 1,
		}, nil
	}

	// Search by title/filename
	mediaFiles, err := uc.mediaRepo.Search(ctx, query, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search media: %w", err)
	}

	results := make([]MediaSearchResult, len(mediaFiles))
	for i, m := range mediaFiles {
		title := m.Title
		if title == "" {
			title = m.Filename
		}
		results[i] = MediaSearchResult{
			ID:         m.ID,
			Title:      title,
			Filename:   m.Filename,
			Resolution: m.Resolution,
		}
	}

	return &SearchMediaOutput{
		Results: results,
		Count:   len(results),
	}, nil
}

// isUUID checks if a string looks like a UUID
func isUUID(s string) bool {
	// Simple check: UUIDs are 36 characters with hyphens at specific positions
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// GetQualityProfilesOutput is the output of GetQualityProfilesUseCase
type GetQualityProfilesOutput struct {
	Profiles []config.QualityProfile `json:"quality_profiles"`
}

// GetQualityProfilesUseCase returns all quality profiles from config
type GetQualityProfilesUseCase struct {
	config *config.Config
}

// NewGetQualityProfilesUseCase creates a new GetQualityProfilesUseCase
func NewGetQualityProfilesUseCase(cfg *config.Config) *GetQualityProfilesUseCase {
	return &GetQualityProfilesUseCase{config: cfg}
}

// Execute returns all quality profiles
func (uc *GetQualityProfilesUseCase) Execute(ctx context.Context) (*GetQualityProfilesOutput, error) {
	return &GetQualityProfilesOutput{
		Profiles: uc.config.Transcoding.QualityProfiles,
	}, nil
}

// getTranscodeOutputDir returns the output directory path for a transcode
func getTranscodeOutputDir(mediaPath, mediaID string, trackType domain.TrackType, quality string, trackIndex int) string {
	baseDir := filepath.Join(mediaPath, mediaID, "transcoded")

	switch trackType {
	case domain.TrackTypeVideo:
		return filepath.Join(baseDir, quality, "video")
	case domain.TrackTypeAudio:
		return filepath.Join(baseDir, fmt.Sprintf("audio-%d", trackIndex))
	case domain.TrackTypeSubtitle:
		return filepath.Join(baseDir, fmt.Sprintf("subtitle-%d.vtt", trackIndex))
	}
	return baseDir
}

// parseResolutionHeightFromName extracts height from a quality name like "720p"
func parseResolutionHeightFromName(name string) (int, error) {
	// Handle "original" as a special case
	if name == "original" {
		return 0, nil
	}

	// Try to parse "720p" format
	if strings.HasSuffix(name, "p") {
		heightStr := strings.TrimSuffix(name, "p")
		height, err := strconv.Atoi(heightStr)
		if err != nil {
			return 0, fmt.Errorf("invalid quality name format: %s", name)
		}
		return height, nil
	}

	return 0, fmt.Errorf("invalid quality name format: %s", name)
}
