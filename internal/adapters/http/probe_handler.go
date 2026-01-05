package http

import (
	"crypto/rand"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

const (
	// DefaultProbeBytes is the default size of probe data (2 MB)
	DefaultProbeBytes = 2_000_000
	// MaxProbeBytes is the maximum allowed probe size (10 MB)
	MaxProbeBytes = 10_000_000
)

// ProbeHandler handles bandwidth measurement requests
type ProbeHandler struct{}

// NewProbeHandler creates a new probe handler
func NewProbeHandler() *ProbeHandler {
	return &ProbeHandler{}
}

// Serve handles GET /stream/probe?bytes=N
// Returns N bytes of random data for bandwidth measurement
func (h *ProbeHandler) Serve(c *gin.Context) {
	bytesParam := c.DefaultQuery("bytes", strconv.Itoa(DefaultProbeBytes))
	numBytes, err := strconv.Atoi(bytesParam)
	if err != nil || numBytes <= 0 {
		numBytes = DefaultProbeBytes
	}
	if numBytes > MaxProbeBytes {
		numBytes = MaxProbeBytes
	}

	logger.Debug().
		Int("bytes", numBytes).
		Msg("bandwidth probe request")

	// Generate random bytes
	data := make([]byte, numBytes)
	_, err = rand.Read(data)
	if err != nil {
		// Fallback to zeros if random fails (shouldn't happen)
		logger.Warn().Err(err).Msg("failed to generate random data, using zeros")
		data = make([]byte, numBytes)
	}

	// Set headers to prevent caching
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.Itoa(numBytes))
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	c.Data(http.StatusOK, "application/octet-stream", data)
}
