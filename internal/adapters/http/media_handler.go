package http

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/media"
)

// MediaHandler handles media API endpoints
type MediaHandler struct {
	listMediaUC   *media.ListMediaUseCase
	getMediaUC    *media.GetMediaUseCase
	deleteMediaUC *media.DeleteMediaUseCase
	config        *config.Config
}

// NewMediaHandler creates a new media handler
func NewMediaHandler(
	listMediaUC *media.ListMediaUseCase,
	getMediaUC *media.GetMediaUseCase,
	deleteMediaUC *media.DeleteMediaUseCase,
	config *config.Config,
) *MediaHandler {
	return &MediaHandler{
		listMediaUC:   listMediaUC,
		getMediaUC:    getMediaUC,
		deleteMediaUC: deleteMediaUC,
		config:        config,
	}
}

// RegisterRoutes registers media routes on a protected router group
func (h *MediaHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/media", h.ListMedia)
	router.GET("/media/:id", h.GetMedia)
	logger.Debug().Msg("Media routes registered")
}

// RegisterAdminRoutes registers admin-only media routes (delete operations)
func (h *MediaHandler) RegisterAdminRoutes(router *gin.RouterGroup) {
	adminGroup := router.Group("/media")
	adminGroup.Use(h.requireAdmin())
	{
		adminGroup.DELETE("/:id", h.DeleteMedia)
	}
	logger.Debug().Msg("Media admin routes registered")
}

// requireAdmin middleware checks if the user has admin role
func (h *MediaHandler) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, server.ErrorResponse(
				"FORBIDDEN",
				"Access denied",
				"",
			))
			return
		}

		if role != string(shared.RoleAdmin) {
			c.AbortWithStatusJSON(http.StatusForbidden, server.ErrorResponse(
				"FORBIDDEN",
				"Admin access required",
				"",
			))
			return
		}

		c.Next()
	}
}

// MediaDetailResponse represents the detailed media response for a single item
type MediaDetailResponse struct {
	ID                 string           `json:"id"`
	Title              string           `json:"title"`
	Filename           string           `json:"filename"`
	Duration           int              `json:"duration"`
	Resolution         string           `json:"resolution"`
	Width              int              `json:"width"`
	Height             int              `json:"height"`
	Status             string           `json:"status"`
	DashManifestURL    string           `json:"dash_manifest_url"`
	AudioStreams       []AudioStreamDTO `json:"audio_streams"`
	SubtitleStreams    []SubtitleDTO    `json:"subtitle_streams"`
	AvailableQualities []string         `json:"available_qualities"`
}

// AudioStreamDTO represents an audio stream in the API response
type AudioStreamDTO struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Channels int    `json:"channels"`
}

// SubtitleDTO represents a subtitle stream in the API response
type SubtitleDTO struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Title    string `json:"title"`
}

// ListMedia handles GET /api/v1/media
func (h *MediaHandler) ListMedia(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	logger.Debug().
		Int("page", page).
		Int("per_page", perPage).
		Msg("listing media")

	// Execute use case
	result, err := h.listMediaUC.Execute(c.Request.Context(), media.ListMediaInput{
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to list media")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"LIST_FAILED",
			"Failed to list media",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// GetMedia handles GET /api/v1/media/:id
func (h *MediaHandler) GetMedia(c *gin.Context) {
	id := c.Param("id")

	logger.Debug().
		Str("media_id", id).
		Msg("getting media details")

	// Execute use case
	result, err := h.getMediaUC.Execute(c.Request.Context(), media.GetMediaInput{
		MediaID: id,
	})
	if err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to get media")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"MEDIA_NOT_FOUND",
			"Media not found",
			err.Error(),
		))
		return
	}

	// Build response
	response := h.buildMediaDetailResponse(result)

	c.JSON(http.StatusOK, server.SuccessResponse(response))
}

// buildMediaDetailResponse converts the use case output to API response
func (h *MediaHandler) buildMediaDetailResponse(result *media.GetMediaOutput) MediaDetailResponse {
	m := result.Media

	title := m.Title
	if title == "" {
		title = m.Filename
	}

	// Build audio streams
	audioStreams := make([]AudioStreamDTO, len(result.AudioStreams))
	for i, as := range result.AudioStreams {
		title := as.Title
		if title == "" {
			if as.Language != "" {
				title = shared.LanguageCodeToName(as.Language)
			} else {
				title = fmt.Sprintf("Audio %d", as.StreamIndex)
			}
		}
		audioStreams[i] = AudioStreamDTO{
			Index:    as.StreamIndex,
			Language: as.Language,
			Title:    title,
			Channels: as.Channels,
		}
	}

	// Build subtitle streams
	subtitleStreams := make([]SubtitleDTO, len(result.SubtitleStreams))
	for i, ss := range result.SubtitleStreams {
		title := ss.Title
		if title == "" {
			if ss.Language != "" {
				title = shared.LanguageCodeToName(ss.Language)
			} else {
				title = fmt.Sprintf("Subtitle %d", ss.StreamIndex)
			}
		}
		subtitleStreams[i] = SubtitleDTO{
			Index:    ss.StreamIndex,
			Language: ss.Language,
			Title:    title,
		}
	}

	// Get available qualities from completed transcodes
	qualitySet := make(map[string]bool)
	for _, t := range result.Transcodes {
		if t.IsCompleted() && t.TrackType == domain.TrackTypeVideo && t.Quality != "" {
			qualitySet[t.Quality] = true
		}
	}
	qualities := make([]string, 0, len(qualitySet))
	for q := range qualitySet {
		qualities = append(qualities, q)
	}

	return MediaDetailResponse{
		ID:                 m.ID,
		Title:              title,
		Filename:           m.Filename,
		Duration:           m.Duration,
		Resolution:         m.Resolution,
		Width:              m.Width,
		Height:             m.Height,
		Status:             m.Status,
		DashManifestURL:    fmt.Sprintf("/stream/dash/%s/manifest.mpd", m.ID),
		AudioStreams:       audioStreams,
		SubtitleStreams:    subtitleStreams,
		AvailableQualities: qualities,
	}
}

// DeleteMedia handles DELETE /api/v1/media/:id (admin only)
// Moves source file to trash and permanently deletes transcoded files
func (h *MediaHandler) DeleteMedia(c *gin.Context) {
	id := c.Param("id")

	logger.Info().
		Str("media_id", id).
		Msg("deleting media")

	err := h.deleteMediaUC.Execute(c.Request.Context(), media.DeleteMediaInput{
		MediaID: id,
	})
	if err != nil {
		if errors.Is(err, shared.ErrMediaNotFound) {
			c.JSON(http.StatusNotFound, server.ErrorResponse(
				"MEDIA_NOT_FOUND",
				"Media not found",
				err.Error(),
			))
			return
		}
		if errors.Is(err, shared.ErrMediaHasRunningJobs) {
			c.JSON(http.StatusConflict, server.ErrorResponse(
				"MEDIA_HAS_RUNNING_JOBS",
				"Cannot delete media with running transcode jobs",
				"Wait for transcode jobs to complete before deleting",
			))
			return
		}
		logger.Error().Err(err).Str("media_id", id).Msg("failed to delete media")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"DELETE_FAILED",
			"Failed to delete media",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(gin.H{"deleted": true}))
}
