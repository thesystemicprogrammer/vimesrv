package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

func initializeHTTPServer(config config.ServerConfig) *server.HTTPServer {
	logger.Debug().Msg("creating HTTP server")
	return server.NewHTTPServer(server.HTTPServerConfig{
		Host: config.Host,
		Port: config.Port,
	})
}
