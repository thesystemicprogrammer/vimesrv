package websocket

import (
	"encoding/json"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

// Message types for job notifications
const (
	MessageTypeJobCompleted = "job_completed"
	MessageTypeJobFailed    = "job_failed"
	MessageTypeJobRetrying  = "job_retrying"
)

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

// WebSocketJobNotifier implements JobNotifier using WebSocket broadcasts
type WebSocketJobNotifier struct {
	hub *Hub
}

// NewWebSocketJobNotifier creates a new WebSocket-based job notifier
func NewWebSocketJobNotifier(hub *Hub) *WebSocketJobNotifier {
	return &WebSocketJobNotifier{
		hub: hub,
	}
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
