package shared

import "time"

type JobStatus string

const (
	StatusQueued    JobStatus = "queued"
	StatusRunning   JobStatus = "running"
	StatusSucceeded JobStatus = "succeeded"
	StatusDead      JobStatus = "dead"
)

// Job types
const (
	JobTypeScanLibrary    = "scan_library"
	JobTypeTranscodeVideo = "transcode_video"
)

// ErrorBackoffDuration is the wait time after a job processing error before retrying
const ErrorBackoffDuration = 500 * time.Millisecond
