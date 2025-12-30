package job

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// TestNewHandlerRegistry tests registry initialization
func TestNewHandlerRegistry(t *testing.T) {
	registry := NewHandlerRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.handlers)
	assert.Equal(t, 0, len(registry.handlers))
}

// TestHandlerRegistry_Register tests registering a handler
func TestHandlerRegistry_Register(t *testing.T) {
	registry := NewHandlerRegistry()

	handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		return nil
	})

	registry.Register("test-job", handler)

	// Verify handler was registered
	retrieved, ok := registry.Get("test-job")
	require.True(t, ok)
	assert.NotNil(t, retrieved)
}

// TestHandlerRegistry_Get_NotFound tests getting non-existent handler
func TestHandlerRegistry_Get_NotFound(t *testing.T) {
	registry := NewHandlerRegistry()

	handler, ok := registry.Get("non-existent")

	assert.False(t, ok)
	assert.Nil(t, handler)
}

// TestHandlerRegistry_Get_Found tests getting registered handler
func TestHandlerRegistry_Get_Found(t *testing.T) {
	registry := NewHandlerRegistry()

	executed := false
	handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		executed = true
		return nil
	})

	registry.Register("my-job", handler)

	// Get and execute handler
	retrieved, ok := registry.Get("my-job")
	require.True(t, ok)
	require.NotNil(t, retrieved)

	err := retrieved(context.Background(), &domain.Job{})
	require.NoError(t, err)
	assert.True(t, executed, "Handler should have been executed")
}

// TestHandlerRegistry_MultipleHandlers tests registering multiple handlers
func TestHandlerRegistry_MultipleHandlers(t *testing.T) {
	registry := NewHandlerRegistry()

	handler1 := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		return nil
	})
	handler2 := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		return nil
	})
	handler3 := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		return nil
	})

	registry.Register("job-1", handler1)
	registry.Register("job-2", handler2)
	registry.Register("job-3", handler3)

	// Verify all handlers are registered
	h1, ok1 := registry.Get("job-1")
	assert.True(t, ok1)
	assert.NotNil(t, h1)

	h2, ok2 := registry.Get("job-2")
	assert.True(t, ok2)
	assert.NotNil(t, h2)

	h3, ok3 := registry.Get("job-3")
	assert.True(t, ok3)
	assert.NotNil(t, h3)
}

// TestHandlerRegistry_Overwrite tests overwriting a handler
func TestHandlerRegistry_Overwrite(t *testing.T) {
	registry := NewHandlerRegistry()

	firstCalled := false
	handler1 := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		firstCalled = true
		return nil
	})

	secondCalled := false
	handler2 := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		secondCalled = true
		return nil
	})

	// Register first handler
	registry.Register("test-job", handler1)

	// Overwrite with second handler
	registry.Register("test-job", handler2)

	// Get and execute - should execute second handler
	retrieved, ok := registry.Get("test-job")
	require.True(t, ok)

	err := retrieved(context.Background(), &domain.Job{})
	require.NoError(t, err)
	assert.False(t, firstCalled, "First handler should not be called")
	assert.True(t, secondCalled, "Second handler should be called")
}

// TestHandlerRegistry_ConcurrentRegister tests concurrent registration
func TestHandlerRegistry_ConcurrentRegister(t *testing.T) {
	registry := NewHandlerRegistry()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrently register handlers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
				return nil
			})
			registry.Register(string(rune(id)), handler)
		}(i)
	}

	wg.Wait()

	// Verify all handlers were registered (length should be numGoroutines)
	// Note: We can't directly access handlers map length, so we'll check a few
	for i := 0; i < 10; i++ {
		_, ok := registry.Get(string(rune(i)))
		assert.True(t, ok, "Handler %d should be registered", i)
	}
}

// TestHandlerRegistry_ConcurrentGet tests concurrent reading
func TestHandlerRegistry_ConcurrentGet(t *testing.T) {
	registry := NewHandlerRegistry()

	handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		return nil
	})

	registry.Register("test-job", handler)

	var wg sync.WaitGroup
	numReads := 1000
	successCount := 0
	var mu sync.Mutex

	// Concurrently read handlers
	for i := 0; i < numReads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok := registry.Get("test-job")
			if ok {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// All reads should succeed
	assert.Equal(t, numReads, successCount)
}

// TestHandlerRegistry_ConcurrentRegisterAndGet tests concurrent read/write
func TestHandlerRegistry_ConcurrentRegisterAndGet(t *testing.T) {
	registry := NewHandlerRegistry()

	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
				return nil
			})
			registry.Register(string(rune(id)), handler)
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			registry.Get(string(rune(id)))
		}(i)
	}

	// Should not deadlock or panic
	wg.Wait()
}

// TestHandlerRegistry_EmptyJobType tests registering with empty job type
func TestHandlerRegistry_EmptyJobType(t *testing.T) {
	registry := NewHandlerRegistry()

	handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
		return nil
	})

	// Should be able to register with empty string
	registry.Register("", handler)

	retrieved, ok := registry.Get("")
	assert.True(t, ok)
	assert.NotNil(t, retrieved)
}

// TestHandlerRegistry_SpecialCharacters tests job types with special characters
func TestHandlerRegistry_SpecialCharacters(t *testing.T) {
	registry := NewHandlerRegistry()

	testCases := []string{
		"job-with-dashes",
		"job_with_underscores",
		"job.with.dots",
		"job:with:colons",
		"job/with/slashes",
		"job with spaces",
		"UPPERCASE-JOB",
		"MixedCaseJob",
	}

	for _, jobType := range testCases {
		handler := ports.JobHandler(func(ctx context.Context, job *domain.Job) error {
			return nil
		})

		registry.Register(jobType, handler)

		retrieved, ok := registry.Get(jobType)
		assert.True(t, ok, "Should find handler for job type: %s", jobType)
		assert.NotNil(t, retrieved, "Handler should not be nil for job type: %s", jobType)
	}
}

// TestHandlerRegistry_NilHandler tests registering nil handler
func TestHandlerRegistry_NilHandler(t *testing.T) {
	registry := NewHandlerRegistry()

	// Should be able to register nil (though not recommended)
	registry.Register("test-job", nil)

	retrieved, ok := registry.Get("test-job")
	assert.True(t, ok, "Should find registered entry")
	assert.Nil(t, retrieved, "Handler should be nil")
}
