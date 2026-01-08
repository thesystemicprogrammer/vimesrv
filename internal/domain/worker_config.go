package domain

import "time"

// WorkerType distinguishes between local (server) and distributed (remote) workers
type WorkerType string

const (
	WorkerTypeLocal       WorkerType = "local"
	WorkerTypeDistributed WorkerType = "distributed"
)

// WorkerConfig represents the configuration for a job processing worker.
// For local workers, this controls whether the worker is active and what job types it accepts.
// For distributed workers, these settings are configured via the admin UI.
type WorkerConfig struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	WorkerType   WorkerType `json:"worker_type"`
	AcceptsVideo bool       `json:"accepts_video"`
	AcceptsAudio bool       `json:"accepts_audio"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// AllowedTranscodeJobTypes returns the transcode job types this worker can process.
// Subtitles are always processed regardless of configuration.
func (c *WorkerConfig) AllowedTranscodeJobTypes() []string {
	// Always process subtitles
	types := []string{"transcode_subtitle"}
	if c.AcceptsVideo {
		types = append(types, "transcode_video")
	}
	if c.AcceptsAudio {
		types = append(types, "transcode_audio")
	}
	return types
}

// IsEnabled returns true if the worker is configured to process any transcode jobs.
// A worker that accepts neither video nor audio will still process subtitles.
func (c *WorkerConfig) IsEnabled() bool {
	// Workers always process subtitles, so they're always "enabled" in that sense
	// This method indicates if they process video or audio
	return c.AcceptsVideo || c.AcceptsAudio
}

// Copy returns a deep copy of the WorkerConfig
func (c *WorkerConfig) Copy() *WorkerConfig {
	return &WorkerConfig{
		ID:           c.ID,
		Name:         c.Name,
		WorkerType:   c.WorkerType,
		AcceptsVideo: c.AcceptsVideo,
		AcceptsAudio: c.AcceptsAudio,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}
