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

func NewJobRepository(db *database.DB) *JobRepository {
	return &JobRepository{db: db.DB}
}

// scanJobRow scans a row into a Job struct and converts the payload string to json.RawMessage
func (repo *JobRepository) scanJobRow(s scanner) (*domain.Job, error) {
	var job domain.Job
	var payloadStr string

	err := s.Scan(
		&job.ID,
		&job.Type,
		&payloadStr,
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
	if err != nil {
		return nil, err
	}

	job.Payload = []byte(payloadStr)
	return &job, nil
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

// ClaimNextJobDue atomically claims the next job due for processing.
// This uses SQLite's atomic UPDATE...RETURNING behavior to claim a job in a single query.
// NOTE: This implementation is SQLite-specific. When migrating to PostgreSQL/MySQL,
// wrap the UPDATE and SELECT in an explicit transaction with appropriate isolation level
// (e.g., FOR UPDATE SKIP LOCKED in PostgreSQL).
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
	job, err := repo.scanJobRow(repo.db.QueryRowContext(ctx, command, workerID))
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		logger.Error().Err(err).Msg("sql error claiming next job")
		return nil, false, err
	}
	return job, true, nil
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

// FindStuckJobs finds jobs that have been in 'running' state longer than the threshold
func (repo *JobRepository) FindStuckJobs(ctx context.Context, threshold time.Duration) ([]*domain.Job, error) {
	thresholdTime := time.Now().Add(-threshold).UTC()

	const command = `
	SELECT id, type, payload, status, priority, run_at, attempt, max_attempts, last_error, worker_id, scheduled_id, created_at, started_at, finished_at, updated_at
	FROM jobs
	WHERE status='running' AND started_at < ?
	ORDER BY started_at ASC
	`
	rows, err := repo.db.QueryContext(ctx, command, thresholdTime)
	if err != nil {
		logger.Error().Err(err).Msg("sql error finding stuck jobs")
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		job, err := repo.scanJobRow(rows)
		if err != nil {
			logger.Error().Err(err).Msg("sql error scanning stuck job")
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// ResetStuckJob resets a stuck job back to 'queued' status so it can be retried
func (repo *JobRepository) ResetStuckJob(ctx context.Context, jobID int64) error {
	const command = `
	UPDATE jobs
	SET status='queued', worker_id=NULL, started_at=NULL, updated_at=CURRENT_TIMESTAMP
	WHERE id=? AND status='running'
	`
	result, err := repo.db.ExecContext(ctx, command, jobID)
	if err != nil {
		logger.Error().Err(err).Int64("jobID", jobID).Msg("sql error resetting stuck job")
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logger.Warn().Int64("jobID", jobID).Msg("no rows affected when resetting stuck job - job may have already completed")
	}

	return nil
}

// ExistsPendingJobByType checks if there's already a queued or running job of the given type
// with a payload containing the specified language value
func (repo *JobRepository) ExistsPendingJobByType(ctx context.Context, jobType string, language string) (bool, error) {
	const query = `
	SELECT EXISTS(
		SELECT 1 FROM jobs
		WHERE type = ?
		AND status IN ('queued', 'running')
		AND payload LIKE ?
	)
	`
	// Build the LIKE pattern to match the language in JSON payload
	likePattern := `%"language":"` + language + `"%`

	var exists bool
	err := repo.db.QueryRowContext(ctx, query, jobType, likePattern).Scan(&exists)
	if err != nil {
		logger.Error().Err(err).Str("jobType", jobType).Str("language", language).Msg("sql error checking for pending job")
		return false, err
	}

	return exists, nil
}
