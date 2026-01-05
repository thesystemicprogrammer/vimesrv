package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/transcode"
)

// TranscodeAdminHandler handles transcoding administration API endpoints
type TranscodeAdminHandler struct {
	getDetailsUC  *transcode.GetTranscodingDetailsUseCase
	addUC         *transcode.AddTranscodingUseCase
	recreateUC    *transcode.RecreateTranscodingUseCase
	deleteUC      *transcode.DeleteTranscodingUseCase
	searchMediaUC *transcode.SearchMediaForTranscodingsUseCase
	getProfilesUC *transcode.GetQualityProfilesUseCase
}

// NewTranscodeAdminHandler creates a new TranscodeAdminHandler
func NewTranscodeAdminHandler(
	getDetailsUC *transcode.GetTranscodingDetailsUseCase,
	addUC *transcode.AddTranscodingUseCase,
	recreateUC *transcode.RecreateTranscodingUseCase,
	deleteUC *transcode.DeleteTranscodingUseCase,
	searchMediaUC *transcode.SearchMediaForTranscodingsUseCase,
	getProfilesUC *transcode.GetQualityProfilesUseCase,
) *TranscodeAdminHandler {
	return &TranscodeAdminHandler{
		getDetailsUC:  getDetailsUC,
		addUC:         addUC,
		recreateUC:    recreateUC,
		deleteUC:      deleteUC,
		searchMediaUC: searchMediaUC,
		getProfilesUC: getProfilesUC,
	}
}

// RegisterRoutes registers transcoding admin routes on a protected router group
// These routes require manager or admin role
func (h *TranscodeAdminHandler) RegisterRoutes(router *gin.RouterGroup) {
	adminGroup := router.Group("/admin/transcodings")
	adminGroup.Use(h.requireManager())
	{
		// Get quality profiles configuration
		adminGroup.GET("/config", h.GetConfig)

		// Search media files
		adminGroup.GET("/search", h.SearchMedia)

		// Get transcoding details for a media file
		adminGroup.GET("/:mediaId", h.GetDetails)

		// Add new transcoding
		adminGroup.POST("/:mediaId", h.AddTranscoding)

		// Recreate existing transcoding
		adminGroup.POST("/:transcodeId/recreate", h.RecreateTranscoding)

		// Delete transcoding
		adminGroup.DELETE("/:transcodeId", h.DeleteTranscoding)
	}
	logger.Debug().Msg("Transcode admin routes registered")
}

// requireManager middleware checks if the user has admin or manager role
func (h *TranscodeAdminHandler) requireManager() gin.HandlerFunc {
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

		roleStr := role.(string)
		if roleStr != string(shared.RoleAdmin) && roleStr != string(shared.RoleManager) {
			c.AbortWithStatusJSON(http.StatusForbidden, server.ErrorResponse(
				"FORBIDDEN",
				"Manager or admin access required",
				"",
			))
			return
		}

		c.Next()
	}
}

