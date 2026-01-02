package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/library"
)

// LibraryHandler handles library browsing API endpoints
type LibraryHandler struct {
	listMoviesUC       *library.ListMoviesUseCase
	getMovieUC         *library.GetMovieUseCase
	getMovieCreditsUC  *library.GetMovieCreditsUseCase
	listSeriesUC       *library.ListSeriesUseCase
	getSeriesUC        *library.GetSeriesUseCase
	getSeriesCreditsUC *library.GetSeriesCreditsUseCase
	listRecentUC       *library.ListRecentUseCase
	listUnmatchedUC    *library.ListUnmatchedUseCase
}

// NewLibraryHandler creates a new library handler
func NewLibraryHandler(
	listMoviesUC *library.ListMoviesUseCase,
	getMovieUC *library.GetMovieUseCase,
	getMovieCreditsUC *library.GetMovieCreditsUseCase,
	listSeriesUC *library.ListSeriesUseCase,
	getSeriesUC *library.GetSeriesUseCase,
	getSeriesCreditsUC *library.GetSeriesCreditsUseCase,
	listRecentUC *library.ListRecentUseCase,
	listUnmatchedUC *library.ListUnmatchedUseCase,
) *LibraryHandler {
	return &LibraryHandler{
		listMoviesUC:       listMoviesUC,
		getMovieUC:         getMovieUC,
		getMovieCreditsUC:  getMovieCreditsUC,
		listSeriesUC:       listSeriesUC,
		getSeriesUC:        getSeriesUC,
		getSeriesCreditsUC: getSeriesCreditsUC,
		listRecentUC:       listRecentUC,
		listUnmatchedUC:    listUnmatchedUC,
	}
}

// RegisterRoutes registers library routes on a protected router group
func (h *LibraryHandler) RegisterRoutes(router *gin.RouterGroup) {
	libraryGroup := router.Group("/library")
	{
		libraryGroup.GET("/movies", h.ListMovies)
		libraryGroup.GET("/movies/:id", h.GetMovie)
		libraryGroup.GET("/movies/:id/credits", h.GetMovieCredits)
		libraryGroup.GET("/series", h.ListSeries)
		libraryGroup.GET("/series/:id", h.GetSeries)
		libraryGroup.GET("/series/:id/credits", h.GetSeriesCredits)
		libraryGroup.GET("/recent", h.ListRecent)
		libraryGroup.GET("/unmatched", h.ListUnmatched)
	}
	logger.Debug().Msg("Library routes registered")
}

