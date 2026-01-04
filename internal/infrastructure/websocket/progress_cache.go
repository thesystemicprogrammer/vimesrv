package websocket

import (
	"sync"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

const (
	// DefaultProgressTTL is the default time-to-live for progress entries.
	// Entries not updated within this duration will be removed.
	DefaultProgressTTL = 10 * time.Minute

	// DefaultCleanupInterval is the interval at which stale entries are cleaned up.
	DefaultCleanupInterval = 5 * time.Minute
)

// progressEntry holds a progress value along with its last update time.
type progressEntry struct {
	progress  ports.JobProgress
	updatedAt time.Time
}

// ProgressCache is a thread-safe in-memory cache for job progress.
// It stores the latest progress for running jobs and automatically
// cleans up stale entries based on TTL.
type ProgressCache struct {
	mu       sync.RWMutex
	entries  map[int64]progressEntry
	ttl      time.Duration
	stopChan chan struct{}
	stopped  bool
}

// NewProgressCache creates a new ProgressCache with default TTL and starts
// a background cleanup goroutine.
func NewProgressCache() *ProgressCache {
	c := &ProgressCache{
		entries:  make(map[int64]progressEntry),
		ttl:      DefaultProgressTTL,
		stopChan: make(chan struct{}),
	}
	go c.cleanupRoutine()
	return c
}

// Set stores or updates the progress for a job.
// This also refreshes the entry's timestamp.
func (c *ProgressCache) Set(jobID int64, progress ports.JobProgress) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[jobID] = progressEntry{
		progress:  progress,
		updatedAt: time.Now(),
	}
}

// Get retrieves the progress for a job.
// Returns the progress and true if found, or zero value and false if not found.
func (c *ProgressCache) Get(jobID int64) (ports.JobProgress, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[jobID]
	if !ok {
		return ports.JobProgress{}, false
	}
	return entry.progress, true
}

// Delete removes the progress entry for a job.
// This should be called when a job completes or fails.
func (c *ProgressCache) Delete(jobID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, jobID)
}

// GetAll returns a copy of all current progress entries.
// This is useful for batch retrieval when listing jobs.
func (c *ProgressCache) GetAll() map[int64]ports.JobProgress {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[int64]ports.JobProgress, len(c.entries))
	for jobID, entry := range c.entries {
		result[jobID] = entry.progress
	}
	return result
}

// Stop gracefully shuts down the cleanup goroutine.
// This should be called during application shutdown.
func (c *ProgressCache) Stop() {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return
	}
	c.stopped = true
	c.mu.Unlock()

	close(c.stopChan)
}

// cleanupRoutine periodically removes stale entries from the cache.
func (c *ProgressCache) cleanupRoutine() {
	ticker := time.NewTicker(DefaultCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			logger.Debug().Msg("Progress cache cleanup routine stopped")
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

// cleanup removes entries that haven't been updated within the TTL.
func (c *ProgressCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := 0

	for jobID, entry := range c.entries {
		if now.Sub(entry.updatedAt) > c.ttl {
			delete(c.entries, jobID)
			expired++
		}
	}

	if expired > 0 {
		logger.Debug().
			Int("expired", expired).
			Int("remaining", len(c.entries)).
			Msg("Cleaned up expired progress cache entries")
	}
}
