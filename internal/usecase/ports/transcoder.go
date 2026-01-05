package ports

import "context"

// TranscodeOptions contains options for transcoding operations
type TranscodeOptions struct {
	// Input configuration
	InputPath         string // Path to the source media file
	SourceStreamIndex int    // Source stream index (for audio/subtitle tracks)

	// Output configuration
	OutputPath string // Path to output directory (for video/audio) or file (for subtitles)

	// Video encoding options (for video tracks)
	Width        int    // Target video width in pixels
	Height       int    // Target video height in pixels
	VideoCodec   string // Video codec (e.g., "libx264")
	CRF          int    // Constant Rate Factor (quality-based encoding, 18-28 typical)
	MaxBitrate   int    // Maximum bitrate in kbps
	VideoBitrate int    // Target video bitrate in kbps (legacy, use CRF instead)
	Preset       string // Encoding preset (e.g., "medium", "fast")

	// Audio encoding options (for audio tracks)
	AudioCodec    string // Audio codec (e.g., "aac")
	AudioBitrate  int    // Audio bitrate in kbps
	AudioChannels int    // Number of audio channels (0 = preserve source)

	// Segmentation options (for CMAF output)
	SegmentTime    int    // Segment duration in seconds (default 4)
	SegmentPattern string // Segment filename pattern (default "chunk-%05d.m4s")

	// Track type identifier
	TrackType string // Track type: "video", "audio-0", "audio-1", "subtitle-0", etc.
}

// TranscodeProgress represents the current progress of a transcoding operation
type TranscodeProgress struct {
	Frame      int64   // Current frame number
	FPS        float64 // Current frames per second
	Bitrate    string  // Current bitrate (e.g., "1500kbits/s")
	Time       string  // Current timestamp (HH:MM:SS.ms)
	Speed      string  // Processing speed (e.g., "1.5x")
	Percentage float64 // Progress percentage (0-100)
}

// ProgressCallback is called periodically during transcoding to report progress
type ProgressCallback func(progress TranscodeProgress)

// SegmentInfo contains timing information for a media segment
type SegmentInfo struct {
	Number   int   // Segment number (0-based)
	Duration int64 // Segment duration in milliseconds
}

// Transcoder provides video transcoding capabilities using FFmpeg
type Transcoder interface {
	// IsAvailable checks if the transcoder (FFmpeg) is available and executable
	IsAvailable() error

	// TranscodeVideo transcodes a video stream to the specified quality with CMAF segmentation
	// Creates init.mp4 + chunk-XXX.m4s files in the output directory
	// Returns an error if transcoding fails
	TranscodeVideo(ctx context.Context, opts TranscodeOptions, callback ProgressCallback) error

	// TranscodeAudio transcodes an audio stream to AAC with CMAF segmentation
	// Creates init.mp4 + chunk-XXX.m4s files in the output directory
	// Returns an error if transcoding fails
	TranscodeAudio(ctx context.Context, opts TranscodeOptions, callback ProgressCallback) error

	// ExtractSubtitle extracts a subtitle stream and converts it to WebVTT format
	// Creates a single .vtt file at the specified output path
	// Returns an error if extraction fails
	ExtractSubtitle(ctx context.Context, opts TranscodeOptions) error

	// ProbeSegmentDurations probes all segment files in the output directory
	// and returns their exact durations for accurate DASH manifest generation
	// Returns a slice of SegmentInfo or an error if probing fails
	ProbeSegmentDurations(ctx context.Context, outputPath string) ([]SegmentInfo, error)
}
