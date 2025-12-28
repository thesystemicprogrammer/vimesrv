package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

type JobRepository struct {
	db *sql.DB
}

func NewJobRepository(db database.DB) *JobRepository {
	return &JobRepository{db: db.DB}
}

func (repo *JobRepository) Enqueue(ctx context.Context, j *domain.Job) (int64, error) {
	const command = `
	INSERT INTO jobs (type, payload, status, priority, run_at, attempt, max_attempts, created_at, updated_at)
	VALUES (?, ?, 'queued', ?, ?, 0, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	res, err := repo.db.ExecContext(ctx, command, j.Type, string(j.Payload), j.Priority, j.RunAt.UTC(), j.MaxAttempts)
	if err != nil {
		logger.Error().Err(err).Msg("sql error insert while enqueuing job")
		return 0, err
	}
	return res.LastInsertId()
}

func (repo *JobRepository) ClaimNextJobDue(ctx context.Context, workerID string) (*domain.Job, bool, error) {
	const command = `
	UPDATE jobs
	SET status='running',
	    worker_id=?,
	    attempt=attempt+1,
	    started_at=CURRENT_TIMESTAMP,
	    updated_at=CURRENT_TIMESTAMP
	WHERE id = (
	  SELECT id FROM jobs
	  WHERE status='queued' AND run_at <= CURRENT_TIMESTAMP
	  ORDER BY priority DESC, run_at ASC, id ASC
	  LIMIT 1
	)
	RETURNING id, type, payload, status, priority, run_at, attempt, max_attempts, last_error, worker_id, scheduled_id, created_at, started_at, finished_at, updated_at
	`
	row := repo.db.QueryRowContext(ctx, command, workerID)
	var job domain.Job
	err := row.Scan(
		&job.ID,
		&job.Type,
		&job.Payload,
		&job.Status,
		&job.Priority,
		&job.RunAt,
		&job.Attempts,
		&job.MaxAttempts,
		&job.LastError,
		&job.WorkerID,
		&job.ScheduledID,
		&job.CreatedAt,
		&job.StartedAt,
		&job.FinishedAt,
		&job.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		logger.Error().Err(err).Msg("sql error claiming next job")
		return nil, false, err
	}
	return &job, true, nil
}

func (repo *JobRepository) MarkSuccess(ctx context.Context, jobID int64) error {
	const command = `
	UPDATE jobs
	SET status='succeeded', finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP, last_error=NULL
	WHERE id=?
	`
	_, err := repo.db.ExecContext(ctx, command, jobID)
	if err != nil {
		logger.Error().Err(err).Msg("sql error marking job successful")
	}

	return err
}

func (repo *JobRepository) Reschedule(ctx context.Context, jobID int64, runAt time.Time, lastErr string) error {
	const command = `
	UPDATE jobs
	SET status='queued', run_at=?, updated_at=CURRENT_TIMESTAMP, last_error=?
	WHERE id=?
	`
	_, err := repo.db.ExecContext(ctx, command, runAt.UTC(), shorten(lastErr, 2000), jobID)
	if err != nil {
		logger.Error().Err(err).Msg("sql error rescheduling job")
	}

	return err
}

func (repo *JobRepository) MarkDead(ctx context.Context, jobID int64, lastErr string) error {
	const command = `
	UPDATE jobs
	SET status='dead', finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP, last_error=?
	WHERE id=?
	`
	_, err := repo.db.ExecContext(ctx, command, shorten(lastErr, 2000), jobID)
	if err != nil {
		logger.Error().Err(err).Msg("sql error marking job dead")
	}

	return err
}

func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
