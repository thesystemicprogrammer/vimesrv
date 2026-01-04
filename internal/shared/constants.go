package shared

import "time"

type JobStatus string

const (
	StatusQueued    JobStatus = "queued"
	StatusRunning   JobStatus = "running"
	StatusSucceeded JobStatus = "succeeded"
	StatusDead      JobStatus = "dead"
)

// User roles
type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleManager UserRole = "manager"
	RoleUser    UserRole = "user"
)

// ValidUserRoles returns all valid user roles
func ValidUserRoles() []UserRole {
	return []UserRole{RoleAdmin, RoleManager, RoleUser}
}

// IsValidRole checks if the given role is valid
func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleManager, RoleUser:
		return true
	default:
		return false
	}
}

// Job types
const (
	JobTypeScanLibrary       = "scan_library"
	JobTypeTranscodeVideo    = "transcode_video"
	JobTypeTranscodeAudio    = "transcode_audio"
	JobTypeTranscodeSubtitle = "transcode_subtitle"
	JobTypeEnrichMetadata    = "enrich_metadata"
	JobTypeFetchTranslations = "fetch_translations"
)

// Job priorities (lower value = higher priority)
const (
	JobPriorityLibraryScan       = 0
	JobPriorityTranscode         = 5
	JobPriorityEnrichMetadata    = 3
	JobPriorityFetchTranslations = 4
)

// ErrorBackoffDuration is the wait time after a job processing error before retrying
const ErrorBackoffDuration = 500 * time.Millisecond
