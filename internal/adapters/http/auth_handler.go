package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	config *config.AuthConfig
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(cfg *config.AuthConfig) *AuthHandler {
	return &AuthHandler{
		config: cfg,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"` // seconds until expiry
}

// StreamTokenResponse represents the stream token response
type StreamTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"` // seconds until expiry
}

// RegisterRoutes registers auth routes
func (h *AuthHandler) RegisterRoutes(router *gin.Engine) {
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/login", h.Login)
	}
	logger.Debug().Msg("Auth routes registered")
}

// RegisterProtectedRoutes registers protected auth routes (require authentication)
func (h *AuthHandler) RegisterProtectedRoutes(router *gin.RouterGroup) {
	router.GET("/auth/me", h.Me)
	router.POST("/auth/stream-token", h.GenerateStreamToken)
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	// If auth is disabled, return an error
	if !h.config.Enabled {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"AUTH_DISABLED",
			"Authentication is not enabled",
			"",
		))
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Check username
	if req.Username != h.config.Username {
		logger.Debug().
			Str("username", req.Username).
			Msg("login failed: invalid username")
		c.JSON(http.StatusUnauthorized, server.ErrorResponse(
			"INVALID_CREDENTIALS",
			"Invalid username or password",
			"",
		))
		return
	}

	// Check password against bcrypt hash
	if err := bcrypt.CompareHashAndPassword([]byte(h.config.PasswordHash), []byte(req.Password)); err != nil {
		logger.Debug().
			Str("username", req.Username).
			Msg("login failed: invalid password")
		c.JSON(http.StatusUnauthorized, server.ErrorResponse(
			"INVALID_CREDENTIALS",
			"Invalid username or password",
			"",
		))
		return
	}

	// Generate JWT token
	expiryDuration := time.Duration(h.config.TokenExpiryHours) * time.Hour
	token, err := server.GenerateToken(req.Username, server.TokenTypeAPI, h.config.JWTSecret, expiryDuration)
	if err != nil {
		logger.Error().Err(err).Msg("failed to generate token")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"TOKEN_ERROR",
			"Failed to generate token",
			"",
		))
		return
	}

	logger.Info().
		Str("username", req.Username).
		Msg("user logged in successfully")

	c.JSON(http.StatusOK, server.SuccessResponse(LoginResponse{
		Token:     token,
		ExpiresIn: int(expiryDuration.Seconds()),
	}))
}

// Me handles GET /api/v1/auth/me - returns current user info
func (h *AuthHandler) Me(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, server.ErrorResponse(
			"UNAUTHORIZED",
			"Not authenticated",
			"",
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(gin.H{
		"username": username,
	}))
}

// GenerateStreamToken handles POST /api/v1/auth/stream-token
// Generates a short-lived token for stream URLs
func (h *AuthHandler) GenerateStreamToken(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, server.ErrorResponse(
			"UNAUTHORIZED",
			"Not authenticated",
			"",
		))
		return
	}

	// Generate stream token with shorter expiry
	expiryDuration := time.Duration(h.config.StreamTokenMins) * time.Minute
	token, err := server.GenerateToken(username.(string), server.TokenTypeStream, h.config.JWTSecret, expiryDuration)
	if err != nil {
		logger.Error().Err(err).Msg("failed to generate stream token")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"TOKEN_ERROR",
			"Failed to generate stream token",
			"",
		))
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(StreamTokenResponse{
		Token:     token,
		ExpiresIn: int(expiryDuration.Seconds()),
	}))
}
