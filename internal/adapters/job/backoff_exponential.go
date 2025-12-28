package job

import (
	"math"
	"math/rand"
	"time"
)

type ExponentialBackoff struct {
	Base time.Duration
	Max  time.Duration
	rand *rand.Rand
}

func NewExponentialBackoff(base, max time.Duration) *ExponentialBackoff {
	return &ExponentialBackoff{
		Base: base,
		Max:  max,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (backoff *ExponentialBackoff) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}

	// base * 2^(attempt-1) with +/-20% jitter
	raw := min(time.Duration(float64(backoff.Base)*math.Pow(2, float64(attempt-1))), backoff.Max)
	jitter := 0.2
	factor := 1 + (backoff.rand.Float64()*2-1)*jitter
	duration := min(max(time.Duration(float64(raw)*factor), backoff.Base), backoff.Max)
	return duration
}
