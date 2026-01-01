package domain

import "time"

// AudioStream represents an audio track extracted from a media file
type AudioStream struct {
	ID            int64
	MediaID       string
	StreamIndex   int
	Codec         string
	Language      string
	Channels      int
	ChannelLayout string
	SampleRate    int
	Title         string
	CreatedAt     time.Time
}

// NewAudioStream creates a new AudioStream with the given parameters
func NewAudioStream(mediaID string, streamIndex int, codec, language string, channels int, channelLayout string, sampleRate int, title string) *AudioStream {
	return &AudioStream{
		MediaID:       mediaID,
		StreamIndex:   streamIndex,
		Codec:         codec,
		Language:      language,
		Channels:      channels,
		ChannelLayout: channelLayout,
		SampleRate:    sampleRate,
		Title:         title,
		CreatedAt:     time.Now(),
	}
}
