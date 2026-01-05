package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/media"
)

// DirectHandler handles direct file streaming with Range request support
type DirectHandler struct {
	getMediaUC *media.GetMediaUseCase
	config     *config.Config
}

// NewDirectHandler creates a new direct streaming handler
func NewDirectHandler(getMediaUC *media.GetMediaUseCase, config *config.Config) *DirectHandler {
	return &DirectHandler{
		getMediaUC: getMediaUC,
		config:     config,
	}
}

// Serve handles GET /stream/direct/:id
// Serves the original media file with full HTTP Range request support
func (h *DirectHandler) Serve(c *gin.Context) {
	id := c.Param("id")

	logger.Debug().
		Str("media_id", id).
		Str("method", c.Request.Method).
		Str("range", c.GetHeader("Range")).
		Msg("direct play request")

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

	// Validate format is suitable for direct play
	format := strings.ToLower(m.Format)
	if format != "mp4" && format != "mov" && format != "m4v" {
		logger.Warn().
			Str("media_id", id).
			Str("format", m.Format).
			Msg("format not suitable for direct play")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_FORMAT",
			"Media format not suitable for direct play",
			"Use /stream/remux/ for non-MP4 formats",
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
	fileInfo, err := os.Stat(absFilePath)
	if os.IsNotExist(err) {
		logger.Error().Str("path", absFilePath).Msg("source file not found")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"FILE_NOT_FOUND",
			"Source file not found",
			"",
		))
		return
	}
	if err != nil {
		logger.Error().Err(err).Str("path", absFilePath).Msg("failed to stat file")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"FILE_ERROR",
			"Failed to access file",
			"",
		))
		return
	}

	// Open file
	file, err := os.Open(absFilePath)
	if err != nil {
		logger.Error().Err(err).Str("path", absFilePath).Msg("failed to open file")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"FILE_ERROR",
			"Failed to open file",
			"",
		))
		return
	}
	defer file.Close()

	// Determine content type
	contentType := "video/mp4"
	if format == "mov" {
		contentType = "video/quicktime"
	}

	// Set headers
	c.Header("Content-Type", contentType)
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "private, max-age=3600")

	// Use http.ServeContent for automatic Range request handling
	http.ServeContent(c.Writer, c.Request, m.Filename, fileInfo.ModTime(), file)
}
