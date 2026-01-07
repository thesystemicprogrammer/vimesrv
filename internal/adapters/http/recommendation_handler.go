package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/recommendation"
)

// RecommendationHandler handles recommendation endpoints
type RecommendationHandler struct {
	buildModelUseCase          *recommendation.BuildRecommendationModelUseCase
	getPersonalizedRecsUseCase *recommendation.GetPersonalizedRecommendationsUseCase
	recommendationRepo         RecommendationQueryRepo
}

// RecommendationQueryRepo interface for querying recommendations
type RecommendationQueryRepo interface {
	GetMovieRecommendations(ctx context.Context, sourceID int64, limit int) ([]domain.MovieRecommendation, error)
	GetSeriesRecommendations(ctx context.Context, sourceID int64, limit int) ([]domain.SeriesRecommendation, error)
	GetModelMetadata(ctx context.Context, modelType string) (*domain.RecommendationModelMetadata, error)
}

// NewRecommendationHandler creates a new recommendation handler
func NewRecommendationHandler(
	buildModelUseCase *recommendation.BuildRecommendationModelUseCase,
	getPersonalizedRecsUseCase *recommendation.GetPersonalizedRecommendationsUseCase,
	recommendationRepo RecommendationQueryRepo,
) *RecommendationHandler {
	return &RecommendationHandler{
		buildModelUseCase:          buildModelUseCase,
		getPersonalizedRecsUseCase: getPersonalizedRecsUseCase,
		recommendationRepo:         recommendationRepo,
	}
}

// RegisterRoutes registers user-facing recommendation routes
func (h *RecommendationHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/recommendations", h.GetPersonalizedRecommendations)
	router.GET("/movies/:id/similar", h.GetSimilarMovies)
	router.GET("/series/:id/similar", h.GetSimilarSeries)

	logger.Debug().Msg("Recommendation routes registered")
}

// RegisterAdminRoutes registers admin-only recommendation routes
func (h *RecommendationHandler) RegisterAdminRoutes(router *gin.RouterGroup) {
	adminGroup := router.Group("/admin/recommendations")
	adminGroup.Use(h.requireAdmin())
	{
		adminGroup.POST("/rebuild", h.RebuildModels)
		adminGroup.GET("/status", h.GetModelStatus)
	}
	logger.Debug().Msg("Recommendation admin routes registered")
}

// requireAdmin middleware checks if the user has admin role
func (h *RecommendationHandler) requireAdmin() gin.HandlerFunc {
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

// RecommendationResponse represents a recommendation in API responses
type RecommendationResponse struct {
	ItemID       int64   `json:"item_id"`
	ItemType     string  `json:"item_type"`
	MediaID      string  `json:"media_id,omitempty"`
	Title        string  `json:"title"`
	Year         string  `json:"year,omitempty"`
	PosterPath   string  `json:"poster_path,omitempty"`
	BackdropPath string  `json:"backdrop_path,omitempty"`
	VoteAverage  float64 `json:"vote_average"`
	Score        float64 `json:"score"`
}

// SimilarItemResponse represents a similar item in API responses
type SimilarItemResponse struct {
	MetadataID      int64   `json:"metadata_id"`
	SimilarityScore float64 `json:"similarity_score"`
	Rank            int     `json:"rank"`
}

// ModelStatusResponse represents model metadata in API responses
type ModelStatusResponse struct {
	ModelType       string `json:"model_type"`
	TotalItems      int    `json:"total_items"`
	FeatureCount    int    `json:"feature_count"`
	LastBuiltAt     string `json:"last_built_at"`
	BuildDurationMs int    `json:"build_duration_ms"`
}

// RebuildModelsResponse represents the result of rebuilding models
type RebuildModelsResponse struct {
	MovieModelBuilt  bool `json:"movie_model_built"`
	SeriesModelBuilt bool `json:"series_model_built"`
	MovieItems       int  `json:"movie_items"`
	SeriesItems      int  `json:"series_items"`
	MovieDurationMs  int  `json:"movie_duration_ms"`
	SeriesDurationMs int  `json:"series_duration_ms"`
}

// GetPersonalizedRecommendations handles GET /api/recommendations
func (h *RecommendationHandler) GetPersonalizedRecommendations(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, server.ErrorResponse("UNAUTHORIZED", "User not authenticated", ""))
		return
	}

	// Parse query parameters
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	contentType := c.Query("type") // "movie", "series", or empty for all

	input := recommendation.GetPersonalizedRecommendationsInput{
		UserID: userID.(string),
		Limit:  limit,
		Type:   contentType,
	}

	recs, err := h.getPersonalizedRecsUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get personalized recommendations")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse("INTERNAL_ERROR", "Failed to get recommendations", ""))
		return
	}

	// Convert to response format
	response := make([]RecommendationResponse, len(recs))
	for i, rec := range recs {
		response[i] = RecommendationResponse{
			ItemID:       rec.ItemID,
			ItemType:     rec.ItemType,
			MediaID:      rec.MediaID,
			Title:        rec.Title,
			Year:         rec.Year,
			PosterPath:   rec.PosterPath,
			BackdropPath: rec.BackdropPath,
			VoteAverage:  rec.VoteAverage,
			Score:        rec.Score,
		}
	}

	c.JSON(http.StatusOK, server.SuccessResponse(response))
}

