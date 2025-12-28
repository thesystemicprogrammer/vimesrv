package ports

import "time"

// Clock allows injecting time in tests.
type Clock interface {
	Now() time.Time
}

// RealClock is a production clock.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }
