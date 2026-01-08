package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/watch_progress"
)

// WatchProgressHandler handles watch progress endpoints
type WatchProgressHandler struct {
	recordProgressUseCase             *watch_progress.RecordWatchProgressUseCase
	getProgressUseCase                *watch_progress.GetWatchProgressUseCase
	getContinueWatchingUseCase        *watch_progress.GetContinueWatchingUseCase
	removeFromContinueWatchingUseCase *watch_progress.RemoveFromContinueWatchingUseCase
}

// NewWatchProgressHandler creates a new watch progress handler
func NewWatchProgressHandler(
	recordProgressUseCase *watch_progress.RecordWatchProgressUseCase,
	getProgressUseCase *watch_progress.GetWatchProgressUseCase,
	getContinueWatchingUseCase *watch_progress.GetContinueWatchingUseCase,
	removeFromContinueWatchingUseCase *watch_progress.RemoveFromContinueWatchingUseCase,
) *WatchProgressHandler {
	return &WatchProgressHandler{
		recordProgressUseCase:             recordProgressUseCase,
		getProgressUseCase:                getProgressUseCase,
		getContinueWatchingUseCase:        getContinueWatchingUseCase,
		removeFromContinueWatchingUseCase: removeFromContinueWatchingUseCase,
	}
}

// RegisterRoutes registers watch progress routes
func (h *WatchProgressHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/playback/progress", h.SaveProgress)
	router.GET("/playback/progress/:media_id", h.GetProgress)
	router.GET("/library/continue-watching", h.GetContinueWatching)
	router.DELETE("/playback/progress/:media_id", h.RemoveFromContinueWatching)
}

// SaveProgressRequest represents the request to save watch progress
type SaveProgressRequest struct {
	MediaID         *string `json:"media_id"`
	EpisodeID       *int64  `json:"episode_metadata_id"`
	PositionSeconds int     `json:"position_seconds" binding:"required"`
	DurationSeconds int     `json:"duration_seconds" binding:"required"`
}

// WatchProgressResponse represents watch progress in API responses
type WatchProgressResponse struct {
	MediaID           *string `json:"media_id,omitempty"`
	EpisodeMetadataID *int64  `json:"episode_metadata_id,omitempty"`
	PositionSeconds   int     `json:"position_seconds"`
	DurationSeconds   int     `json:"duration_seconds"`
	ProgressPercent   float64 `json:"progress_percent"`
	LastWatchedAt     string  `json:"last_watched_at"`
	Completed         bool    `json:"completed"`
}

// ContinueWatchingItemResponse represents a continue watching item
type ContinueWatchingItemResponse struct {
	ID                string  `json:"id"`
	MediaID           *string `json:"media_id,omitempty"`
	EpisodeMetadataID *int64  `json:"episode_metadata_id,omitempty"`
	Title             string  `json:"title"`
	PosterPath        *string `json:"poster_path,omitempty"`
	BackdropPath      *string `json:"backdrop_path,omitempty"`
	MediaType         string  `json:"media_type"`
	Year              *int64  `json:"year,omitempty"`
	Resolution        *string `json:"resolution,omitempty"`
	PositionSeconds   int     `json:"position_seconds"`
	DurationSeconds   int     `json:"duration_seconds"`
	ProgressPercent   float64 `json:"progress_percent"`
	LastWatchedAt     string  `json:"last_watched_at"`
	// Episode-specific fields
	SeriesName       *string `json:"series_name,omitempty"`
	SeriesMetadataID *int64  `json:"series_metadata_id,omitempty"`
	SeasonNumber     *int64  `json:"season_number,omitempty"`
	EpisodeNumber    *int64  `json:"episode_number,omitempty"`
	EpisodeName      *string `json:"episode_name,omitempty"`
}

// toWatchProgressResponse converts domain watch progress to API response
func toWatchProgressResponse(wp *domain.WatchProgress) WatchProgressResponse {
	resp := WatchProgressResponse{
		PositionSeconds: wp.PositionSeconds,
		DurationSeconds: wp.DurationSeconds,
		ProgressPercent: wp.ProgressPercent,
		LastWatchedAt:   wp.LastWatchedAt.Format("2006-01-02T15:04:05Z"),
		Completed:       wp.Completed,
	}
	if wp.MediaID.Valid {
		resp.MediaID = &wp.MediaID.String
	}
	if wp.EpisodeMetadataID.Valid {
		resp.EpisodeMetadataID = &wp.EpisodeMetadataID.Int64
	}
	return resp
}

