package domain

import "time"

// WorkerState represents a registered transcoding worker's current state
// This is an in-memory representation, not persisted to database
type WorkerState struct {
	// ID is the unique identifier for the worker
	ID string

	// Name is a human-readable name for the worker
	Name string

	// LastSeen is the last time the worker sent a heartbeat or progress report
	LastSeen time.Time

	// ActiveJobs is the number of jobs currently being processed by this worker
	ActiveJobs int

	// Capacity is the maximum number of concurrent jobs this worker can handle
	Capacity int

	// RegisteredAt is when the worker first registered with the server
	RegisteredAt time.Time
}

// IsAlive checks if the worker has been seen within the given timeout duration
func (w *WorkerState) IsAlive(timeout time.Duration) bool {
	return time.Since(w.LastSeen) < timeout
}

// Copy returns a copy of the WorkerState to avoid race conditions
func (w *WorkerState) Copy() *WorkerState {
	return &WorkerState{
		ID:           w.ID,
		Name:         w.Name,
		LastSeen:     w.LastSeen,
		ActiveJobs:   w.ActiveJobs,
		Capacity:     w.Capacity,
		RegisteredAt: w.RegisteredAt,
	}
}
