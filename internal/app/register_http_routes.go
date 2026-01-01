package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/http"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/web"
)

func registerHTTPHandlers(useCases *UseCases, httpServer *server.HTTPServer, cfg *config.Config) {
	// Register static web files (HTML players)
	if err := httpServer.RegisterStaticFiles(web.StaticFiles); err != nil {
		logger.Error().Err(err).Msg("failed to register static files")
	}

	// Scan Library API
	scanLibraryHttpHandler := http.NewScanLibraryHTTPHandler(useCases.EnqueueJobUseCase)
	httpServer.RegisterRoutes(scanLibraryHttpHandler)

	// DASH Streaming
	dashHandler := http.NewDASHHandler(useCases.GetMediaUseCase, cfg)
	dashHandler.RegisterRoutes(httpServer.Router())

	// HLS Streaming
	hlsHandler := http.NewHLSHandler(useCases.GetMediaUseCase, cfg)
	hlsHandler.RegisterRoutes(httpServer.Router())
}
