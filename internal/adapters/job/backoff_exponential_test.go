package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewExponentialBackoff tests backoff initialization
func TestNewExponentialBackoff(t *testing.T) {
	backoff := NewExponentialBackoff(1, 60)

	assert.Equal(t, 1*time.Second, backoff.Base)
	assert.Equal(t, 60*time.Second, backoff.Max)
	assert.NotNil(t, backoff.rand)
}

// TestExponentialBackoff_NextDelay_FirstAttempt tests first attempt delay
func TestExponentialBackoff_NextDelay_FirstAttempt(t *testing.T) {
	backoff := NewExponentialBackoff(1, 60)

	delay := backoff.NextDelay(1)

	// First attempt should be close to base (1s) with +/-20% jitter
	// So range is 0.8s to 1.2s
	assert.GreaterOrEqual(t, delay, 800*time.Millisecond)
	assert.LessOrEqual(t, delay, 1200*time.Millisecond)
}

// TestExponentialBackoff_NextDelay_ExponentialGrowth tests exponential growth
func TestExponentialBackoff_NextDelay_ExponentialGrowth(t *testing.T) {
	backoff := NewExponentialBackoff(1, 60)

	// Test multiple attempts to verify exponential growth
	// Attempt 1: 1s * 2^0 = 1s
	// Attempt 2: 1s * 2^1 = 2s
	// Attempt 3: 1s * 2^2 = 4s
	// Attempt 4: 1s * 2^3 = 8s
	// Attempt 5: 1s * 2^4 = 16s

	delays := make([]time.Duration, 5)
	for i := 1; i <= 5; i++ {
		delays[i-1] = backoff.NextDelay(i)
	}

	// Each delay should generally be larger than the previous (accounting for jitter)
	// We'll check that delay grows overall
	assert.Less(t, delays[0], 2*time.Second)    // Attempt 1: ~1s
	assert.Greater(t, delays[1], 1*time.Second) // Attempt 2: ~2s
	assert.Greater(t, delays[2], 2*time.Second) // Attempt 3: ~4s
	assert.Greater(t, delays[3], 4*time.Second) // Attempt 4: ~8s
	assert.Greater(t, delays[4], 8*time.Second) // Attempt 5: ~16s
}

// TestExponentialBackoff_NextDelay_MaxCap tests max duration cap
func TestExponentialBackoff_NextDelay_MaxCap(t *testing.T) {
	backoff := NewExponentialBackoff(1, 10)

	// After enough attempts, delay should be capped at max
	// Attempt 10: 1s * 2^9 = 512s, but capped at 10s
	delay := backoff.NextDelay(10)

	// Should be at or near max (10s) with jitter
	// Max with jitter could be up to 12s (10s * 1.2)
	assert.LessOrEqual(t, delay, 12*time.Second)
	assert.GreaterOrEqual(t, delay, 8*time.Second) // At least 80% of max
}

// TestExponentialBackoff_NextDelay_ZeroAttempt tests zero attempt handling
func TestExponentialBackoff_NextDelay_ZeroAttempt(t *testing.T) {
	backoff := NewExponentialBackoff(1, 60)

	// Zero or negative attempts should be treated as attempt 1
	delay := backoff.NextDelay(0)

	assert.GreaterOrEqual(t, delay, 800*time.Millisecond)
	assert.LessOrEqual(t, delay, 1200*time.Millisecond)
}

// TestExponentialBackoff_NextDelay_NegativeAttempt tests negative attempt handling
func TestExponentialBackoff_NextDelay_NegativeAttempt(t *testing.T) {
	backoff := NewExponentialBackoff(1, 60)

	delay := backoff.NextDelay(-1)

	assert.GreaterOrEqual(t, delay, 800*time.Millisecond)
	assert.LessOrEqual(t, delay, 1200*time.Millisecond)
}

// TestExponentialBackoff_NextDelay_Jitter tests jitter randomization
func TestExponentialBackoff_NextDelay_Jitter(t *testing.T) {
	backoff := NewExponentialBackoff(5, 300)

	// Get multiple delays for same attempt - should vary due to jitter
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = backoff.NextDelay(3) // Attempt 3: 5s * 2^2 = 20s base
	}

	// Check that we get variation (not all the same)
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}
	assert.False(t, allSame, "Expected jitter to produce different values")

	// All values should be within jitter range of 20s (16s to 24s)
	for _, delay := range delays {
		assert.GreaterOrEqual(t, delay, 16*time.Second)
		assert.LessOrEqual(t, delay, 24*time.Second)
	}
}

// TestExponentialBackoff_NextDelay_DifferentBaseAndMax tests different configurations
func TestExponentialBackoff_NextDelay_DifferentBaseAndMax(t *testing.T) {
	testCases := []struct {
		name        string
		base        int
		max         int
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name:        "Small base, small max",
			base:        1,
			max:         5,
			attempt:     3,
			expectedMin: 3 * time.Second, // 4s * 0.8
			expectedMax: 6 * time.Second, // 5s max * 1.2 (with jitter)
		},
		{
			name:        "Large base",
			base:        10,
			max:         600,
			attempt:     1,
			expectedMin: 8 * time.Second,  // 10s * 0.8
			expectedMax: 12 * time.Second, // 10s * 1.2
		},
		{
			name:        "Very large max",
			base:        2,
			max:         3600,
			attempt:     5,
			expectedMin: 25 * time.Second, // 32s * 0.8
			expectedMax: 39 * time.Second, // 32s * 1.2
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			backoff := NewExponentialBackoff(tc.base, tc.max)
			delay := backoff.NextDelay(tc.attempt)

			assert.GreaterOrEqual(t, delay, tc.expectedMin,
				"Delay should be >= %v, got %v", tc.expectedMin, delay)
			assert.LessOrEqual(t, delay, tc.expectedMax,
				"Delay should be <= %v, got %v", tc.expectedMax, delay)
		})
	}
}

// TestExponentialBackoff_NextDelay_ConsistentBase tests that delay never goes below base
func TestExponentialBackoff_NextDelay_ConsistentBase(t *testing.T) {
	backoff := NewExponentialBackoff(5, 300)

	// Test many attempts - delay should never go below base
	for attempt := 1; attempt <= 20; attempt++ {
		delay := backoff.NextDelay(attempt)
		// Even with negative jitter, should not go below base
		assert.GreaterOrEqual(t, delay, 5*time.Second,
			"Delay for attempt %d should be >= base (5s), got %v", attempt, delay)
	}
}

// TestExponentialBackoff_NextDelay_ConsistentMax tests that delay never exceeds max
func TestExponentialBackoff_NextDelay_ConsistentMax(t *testing.T) {
	backoff := NewExponentialBackoff(1, 30)

	// Test many high attempts - delay should never exceed max
	for attempt := 10; attempt <= 50; attempt++ {
		delay := backoff.NextDelay(attempt)
		// Even with positive jitter, should not exceed max
		assert.LessOrEqual(t, delay, 30*time.Second,
			"Delay for attempt %d should be <= max (30s), got %v", attempt, delay)
	}
}

// TestExponentialBackoff_NextDelay_SequentialAttempts tests realistic retry sequence
func TestExponentialBackoff_NextDelay_SequentialAttempts(t *testing.T) {
	backoff := NewExponentialBackoff(1, 60)

	// Simulate a job failing multiple times
	for attempt := 1; attempt <= 6; attempt++ {
		delay := backoff.NextDelay(attempt)

		// Log for debugging
		t.Logf("Attempt %d: delay = %v", attempt, delay)

		// Delay should be positive
		assert.Greater(t, delay, time.Duration(0))
	}
}
