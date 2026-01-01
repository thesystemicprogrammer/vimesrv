package media

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ListMediaInput contains the input parameters for listing media
type ListMediaInput struct {
	Page    int
	PerPage int
}

// MediaListItem represents a single media item in the list response
type MediaListItem struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Filename       string `json:"filename"`
	Duration       int    `json:"duration"`
	Resolution     string `json:"resolution"`
	Status         string `json:"status"`
	HasSubtitles   bool   `json:"has_subtitles"`
	AudioTracks    int    `json:"audio_tracks"`
	SubtitleTracks int    `json:"subtitle_tracks"`
}

// ListMediaOutput contains the paginated media list
type ListMediaOutput struct {
	Items   []MediaListItem `json:"items"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	PerPage int             `json:"per_page"`
}

// ListMediaUseCase retrieves a paginated list of media files
type ListMediaUseCase struct {
	mediaRepository ports.MediaRepository
}

// NewListMediaUseCase creates a new ListMediaUseCase instance
func NewListMediaUseCase(mediaRepository ports.MediaRepository) *ListMediaUseCase {
	return &ListMediaUseCase{
		mediaRepository: mediaRepository,
	}
}

// Execute retrieves a paginated list of media files
func (uc *ListMediaUseCase) Execute(ctx context.Context, input ListMediaInput) (*ListMediaOutput, error) {
	// Set defaults
	page := input.Page
	perPage := input.PerPage
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100 // Cap at 100
	}

	// Get media list from repository
	mediaFiles, total, err := uc.mediaRepository.List(ctx, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("failed to list media: %w", err)
	}

	// Convert to list items
	items := make([]MediaListItem, len(mediaFiles))
	for i, m := range mediaFiles {
		items[i] = uc.toListItem(m)
	}

	return &ListMediaOutput{
		Items:   items,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

// toListItem converts a domain MediaFile to a MediaListItem
func (uc *ListMediaUseCase) toListItem(m *domain.MediaFile) MediaListItem {
	title := m.Title
	if title == "" {
		title = m.Filename
	}

	return MediaListItem{
		ID:             m.ID,
		Title:          title,
		Filename:       m.Filename,
		Duration:       m.Duration,
		Resolution:     m.Resolution,
		Status:         m.Status,
		HasSubtitles:   m.SubtitleTracks > 0,
		AudioTracks:    m.AudioTracks,
		SubtitleTracks: m.SubtitleTracks,
	}
}
