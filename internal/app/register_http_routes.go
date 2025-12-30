package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/http"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
)

func registerHTTPHandlers(useCases *UseCases, httpServer *server.HTTPServer) {
	// Scan Library
	scanLibraryHttpHandler := http.NewScanLibraryHTTPHandler(useCases.EnqueueJobUseCase)
	httpServer.RegisterRoutes(scanLibraryHttpHandler)
}
