package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/favorite"
)

// FavoriteHandler handles favorite endpoints
type FavoriteHandler struct {
	toggleFavoriteUseCase   *favorite.ToggleFavoriteUseCase
	getUserFavoritesUseCase *favorite.GetUserFavoritesUseCase
}

// NewFavoriteHandler creates a new favorite handler
func NewFavoriteHandler(
	toggleFavoriteUseCase *favorite.ToggleFavoriteUseCase,
	getUserFavoritesUseCase *favorite.GetUserFavoritesUseCase,
) *FavoriteHandler {
	return &FavoriteHandler{
		toggleFavoriteUseCase:   toggleFavoriteUseCase,
		getUserFavoritesUseCase: getUserFavoritesUseCase,
	}
}

// RegisterRoutes registers favorite routes
func (h *FavoriteHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/favorites", h.ToggleFavorite)
	router.GET("/favorites", h.GetFavorites)
}

// ToggleFavoriteRequest represents the request to toggle a favorite
type ToggleFavoriteRequest struct {
	MediaType  string `json:"media_type" binding:"required"`
	MetadataID int64  `json:"metadata_id" binding:"required"`
}

// FavoriteItemResponse represents a favorite item in API responses
type FavoriteItemResponse struct {
	ID           string   `json:"id"`
	MediaType    string   `json:"media_type"`
	MetadataID   int64    `json:"metadata_id"`
	Title        string   `json:"title"`
	PosterPath   *string  `json:"poster_path,omitempty"`
	BackdropPath *string  `json:"backdrop_path,omitempty"`
	Year         *int64   `json:"year,omitempty"`
	Rating       *float64 `json:"rating,omitempty"`
	Genres       []string `json:"genres,omitempty"`
	AddedAt      string   `json:"added_at"`
}

// toFavoriteItemResponse converts domain favorite item to API response
func toFavoriteItemResponse(item domain.FavoriteItem) FavoriteItemResponse {
	resp := FavoriteItemResponse{
		ID:        item.ID,
		MediaType: item.MediaType,
		Title:     item.Title,
		AddedAt:   item.AddedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Set metadata ID based on type
	if item.MediaType == "movie" && item.MovieMetadataID.Valid {
		resp.MetadataID = item.MovieMetadataID.Int64
	} else if item.MediaType == "series" && item.SeriesMetadataID.Valid {
		resp.MetadataID = item.SeriesMetadataID.Int64
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
	if item.Rating.Valid {
		resp.Rating = &item.Rating.Float64
	}

	// Parse genres from JSON string (if needed)
	// For now, return empty array - genres parsing can be enhanced
	resp.Genres = []string{}

	return resp
}

// ToggleFavorite handles POST /api/favorites
func (h *FavoriteHandler) ToggleFavorite(c *gin.Context) {
	var req ToggleFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn().Err(err).Msg("invalid request body")
		c.JSON(http.StatusBadRequest, server.ErrorResponse("INVALID_REQUEST", "Invalid request body", ""))
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, server.ErrorResponse("UNAUTHORIZED", "User not authenticated", ""))
		return
	}

	input := favorite.ToggleFavoriteInput{
		UserID:     userID.(string),
		MediaType:  req.MediaType,
		MetadataID: req.MetadataID,
	}

	isFavorited, err := h.toggleFavoriteUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		logger.Error().Err(err).Msg("failed to toggle favorite")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse("INTERNAL_ERROR", "Failed to toggle favorite", ""))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(gin.H{"favorited": isFavorited}))
}

// GetFavorites handles GET /api/favorites
func (h *FavoriteHandler) GetFavorites(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, server.ErrorResponse("UNAUTHORIZED", "User not authenticated", ""))
		return
	}

	// Get limit from query param (default 50)
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	items, err := h.getUserFavoritesUseCase.Execute(c.Request.Context(), userID.(string), limit)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get user favorites")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse("INTERNAL_ERROR", "Failed to get favorites", ""))
		return
	}

	// Convert to response format
	respItems := make([]FavoriteItemResponse, len(items))
	for i, item := range items {
		respItems[i] = toFavoriteItemResponse(item)
	}

	c.JSON(http.StatusOK, server.SuccessResponse(respItems))
}
