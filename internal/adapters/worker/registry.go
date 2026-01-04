package worker

import (
	"sync"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// Registry is an in-memory store that tracks active workers
type Registry struct {
	mu      sync.RWMutex
	workers map[string]*domain.WorkerState
	timeout time.Duration
}

// NewRegistry creates a new worker registry with the specified heartbeat timeout
func NewRegistry(timeout time.Duration) *Registry {
	return &Registry{
		workers: make(map[string]*domain.WorkerState),
		timeout: timeout,
	}
}

// Register adds or updates a worker in the registry
func (r *Registry) Register(workerID, name string, capacity int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if w, ok := r.workers[workerID]; ok {
		w.Name = name
		w.LastSeen = now
		w.Capacity = capacity
	} else {
		r.workers[workerID] = &domain.WorkerState{
			ID:           workerID,
			Name:         name,
			LastSeen:     now,
			Capacity:     capacity,
			RegisteredAt: now,
		}
	}
}

// Touch updates the LastSeen timestamp for a worker
// Called on heartbeat and progress reports
// Returns true if the worker was found, false otherwise
func (r *Registry) Touch(workerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if w, ok := r.workers[workerID]; ok {
		w.LastSeen = time.Now()
		return true
	}
	return false
}

// SetActiveJobs updates the active job count for a worker
func (r *Registry) SetActiveJobs(workerID string, count int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if w, ok := r.workers[workerID]; ok {
		w.ActiveJobs = count
	}
}

// IncrementActiveJobs increments the active job count for a worker
func (r *Registry) IncrementActiveJobs(workerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if w, ok := r.workers[workerID]; ok {
		w.ActiveJobs++
	}
}

// DecrementActiveJobs decrements the active job count for a worker
func (r *Registry) DecrementActiveJobs(workerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if w, ok := r.workers[workerID]; ok {
		if w.ActiveJobs > 0 {
			w.ActiveJobs--
		}
	}
}

// HasAliveWorkers returns true if any worker has been seen within the timeout
func (r *Registry) HasAliveWorkers() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, w := range r.workers {
		if w.IsAlive(r.timeout) {
			return true
		}
	}
	return false
}

// AliveWorkerCount returns the number of alive workers
func (r *Registry) AliveWorkerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, w := range r.workers {
		if w.IsAlive(r.timeout) {
			count++
		}
	}
	return count
}

// GetWorker returns a worker by ID, or nil if not found
// Returns a copy to avoid race conditions
func (r *Registry) GetWorker(workerID string) *domain.WorkerState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if w, ok := r.workers[workerID]; ok {
		return w.Copy()
	}
	return nil
}

// ListAliveWorkers returns all workers that are currently alive
// Returns copies to avoid race conditions
func (r *Registry) ListAliveWorkers() []*domain.WorkerState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.WorkerState
	for _, w := range r.workers {
		if w.IsAlive(r.timeout) {
			result = append(result, w.Copy())
		}
	}
	return result
}

// ListAllWorkers returns all registered workers (alive or dead)
// Returns copies to avoid race conditions
func (r *Registry) ListAllWorkers() []*domain.WorkerState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.WorkerState, 0, len(r.workers))
	for _, w := range r.workers {
		result = append(result, w.Copy())
	}
	return result
}

// Cleanup removes workers that haven't been seen for a long time
// Should be called periodically to prevent memory leaks
// Returns the number of removed workers
func (r *Registry) Cleanup(maxAge time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	removed := 0
	for id, w := range r.workers {
		if time.Since(w.LastSeen) > maxAge {
			delete(r.workers, id)
			removed++
		}
	}
	return removed
}

// IsWorkerAlive checks if a specific worker is alive
func (r *Registry) IsWorkerAlive(workerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if w, ok := r.workers[workerID]; ok {
		return w.IsAlive(r.timeout)
	}
	return false
}

// Remove removes a worker from the registry
func (r *Registry) Remove(workerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.workers, workerID)
}
