package domain

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
)

type Job struct {
	ID          int64
	Type        string
	Payload     json.RawMessage
	Status      shared.JobStatus
	Priority    int
	RunAt       time.Time
	Attempts    int
	MaxAttempts int
	LastError   sql.NullString
	WorkerID    sql.NullString
	ScheduledID sql.NullInt64
	CreatedAt   time.Time
	StartedAt   sql.NullTime
	FinishedAt  sql.NullTime
	UpdatedAt   time.Time
}
