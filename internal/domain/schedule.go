package domain

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Schedule struct {
	ID             int64
	Name           string
	CronSpec       string
	JobType        string
	Payload        json.RawMessage
	Priority       int
	MaxAttempts    int
	Enabled        bool
	NextRunAt      sql.NullTime
	LastEnqueuedAt sql.NullTime
	UpdatedAt      time.Time
}
