package ports

import (
	"context"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// JobListFilter defines filtering options for listing jobs
type JobListFilter struct {
	Statuses []string  // Filter by status (queued, running, succeeded, dead)
	Types    []string  // Filter by job type
	Since    time.Time // Only include jobs created/updated after this time
	Limit    int       // Maximum number of results
	Offset   int       // Offset for pagination
}

// JobListResult contains the result of a job list query
type JobListResult struct {
	Jobs  []*domain.Job
	Total int
}

type JobRepository interface {
	Enqueue(ctx context.Context, job *domain.Job) (int64, error)
	ClaimNextJobDue(ctx context.Context, workerID string) (*domain.Job, bool, error)
	// ClaimNextJobDueExcludingTypes claims the next due job, excluding specified job types.
	// Used when distributed workers are enabled to prevent local processing of transcode jobs.
	ClaimNextJobDueExcludingTypes(ctx context.Context, workerID string, excludeTypes []string) (*domain.Job, bool, error)
	MarkSuccess(ctx context.Context, jobID int64) error
	Reschedule(ctx context.Context, jobID int64, runAt time.Time, lastErr string) error
	MarkDead(ctx context.Context, jobID int64, lastErr string) error
	FindStuckJobs(ctx context.Context, threshold time.Duration) ([]*domain.Job, error)
	ResetStuckJob(ctx context.Context, jobID int64) error
	// ExistsPendingJobByType checks if there's already a queued or running job of the given type
	// with a payload containing the specified language value
	ExistsPendingJobByType(ctx context.Context, jobType string, language string) (bool, error)
	// ListJobs returns jobs matching the filter criteria
	ListJobs(ctx context.Context, filter JobListFilter) (*JobListResult, error)

	// Get retrieves a job by its ID
	Get(ctx context.Context, jobID int64) (*domain.Job, error)
	// ClaimNextTranscodeJob atomically claims the next queued transcode_video job
	// for processing by a distributed worker. Returns (nil, nil) if no jobs available.
	ClaimNextTranscodeJob(ctx context.Context, workerID string) (*domain.Job, error)
	// CountQueuedTranscodeJobs returns the number of queued transcode_video jobs
	CountQueuedTranscodeJobs(ctx context.Context) (int, error)
}
