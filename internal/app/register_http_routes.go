package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/http"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/web"
)

func registerHTTPHandlers(useCases *UseCases, httpServer *server.HTTPServer, cfg *config.Config) {
	router := httpServer.Router()

	// Register PWA (Angular frontend)
	if err := httpServer.RegisterPWA(web.PWAFiles); err != nil {
		logger.Error().Err(err).Msg("failed to register PWA")
	}

	// === Public Routes (no auth required) ===

	// Auth handler - login is public
	authHandler := http.NewAuthHandler(&cfg.Auth)
	authHandler.RegisterRoutes(router)

	// === Protected API Routes ===

	// Create API group with auth middleware
	apiGroup := router.Group("/api/v1")
	apiGroup.Use(server.AuthMiddleware(&cfg.Auth))
	{
		// Protected auth routes (me, stream-token)
		authHandler.RegisterProtectedRoutes(apiGroup)

		// Scan Library API
		scanLibraryHttpHandler := http.NewScanLibraryHTTPHandler(useCases.EnqueueJobUseCase)
		scanLibraryHttpHandler.RegisterRoutes(apiGroup)

		// Media API
		mediaHandler := http.NewMediaHandler(useCases.ListMediaUseCase, useCases.GetMediaUseCase, cfg)
		mediaHandler.RegisterRoutes(apiGroup)
	}

	// === Protected Streaming Routes ===

	// DASH Streaming with stream auth middleware
	dashHandler := http.NewDASHHandler(useCases.GetMediaUseCase, cfg)

	// Stream routes require stream token auth
	streamGroup := router.Group("/stream")
	streamGroup.Use(server.StreamAuthMiddleware(&cfg.Auth))
	{
		// Manifest endpoint
		streamGroup.GET("/dash/:id/manifest.mpd", dashHandler.ServeManifest)

		// Content endpoint (video/audio segments, subtitles)
		streamGroup.Match([]string{"GET", "HEAD"}, "/dash/content/:id/*path", dashHandler.ServeContent)
	}

	logger.Info().
		Bool("auth_enabled", cfg.Auth.Enabled).
		Msg("HTTP routes registered")
}
