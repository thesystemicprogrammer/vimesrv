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
	JobTypeEnrichMetadata = "enrich_metadata"
)

// Job priorities (lower value = higher priority)
const (
	JobPriorityLibraryScan    = 0
	JobPriorityTranscode      = 5
	JobPriorityEnrichMetadata = 3
)

// ErrorBackoffDuration is the wait time after a job processing error before retrying
const ErrorBackoffDuration = 500 * time.Millisecond