// GetSimilarMovies handles GET /api/movies/:id/similar
func (h *RecommendationHandler) GetSimilarMovies(c *gin.Context) {
	// Parse movie metadata ID
	idStr := c.Param("id")
	movieID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse("INVALID_ID", "Invalid movie ID", ""))
		return
	}

	// Parse limit
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	recs, err := h.recommendationRepo.GetMovieRecommendations(c.Request.Context(), movieID, limit)
	if err != nil {
		logger.Error().Err(err).Int64("movie_id", movieID).Msg("failed to get similar movies")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse("INTERNAL_ERROR", "Failed to get similar movies", ""))
		return
	}

	// Convert to response format
	response := make([]SimilarItemResponse, len(recs))
	for i, rec := range recs {
		response[i] = SimilarItemResponse{
			MetadataID:      rec.RecommendedMovieMetadataID,
			SimilarityScore: rec.SimilarityScore,
			Rank:            rec.RankOrder,
		}
	}

	c.JSON(http.StatusOK, server.SuccessResponse(response))
}

// GetSimilarSeries handles GET /api/series/:id/similar
func (h *RecommendationHandler) GetSimilarSeries(c *gin.Context) {
	// Parse series metadata ID
	idStr := c.Param("id")
	seriesID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse("INVALID_ID", "Invalid series ID", ""))
		return
	}

	// Parse limit
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	recs, err := h.recommendationRepo.GetSeriesRecommendations(c.Request.Context(), seriesID, limit)
	if err != nil {
		logger.Error().Err(err).Int64("series_id", seriesID).Msg("failed to get similar series")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse("INTERNAL_ERROR", "Failed to get similar series", ""))
		return
	}

	// Convert to response format
	response := make([]SimilarItemResponse, len(recs))
	for i, rec := range recs {
		response[i] = SimilarItemResponse{
			MetadataID:      rec.RecommendedSeriesMetadataID,
			SimilarityScore: rec.SimilarityScore,
			Rank:            rec.RankOrder,
		}
	}

	c.JSON(http.StatusOK, server.SuccessResponse(response))
}

// RebuildModels handles POST /api/admin/recommendations/rebuild
func (h *RecommendationHandler) RebuildModels(c *gin.Context) {
	// Parse optional model type
	modelType := c.Query("type") // "movie", "series", or empty for all

	input := recommendation.BuildRecommendationModelInput{
		ModelType: modelType,
	}

	output, err := h.buildModelUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		logger.Error().Err(err).Msg("failed to rebuild recommendation models")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse("INTERNAL_ERROR", "Failed to rebuild models", ""))
		return
	}

	response := RebuildModelsResponse{
		MovieModelBuilt:  output.MovieModelBuilt,
		SeriesModelBuilt: output.SeriesModelBuilt,
		MovieItems:       output.MovieItems,
		SeriesItems:      output.SeriesItems,
		MovieDurationMs:  output.MovieDurationMs,
		SeriesDurationMs: output.SeriesDurationMs,
	}

	c.JSON(http.StatusOK, server.SuccessResponse(response))
}

// GetModelStatus handles GET /api/admin/recommendations/status
func (h *RecommendationHandler) GetModelStatus(c *gin.Context) {
	var status []ModelStatusResponse

	// Get movie model status
	movieMeta, err := h.recommendationRepo.GetModelMetadata(c.Request.Context(), "movie")
	if err != nil {
		logger.Warn().Err(err).Msg("failed to get movie model metadata")
	} else if movieMeta != nil {
		status = append(status, ModelStatusResponse{
			ModelType:       movieMeta.ModelType,
			TotalItems:      movieMeta.TotalItems,
			FeatureCount:    movieMeta.FeatureCount,
			LastBuiltAt:     movieMeta.LastBuiltAt.Format("2006-01-02T15:04:05Z"),
			BuildDurationMs: movieMeta.BuildDurationMs,
		})
	}

	// Get series model status
	seriesMeta, err := h.recommendationRepo.GetModelMetadata(c.Request.Context(), "series")
	if err != nil {
		logger.Warn().Err(err).Msg("failed to get series model metadata")
	} else if seriesMeta != nil {
		status = append(status, ModelStatusResponse{
			ModelType:       seriesMeta.ModelType,
			TotalItems:      seriesMeta.TotalItems,
			FeatureCount:    seriesMeta.FeatureCount,
			LastBuiltAt:     seriesMeta.LastBuiltAt.Format("2006-01-02T15:04:05Z"),
			BuildDurationMs: seriesMeta.BuildDurationMs,
		})
	}

	c.JSON(http.StatusOK, server.SuccessResponse(status))
}
