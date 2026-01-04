package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// DefaultQueuedJobsFlushInterval is the default interval for batching job_queued notifications
const DefaultQueuedJobsFlushInterval = 500 * time.Millisecond

// Message types for job notifications
const (
	MessageTypeJobsQueued   = "jobs_queued"
	MessageTypeJobStarted   = "job_started"
	MessageTypeJobProgress  = "job_progress"
	MessageTypeJobCompleted = "job_completed"
	MessageTypeJobFailed    = "job_failed"
	MessageTypeJobRetrying  = "job_retrying"
)

// JobStartedPayload is the payload sent when a job starts processing
type JobStartedPayload struct {
	JobID       int64           `json:"job_id"`
	JobType     string          `json:"job_type"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Attempt     int             `json:"attempt"`
	MaxAttempts int             `json:"max_attempts"`
}

// JobProgressPayload is the payload sent during job progress updates
type JobProgressPayload struct {
	JobID      int64   `json:"job_id"`
	JobType    string  `json:"job_type"`
	Frame      int64   `json:"frame"`
	FPS        float64 `json:"fps"`
	Bitrate    string  `json:"bitrate"`
	TotalSize  int64   `json:"total_size"`
	Time       string  `json:"time"`
	Speed      string  `json:"speed"`
	Percentage float64 `json:"percentage"`
	Message    string  `json:"message,omitempty"`
}

// JobCompletedPayload is the payload sent when a job completes successfully
type JobCompletedPayload struct {
	JobID   int64           `json:"job_id"`
	JobType string          `json:"job_type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// JobFailedPayload is the payload sent when a job fails
type JobFailedPayload struct {
	JobID        int64           `json:"job_id"`
	JobType      string          `json:"job_type"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	ErrorMessage string          `json:"error_message"`
}

// JobRetryingPayload is the payload sent when a job is being retried
type JobRetryingPayload struct {
	JobID       int64           `json:"job_id"`
	JobType     string          `json:"job_type"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Attempt     int             `json:"attempt"`
	MaxAttempts int             `json:"max_attempts"`
}

// JobQueuedPayload represents a single queued job in the jobs_queued message
type JobQueuedPayload struct {
	JobID       int64           `json:"job_id"`
	JobType     string          `json:"job_type"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Priority    int             `json:"priority"`
	MaxAttempts int             `json:"max_attempts"`
	CreatedAt   string          `json:"created_at"`
}

// JobsQueuedPayload is the payload sent when jobs are queued (batched)
type JobsQueuedPayload struct {
	Jobs []JobQueuedPayload `json:"jobs"`
}

// WebSocketJobNotifier implements JobNotifier using WebSocket broadcasts
type WebSocketJobNotifier struct {
	hub           *Hub
	flushInterval time.Duration

	// Batching for job_queued notifications
	pendingMu    sync.Mutex
	pendingJobs  []JobQueuedPayload
	flushTimer   *time.Timer
	timerRunning bool
}

// NewWebSocketJobNotifier creates a new WebSocket-based job notifier
func NewWebSocketJobNotifier(hub *Hub) *WebSocketJobNotifier {
	return &WebSocketJobNotifier{
		hub:           hub,
		flushInterval: DefaultQueuedJobsFlushInterval,
		pendingJobs:   make([]JobQueuedPayload, 0),
	}
}

// NewWebSocketJobNotifierWithInterval creates a new WebSocket-based job notifier with custom flush interval
func NewWebSocketJobNotifierWithInterval(hub *Hub, flushInterval time.Duration) *WebSocketJobNotifier {
	return &WebSocketJobNotifier{
		hub:           hub,
		flushInterval: flushInterval,
		pendingJobs:   make([]JobQueuedPayload, 0),
	}
}

// NotifyJobQueued adds a job to the pending batch and schedules a flush.
// Jobs are batched and sent together after the flush interval.
func (n *WebSocketJobNotifier) NotifyJobQueued(job *domain.Job) {
	if n.hub == nil {
		return
	}

	payload := JobQueuedPayload{
		JobID:       job.ID,
		JobType:     job.Type,
		Payload:     job.Payload,
		Priority:    job.Priority,
		MaxAttempts: job.MaxAttempts,
		CreatedAt:   job.CreatedAt.Format(time.RFC3339),
	}

	n.pendingMu.Lock()
	defer n.pendingMu.Unlock()

	n.pendingJobs = append(n.pendingJobs, payload)

	// Start timer on first job in batch
	if !n.timerRunning {
		n.timerRunning = true
		n.flushTimer = time.AfterFunc(n.flushInterval, n.flushQueuedJobs)
	}
}

// flushQueuedJobs broadcasts all pending queued jobs and resets the batch
func (n *WebSocketJobNotifier) flushQueuedJobs() {
	n.pendingMu.Lock()
	jobs := n.pendingJobs
	n.pendingJobs = make([]JobQueuedPayload, 0)
	n.timerRunning = false
	n.flushTimer = nil
	n.pendingMu.Unlock()

	if len(jobs) == 0 {
		return
	}

	msg := Message{
		Type:    MessageTypeJobsQueued,
		Payload: JobsQueuedPayload{Jobs: jobs},
	}

	n.hub.Broadcast(msg)
	logger.Debug().
		Int("count", len(jobs)).
		Msg("Broadcast jobs queued notification")
}

// Stop flushes any pending notifications. Call this on shutdown.
func (n *WebSocketJobNotifier) Stop() {
	n.pendingMu.Lock()
	if n.flushTimer != nil {
		n.flushTimer.Stop()
	}
	n.pendingMu.Unlock()

	// Flush remaining jobs
	n.flushQueuedJobs()
}

// NotifyJobStarted broadcasts that a job has started processing
func (n *WebSocketJobNotifier) NotifyJobStarted(job *domain.Job) {
	if n.hub == nil {
		return
	}

	msg := Message{
		Type: MessageTypeJobStarted,
		Payload: JobStartedPayload{
			JobID:       job.ID,
			JobType:     job.Type,
			Payload:     job.Payload,
			Attempt:     job.Attempts,
			MaxAttempts: job.MaxAttempts,
		},
	}

	n.hub.Broadcast(msg)
	logger.Debug().
		Int64("job_id", job.ID).
		Str("job_type", job.Type).
		Int("attempt", job.Attempts).
		Msg("Broadcast job started notification")
}

// NotifyJobProgress broadcasts progress updates for a running job
func (n *WebSocketJobNotifier) NotifyJobProgress(jobID int64, jobType string, progress ports.JobProgress) {
	if n.hub == nil {
		return
	}

	msg := Message{
		Type: MessageTypeJobProgress,
		Payload: JobProgressPayload{
			JobID:      jobID,
			JobType:    jobType,
			Frame:      progress.Frame,
			FPS:        progress.FPS,
			Bitrate:    progress.Bitrate,
			TotalSize:  progress.TotalSize,
			Time:       progress.Time,
			Speed:      progress.Speed,
			Percentage: progress.Percentage,
			Message:    progress.Message,
		},
	}

	n.hub.Broadcast(msg)
	// No logging here to avoid flooding logs - progress is already throttled
}

// NotifyJobCompleted broadcasts that a job has completed successfully
func (n *WebSocketJobNotifier) NotifyJobCompleted(job *domain.Job) {
	if n.hub == nil {
		return
	}

	msg := Message{
		Type: MessageTypeJobCompleted,
		Payload: JobCompletedPayload{
			JobID:   job.ID,
			JobType: job.Type,
			Payload: job.Payload,
		},
	}

	n.hub.Broadcast(msg)
	logger.Debug().
		Int64("job_id", job.ID).
		Str("job_type", job.Type).
		Msg("Broadcast job completed notification")
}

// NotifyJobFailed broadcasts that a job has failed (max attempts exceeded)
func (n *WebSocketJobNotifier) NotifyJobFailed(job *domain.Job, errorMessage string) {
	if n.hub == nil {
		return
	}

	msg := Message{
		Type: MessageTypeJobFailed,
		Payload: JobFailedPayload{
			JobID:        job.ID,
			JobType:      job.Type,
			Payload:      job.Payload,
			ErrorMessage: errorMessage,
		},
	}

	n.hub.Broadcast(msg)
	logger.Debug().
		Int64("job_id", job.ID).
		Str("job_type", job.Type).
		Str("error", errorMessage).
		Msg("Broadcast job failed notification")
}

// NotifyJobRetrying broadcasts that a job is being retried
func (n *WebSocketJobNotifier) NotifyJobRetrying(job *domain.Job, attempt int, maxAttempts int) {
	if n.hub == nil {
		return
	}

	msg := Message{
		Type: MessageTypeJobRetrying,
		Payload: JobRetryingPayload{
			JobID:       job.ID,
			JobType:     job.Type,
			Payload:     job.Payload,
			Attempt:     attempt,
			MaxAttempts: maxAttempts,
		},
	}

	n.hub.Broadcast(msg)
	logger.Debug().
		Int64("job_id", job.ID).
		Str("job_type", job.Type).
		Int("attempt", attempt).
		Int("max_attempts", maxAttempts).
		Msg("Broadcast job retrying notification")
}
