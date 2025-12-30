package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

type ScheduleRepository struct {
	db *sql.DB
}

func NewScheduleRepository(db *database.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db.DB}
}

func (repo *ScheduleRepository) Upsert(ctx context.Context, schedule *domain.Schedule) (int64, error) {
	const command = `
	INSERT INTO schedules (name, cron_spec, job_type, payload, priority, max_attempts, enabled, next_run_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, NULL, CURRENT_TIMESTAMP)
	ON CONFLICT(name) DO UPDATE SET
		cron_spec=excluded.cron_spec,
		job_type=excluded.job_type,
		payload=excluded.payload,
		priority=excluded.priority,
		max_attempts=excluded.max_attempts,
		enabled=excluded.enabled,
		updated_at=CURRENT_TIMESTAMP
	`
	_, err := repo.db.ExecContext(ctx, command, schedule.Name, schedule.CronSpec, schedule.JobType, string(schedule.Payload), schedule.Priority, schedule.MaxAttempts, boolToInt(schedule.Enabled))
	if err != nil {
		logger.Error().Err(err).Int64("ID", schedule.ID).Str("name", schedule.Name).Msg("sql error upsert while ExecContext")
		return 0, err
	}

	var id int64
	err = repo.db.QueryRowContext(ctx, `SELECT id FROM schedules WHERE name = ?`, schedule.Name).Scan(&id)
	if err != nil {
		logger.Error().Err(err).Int64("ID", schedule.ID).Str("name", schedule.Name).Msg("sql error upsert while QueryRowContext")
		return 0, err
	}

	return id, nil
}

func (repo *ScheduleRepository) GetByName(ctx context.Context, name string) (*domain.Schedule, error) {
	const command = `
	SELECT id, name, cron_spec, job_type, payload, priority, max_attempts, enabled, next_run_at, last_enqueued_at, updated_at
	FROM schedules WHERE name = ?
	`
	var schedule domain.Schedule
	var payload sql.NullString
	var enabledInt int
	err := repo.db.QueryRowContext(ctx, command, name).Scan(
		&schedule.ID, &schedule.Name, &schedule.CronSpec, &schedule.JobType, &payload, &schedule.Priority, &schedule.MaxAttempts, &enabledInt,
		&schedule.NextRunAt, &schedule.LastEnqueuedAt, &schedule.UpdatedAt,
	)
	if err != nil {
		logger.Error().Err(err).Str("name", name).Msg("sql error while getting schedule by name")
		return nil, err
	}
	if payload.Valid {
		schedule.Payload = []byte(payload.String)
	}
	schedule.Enabled = enabledInt == 1
	return &schedule, nil
}

func (repo *ScheduleRepository) SetNextRunIfNull(ctx context.Context, id int64, nextRunAt time.Time) error {
	const command = `
	UPDATE schedules
	SET next_run_at=?, updated_at=CURRENT_TIMESTAMP
	WHERE id=? AND next_run_at IS NULL
	`
	_, err := repo.db.ExecContext(ctx, command, nextRunAt.UTC(), id)
	if err != nil {
		logger.Error().Err(err).Int64("ID", id).Msg("sql error while setting schedule for next run")
	}

	return err
}

// SetNextRun unconditionally updates the next_run_at for a schedule.
// This is used when ForceNextRunNow is true to always reset the schedule to run immediately.
func (repo *ScheduleRepository) SetNextRun(ctx context.Context, id int64, nextRunAt time.Time) error {
	const command = `
	UPDATE schedules
	SET next_run_at=?, updated_at=CURRENT_TIMESTAMP
	WHERE id=?
	`
	_, err := repo.db.ExecContext(ctx, command, nextRunAt.UTC(), id)
	if err != nil {
		logger.Error().Err(err).Int64("ID", id).Msg("sql error while unconditionally setting next run")
	}

	return err
}

func (repo *ScheduleRepository) ListDue(ctx context.Context, limit int) ([]*domain.Schedule, error) {
	const command = `
	SELECT id, name, cron_spec, job_type, payload, priority, max_attempts, enabled, next_run_at, last_enqueued_at, updated_at
	FROM schedules
	WHERE enabled=1 AND (next_run_at IS NULL OR next_run_at <= CURRENT_TIMESTAMP)
	ORDER BY COALESCE(next_run_at, '1970-01-01')
	LIMIT ?
	`
	rows, err := repo.db.QueryContext(ctx, command, limit)
	if err != nil {
		logger.Error().Err(err).Msg("sql error while listing next due jobs")
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Schedule
	for rows.Next() {
		var schedule domain.Schedule
		var payload sql.NullString
		var enabledInt int
		if err := rows.Scan(
			&schedule.ID, &schedule.Name, &schedule.CronSpec, &schedule.JobType, &payload, &schedule.Priority, &schedule.MaxAttempts, &enabledInt,
			&schedule.NextRunAt, &schedule.LastEnqueuedAt, &schedule.UpdatedAt,
		); err != nil {
			logger.Error().Err(err).Msg("sql error while scanning next due jobs")
			return nil, err
		}
		if payload.Valid {
			schedule.Payload = []byte(payload.String)
		}
		schedule.Enabled = enabledInt == 1
		out = append(out, &schedule)
	}
	return out, rows.Err()
}

func (repo *ScheduleRepository) AdvanceAndEnqueue(ctx context.Context, scheduleID int64, next time.Time, jobProto *domain.Job) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error().Err(err).Msg("sql error beginning transaction")
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Advance next_run_at only if still due (avoids duplicate enqueues with multiple schedulers).
	res, err := tx.ExecContext(ctx, `
	UPDATE schedules
	SET last_enqueued_at=CURRENT_TIMESTAMP, next_run_at=?, updated_at=CURRENT_TIMESTAMP
	WHERE id=? AND enabled=1 AND (next_run_at IS NULL OR next_run_at <= CURRENT_TIMESTAMP)
	`, next.UTC(), scheduleID)
	if err != nil {
		logger.Error().Err(err).Msg("sql error updating schedules")
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return tx.Commit() // someone else advanced it
	}

	// Enqueue job
	_, err = tx.ExecContext(ctx, `
	INSERT INTO jobs (type, payload, status, priority, run_at, attempt, max_attempts, scheduled_id, created_at, updated_at)
	VALUES (?, ?, 'queued', ?, CURRENT_TIMESTAMP, 0, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, jobProto.Type, string(jobProto.Payload), jobProto.Priority, jobProto.MaxAttempts, scheduleID)
	if err != nil {
		logger.Error().Err(err).Msg("sql error enqueuing jobs")
		return err
	}
	return tx.Commit()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
