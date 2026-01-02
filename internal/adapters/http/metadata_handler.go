package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	usecasejob "github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/metadata"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// MetadataHandler handles metadata enrichment API endpoints
type MetadataHandler struct {
	getCandidatesUC   *metadata.GetCandidatesUseCase
	linkMetadataUC    *metadata.LinkMetadataUseCase
	searchMetadataUC  *metadata.SearchMetadataUseCase
	linkFromSearchUC  *metadata.LinkFromSearchUseCase
	skipEnrichmentUC  *metadata.SkipEnrichmentUseCase
	resetEnrichmentUC *metadata.ResetEnrichmentUseCase
	enqueueJobUC      *usecasejob.EnqueueJobUseCase
	jobRepository     ports.JobRepository
}

// NewMetadataHandler creates a new metadata handler
func NewMetadataHandler(
	getCandidatesUC *metadata.GetCandidatesUseCase,
	linkMetadataUC *metadata.LinkMetadataUseCase,
	searchMetadataUC *metadata.SearchMetadataUseCase,
	linkFromSearchUC *metadata.LinkFromSearchUseCase,
	skipEnrichmentUC *metadata.SkipEnrichmentUseCase,
	resetEnrichmentUC *metadata.ResetEnrichmentUseCase,
	enqueueJobUC *usecasejob.EnqueueJobUseCase,
	jobRepository ports.JobRepository,
) *MetadataHandler {
	return &MetadataHandler{
		getCandidatesUC:   getCandidatesUC,
		linkMetadataUC:    linkMetadataUC,
		searchMetadataUC:  searchMetadataUC,
		linkFromSearchUC:  linkFromSearchUC,
		skipEnrichmentUC:  skipEnrichmentUC,
		resetEnrichmentUC: resetEnrichmentUC,
		enqueueJobUC:      enqueueJobUC,
		jobRepository:     jobRepository,
	}
}

// RegisterRoutes registers metadata routes on a protected router group
func (h *MetadataHandler) RegisterRoutes(router *gin.RouterGroup) {
	// GET /api/v1/media/:id/candidates - List metadata candidates
	router.GET("/media/:id/candidates", h.GetCandidates)

	// POST /api/v1/media/:id/link - Link to a candidate
	router.POST("/media/:id/link", h.LinkMetadata)

	// POST /api/v1/media/:id/search - Manual TMDB search
	router.POST("/media/:id/search", h.SearchMetadata)

	// POST /api/v1/media/:id/link-search - Link directly from search result
	router.POST("/media/:id/link-search", h.LinkFromSearch)

	// POST /api/v1/media/:id/skip - Skip enrichment
	router.POST("/media/:id/skip", h.SkipEnrichment)

	// POST /api/v1/media/:id/reset - Reset enrichment status
	router.POST("/media/:id/reset", h.ResetEnrichment)

	// POST /api/v1/translations/fetch - Fetch translations for a language
	router.POST("/translations/fetch", h.FetchTranslations)

	logger.Debug().Msg("Metadata routes registered")
}

