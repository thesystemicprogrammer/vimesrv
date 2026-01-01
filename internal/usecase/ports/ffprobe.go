package ports

// VideoMetadata contains metadata extracted from a video file
type VideoMetadata struct {
	Duration          int      // Duration in seconds
	FileSize          int64    // File size in bytes
	Format            string   // Container format (e.g., "matroska,webm", "mov,mp4,m4a,3gp,3g2,mj2")
	VideoCodec        string   // Video codec (e.g., "h264", "hevc")
	AudioCodecs       []string // Audio codecs (e.g., ["aac", "ac3"])
	Resolution        string   // Resolution string (e.g., "1920x1080")
	Width             int      // Video width in pixels
	Height            int      // Video height in pixels
	Bitrate           int      // Overall bitrate in bits per second
	AudioTracks       int      // Number of audio tracks
	SubtitleTracks    int      // Number of subtitle tracks
	SubtitleLanguages []string // Subtitle languages (e.g., ["eng", "spa"])
}

// AudioStreamInfo contains detailed information about an audio stream
type AudioStreamInfo struct {
	StreamIndex   int    // Stream index in the source file
	Codec         string // Audio codec (e.g., "aac", "ac3", "dts")
	Language      string // Language code (e.g., "eng", "spa")
	Channels      int    // Number of audio channels (e.g., 2 for stereo, 6 for 5.1)
	ChannelLayout string // Channel layout (e.g., "stereo", "5.1(side)")
	SampleRate    int    // Sample rate in Hz (e.g., 48000)
	Title         string // Stream title/name if available
	Bitrate       int    // Bitrate in bits per second (0 if unknown)
}

// SubtitleStreamInfo contains detailed information about a subtitle stream
type SubtitleStreamInfo struct {
	StreamIndex int    // Stream index in the source file
	Codec       string // Subtitle codec (e.g., "srt", "ass", "subrip")
	Language    string // Language code (e.g., "eng", "spa")
	Title       string // Stream title/name if available
	Forced      bool   // Whether this is a forced subtitle track
}

// FFProbeService provides video validation and metadata extraction using ffprobe
type FFProbeService interface {
	// IsAvailable checks if ffprobe is available and executable
	// Returns an error if ffprobe is not available or not working
	IsAvailable() error

	// ValidateVideo checks if the file is a valid video file
	// Returns true if valid, false otherwise
	ValidateVideo(filePath string) (bool, error)

	// ExtractMetadata extracts metadata from a video file
	// Returns VideoMetadata or an error if extraction fails
	ExtractMetadata(filePath string) (*VideoMetadata, error)

	// GetAudioStreams extracts detailed information about all audio streams
	// Returns a slice of AudioStreamInfo or an error if extraction fails
	GetAudioStreams(filePath string) ([]*AudioStreamInfo, error)

	// GetSubtitleStreams extracts detailed information about all subtitle streams
	// Returns a slice of SubtitleStreamInfo or an error if extraction fails
	GetSubtitleStreams(filePath string) ([]*SubtitleStreamInfo, error)
}
