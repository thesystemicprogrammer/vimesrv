package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle errors if any occurred
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			logger.Error().
				Err(err).
				Str("path", c.Request.URL.Path).
				Msg("request error")

			// Map shared errors to HTTP status codes
			statusCode, code, message := mapError(err)

			c.JSON(statusCode, ErrorResponse(code, message, err.Error()))
		}
	}
}

func mapError(err error) (statusCode int, code string, message string) {
	switch err {
	case shared.ErrNotFound, shared.ErrMediaNotFound, shared.ErrTranscodeJobNotFound:
		return http.StatusNotFound, "NOT_FOUND", "Resource not found"
	case shared.ErrAlreadyExists:
		return http.StatusConflict, "ALREADY_EXISTS", "Resource already exists"
	case shared.ErrInvalidInput, shared.ErrInvalidMediaPath, shared.ErrUnsupportedFormat:
		return http.StatusBadRequest, "INVALID_INPUT", "Invalid input"
	case shared.ErrInvalidState, shared.ErrTranscodingInProgress:
		return http.StatusConflict, "INVALID_STATE", "Invalid state"
	case shared.ErrTranscodingFailed:
		return http.StatusInternalServerError, "TRANSCODING_FAILED", "Transcoding failed"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error"
	}
}
