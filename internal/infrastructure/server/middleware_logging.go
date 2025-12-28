package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		var logEvent *zerolog.Event
		if status >= 500 {
			logEvent = logger.Error()
		} else if status >= 400 {
			logEvent = logger.Warn()
		} else {
			logEvent = logger.Debug()
		}

		logEvent.
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", status).
			Dur("duration", duration).
			Str("client_ip", c.ClientIP())

		if len(c.Errors) > 0 {
			logEvent.Str("errors", c.Errors.String())
		}

		logEvent.Msg("HTTP request")
	}
}