// toContinueWatchingResponse converts domain continue watching item to API response
func toContinueWatchingResponse(item domain.ContinueWatchingItem) ContinueWatchingItemResponse {
	resp := ContinueWatchingItemResponse{
		ID:              item.ID,
		Title:           item.Title,
		MediaType:       item.MediaType,
		PositionSeconds: item.PositionSeconds,
		DurationSeconds: item.DurationSeconds,
		ProgressPercent: item.ProgressPercent,
		LastWatchedAt:   item.LastWatchedAt.Format("2006-01-02T15:04:05Z"),
	}

	if item.MediaID.Valid {
		resp.MediaID = &item.MediaID.String
	}
	if item.EpisodeMetadataID.Valid {
		resp.EpisodeMetadataID = &item.EpisodeMetadataID.Int64
	}
	if item.PosterPath.Valid {
		resp.PosterPath = &item.PosterPath.String
	}
	if item.BackdropPath.Valid {
		resp.BackdropPath = &item.BackdropPath.String
	}
	if item.Year.Valid {
		resp.Year = &item.Year.Int64
	}
	if item.Resolution.Valid {
		resp.Resolution = &item.Resolution.String
	}
	if item.SeriesName.Valid {
		resp.SeriesName = &item.SeriesName.String
	}
	if item.SeriesMetadataID.Valid {
		resp.SeriesMetadataID = &item.SeriesMetadataID.Int64
	}
	if item.SeasonNumber.Valid {
		resp.SeasonNumber = &item.SeasonNumber.Int64
	}
	if item.EpisodeNumber.Valid {
		resp.EpisodeNumber = &item.EpisodeNumber.Int64
	}
	if item.EpisodeName.Valid {
		resp.EpisodeName = &item.EpisodeName.String
	}

	return resp
}

// SaveProgress handles POST /api/playback/progress
func (h *WatchProgressHandler) SaveProgress(c *gin.Context) {
	var req SaveProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn().Err(err).Msg("invalid request body")
		c.JSON(http.StatusBadRequest, server.ErrorResponse("INVALID_REQUEST", "Invalid request body", ""))
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, server.ErrorResponse("UNAUTHORIZED", "User not authenticated", ""))
		return
	}

	input := watch_progress.RecordProgressInput{
		UserID:          userID.(string),
		MediaID:         req.MediaID,
		EpisodeID:       req.EpisodeID,
		PositionSeconds: req.PositionSeconds,
		DurationSeconds: req.DurationSeconds,
	}

	err := h.recordProgressUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		logger.Error().Err(err).Msg("failed to save watch progress")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse("INTERNAL_ERROR", "Failed to save watch progress", ""))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(gin.H{"message": "Progress saved"}))
}

// GetProgress handles GET /api/playback/progress/:media_id
func (h *WatchProgressHandler) GetProgress(c *gin.Context) {
	mediaID := c.Param("media_id")
	if mediaID == "" {
		c.JSON(http.StatusBadRequest, server.ErrorResponse("INVALID_REQUEST", "Missing media_id parameter", ""))
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, server.ErrorResponse("UNAUTHORIZED", "User not authenticated", ""))
		return
	}

	// Optional episode_id query param
	var episodeID *int64
	if episodeIDStr := c.Query("episode_id"); episodeIDStr != "" {
		id, err := strconv.ParseInt(episodeIDStr, 10, 64)
		if err == nil {
			episodeID = &id
		}
	}

	progress, err := h.getProgressUseCase.Execute(c.Request.Context(), userID.(string), mediaID, episodeID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get watch progress")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse("INTERNAL_ERROR", "Failed to get watch progress", ""))
		return
	}

	if progress == nil {
		c.JSON(http.StatusNotFound, server.ErrorResponse("NOT_FOUND", "No watch progress found", ""))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(toWatchProgressResponse(progress)))
}

// GetContinueWatching handles GET /api/library/continue-watching
func (h *WatchProgressHandler) GetContinueWatching(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, server.ErrorResponse("UNAUTHORIZED", "User not authenticated", ""))
		return
	}

	// Get limit from query param (default 20)
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	items, err := h.getContinueWatchingUseCase.Execute(c.Request.Context(), userID.(string), limit)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get continue watching items")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse("INTERNAL_ERROR", "Failed to get continue watching items", ""))
		return
	}

	// Convert to response format
	respItems := make([]ContinueWatchingItemResponse, len(items))
	for i, item := range items {
		respItems[i] = toContinueWatchingResponse(item)
	}

	c.JSON(http.StatusOK, server.SuccessResponse(respItems))
}

// RemoveFromContinueWatching handles DELETE /api/playback/progress/:media_id
func (h *WatchProgressHandler) RemoveFromContinueWatching(c *gin.Context) {
	mediaID := c.Param("media_id")
	if mediaID == "" {
		c.JSON(http.StatusBadRequest, server.ErrorResponse("INVALID_REQUEST", "Missing media_id parameter", ""))
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, server.ErrorResponse("UNAUTHORIZED", "User not authenticated", ""))
		return
	}

	// Optional episode_id query param
	var episodeID *int64
	if episodeIDStr := c.Query("episode_id"); episodeIDStr != "" {
		id, err := strconv.ParseInt(episodeIDStr, 10, 64)
		if err == nil {
			episodeID = &id
		}
	}

	err := h.removeFromContinueWatchingUseCase.Execute(c.Request.Context(), userID.(string), mediaID, episodeID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to remove from continue watching")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse("INTERNAL_ERROR", "Failed to remove from continue watching", ""))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(gin.H{"message": "Removed from continue watching"}))
}
