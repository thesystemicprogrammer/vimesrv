package domain

import "time"

// SubtitleStream represents a subtitle track extracted from a media file
type SubtitleStream struct {
	ID          int64
	MediaID     string
	StreamIndex int
	Codec       string
	Language    string
	Title       string
	Forced      bool
	CreatedAt   time.Time
}

// NewSubtitleStream creates a new SubtitleStream with the given parameters
func NewSubtitleStream(mediaID string, streamIndex int, codec, language, title string, forced bool) *SubtitleStream {
	return &SubtitleStream{
		MediaID:     mediaID,
		StreamIndex: streamIndex,
		Codec:       codec,
		Language:    language,
		Title:       title,
		Forced:      forced,
		CreatedAt:   time.Now(),
	}
}
