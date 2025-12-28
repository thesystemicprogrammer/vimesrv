package ports

import "time"

type BackoffStrategy interface {
	NextDelay(attempt int) time.Duration
}
