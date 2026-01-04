package ports

import (
	"context"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

type ScheduleRepository interface {
	Upsert(ctx context.Context, s *domain.Schedule) (int64, error)
	GetByName(ctx context.Context, name string) (*domain.Schedule, error)
	SetNextRunIfNull(ctx context.Context, id int64, next time.Time) error
	SetNextRun(ctx context.Context, id int64, next time.Time) error
	ListDue(ctx context.Context, limit int) ([]*domain.Schedule, error)
	// AdvanceAndEnqueue atomically advances the schedule's next_run_at and enqueues a job.
	// Returns the created job with its ID populated, or nil if the schedule was already advanced by another process.
	AdvanceAndEnqueue(ctx context.Context, scheduleID int64, next time.Time, jobProto *domain.Job) (*domain.Job, error)
}