// ListMovies handles GET /api/v1/library/movies
func (h *LibraryHandler) ListMovies(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	language := c.DefaultQuery("lang", "en")

	logger.Debug().
		Int("page", page).
		Int("per_page", perPage).
		Str("language", language).
		Msg("listing movies")

	result, err := h.listMoviesUC.Execute(c.Request.Context(), library.ListMoviesInput{
		Language: language,
		Page:     page,
		PerPage:  perPage,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to list movies")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"LIST_FAILED",
			"Failed to list movies",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// GetMovie handles GET /api/v1/library/movies/:id
func (h *LibraryHandler) GetMovie(c *gin.Context) {
	mediaID := c.Param("id")
	language := c.DefaultQuery("lang", "en")

	logger.Debug().
		Str("media_id", mediaID).
		Str("language", language).
		Msg("getting movie details")

	result, err := h.getMovieUC.Execute(c.Request.Context(), library.GetMovieInput{
		MediaID:  mediaID,
		Language: language,
	})
	if err != nil {
		if err == shared.ErrNotFound {
			c.JSON(http.StatusNotFound, server.ErrorResponse(
				"MOVIE_NOT_FOUND",
				"Movie not found",
				"The requested movie was not found",
			))
			return
		}
		logger.Error().Err(err).Str("media_id", mediaID).Msg("failed to get movie")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"GET_FAILED",
			"Failed to get movie",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// ListSeries handles GET /api/v1/library/series
func (h *LibraryHandler) ListSeries(c *gin.Context) {
	language := c.DefaultQuery("lang", "en")
	includeEmpty := c.DefaultQuery("include_empty", "false") == "true"

	logger.Debug().
		Str("language", language).
		Bool("include_empty", includeEmpty).
		Msg("listing series")

	result, err := h.listSeriesUC.Execute(c.Request.Context(), library.ListSeriesInput{
		Language:     language,
		IncludeEmpty: includeEmpty,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to list series")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"LIST_FAILED",
			"Failed to list series",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// GetSeries handles GET /api/v1/library/series/:id
func (h *LibraryHandler) GetSeries(c *gin.Context) {
	seriesID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_ID",
			"Invalid series ID",
			"Series ID must be a valid integer",
		))
		return
	}
	language := c.DefaultQuery("lang", "en")

	logger.Debug().
		Int64("series_id", seriesID).
		Str("language", language).
		Msg("getting series details")

	result, err := h.getSeriesUC.Execute(c.Request.Context(), library.GetSeriesInput{
		SeriesID: seriesID,
		Language: language,
	})
	if err != nil {
		if err == shared.ErrNotFound {
			c.JSON(http.StatusNotFound, server.ErrorResponse(
				"SERIES_NOT_FOUND",
				"Series not found",
				"The requested series was not found",
			))
			return
		}
		logger.Error().Err(err).Int64("series_id", seriesID).Msg("failed to get series")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"GET_FAILED",
			"Failed to get series",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// ListRecent handles GET /api/v1/library/recent
func (h *LibraryHandler) ListRecent(c *gin.Context) {
	language := c.DefaultQuery("lang", "en")

	logger.Debug().
		Str("language", language).
		Msg("listing recently added media")

	result, err := h.listRecentUC.Execute(c.Request.Context(), library.ListRecentInput{
		Language: language,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to list recently added")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"LIST_FAILED",
			"Failed to list recently added media",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// ListUnmatched handles GET /api/v1/library/unmatched
func (h *LibraryHandler) ListUnmatched(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	logger.Debug().
		Int("page", page).
		Int("per_page", perPage).
		Msg("listing unmatched media")

	result, err := h.listUnmatchedUC.Execute(c.Request.Context(), library.ListUnmatchedInput{
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to list unmatched media")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"LIST_FAILED",
			"Failed to list unmatched media",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// GetMovieCredits handles GET /api/v1/library/movies/:id/credits
func (h *LibraryHandler) GetMovieCredits(c *gin.Context) {
	movieMetadataID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_ID",
			"Invalid movie metadata ID",
			"Movie metadata ID must be a valid integer",
		))
		return
	}

	logger.Debug().
		Int64("movie_metadata_id", movieMetadataID).
		Msg("getting movie credits")

	result, err := h.getMovieCreditsUC.Execute(c.Request.Context(), library.GetMovieCreditsInput{
		MovieMetadataID: movieMetadataID,
	})
	if err != nil {
		if err == shared.ErrNotFound {
			c.JSON(http.StatusNotFound, server.ErrorResponse(
				"MOVIE_NOT_FOUND",
				"Movie not found",
				"The requested movie was not found",
			))
			return
		}
		logger.Error().Err(err).Int64("movie_metadata_id", movieMetadataID).Msg("failed to get movie credits")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"GET_FAILED",
			"Failed to get movie credits",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}

// GetSeriesCredits handles GET /api/v1/library/series/:id/credits
func (h *LibraryHandler) GetSeriesCredits(c *gin.Context) {
	seriesMetadataID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_ID",
			"Invalid series metadata ID",
			"Series metadata ID must be a valid integer",
		))
		return
	}

	logger.Debug().
		Int64("series_metadata_id", seriesMetadataID).
		Msg("getting series credits")

	result, err := h.getSeriesCreditsUC.Execute(c.Request.Context(), library.GetSeriesCreditsInput{
		SeriesMetadataID: seriesMetadataID,
	})
	if err != nil {
		if err == shared.ErrNotFound {
			c.JSON(http.StatusNotFound, server.ErrorResponse(
				"SERIES_NOT_FOUND",
				"Series not found",
				"The requested series was not found",
			))
			return
		}
		logger.Error().Err(err).Int64("series_metadata_id", seriesMetadataID).Msg("failed to get series credits")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"GET_FAILED",
			"Failed to get series credits",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(result))
}
