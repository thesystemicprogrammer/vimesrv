package ports

import (
	"context"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

type JobRepository interface {
	Enqueue(ctx context.Context, job *domain.Job) (int64, error)
	ClaimNextJobDue(ctx context.Context, workerID string) (*domain.Job, bool, error)
	MarkSuccess(ctx context.Context, jobID int64) error
	Reschedule(ctx context.Context, jobID int64, runAt time.Time, lastErr string) error
	MarkDead(ctx context.Context, jobID int64, lastErr string) error
	FindStuckJobs(ctx context.Context, threshold time.Duration) ([]*domain.Job, error)
	ResetStuckJob(ctx context.Context, jobID int64) error
}
