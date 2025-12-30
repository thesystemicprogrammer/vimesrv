package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

type HTTPServer struct {
	router     *gin.Engine
	httpServer *http.Server
	host       string
	port       int
}

type HTTPServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func NewHTTPServer(config HTTPServerConfig) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Add middleware
	router.Use(RecoveryMiddleware())
	router.Use(LoggingMiddleware())
	router.Use(CORSMiddleware())
	router.Use(ErrorHandlerMiddleware())

	// Set default timeouts if not provided
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 120 * time.Second
	}

	return &HTTPServer{
		router: router,
		host:   config.Host,
		port:   config.Port,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
			Handler:      router,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			IdleTimeout:  config.IdleTimeout,
		},
	}
}

func (s *HTTPServer) Start() error {
	s.setupHealthEndpoint()

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error().Err(err).Msg("error while server start")
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	logger.Info().Msg("shutting down HTTP server")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	logger.Info().Msg("HTTP server stopped")
	return nil
}

func (s *HTTPServer) Addr() string {
	return s.httpServer.Addr
}

func (s *HTTPServer) setupHealthEndpoint() {
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, SuccessResponse(map[string]interface{}{
			"status": "healthy",
		}))
	})

	logger.Debug().Msg("Health endpoint setup conducted")
}

// RegisterRoutes registers all HTTP routes with their handlers.
// This should be called after the server is created but before it starts.
func (s *HTTPServer) RegisterRoutes(scanLibraryHandler interface{ Handle(*gin.Context) }) {
	api := s.router.Group("/api/v1")
	{
		api.POST("/scanlib", scanLibraryHandler.Handle)
	}

	logger.Debug().Msg("API routes registered")
}
