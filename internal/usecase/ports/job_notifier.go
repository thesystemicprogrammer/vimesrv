package ports

import "github.com/thesystemicprogrammer/vimesrv/internal/domain"

// JobNotifier provides real-time notifications about job state changes.
// Implementations can use WebSocket, Server-Sent Events, or other push mechanisms.
type JobNotifier interface {
	// NotifyJobCompleted broadcasts that a job has completed successfully
	NotifyJobCompleted(job *domain.Job)

	// NotifyJobFailed broadcasts that a job has failed (max attempts exceeded)
	NotifyJobFailed(job *domain.Job, errorMessage string)

	// NotifyJobRetrying broadcasts that a job is being retried
	NotifyJobRetrying(job *domain.Job, attempt int, maxAttempts int)
}

// NoOpJobNotifier is a no-op implementation for when notifications are disabled
type NoOpJobNotifier struct{}

func (n *NoOpJobNotifier) NotifyJobCompleted(job *domain.Job) {}

func (n *NoOpJobNotifier) NotifyJobFailed(job *domain.Job, errorMessage string) {}

func (n *NoOpJobNotifier) NotifyJobRetrying(job *domain.Job, attempt int, maxAttempts int) {}
