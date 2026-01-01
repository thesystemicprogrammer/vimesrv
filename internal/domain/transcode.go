package domain

import "time"

// TranscodeStatus represents the status of a transcode task
type TranscodeStatus string

const (
	TranscodePending    TranscodeStatus = "pending"
	TranscodeProcessing TranscodeStatus = "processing"
	TranscodeCompleted  TranscodeStatus = "completed"
	TranscodeFailed     TranscodeStatus = "failed"
	TranscodeCancelled  TranscodeStatus = "cancelled"
)

// TrackType represents the type of media track being transcoded
type TrackType string

const (
	TrackTypeVideo    TrackType = "video"
	TrackTypeAudio    TrackType = "audio"
	TrackTypeSubtitle TrackType = "subtitle"
)

// Transcode represents an individual transcoding task
// Note: Progress, timing, and error details are stored in the associated job
type Transcode struct {
	ID         string
	MediaID    string
	Quality    string // "360p", "720p", "" for shared audio/subtitle
	TrackType  TrackType
	TrackIndex int
	Status     TranscodeStatus
	OutputPath string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewTranscode creates a new Transcode with default values
func NewTranscode(id, mediaID, quality string, trackType TrackType, trackIndex int) *Transcode {
	now := time.Now()
	return &Transcode{
		ID:         id,
		MediaID:    mediaID,
		Quality:    quality,
		TrackType:  trackType,
		TrackIndex: trackIndex,
		Status:     TranscodePending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// MarkProcessing marks the transcode as currently processing
func (t *Transcode) MarkProcessing() {
	t.Status = TranscodeProcessing
	t.UpdatedAt = time.Now()
}

// MarkCompleted marks the transcode as successfully completed
func (t *Transcode) MarkCompleted(outputPath string) {
	t.Status = TranscodeCompleted
	t.OutputPath = outputPath
	t.UpdatedAt = time.Now()
}

// MarkFailed marks the transcode as failed
func (t *Transcode) MarkFailed() {
	t.Status = TranscodeFailed
	t.UpdatedAt = time.Now()
}

// MarkCancelled marks the transcode as cancelled
func (t *Transcode) MarkCancelled() {
	t.Status = TranscodeCancelled
	t.UpdatedAt = time.Now()
}

// IsPending returns true if the transcode is pending
func (t *Transcode) IsPending() bool {
	return t.Status == TranscodePending
}

// IsProcessing returns true if the transcode is currently processing
func (t *Transcode) IsProcessing() bool {
	return t.Status == TranscodeProcessing
}

// IsCompleted returns true if the transcode completed successfully
func (t *Transcode) IsCompleted() bool {
	return t.Status == TranscodeCompleted
}

// IsFailed returns true if the transcode failed
func (t *Transcode) IsFailed() bool {
	return t.Status == TranscodeFailed
}

// IsCancelled returns true if the transcode was cancelled
func (t *Transcode) IsCancelled() bool {
	return t.Status == TranscodeCancelled
}
