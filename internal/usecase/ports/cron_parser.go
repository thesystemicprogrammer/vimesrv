package ports

import "time"

type CronSchedule interface {
	Next(from time.Time) time.Time
}

type CronParser interface {
	Parse(spec string) (CronSchedule, error)
}
