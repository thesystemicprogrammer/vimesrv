package websocket

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

func TestProgressCache_SetAndGet(t *testing.T) {
	cache := NewProgressCache()
	defer cache.Stop()

	progress := ports.JobProgress{
		Percentage: 50.5,
		Frame:      1000,
		FPS:        30.0,
		Speed:      "2x",
		Message:    "Encoding...",
	}

	// Set and retrieve
	cache.Set(123, progress)
	got, ok := cache.Get(123)

	assert.True(t, ok)
	assert.Equal(t, progress.Percentage, got.Percentage)
	assert.Equal(t, progress.Frame, got.Frame)
	assert.Equal(t, progress.FPS, got.FPS)
	assert.Equal(t, progress.Speed, got.Speed)
	assert.Equal(t, progress.Message, got.Message)
}

func TestProgressCache_GetNonExistent(t *testing.T) {
	cache := NewProgressCache()
	defer cache.Stop()

	got, ok := cache.Get(999)

	assert.False(t, ok)
	assert.Equal(t, ports.JobProgress{}, got)
}

func TestProgressCache_Delete(t *testing.T) {
	cache := NewProgressCache()
	defer cache.Stop()

	progress := ports.JobProgress{Percentage: 75.0}
	cache.Set(123, progress)

	// Verify it exists
	_, ok := cache.Get(123)
	assert.True(t, ok)

	// Delete and verify it's gone
	cache.Delete(123)
	_, ok = cache.Get(123)
	assert.False(t, ok)
}

func TestProgressCache_DeleteNonExistent(t *testing.T) {
	cache := NewProgressCache()
	defer cache.Stop()

	// Should not panic
	cache.Delete(999)
}

func TestProgressCache_GetAll(t *testing.T) {
	cache := NewProgressCache()
	defer cache.Stop()

	cache.Set(1, ports.JobProgress{Percentage: 10.0})
	cache.Set(2, ports.JobProgress{Percentage: 20.0})
	cache.Set(3, ports.JobProgress{Percentage: 30.0})

	all := cache.GetAll()

	assert.Len(t, all, 3)
	assert.Equal(t, 10.0, all[1].Percentage)
	assert.Equal(t, 20.0, all[2].Percentage)
	assert.Equal(t, 30.0, all[3].Percentage)
}

func TestProgressCache_GetAllReturnsACopy(t *testing.T) {
	cache := NewProgressCache()
	defer cache.Stop()

	cache.Set(1, ports.JobProgress{Percentage: 10.0})

	all := cache.GetAll()
	// Modify the returned map
	all[1] = ports.JobProgress{Percentage: 99.0}
	all[999] = ports.JobProgress{Percentage: 100.0}

	// Original cache should be unchanged
	got, _ := cache.Get(1)
	assert.Equal(t, 10.0, got.Percentage)

	_, ok := cache.Get(999)
	assert.False(t, ok)
}

func TestProgressCache_UpdateExisting(t *testing.T) {
	cache := NewProgressCache()
	defer cache.Stop()

	cache.Set(123, ports.JobProgress{Percentage: 25.0})
	cache.Set(123, ports.JobProgress{Percentage: 50.0})
	cache.Set(123, ports.JobProgress{Percentage: 75.0})

	got, ok := cache.Get(123)

	assert.True(t, ok)
	assert.Equal(t, 75.0, got.Percentage)
}

func TestProgressCache_ConcurrentAccess(t *testing.T) {
	cache := NewProgressCache()
	defer cache.Stop()

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.Set(int64(id), ports.JobProgress{Percentage: float64(j)})
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.Get(int64(id))
			}
		}(i)
	}

	// Concurrent GetAll
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.GetAll()
			}
		}()
	}

	// Concurrent deletes
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.Delete(int64(id))
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race conditions or deadlocks occurred
}

func TestProgressCache_StopIsIdempotent(t *testing.T) {
	cache := NewProgressCache()

	// Multiple calls to Stop should not panic
	cache.Stop()
	cache.Stop()
	cache.Stop()
}

func TestProgressCache_CleanupRemovesStaleEntries(t *testing.T) {
	// Create cache with very short TTL for testing
	cache := &ProgressCache{
		entries:  make(map[int64]progressEntry),
		ttl:      50 * time.Millisecond,
		stopChan: make(chan struct{}),
	}
	defer cache.Stop()

	// Add an entry
	cache.Set(123, ports.JobProgress{Percentage: 50.0})

	// Verify it exists
	_, ok := cache.Get(123)
	assert.True(t, ok)

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup
	cache.cleanup()

	// Verify entry was removed
	_, ok = cache.Get(123)
	assert.False(t, ok)
}

func TestProgressCache_CleanupKeepsFreshEntries(t *testing.T) {
	// Create cache with short TTL for testing
	cache := &ProgressCache{
		entries:  make(map[int64]progressEntry),
		ttl:      200 * time.Millisecond,
		stopChan: make(chan struct{}),
	}
	defer cache.Stop()

	// Add an entry
	cache.Set(123, ports.JobProgress{Percentage: 50.0})

	// Wait less than TTL
	time.Sleep(50 * time.Millisecond)

	// Manually trigger cleanup
	cache.cleanup()

	// Verify entry still exists
	got, ok := cache.Get(123)
	assert.True(t, ok)
	assert.Equal(t, 50.0, got.Percentage)
}

func TestProgressCache_SetRefreshesTTL(t *testing.T) {
	// Create cache with short TTL for testing
	cache := &ProgressCache{
		entries:  make(map[int64]progressEntry),
		ttl:      100 * time.Millisecond,
		stopChan: make(chan struct{}),
	}
	defer cache.Stop()

	// Add an entry
	cache.Set(123, ports.JobProgress{Percentage: 25.0})

	// Wait 60ms (past half of TTL)
	time.Sleep(60 * time.Millisecond)

	// Update the entry (should refresh TTL)
	cache.Set(123, ports.JobProgress{Percentage: 50.0})

	// Wait another 60ms (would be past original TTL, but not refreshed TTL)
	time.Sleep(60 * time.Millisecond)

	// Manually trigger cleanup
	cache.cleanup()

	// Verify entry still exists due to TTL refresh
	got, ok := cache.Get(123)
	assert.True(t, ok)
	assert.Equal(t, 50.0, got.Percentage)
}
