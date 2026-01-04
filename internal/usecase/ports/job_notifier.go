package ports

import "github.com/thesystemicprogrammer/vimesrv/internal/domain"

// JobProgress contains detailed progress information for a running job.
// Currently used primarily for transcoding jobs which report detailed ffmpeg stats.
type JobProgress struct {
	Frame      int64   `json:"frame"`             // Current frame number
	FPS        float64 `json:"fps"`               // Current encoding FPS
	Bitrate    string  `json:"bitrate"`           // Current bitrate (e.g. "1500kbits/s")
	TotalSize  int64   `json:"total_size"`        // Output size in bytes
	Time       string  `json:"time"`              // Current time position (e.g. "00:01:30.50")
	Speed      string  `json:"speed"`             // Encoding speed (e.g. "2.5x")
	Percentage float64 `json:"percentage"`        // Overall percentage complete (0-100)
	Message    string  `json:"message,omitempty"` // Optional status message
}

// JobNotifier provides real-time notifications about job state changes.
// Implementations can use WebSocket, Server-Sent Events, or other push mechanisms.
type JobNotifier interface {
	// NotifyJobQueued broadcasts that a job has been queued.
	// Implementations may batch multiple calls and send them together.
	NotifyJobQueued(job *domain.Job)

	// NotifyJobStarted broadcasts that a job has started processing
	NotifyJobStarted(job *domain.Job)

	// NotifyJobProgress broadcasts progress updates for a running job.
	// This should be called periodically (throttled) during long-running jobs.
	NotifyJobProgress(jobID int64, jobType string, progress JobProgress)

	// NotifyJobCompleted broadcasts that a job has completed successfully
	NotifyJobCompleted(job *domain.Job)

	// NotifyJobFailed broadcasts that a job has failed (max attempts exceeded)
	NotifyJobFailed(job *domain.Job, errorMessage string)

	// NotifyJobRetrying broadcasts that a job is being retried
	NotifyJobRetrying(job *domain.Job, attempt int, maxAttempts int)
}

// NoOpJobNotifier is a no-op implementation for when notifications are disabled
type NoOpJobNotifier struct{}

func (n *NoOpJobNotifier) NotifyJobQueued(job *domain.Job) {}

func (n *NoOpJobNotifier) NotifyJobStarted(job *domain.Job) {}

func (n *NoOpJobNotifier) NotifyJobProgress(jobID int64, jobType string, progress JobProgress) {}

func (n *NoOpJobNotifier) NotifyJobCompleted(job *domain.Job) {}

func (n *NoOpJobNotifier) NotifyJobFailed(job *domain.Job, errorMessage string) {}

func (n *NoOpJobNotifier) NotifyJobRetrying(job *domain.Job, attempt int, maxAttempts int) {}
