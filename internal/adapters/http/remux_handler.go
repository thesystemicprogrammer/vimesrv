package http

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/media"
)

// Browser-compatible audio codecs that can be copied without transcoding
var browserCompatibleAudioCodecs = map[string]bool{
	"aac":    true,
	"mp3":    true,
	"opus":   true,
	"vorbis": true,
	"flac":   true,
}

// isBrowserCompatibleAudio checks if an audio codec can be played natively by browsers
func isBrowserCompatibleAudio(codec string) bool {
	return browserCompatibleAudioCodecs[strings.ToLower(codec)]
}

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
// Query params:
//   - audio: audio stream index to use (defaults to first audio stream)
//
// Note: This is a streaming response with no Content-Length or Range support
func (h *RemuxHandler) Serve(c *gin.Context) {
	id := c.Param("id")
	audioParam := c.Query("audio")

	logger.Debug().
		Str("media_id", id).
		Str("audio_param", audioParam).
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
	// Note: ffprobe may return comma-separated formats like "matroska,webm", so we use contains
	format := strings.ToLower(m.Format)
	if !strings.Contains(format, "mkv") && !strings.Contains(format, "matroska") &&
		!strings.Contains(format, "avi") && !strings.Contains(format, "webm") {
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

	// Determine which audio stream to use
	selectedAudio, err := h.selectAudioStream(result.AudioStreams, audioParam)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("media_id", id).
			Str("audio_param", audioParam).
			Msg("invalid audio stream selection")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_AUDIO",
			"Invalid audio stream selection",
			err.Error(),
		))
		return
	}

	// Validate audio codec is browser-compatible (required for direct-stream)
	if selectedAudio != nil && !isBrowserCompatibleAudio(selectedAudio.Codec) {
		logger.Warn().
			Str("media_id", id).
			Int("audio_stream", selectedAudio.StreamIndex).
			Str("codec", selectedAudio.Codec).
			Msg("audio codec not browser-compatible")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INCOMPATIBLE_AUDIO",
			"Audio codec not compatible with browser playback",
			fmt.Sprintf("Audio stream %d uses %s codec which browsers cannot decode. Use DASH streaming instead.", selectedAudio.StreamIndex, selectedAudio.Codec),
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
	args := h.buildFFmpegArgs(absFilePath, selectedAudio)

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
		Int("audio_stream", h.getAudioStreamIndex(selectedAudio)).
		Str("audio_codec", h.getAudioCodec(selectedAudio)).
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

// selectAudioStream determines which audio stream to use based on the query param
// If no param is provided, defaults to the first audio stream
// Returns nil if no audio streams exist
func (h *RemuxHandler) selectAudioStream(audioStreams []*domain.AudioStream, audioParam string) (*domain.AudioStream, error) {
	if len(audioStreams) == 0 {
		return nil, nil
	}

	// Default to first audio stream
	if audioParam == "" {
		return audioStreams[0], nil
	}

	// Parse audio stream index from query param
	requestedIndex, err := strconv.Atoi(audioParam)
	if err != nil {
		return nil, fmt.Errorf("audio parameter must be a number: %w", err)
	}

	// Find the audio stream with matching stream index
	for _, as := range audioStreams {
		if as.StreamIndex == requestedIndex {
			return as, nil
		}
	}

	return nil, fmt.Errorf("audio stream with index %d not found", requestedIndex)
}

// buildFFmpegArgs constructs the ffmpeg command arguments for remuxing
// Copies video and audio streams from source to fragmented MP4
func (h *RemuxHandler) buildFFmpegArgs(inputPath string, audioStream *domain.AudioStream) []string {
	args := []string{
		"-i", inputPath,
	}

	// Map video stream (copy)
	args = append(args, "-map", "0:v:0", "-c:v", "copy")

	// Map and copy audio stream
	if audioStream != nil {
		// Map the specific audio stream by its index in the file
		args = append(args, "-map", fmt.Sprintf("0:%d", audioStream.StreamIndex))
		args = append(args, "-c:a", "copy")
	}

	// Output format settings
	// -f mp4: output as MP4 container
	// -movflags: flags for streaming fragmented MP4
	//   - frag_keyframe: start new fragment at each keyframe
	//   - empty_moov: write empty moov atom at start (allows streaming without seeking)
	//   - default_base_moof: required for DASH/fragmented MP4 compatibility
	// pipe:1: output to stdout
	args = append(args,
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"pipe:1",
	)

	return args
}

// getAudioStreamIndex safely gets the stream index from an audio stream
func (h *RemuxHandler) getAudioStreamIndex(as *domain.AudioStream) int {
	if as == nil {
		return -1
	}
	return as.StreamIndex
}

// getAudioCodec safely gets the codec from an audio stream
func (h *RemuxHandler) getAudioCodec(as *domain.AudioStream) string {
	if as == nil {
		return ""
	}
	return as.Codec
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