// GetConfig handles GET /api/v1/admin/transcodings/config
func (h *TranscodeAdminHandler) GetConfig(c *gin.Context) {
	result, err := h.getProfilesUC.Execute(c.Request.Context())
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get quality profiles")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"GET_CONFIG_FAILED",
			"Failed to get quality profiles",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// SearchMedia handles GET /api/v1/admin/transcodings/search
func (h *TranscodeAdminHandler) SearchMedia(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusOK, server.SuccessResponse(transcode.SearchMediaOutput{
			Results: []transcode.MediaSearchResult{},
			Count:   0,
		}))
		return
	}

	result, err := h.searchMediaUC.Execute(c.Request.Context(), transcode.SearchMediaInput{
		Query: query,
		Limit: 20,
	})
	if err != nil {
		logger.Error().Err(err).Str("query", query).Msg("Failed to search media")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"SEARCH_FAILED",
			"Failed to search media",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// GetDetails handles GET /api/v1/admin/transcodings/:mediaId
func (h *TranscodeAdminHandler) GetDetails(c *gin.Context) {
	mediaID := c.Param("mediaId")

	result, err := h.getDetailsUC.Execute(c.Request.Context(), transcode.GetTranscodingDetailsInput{
		MediaID: mediaID,
	})
	if err != nil {
		logger.Error().Err(err).Str("media_id", mediaID).Msg("Failed to get transcoding details")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"MEDIA_NOT_FOUND",
			"Media not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// AddTranscodingRequest is the request body for adding a new transcoding
type AddTranscodingRequest struct {
	Type       string `json:"type" binding:"required,oneof=video audio subtitle"`
	Quality    string `json:"quality,omitempty"`
	TrackIndex int    `json:"track_index,omitempty"`
}

// AddTranscoding handles POST /api/v1/admin/transcodings/:mediaId
func (h *TranscodeAdminHandler) AddTranscoding(c *gin.Context) {
	mediaID := c.Param("mediaId")

	var req AddTranscodingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Validate: video requires quality, audio/subtitle require track_index
	if req.Type == "video" && req.Quality == "" {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_REQUEST",
			"Quality is required for video transcoding",
			"",
		))
		return
	}

	result, err := h.addUC.Execute(c.Request.Context(), transcode.AddTranscodingInput{
		MediaID:    mediaID,
		Type:       req.Type,
		Quality:    req.Quality,
		TrackIndex: req.TrackIndex,
	})
	if err != nil {
		logger.Error().Err(err).
			Str("media_id", mediaID).
			Str("type", req.Type).
			Msg("Failed to add transcoding")

		// Check for specific error types
		errMsg := err.Error()
		if contains(errMsg, "not found") {
			c.JSON(http.StatusNotFound, server.ErrorResponse(
				"NOT_FOUND",
				"Resource not found",
				errMsg,
			))
			return
		}

		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"ADD_FAILED",
			"Failed to add transcoding",
			errMsg,
		))
		return
	}

	c.JSON(http.StatusCreated, server.SuccessResponse(result))
}

// RecreateTranscoding handles POST /api/v1/admin/transcodings/:transcodeId/recreate
func (h *TranscodeAdminHandler) RecreateTranscoding(c *gin.Context) {
	transcodeID := c.Param("transcodeId")

	result, err := h.recreateUC.Execute(c.Request.Context(), transcode.RecreateTranscodingInput{
		TranscodeID: transcodeID,
	})
	if err != nil {
		logger.Error().Err(err).Str("transcode_id", transcodeID).Msg("Failed to recreate transcoding")

		errMsg := err.Error()
		if contains(errMsg, "not found") {
			c.JSON(http.StatusNotFound, server.ErrorResponse(
				"TRANSCODE_NOT_FOUND",
				"Transcode not found",
				errMsg,
			))
			return
		}
		if contains(errMsg, "currently processing") {
			c.JSON(http.StatusConflict, server.ErrorResponse(
				"TRANSCODE_PROCESSING",
				"Cannot recreate transcode that is currently processing",
				errMsg,
			))
			return
		}

		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"RECREATE_FAILED",
			"Failed to recreate transcoding",
			errMsg,
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// DeleteTranscoding handles DELETE /api/v1/admin/transcodings/:transcodeId
func (h *TranscodeAdminHandler) DeleteTranscoding(c *gin.Context) {
	transcodeID := c.Param("transcodeId")

	result, err := h.deleteUC.Execute(c.Request.Context(), transcode.DeleteTranscodingInput{
		TranscodeID: transcodeID,
	})
	if err != nil {
		logger.Error().Err(err).Str("transcode_id", transcodeID).Msg("Failed to delete transcoding")

		errMsg := err.Error()
		if contains(errMsg, "not found") {
			c.JSON(http.StatusNotFound, server.ErrorResponse(
				"TRANSCODE_NOT_FOUND",
				"Transcode not found",
				errMsg,
			))
			return
		}
		if contains(errMsg, "currently processing") {
			c.JSON(http.StatusConflict, server.ErrorResponse(
				"TRANSCODE_PROCESSING",
				"Cannot delete transcode that is currently processing",
				errMsg,
			))
			return
		}

		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"DELETE_FAILED",
			"Failed to delete transcoding",
			errMsg,
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
