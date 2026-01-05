package domain

import (
	"strings"
	"time"
)

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

// textBasedSubtitleCodecs contains subtitle codecs that can be converted to WebVTT.
// Bitmap-based subtitles (PGS, DVD, DVB) cannot be converted to text without OCR.
var textBasedSubtitleCodecs = map[string]bool{
	"subrip":   true, // SRT format
	"srt":      true, // SRT format (alternate name)
	"ass":      true, // Advanced SubStation Alpha
	"ssa":      true, // SubStation Alpha
	"webvtt":   true, // WebVTT (already in target format)
	"mov_text": true, // QuickTime text subtitles
	"text":     true, // Generic text subtitles
}

// IsTextBased returns true if this subtitle stream uses a text-based codec
// that can be converted to WebVTT. Bitmap-based subtitles like PGS (hdmv_pgs_subtitle)
// and DVD subtitles cannot be converted to text formats without OCR.
func (s *SubtitleStream) IsTextBased() bool {
	return textBasedSubtitleCodecs[strings.ToLower(s.Codec)]
}
