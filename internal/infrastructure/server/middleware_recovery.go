package server

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error().
					Interface("error", err).
					Str("path", c.Request.URL.Path).
					Str("stack", string(debug.Stack())).
					Msg("panic recovered")

				c.JSON(http.StatusInternalServerError, ErrorResponse(
					"INTERNAL_ERROR",
					"Internal server error",
					"",
				))
				c.Abort()
			}
		}()
		c.Next()
	}
}
