package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
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

// ClaimNextJobDueExcludingTypes atomically claims the next job due for processing,
// excluding jobs of the specified types. Used when distributed workers are enabled
// to prevent local processing of transcode jobs.
func (repo *JobRepository) ClaimNextJobDueExcludingTypes(ctx context.Context, workerID string, excludeTypes []string) (*domain.Job, bool, error) {
	if len(excludeTypes) == 0 {
		// No exclusions, delegate to regular method
		return repo.ClaimNextJobDue(ctx, workerID)
	}

	// Build dynamic query with exclusions
	placeholders := make([]string, len(excludeTypes))
	args := []interface{}{workerID}
	for i, t := range excludeTypes {
		placeholders[i] = "?"
		args = append(args, t)
	}
	exclusionClause := " AND type NOT IN (" + strings.Join(placeholders, ",") + ")"

	command := `
	UPDATE jobs
	SET status='running',
	    worker_id=?,
	    attempt=attempt+1,
	    started_at=CURRENT_TIMESTAMP,
	    updated_at=CURRENT_TIMESTAMP
	WHERE id = (
	  SELECT id FROM jobs
	  WHERE status='queued' AND run_at <= CURRENT_TIMESTAMP` + exclusionClause + `
	  ORDER BY priority DESC, run_at ASC, id ASC
	  LIMIT 1
	)
	RETURNING id, type, payload, status, priority, run_at, attempt, max_attempts, last_error, worker_id, scheduled_id, created_at, started_at, finished_at, updated_at
	`

	job, err := repo.scanJobRow(repo.db.QueryRowContext(ctx, command, args...))
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		logger.Error().Err(err).Msg("sql error claiming next job excluding types")
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

// ListJobs returns jobs matching the filter criteria
func (repo *JobRepository) ListJobs(ctx context.Context, filter ports.JobListFilter) (*ports.JobListResult, error) {
	// Build dynamic query
	var queryBuilder strings.Builder
	var countBuilder strings.Builder
	var args []interface{}

	queryBuilder.WriteString(`
	SELECT id, type, payload, status, priority, run_at, attempt, max_attempts, last_error, worker_id, scheduled_id, created_at, started_at, finished_at, updated_at
	FROM jobs
	WHERE 1=1
	`)

	countBuilder.WriteString(`
	SELECT COUNT(*) FROM jobs WHERE 1=1
	`)

	// Filter by statuses
	if len(filter.Statuses) > 0 {
		placeholders := make([]string, len(filter.Statuses))
		for i, status := range filter.Statuses {
			placeholders[i] = "?"
			args = append(args, status)
		}
		statusClause := " AND status IN (" + strings.Join(placeholders, ",") + ")"
		queryBuilder.WriteString(statusClause)
		countBuilder.WriteString(statusClause)
	}

	// Filter by types
	if len(filter.Types) > 0 {
		placeholders := make([]string, len(filter.Types))
		for i, t := range filter.Types {
			placeholders[i] = "?"
			args = append(args, t)
		}
		typeClause := " AND type IN (" + strings.Join(placeholders, ",") + ")"
		queryBuilder.WriteString(typeClause)
		countBuilder.WriteString(typeClause)
	}

	// Filter by time (jobs updated after this time, to capture recent activity)
	if !filter.Since.IsZero() {
		sinceClause := " AND updated_at >= ?"
		queryBuilder.WriteString(sinceClause)
		countBuilder.WriteString(sinceClause)
		args = append(args, filter.Since.UTC())
	}

	// Get total count first
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)

	var total int
	if err := repo.db.QueryRowContext(ctx, countBuilder.String(), countArgs...).Scan(&total); err != nil {
		logger.Error().Err(err).Msg("sql error counting jobs")
		return nil, err
	}

	// Add ordering: running first, then queued, then by updated_at desc
	queryBuilder.WriteString(`
	ORDER BY
		CASE status
			WHEN 'running' THEN 1
			WHEN 'queued' THEN 2
			WHEN 'succeeded' THEN 3
			WHEN 'dead' THEN 4
			ELSE 5
		END,
		updated_at DESC
	`)

	// Add pagination
	if filter.Limit > 0 {
		queryBuilder.WriteString(" LIMIT ?")
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		queryBuilder.WriteString(" OFFSET ?")
		args = append(args, filter.Offset)
	}

	rows, err := repo.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		logger.Error().Err(err).Msg("sql error listing jobs")
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		job, err := repo.scanJobRow(rows)
		if err != nil {
			logger.Error().Err(err).Msg("sql error scanning job row")
			return nil, err
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("sql error iterating job rows")
		return nil, err
	}

	return &ports.JobListResult{
		Jobs:  jobs,
		Total: total,
	}, nil
}

// Get retrieves a job by its ID
func (repo *JobRepository) Get(ctx context.Context, jobID int64) (*domain.Job, error) {
	const query = `
	SELECT id, type, payload, status, priority, run_at, attempt, max_attempts, last_error, worker_id, scheduled_id, created_at, started_at, finished_at, updated_at
	FROM jobs
	WHERE id = ?
	`
	job, err := repo.scanJobRow(repo.db.QueryRowContext(ctx, query, jobID))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %d", jobID)
	}
	if err != nil {
		logger.Error().Err(err).Int64("jobID", jobID).Msg("sql error getting job")
		return nil, err
	}
	return job, nil
}

// ClaimNextTranscodeJob atomically claims the next queued transcode job
// for processing by a distributed worker. Returns (nil, nil) if no jobs available.
// Handles all transcode job types: video, audio, and subtitle.
func (repo *JobRepository) ClaimNextTranscodeJob(ctx context.Context, workerID string) (*domain.Job, error) {
	const command = `
	UPDATE jobs
	SET status='running',
	    worker_id=?,
	    attempt=attempt+1,
	    started_at=CURRENT_TIMESTAMP,
	    updated_at=CURRENT_TIMESTAMP
	WHERE id = (
	  SELECT id FROM jobs
	  WHERE status='queued' AND type IN ('transcode_video', 'transcode_audio', 'transcode_subtitle') AND run_at <= CURRENT_TIMESTAMP
	  ORDER BY priority DESC, run_at ASC, id ASC
	  LIMIT 1
	)
	RETURNING id, type, payload, status, priority, run_at, attempt, max_attempts, last_error, worker_id, scheduled_id, created_at, started_at, finished_at, updated_at
	`
	job, err := repo.scanJobRow(repo.db.QueryRowContext(ctx, command, workerID))
	if err == sql.ErrNoRows {
		return nil, nil // No jobs available
	}
	if err != nil {
		logger.Error().Err(err).Msg("sql error claiming next transcode job")
		return nil, err
	}
	return job, nil
}

// CountQueuedTranscodeJobs returns the number of queued transcode jobs
// Counts all transcode job types: video, audio, and subtitle.
func (repo *JobRepository) CountQueuedTranscodeJobs(ctx context.Context) (int, error) {
	const query = `
	SELECT COUNT(*) FROM jobs
	WHERE status = 'queued' AND type IN ('transcode_video', 'transcode_audio', 'transcode_subtitle')
	`
	var count int
	if err := repo.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		logger.Error().Err(err).Msg("sql error counting queued transcode jobs")
		return 0, err
	}
	return count, nil
}