// GetCandidates handles GET /api/v1/media/:id/candidates
func (h *MetadataHandler) GetCandidates(c *gin.Context) {
	id := c.Param("id")
	pendingOnly := c.DefaultQuery("pending_only", "false") == "true"

	logger.Debug().
		Str("media_id", id).
		Bool("pending_only", pendingOnly).
		Msg("getting metadata candidates")

	if h.getCandidatesUC == nil {
		c.JSON(http.StatusServiceUnavailable, server.ErrorResponse(
			"TMDB_DISABLED",
			"TMDB integration is not enabled",
			"",
		))
		return
	}

	result, err := h.getCandidatesUC.Execute(c.Request.Context(), metadata.GetCandidatesInput{
		MediaID:     id,
		PendingOnly: pendingOnly,
	})
	if err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to get candidates")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"GET_CANDIDATES_FAILED",
			"Failed to get metadata candidates",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// LinkMetadataRequest is the request body for linking metadata
type LinkMetadataRequest struct {
	CandidateID int64 `json:"candidate_id" binding:"required"`
}

// LinkMetadata handles POST /api/v1/media/:id/link
func (h *MetadataHandler) LinkMetadata(c *gin.Context) {
	id := c.Param("id")

	var req LinkMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	logger.Debug().
		Str("media_id", id).
		Int64("candidate_id", req.CandidateID).
		Msg("linking metadata")

	if h.linkMetadataUC == nil {
		c.JSON(http.StatusServiceUnavailable, server.ErrorResponse(
			"TMDB_DISABLED",
			"TMDB integration is not enabled",
			"",
		))
		return
	}

	result, err := h.linkMetadataUC.Execute(c.Request.Context(), metadata.LinkMetadataInput{
		MediaID:     id,
		CandidateID: req.CandidateID,
	})
	if err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to link metadata")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"LINK_FAILED",
			"Failed to link metadata",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// SearchMetadataRequest is the request body for searching metadata
type SearchMetadataRequest struct {
	Query      string `json:"query" binding:"required"`
	Year       int    `json:"year"`
	MediaType  string `json:"media_type"` // "movie", "tv", or empty for both
	MaxResults int    `json:"max_results"`
}

// SearchMetadata handles POST /api/v1/media/:id/search
func (h *MetadataHandler) SearchMetadata(c *gin.Context) {
	id := c.Param("id")

	var req SearchMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	logger.Debug().
		Str("media_id", id).
		Str("query", req.Query).
		Str("media_type", req.MediaType).
		Int("year", req.Year).
		Msg("searching metadata")

	if h.searchMetadataUC == nil {
		c.JSON(http.StatusServiceUnavailable, server.ErrorResponse(
			"TMDB_DISABLED",
			"TMDB integration is not enabled",
			"",
		))
		return
	}

	result, err := h.searchMetadataUC.Execute(c.Request.Context(), metadata.SearchMetadataInput{
		MediaID:    id,
		Query:      req.Query,
		Year:       req.Year,
		MediaType:  req.MediaType,
		MaxResults: req.MaxResults,
	})
	if err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to search metadata")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"SEARCH_FAILED",
			"Failed to search metadata",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// LinkFromSearchRequest is the request body for linking from a search result
type LinkFromSearchRequest struct {
	TMDBID        int    `json:"tmdb_id" binding:"required"`
	MediaType     string `json:"media_type" binding:"required"` // "movie" or "tv"
	SeasonNumber  int    `json:"season_number"`
	EpisodeNumber int    `json:"episode_number"`
}

// LinkFromSearch handles POST /api/v1/media/:id/link-search
func (h *MetadataHandler) LinkFromSearch(c *gin.Context) {
	id := c.Param("id")

	var req LinkFromSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Validate media type
	if req.MediaType != "movie" && req.MediaType != "tv" {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_MEDIA_TYPE",
			"media_type must be 'movie' or 'tv'",
			"",
		))
		return
	}

	logger.Debug().
		Str("media_id", id).
		Int("tmdb_id", req.TMDBID).
		Str("media_type", req.MediaType).
		Int("season", req.SeasonNumber).
		Int("episode", req.EpisodeNumber).
		Msg("linking from search result")

	if h.linkFromSearchUC == nil {
		c.JSON(http.StatusServiceUnavailable, server.ErrorResponse(
			"TMDB_DISABLED",
			"TMDB integration is not enabled",
			"",
		))
		return
	}

	result, err := h.linkFromSearchUC.Execute(c.Request.Context(), metadata.LinkFromSearchInput{
		MediaID:       id,
		TMDBID:        req.TMDBID,
		MediaType:     req.MediaType,
		SeasonNumber:  req.SeasonNumber,
		EpisodeNumber: req.EpisodeNumber,
	})
	if err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to link from search")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"LINK_FAILED",
			"Failed to link metadata",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// SkipEnrichment handles POST /api/v1/media/:id/skip
func (h *MetadataHandler) SkipEnrichment(c *gin.Context) {
	id := c.Param("id")

	logger.Debug().
		Str("media_id", id).
		Msg("skipping enrichment")

	if h.skipEnrichmentUC == nil {
		c.JSON(http.StatusServiceUnavailable, server.ErrorResponse(
			"TMDB_DISABLED",
			"TMDB integration is not enabled",
			"",
		))
		return
	}

	result, err := h.skipEnrichmentUC.Execute(c.Request.Context(), metadata.SkipEnrichmentInput{
		MediaID: id,
	})
	if err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to skip enrichment")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"SKIP_FAILED",
			"Failed to skip enrichment",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// ResetEnrichment handles POST /api/v1/media/:id/reset
func (h *MetadataHandler) ResetEnrichment(c *gin.Context) {
	id := c.Param("id")

	logger.Debug().
		Str("media_id", id).
		Msg("resetting enrichment")

	if h.resetEnrichmentUC == nil {
		c.JSON(http.StatusServiceUnavailable, server.ErrorResponse(
			"TMDB_DISABLED",
			"TMDB integration is not enabled",
			"",
		))
		return
	}

	result, err := h.resetEnrichmentUC.Execute(c.Request.Context(), metadata.ResetEnrichmentInput{
		MediaID: id,
	})
	if err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to reset enrichment")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"RESET_FAILED",
			"Failed to reset enrichment",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// FetchTranslationsRequest is the request body for fetching translations
type FetchTranslationsRequest struct {
	Language string `json:"language" binding:"required"`
}

// FetchTranslations handles POST /api/v1/translations/fetch
// It enqueues a fetch_translations job for the specified language
func (h *MetadataHandler) FetchTranslations(c *gin.Context) {
	var req FetchTranslationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	logger.Debug().
		Str("language", req.Language).
		Msg("fetching translations")

	// Check if TMDB is enabled (enqueueJobUC will be nil if not)
	if h.enqueueJobUC == nil {
		c.JSON(http.StatusServiceUnavailable, server.ErrorResponse(
			"TMDB_DISABLED",
			"TMDB integration is not enabled",
			"",
		))
		return
	}

	// Check if there's already a pending job for this language
	exists, err := h.jobRepository.ExistsPendingJobByType(c.Request.Context(), shared.JobTypeFetchTranslations, req.Language)
	if err != nil {
		logger.Error().Err(err).Str("language", req.Language).Msg("failed to check for pending translation job")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"CHECK_FAILED",
			"Failed to check for pending jobs",
			err.Error(),
		))
		return
	}

	if exists {
		logger.Debug().Str("language", req.Language).Msg("translation fetch job already pending")
		c.JSON(http.StatusAccepted, server.SuccessResponse(map[string]interface{}{
			"message": "Translation fetch job already in progress",
			"queued":  false,
		}))
		return
	}

	// Enqueue the translation fetch job
	jobInput := usecasejob.EnqueueJobInput{
		Type: shared.JobTypeFetchTranslations,
		Payload: map[string]string{
			"language": req.Language,
		},
		Priority: shared.JobPriorityFetchTranslations,
	}

	jobID, err := h.enqueueJobUC.Execute(c.Request.Context(), jobInput)
	if err != nil {
		logger.Error().Err(err).Str("language", req.Language).Msg("failed to enqueue translation fetch job")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"ENQUEUE_FAILED",
			"Failed to enqueue translation fetch job",
			err.Error(),
		))
		return
	}

	logger.Info().
		Int64("job_id", jobID).
		Str("language", req.Language).
		Msg("translation fetch job enqueued")

	c.JSON(http.StatusAccepted, server.SuccessResponse(map[string]interface{}{
		"message": "Translation fetch job queued",
		"job_id":  jobID,
		"queued":  true,
	}))
}
