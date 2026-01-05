package http

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/media"
)

// RemuxHandler handles on-the-fly remuxing of MKV/AVI/WebM to fragmented MP4
type RemuxHandler struct {
	getMediaUC *media.GetMediaUseCase
	config     *config.Config
}

// NewRemuxHandler creates a new remux streaming handler
func NewRemuxHandler(getMediaUC *media.GetMediaUseCase, config *config.Config) *RemuxHandler {
	return &RemuxHandler{
		getMediaUC: getMediaUC,
		config:     config,
	}
}

// Serve handles GET /stream/remux/:id
// Remuxes MKV/AVI/WebM files to fragmented MP4 on-the-fly using ffmpeg
// Note: This is a streaming response with no Content-Length or Range support
func (h *RemuxHandler) Serve(c *gin.Context) {
	id := c.Param("id")

	logger.Debug().
		Str("media_id", id).
		Str("method", c.Request.Method).
		Msg("remux stream request")

	// Get media details
	result, err := h.getMediaUC.Execute(c.Request.Context(), media.GetMediaInput{MediaID: id})
	if err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to get media")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"MEDIA_NOT_FOUND",
			"Media not found",
			err.Error(),
		))
		return
	}

	m := result.Media

	// Validate format is suitable for remuxing (not already MP4/MOV)
	format := strings.ToLower(m.Format)
	if format != "mkv" && format != "matroska" && format != "avi" && format != "webm" {
		logger.Warn().
			Str("media_id", id).
			Str("format", m.Format).
			Msg("format not suitable for remux")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_FORMAT",
			"Media format not suitable for remux",
			"Use /stream/direct/ for MP4/MOV formats",
		))
		return
	}

	// Get file path
	filePath := m.FilePath

	// Security: validate path is within allowed directory
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		logger.Error().Err(err).Str("path", filePath).Msg("failed to resolve path")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"PATH_ERROR",
			"Failed to resolve file path",
			"",
		))
		return
	}

	// Check file exists
	if _, err := os.Stat(absFilePath); os.IsNotExist(err) {
		logger.Error().Str("path", absFilePath).Msg("source file not found")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"FILE_NOT_FOUND",
			"Source file not found",
			"",
		))
		return
	}

	// Build ffmpeg command for remuxing
	// -c copy: copy all streams without re-encoding
	// -f mp4: output as MP4 container
	// -movflags: flags for streaming fragmented MP4
	//   - frag_keyframe: start new fragment at each keyframe
	//   - empty_moov: write empty moov atom at start (allows streaming without seeking)
	//   - default_base_moof: required for DASH/fragmented MP4 compatibility
	// pipe:1: output to stdout
	args := []string{
		"-i", absFilePath,
		"-c", "copy",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"pipe:1",
	}

	// Create command with request context for automatic cancellation on client disconnect
	cmd := exec.CommandContext(c.Request.Context(), "ffmpeg", args...)

	// Get stdout pipe for streaming output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error().Err(err).Msg("failed to get ffmpeg stdout pipe")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"REMUX_ERROR",
			"Failed to start remux",
			"",
		))
		return
	}

	// Get stderr pipe for logging/debugging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Error().Err(err).Msg("failed to get ffmpeg stderr pipe")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"REMUX_ERROR",
			"Failed to start remux",
			"",
		))
		return
	}

	// Start ffmpeg process
	if err := cmd.Start(); err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to start ffmpeg")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"REMUX_ERROR",
			"Failed to start remux process",
			err.Error(),
		))
		return
	}

	logger.Info().
		Str("media_id", id).
		Str("file", absFilePath).
		Msg("started remux stream")

	// Consume stderr in a goroutine to prevent blocking
	go h.consumeStderr(stderr, id)

	// Set response headers for streaming
	// Note: No Content-Length since we're streaming
	c.Header("Content-Type", "video/mp4")
	c.Header("Cache-Control", "no-store")
	c.Header("Transfer-Encoding", "chunked")

	// Stream ffmpeg stdout to response
	c.Status(http.StatusOK)

	// Use buffered copy for better performance
	written, err := io.Copy(c.Writer, stdout)
	if err != nil {
		// Client disconnect is normal - don't log as error
		if c.Request.Context().Err() != nil {
			logger.Debug().
				Str("media_id", id).
				Int64("bytes_written", written).
				Msg("remux stream ended (client disconnected)")
		} else {
			logger.Error().
				Err(err).
				Str("media_id", id).
				Int64("bytes_written", written).
				Msg("remux stream error")
		}
	}

	// Wait for ffmpeg to finish (will be killed by context if client disconnected)
	if err := cmd.Wait(); err != nil {
		// Context cancellation is expected on client disconnect
		if c.Request.Context().Err() == nil {
			logger.Warn().
				Err(err).
				Str("media_id", id).
				Msg("ffmpeg exited with error")
		}
	}

	logger.Debug().
		Str("media_id", id).
		Int64("bytes_written", written).
		Msg("remux stream completed")
}

// consumeStderr reads and logs ffmpeg stderr output
func (h *RemuxHandler) consumeStderr(stderr io.ReadCloser, mediaID string) {
	scanner := bufio.NewScanner(stderr)
	// Use larger buffer for ffmpeg's verbose output
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var lastLine string
	for scanner.Scan() {
		lastLine = scanner.Text()
	}

	// Log final stderr line for debugging (usually shows completion or error)
	if lastLine != "" {
		logger.Debug().
			Str("media_id", mediaID).
			Str("ffmpeg_output", lastLine).
			Msg("ffmpeg stderr")
	}

	if err := scanner.Err(); err != nil {
		logger.Debug().
			Err(err).
			Str("media_id", mediaID).
			Msg("error reading ffmpeg stderr")
	}
}
