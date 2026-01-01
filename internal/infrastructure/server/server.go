package server

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
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

// RegisterPWA registers the PWA single-page application with SPA fallback routing.
// All routes that don't match API, stream, or health paths will be
// served the PWA's index.html for client-side routing.
func (s *HTTPServer) RegisterPWA(pwaFS fs.FS) error {
	// Create a sub-filesystem for the PWA dist folder
	distFS, err := fs.Sub(pwaFS, "pwa/dist/pwa/browser")
	if err != nil {
		return fmt.Errorf("failed to create PWA filesystem: %w", err)
	}

	// Pre-load index.html content to avoid redirect issues with http.FS
	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		return fmt.Errorf("failed to read index.html: %w", err)
	}

	httpFS := http.FS(distFS)

	// Helper to serve index.html without redirect issues
	serveIndex := func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	}

	// Serve PWA at root
	s.router.GET("/", serveIndex)

	// Serve icons directory
	iconsFS, err := fs.Sub(distFS, "icons")
	if err == nil {
		s.router.StaticFS("/icons", http.FS(iconsFS))
	}

	// Serve manifest and service worker at root
	s.router.GET("/manifest.webmanifest", func(c *gin.Context) {
		c.FileFromFS("manifest.webmanifest", httpFS)
	})
	s.router.GET("/ngsw.json", func(c *gin.Context) {
		c.FileFromFS("ngsw.json", httpFS)
	})
	s.router.GET("/ngsw-worker.js", func(c *gin.Context) {
		c.Header("Service-Worker-Allowed", "/")
		c.FileFromFS("ngsw-worker.js", httpFS)
	})
	s.router.GET("/safety-worker.js", func(c *gin.Context) {
		c.FileFromFS("safety-worker.js", httpFS)
	})
	s.router.GET("/worker-basic.min.js", func(c *gin.Context) {
		c.FileFromFS("worker-basic.min.js", httpFS)
	})
	s.router.GET("/favicon.ico", func(c *gin.Context) {
		c.FileFromFS("favicon.ico", httpFS)
	})

	// SPA fallback: serve index.html for any unmatched routes
	// This must be registered AFTER all other routes
	s.router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Return JSON errors for API and stream routes
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, ErrorResponse("NOT_FOUND", "API endpoint not found", ""))
			return
		}
		if strings.HasPrefix(path, "/stream") {
			c.JSON(http.StatusNotFound, ErrorResponse("NOT_FOUND", "Stream resource not found", ""))
			return
		}

		// Check if this looks like a static file request (has extension)
		// If so, try to serve it from the PWA files
		if strings.Contains(path, ".") && !strings.HasPrefix(path, "/login") && !strings.HasPrefix(path, "/play") {
			// Try to serve as static file
			filePath := path[1:] // Remove leading slash
			if _, err := fs.Stat(distFS, filePath); err == nil {
				c.FileFromFS(filePath, httpFS)
				return
			}
		}

		// For all other paths, serve index.html (SPA routing)
		serveIndex(c)
	})

	logger.Info().Msg("PWA registered with SPA fallback routing")
	return nil
}

// Router returns the underlying gin.Engine for route registration
// This allows handlers to register their own routes
func (s *HTTPServer) Router() *gin.Engine {
	return s.router
}
