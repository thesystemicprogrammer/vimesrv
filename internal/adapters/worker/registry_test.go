package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	timeout := 60 * time.Second
	registry := NewRegistry(timeout)

	require.NotNil(t, registry)
	assert.NotNil(t, registry.workers)
	assert.Equal(t, timeout, registry.timeout)
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry(60 * time.Second)

	// Register a new worker
	registry.Register("worker-1", "test-worker", 4)

	worker := registry.GetWorker("worker-1")
	require.NotNil(t, worker)
	assert.Equal(t, "worker-1", worker.ID)
	assert.Equal(t, "test-worker", worker.Name)
	assert.Equal(t, 4, worker.Capacity)
	assert.Equal(t, 0, worker.ActiveJobs)
}

func TestRegistry_RegisterUpdate(t *testing.T) {
	registry := NewRegistry(60 * time.Second)

	// Register a worker
	registry.Register("worker-1", "test-worker", 4)
	time.Sleep(10 * time.Millisecond) // Small delay

	// Re-register with updated info
	registry.Register("worker-1", "updated-worker", 8)

	worker := registry.GetWorker("worker-1")
	require.NotNil(t, worker)
	assert.Equal(t, "updated-worker", worker.Name)
	assert.Equal(t, 8, worker.Capacity)
}

func TestRegistry_Touch(t *testing.T) {
	registry := NewRegistry(60 * time.Second)

	// Touch non-existent worker returns false
	assert.False(t, registry.Touch("non-existent"))

	// Register and touch
	registry.Register("worker-1", "test-worker", 4)
	worker1 := registry.GetWorker("worker-1")
	lastSeen1 := worker1.LastSeen

	time.Sleep(10 * time.Millisecond)
	assert.True(t, registry.Touch("worker-1"))

	worker2 := registry.GetWorker("worker-1")
	assert.True(t, worker2.LastSeen.After(lastSeen1))
}

func TestRegistry_ActiveJobs(t *testing.T) {
	registry := NewRegistry(60 * time.Second)
	registry.Register("worker-1", "test-worker", 4)

	// Initial active jobs is 0
	worker := registry.GetWorker("worker-1")
	assert.Equal(t, 0, worker.ActiveJobs)

	// Set active jobs
	registry.SetActiveJobs("worker-1", 3)
	worker = registry.GetWorker("worker-1")
	assert.Equal(t, 3, worker.ActiveJobs)

	// Increment
	registry.IncrementActiveJobs("worker-1")
	worker = registry.GetWorker("worker-1")
	assert.Equal(t, 4, worker.ActiveJobs)

	// Decrement
	registry.DecrementActiveJobs("worker-1")
	worker = registry.GetWorker("worker-1")
	assert.Equal(t, 3, worker.ActiveJobs)

	// Decrement below zero doesn't go negative
	registry.SetActiveJobs("worker-1", 0)
	registry.DecrementActiveJobs("worker-1")
	worker = registry.GetWorker("worker-1")
	assert.Equal(t, 0, worker.ActiveJobs)
}

func TestRegistry_HasAliveWorkers(t *testing.T) {
	registry := NewRegistry(50 * time.Millisecond)

	// No workers
	assert.False(t, registry.HasAliveWorkers())

	// Register a worker
	registry.Register("worker-1", "test-worker", 4)
	assert.True(t, registry.HasAliveWorkers())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)
	assert.False(t, registry.HasAliveWorkers())

	// Touch to bring back alive
	registry.Touch("worker-1")
	assert.True(t, registry.HasAliveWorkers())
}

func TestRegistry_AliveWorkerCount(t *testing.T) {
	registry := NewRegistry(50 * time.Millisecond)

	assert.Equal(t, 0, registry.AliveWorkerCount())

	registry.Register("worker-1", "test-1", 4)
	registry.Register("worker-2", "test-2", 4)
	assert.Equal(t, 2, registry.AliveWorkerCount())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, 0, registry.AliveWorkerCount())
}

func TestRegistry_IsWorkerAlive(t *testing.T) {
	registry := NewRegistry(50 * time.Millisecond)

	// Non-existent worker
	assert.False(t, registry.IsWorkerAlive("non-existent"))

	// Register and check
	registry.Register("worker-1", "test-worker", 4)
	assert.True(t, registry.IsWorkerAlive("worker-1"))

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)
	assert.False(t, registry.IsWorkerAlive("worker-1"))
}

func TestRegistry_ListAliveWorkers(t *testing.T) {
	registry := NewRegistry(100 * time.Millisecond)

	// No workers
	workers := registry.ListAliveWorkers()
	assert.Empty(t, workers)

	// Add workers
	registry.Register("worker-1", "test-1", 4)
	registry.Register("worker-2", "test-2", 2)

	workers = registry.ListAliveWorkers()
	assert.Len(t, workers, 2)

	// Wait for timeout
	time.Sleep(110 * time.Millisecond)
	workers = registry.ListAliveWorkers()
	assert.Empty(t, workers)
}

func TestRegistry_ListAllWorkers(t *testing.T) {
	registry := NewRegistry(50 * time.Millisecond)

	registry.Register("worker-1", "test-1", 4)
	registry.Register("worker-2", "test-2", 2)

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// ListAllWorkers returns all workers, even dead ones
	workers := registry.ListAllWorkers()
	assert.Len(t, workers, 2)

	// But ListAliveWorkers returns none
	aliveWorkers := registry.ListAliveWorkers()
	assert.Empty(t, aliveWorkers)
}

func TestRegistry_Cleanup(t *testing.T) {
	registry := NewRegistry(50 * time.Millisecond)

	registry.Register("worker-1", "test-1", 4)
	registry.Register("worker-2", "test-2", 2)

	// No cleanup needed yet
	removed := registry.Cleanup(100 * time.Millisecond)
	assert.Equal(t, 0, removed)
	assert.Len(t, registry.ListAllWorkers(), 2)

	// Wait and cleanup
	time.Sleep(110 * time.Millisecond)
	removed = registry.Cleanup(100 * time.Millisecond)
	assert.Equal(t, 2, removed)
	assert.Empty(t, registry.ListAllWorkers())
}

func TestRegistry_Remove(t *testing.T) {
	registry := NewRegistry(60 * time.Second)

	registry.Register("worker-1", "test-1", 4)
	registry.Register("worker-2", "test-2", 2)

	assert.Len(t, registry.ListAllWorkers(), 2)

	registry.Remove("worker-1")
	assert.Len(t, registry.ListAllWorkers(), 1)
	assert.Nil(t, registry.GetWorker("worker-1"))
	assert.NotNil(t, registry.GetWorker("worker-2"))
}

func TestRegistry_GetWorker_ReturnsNilForNonExistent(t *testing.T) {
	registry := NewRegistry(60 * time.Second)

	worker := registry.GetWorker("non-existent")
	assert.Nil(t, worker)
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry(60 * time.Second)
	done := make(chan bool)

	// Concurrent registrations
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				registry.Register("worker-1", "test", 4)
				registry.Touch("worker-1")
				registry.IncrementActiveJobs("worker-1")
				registry.DecrementActiveJobs("worker-1")
				registry.GetWorker("worker-1")
				registry.HasAliveWorkers()
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify state is consistent
	worker := registry.GetWorker("worker-1")
	require.NotNil(t, worker)
	assert.Equal(t, "worker-1", worker.ID)
}
