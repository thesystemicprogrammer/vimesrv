package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	ws "github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/websocket"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins - in production, you may want to restrict this
		return true
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub        *ws.Hub
	authConfig *config.AuthConfig
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *ws.Hub, authConfig *config.AuthConfig) *WebSocketHandler {
	return &WebSocketHandler{
		hub:        hub,
		authConfig: authConfig,
	}
}

// RegisterRoutes registers WebSocket routes
func (h *WebSocketHandler) RegisterRoutes(router *gin.Engine) {
	// WebSocket endpoint with token-based authentication via query parameter
	router.GET("/api/v1/ws", h.HandleWebSocket)
	logger.Debug().Msg("WebSocket route registered")
}

// HandleWebSocket handles WebSocket upgrade requests
// Authentication is done via JWT token in query parameter: /api/v1/ws?token=<jwt>
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Validate JWT token from query parameter
	tokenString := c.Query("token")

	// If auth is enabled, require valid token
	if h.authConfig.Enabled {
		if tokenString == "" {
			logger.Debug().Msg("WebSocket connection rejected: missing token")
			c.JSON(http.StatusUnauthorized, server.ErrorResponse(
				"UNAUTHORIZED",
				"Missing authentication token",
				"Provide token as query parameter: /api/v1/ws?token=<jwt>",
			))
			return
		}

		// Validate the token (accept API tokens for WebSocket)
		claims, err := server.ValidateToken(tokenString, h.authConfig.JWTSecret, server.TokenTypeAPI)
		if err != nil {
			logger.Debug().Err(err).Msg("WebSocket connection rejected: invalid token")
			c.JSON(http.StatusUnauthorized, server.ErrorResponse(
				"UNAUTHORIZED",
				"Invalid or expired token",
				"",
			))
			return
		}

		// Upgrade to WebSocket connection
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to upgrade WebSocket connection")
			return
		}

		// Create client with user ID from claims
		userID := claims.UserID
		if userID == "" {
			userID = claims.Username // Fallback for config-based auth
		}

		client := ws.NewClient(h.hub, conn, userID)

		// Register client with hub
		h.hub.Register(client)

		logger.Info().
			Str("user_id", userID).
			Str("username", claims.Username).
			Msg("WebSocket client connected")

		// Start client read/write pumps in separate goroutines
		go client.WritePump()
		go client.ReadPump()
	} else {
		// Auth disabled - allow anonymous connections
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to upgrade WebSocket connection")
			return
		}

		client := ws.NewClient(h.hub, conn, "anonymous")
		h.hub.Register(client)

		logger.Info().Msg("WebSocket client connected (auth disabled)")

		go client.WritePump()
		go client.ReadPump()
	}
}
