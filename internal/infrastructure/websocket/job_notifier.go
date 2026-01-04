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
	WorkerID    string          `json:"worker_id,omitempty"`
	StartedAt   string          `json:"started_at,omitempty"`
	UpdatedAt   string          `json:"updated_at"`
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
	JobID      int64           `json:"job_id"`
	JobType    string          `json:"job_type"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	WorkerID   string          `json:"worker_id,omitempty"`
	StartedAt  string          `json:"started_at,omitempty"`
	FinishedAt string          `json:"finished_at,omitempty"`
	UpdatedAt  string          `json:"updated_at"`
}

// JobFailedPayload is the payload sent when a job fails
type JobFailedPayload struct {
	JobID        int64           `json:"job_id"`
	JobType      string          `json:"job_type"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	ErrorMessage string          `json:"error_message"`
	WorkerID     string          `json:"worker_id,omitempty"`
	StartedAt    string          `json:"started_at,omitempty"`
	FinishedAt   string          `json:"finished_at,omitempty"`
	UpdatedAt    string          `json:"updated_at"`
}

// JobRetryingPayload is the payload sent when a job is being retried
type JobRetryingPayload struct {
	JobID       int64           `json:"job_id"`
	JobType     string          `json:"job_type"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Attempt     int             `json:"attempt"`
	MaxAttempts int             `json:"max_attempts"`
	WorkerID    string          `json:"worker_id,omitempty"`
	UpdatedAt   string          `json:"updated_at"`
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
	progressCache *ProgressCache

	// Batching for job_queued notifications
	pendingMu    sync.Mutex
	pendingJobs  []JobQueuedPayload
	flushTimer   *time.Timer
	timerRunning bool
}

// NewWebSocketJobNotifier creates a new WebSocket-based job notifier
func NewWebSocketJobNotifier(hub *Hub, progressCache *ProgressCache) *WebSocketJobNotifier {
	return &WebSocketJobNotifier{
		hub:           hub,
		flushInterval: DefaultQueuedJobsFlushInterval,
		pendingJobs:   make([]JobQueuedPayload, 0),
		progressCache: progressCache,
	}
}

// NewWebSocketJobNotifierWithInterval creates a new WebSocket-based job notifier with custom flush interval
func NewWebSocketJobNotifierWithInterval(hub *Hub, progressCache *ProgressCache, flushInterval time.Duration) *WebSocketJobNotifier {
	return &WebSocketJobNotifier{
		hub:           hub,
		flushInterval: flushInterval,
		pendingJobs:   make([]JobQueuedPayload, 0),
		progressCache: progressCache,
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

	payload := JobStartedPayload{
		JobID:       job.ID,
		JobType:     job.Type,
		Payload:     job.Payload,
		Attempt:     job.Attempts,
		MaxAttempts: job.MaxAttempts,
		UpdatedAt:   job.UpdatedAt.Format(time.RFC3339),
	}
	if job.WorkerID.Valid {
		payload.WorkerID = job.WorkerID.String
	}
	if job.StartedAt.Valid {
		payload.StartedAt = job.StartedAt.Time.Format(time.RFC3339)
	}

	msg := Message{
		Type:    MessageTypeJobStarted,
		Payload: payload,
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
	// Cache the progress for API retrieval
	if n.progressCache != nil {
		n.progressCache.Set(jobID, progress)
	}

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
	// Remove progress from cache
	if n.progressCache != nil {
		n.progressCache.Delete(job.ID)
	}

	if n.hub == nil {
		return
	}

	payload := JobCompletedPayload{
		JobID:     job.ID,
		JobType:   job.Type,
		Payload:   job.Payload,
		UpdatedAt: job.UpdatedAt.Format(time.RFC3339),
	}
	if job.WorkerID.Valid {
		payload.WorkerID = job.WorkerID.String
	}
	if job.StartedAt.Valid {
		payload.StartedAt = job.StartedAt.Time.Format(time.RFC3339)
	}
	if job.FinishedAt.Valid {
		payload.FinishedAt = job.FinishedAt.Time.Format(time.RFC3339)
	}

	msg := Message{
		Type:    MessageTypeJobCompleted,
		Payload: payload,
	}

	n.hub.Broadcast(msg)
	logger.Debug().
		Int64("job_id", job.ID).
		Str("job_type", job.Type).
		Msg("Broadcast job completed notification")
}

// NotifyJobFailed broadcasts that a job has failed (max attempts exceeded)
func (n *WebSocketJobNotifier) NotifyJobFailed(job *domain.Job, errorMessage string) {
	// Remove progress from cache
	if n.progressCache != nil {
		n.progressCache.Delete(job.ID)
	}

	if n.hub == nil {
		return
	}

	payload := JobFailedPayload{
		JobID:        job.ID,
		JobType:      job.Type,
		Payload:      job.Payload,
		ErrorMessage: errorMessage,
		UpdatedAt:    job.UpdatedAt.Format(time.RFC3339),
	}
	if job.WorkerID.Valid {
		payload.WorkerID = job.WorkerID.String
	}
	if job.StartedAt.Valid {
		payload.StartedAt = job.StartedAt.Time.Format(time.RFC3339)
	}
	if job.FinishedAt.Valid {
		payload.FinishedAt = job.FinishedAt.Time.Format(time.RFC3339)
	}

	msg := Message{
		Type:    MessageTypeJobFailed,
		Payload: payload,
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

	payload := JobRetryingPayload{
		JobID:       job.ID,
		JobType:     job.Type,
		Payload:     job.Payload,
		Attempt:     attempt,
		MaxAttempts: maxAttempts,
		UpdatedAt:   job.UpdatedAt.Format(time.RFC3339),
	}
	if job.WorkerID.Valid {
		payload.WorkerID = job.WorkerID.String
	}

	msg := Message{
		Type:    MessageTypeJobRetrying,
		Payload: payload,
	}

	n.hub.Broadcast(msg)
	logger.Debug().
		Int64("job_id", job.ID).
		Str("job_type", job.Type).
		Int("attempt", attempt).
		Int("max_attempts", maxAttempts).
		Msg("Broadcast job retrying notification")
}
