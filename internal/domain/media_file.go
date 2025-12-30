package domain

import "time"

// Media file status constants
const (
	MediaStatusProcessing = "processing"
	MediaStatusReady      = "ready"
	MediaStatusError      = "error"
)

// MediaFile represents a video file in the media library
type MediaFile struct {
	ID                string    `json:"id"`
	Fingerprint       string    `json:"fingerprint"`
	FilePath          string    `json:"file_path"`
	OriginalFilename  string    `json:"original_filename"`
	Filename          string    `json:"filename"`
	Title             string    `json:"title,omitempty"`
	Duration          int       `json:"duration"`
	FileSize          int64     `json:"file_size"`
	Format            string    `json:"format,omitempty"`
	VideoCodec        string    `json:"video_codec,omitempty"`
	AudioCodecs       []string  `json:"audio_codecs,omitempty"`
	Resolution        string    `json:"resolution,omitempty"`
	Width             int       `json:"width"`
	Height            int       `json:"height"`
	Bitrate           int       `json:"bitrate"`
	AudioTracks       int       `json:"audio_tracks"`
	SubtitleTracks    int       `json:"subtitle_tracks"`
	SubtitleLanguages []string  `json:"subtitle_languages,omitempty"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	ScannedAt         time.Time `json:"scanned_at"`
}

// NewMediaFile creates a new MediaFile with default values
func NewMediaFile(id, fingerprint, filePath, originalFilename, filename string) *MediaFile {
	now := time.Now()
	return &MediaFile{
		ID:                id,
		Fingerprint:       fingerprint,
		FilePath:          filePath,
		OriginalFilename:  originalFilename,
		Filename:          filename,
		Status:            MediaStatusReady,
		CreatedAt:         now,
		UpdatedAt:         now,
		ScannedAt:         now,
		AudioCodecs:       []string{},
		SubtitleLanguages: []string{},
	}
}

// IsValid checks if the media file status is valid
func (m *MediaFile) IsValid() bool {
	return m.Status == MediaStatusProcessing ||
		m.Status == MediaStatusReady ||
		m.Status == MediaStatusError
}

// SetProcessing sets the status to processing
func (m *MediaFile) SetProcessing() {
	m.Status = MediaStatusProcessing
	m.UpdatedAt = time.Now()
}

// SetReady sets the status to ready
func (m *MediaFile) SetReady() {
	m.Status = MediaStatusReady
	m.UpdatedAt = time.Now()
}

// SetError sets the status to error
func (m *MediaFile) SetError() {
	m.Status = MediaStatusError
	m.UpdatedAt = time.Now()
}
