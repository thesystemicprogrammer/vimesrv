package ports

// ProgressCache provides read access to cached job progress data.
// This interface is used by HTTP handlers to retrieve current progress
// for running jobs without depending on the WebSocket infrastructure.
type ProgressCache interface {
	// Get retrieves the progress for a specific job.
	// Returns the progress and true if found, or zero value and false if not found.
	Get(jobID int64) (JobProgress, bool)

	// GetAll returns a copy of all current progress entries.
	// This is useful for batch retrieval when listing jobs.
	GetAll() map[int64]JobProgress
}
