package job

import "time"

type JobManagerOptions struct {
	WorkerCount        int
	PollIntervall      time.Duration
	SchedulerIntervall time.Duration
	SchedulerBarch     int
	DefaultMaxAttempts int
}
