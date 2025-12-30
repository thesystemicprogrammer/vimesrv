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
}
