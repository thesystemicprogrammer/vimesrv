package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/user"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	config                *config.AuthConfig
	userRepo              ports.UserRepository
	changePasswordUseCase *user.ChangePasswordUseCase
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(cfg *config.AuthConfig, userRepo ports.UserRepository, changePasswordUseCase *user.ChangePasswordUseCase) *AuthHandler {
	return &AuthHandler{
		config:                cfg,
		userRepo:              userRepo,
		changePasswordUseCase: changePasswordUseCase,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token              string `json:"token"`
	ExpiresIn          int    `json:"expires_in"` // seconds until expiry
	MustChangePassword bool   `json:"must_change_password,omitempty"`
}

// ChangePasswordRequest represents the change password request body
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
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
	router.POST("/auth/change-password", h.ChangePassword)
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

	// Try database-based authentication first
	if h.userRepo != nil {
		if authenticated, response := h.tryDatabaseLogin(c.Request.Context(), req); authenticated {
			if response != nil {
				c.JSON(http.StatusOK, server.SuccessResponse(response))
			}
			return
		}
	}

	// Fall back to config-based authentication (legacy mode)
	h.tryConfigLogin(c, req)
}

// tryDatabaseLogin attempts to authenticate against the database
// Returns (true, response) if handled (success or error sent), (false, nil) if should fall back to config
func (h *AuthHandler) tryDatabaseLogin(ctx context.Context, req LoginRequest) (bool, *LoginResponse) {
	// Check if there are any users in the database
	userCount, err := h.userRepo.Count(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to count users")
		return false, nil
	}

	// If no users in database, fall back to config-based auth
	if userCount == 0 {
		return false, nil
	}

	// Try to find user by username
	foundUser, err := h.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get user by username")
		return false, nil
	}

	if foundUser == nil {
		logger.Debug().
			Str("username", req.Username).
			Msg("login failed: user not found in database")
		return false, nil
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.PasswordHash), []byte(req.Password)); err != nil {
		logger.Debug().
			Str("username", req.Username).
			Msg("login failed: invalid password")
		return false, nil
	}

	// Generate JWT token with user claims
	expiryDuration := time.Duration(h.config.TokenExpiryHours) * time.Hour
	extraClaims := &server.TokenClaims{
		UserID:             foundUser.ID,
		Role:               string(foundUser.Role),
		MustChangePassword: foundUser.MustChangePassword,
	}

	token, err := server.GenerateTokenWithClaims(req.Username, server.TokenTypeAPI, h.config.JWTSecret, expiryDuration, extraClaims)
	if err != nil {
		logger.Error().Err(err).Msg("failed to generate token")
		return false, nil
	}

	logger.Info().
		Str("username", req.Username).
		Str("user_id", foundUser.ID).
		Str("role", string(foundUser.Role)).
		Msg("user logged in successfully (database auth)")

	return true, &LoginResponse{
		Token:              token,
		ExpiresIn:          int(expiryDuration.Seconds()),
		MustChangePassword: foundUser.MustChangePassword,
	}
}

// tryConfigLogin attempts to authenticate against the config file (legacy mode)
func (h *AuthHandler) tryConfigLogin(c *gin.Context, req LoginRequest) {
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

	// Generate JWT token (legacy mode - no user ID or role, treated as admin)
	expiryDuration := time.Duration(h.config.TokenExpiryHours) * time.Hour
	extraClaims := &server.TokenClaims{
		Role: string(shared.RoleAdmin), // Config-based users are treated as admin
	}

	token, err := server.GenerateTokenWithClaims(req.Username, server.TokenTypeAPI, h.config.JWTSecret, expiryDuration, extraClaims)
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
		Msg("user logged in successfully (config auth)")

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

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	mustChangePassword, _ := c.Get("must_change_password")

	response := gin.H{
		"username": username,
	}

	if userID != nil && userID != "" {
		response["user_id"] = userID
	}
	if role != nil && role != "" {
		response["role"] = role
	}
	if mustChangePassword != nil {
		response["must_change_password"] = mustChangePassword
	}

	c.JSON(http.StatusOK, server.SuccessResponse(response))
}

// ChangePassword handles POST /api/v1/auth/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists || userID == nil || userID == "" {
		// Config-based users cannot change password via API
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"NOT_SUPPORTED",
			"Password change not supported for config-based authentication",
			"",
		))
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	input := user.ChangePasswordInput{
		UserID:          userID.(string),
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}

	_, err := h.changePasswordUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrWrongPassword):
			c.JSON(http.StatusUnauthorized, server.ErrorResponse(
				"WRONG_PASSWORD",
				"Current password is incorrect",
				"",
			))
		case errors.Is(err, user.ErrPasswordTooShort):
			c.JSON(http.StatusBadRequest, server.ErrorResponse(
				"PASSWORD_TOO_SHORT",
				"Password must be at least 6 characters",
				"",
			))
		case errors.Is(err, user.ErrUserNotFound):
			c.JSON(http.StatusNotFound, server.ErrorResponse(
				"USER_NOT_FOUND",
				"User not found",
				"",
			))
		default:
			logger.Error().Err(err).Msg("failed to change password")
			c.JSON(http.StatusInternalServerError, server.ErrorResponse(
				"INTERNAL_ERROR",
				"Failed to change password",
				"",
			))
		}
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Msg("user changed password")

	c.JSON(http.StatusOK, server.SuccessResponse(gin.H{"success": true}))
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
