package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/user"
)

// UserHandler handles user management endpoints
type UserHandler struct {
	createUserUseCase    *user.CreateUserUseCase
	listUsersUseCase     *user.ListUsersUseCase
	getUserUseCase       *user.GetUserUseCase
	updateUserUseCase    *user.UpdateUserUseCase
	deleteUserUseCase    *user.DeleteUserUseCase
	resetPasswordUseCase *user.ResetPasswordUseCase
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	createUserUseCase *user.CreateUserUseCase,
	listUsersUseCase *user.ListUsersUseCase,
	getUserUseCase *user.GetUserUseCase,
	updateUserUseCase *user.UpdateUserUseCase,
	deleteUserUseCase *user.DeleteUserUseCase,
	resetPasswordUseCase *user.ResetPasswordUseCase,
) *UserHandler {
	return &UserHandler{
		createUserUseCase:    createUserUseCase,
		listUsersUseCase:     listUsersUseCase,
		getUserUseCase:       getUserUseCase,
		updateUserUseCase:    updateUserUseCase,
		deleteUserUseCase:    deleteUserUseCase,
		resetPasswordUseCase: resetPasswordUseCase,
	}
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID                 string `json:"id"`
	Username           string `json:"username"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"must_change_password"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Role string `json:"role" binding:"required"`
}

// ResetPasswordRequest represents the request to reset a user's password
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required"`
}

// toUserResponse converts a domain user to an API response
func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:                 u.ID,
		Username:           u.Username,
		Role:               string(u.Role),
		MustChangePassword: u.MustChangePassword,
		CreatedAt:          u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// RegisterProtectedRoutes registers user management routes (require admin role)
func (h *UserHandler) RegisterProtectedRoutes(router *gin.RouterGroup) {
	users := router.Group("/users")
	users.Use(h.requireAdmin())
	{
		users.GET("", h.ListUsers)
		users.POST("", h.CreateUser)
		users.GET("/:id", h.GetUser)
		users.PUT("/:id", h.UpdateUser)
		users.DELETE("/:id", h.DeleteUser)
		users.POST("/:id/reset-password", h.ResetPassword)
	}
	logger.Debug().Msg("User management routes registered")
}

// requireAdmin middleware checks if the user has admin role
func (h *UserHandler) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, server.ErrorResponse(
				"FORBIDDEN",
				"Access denied",
				"",
			))
			return
		}

		if role != string(shared.RoleAdmin) {
			c.AbortWithStatusJSON(http.StatusForbidden, server.ErrorResponse(
				"FORBIDDEN",
				"Admin access required",
				"",
			))
			return
		}

		c.Next()
	}
}

// ListUsers handles GET /api/v1/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.listUsersUseCase.Execute(c.Request.Context())
	if err != nil {
		logger.Error().Err(err).Msg("failed to list users")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"INTERNAL_ERROR",
			"Failed to list users",
			"",
		))
		return
	}

	response := make([]UserResponse, len(users))
	for i, u := range users {
		response[i] = toUserResponse(u)
	}

	c.JSON(http.StatusOK, server.SuccessResponse(response))
}

// CreateUser handles POST /api/v1/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	actorID, _ := c.Get("user_id")
	actorIDStr, _ := actorID.(string)

	input := user.CreateUserInput{
		Username:  req.Username,
		Password:  req.Password,
		Role:      shared.UserRole(req.Role),
		CreatedBy: actorIDStr,
	}

	newUser, err := h.createUserUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.handleUserError(c, err, "create user")
		return
	}

	logger.Info().
		Str("username", newUser.Username).
		Str("role", string(newUser.Role)).
		Msg("user created")

	c.JSON(http.StatusCreated, server.SuccessResponse(toUserResponse(newUser)))
}

// GetUser handles GET /api/v1/users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	foundUser, err := h.getUserUseCase.Execute(c.Request.Context(), userID)
	if err != nil {
		h.handleUserError(c, err, "get user")
		return
	}

	c.JSON(http.StatusOK, server.SuccessResponse(toUserResponse(foundUser)))
}

// UpdateUser handles PUT /api/v1/users/:id
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	actorID, _ := c.Get("user_id")
	actorIDStr, _ := actorID.(string)

	input := user.UpdateUserInput{
		UserID:  userID,
		Role:    shared.UserRole(req.Role),
		ActorID: actorIDStr,
	}

	updatedUser, err := h.updateUserUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.handleUserError(c, err, "update user")
		return
	}

	logger.Info().
		Str("user_id", userID).
		Str("new_role", string(updatedUser.Role)).
		Msg("user updated")

	c.JSON(http.StatusOK, server.SuccessResponse(toUserResponse(updatedUser)))
}

// DeleteUser handles DELETE /api/v1/users/:id
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	actorID, _ := c.Get("user_id")
	actorIDStr, _ := actorID.(string)

	input := user.DeleteUserInput{
		UserID:  userID,
		ActorID: actorIDStr,
	}

	if err := h.deleteUserUseCase.Execute(c.Request.Context(), input); err != nil {
		h.handleUserError(c, err, "delete user")
		return
	}

	logger.Info().
		Str("user_id", userID).
		Msg("user deleted")

	c.JSON(http.StatusOK, server.SuccessResponse(gin.H{"deleted": true}))
}

// ResetPassword handles POST /api/v1/users/:id/reset-password
func (h *UserHandler) ResetPassword(c *gin.Context) {
	userID := c.Param("id")

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	input := user.ResetPasswordInput{
		UserID:      userID,
		NewPassword: req.NewPassword,
	}

	updatedUser, err := h.resetPasswordUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.handleUserError(c, err, "reset password")
		return
	}

	logger.Info().
		Str("user_id", userID).
		Msg("user password reset")

	c.JSON(http.StatusOK, server.SuccessResponse(toUserResponse(updatedUser)))
}

// handleUserError handles common user errors and returns appropriate HTTP responses
func (h *UserHandler) handleUserError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, user.ErrUserNotFound):
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"NOT_FOUND",
			"User not found",
			"",
		))
	case errors.Is(err, user.ErrUsernameExists):
		c.JSON(http.StatusConflict, server.ErrorResponse(
			"USERNAME_EXISTS",
			"Username already exists",
			"",
		))
	case errors.Is(err, user.ErrInvalidRole):
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_ROLE",
			"Invalid role. Must be admin, manager, or user",
			"",
		))
	case errors.Is(err, user.ErrPasswordTooShort):
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"PASSWORD_TOO_SHORT",
			"Password must be at least 6 characters",
			"",
		))
	case errors.Is(err, user.ErrCannotDeleteSelf):
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"CANNOT_DELETE_SELF",
			"Cannot delete yourself",
			"",
		))
	case errors.Is(err, user.ErrLastAdmin):
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"LAST_ADMIN",
			"Cannot delete or demote the last admin user",
			"",
		))
	default:
		logger.Error().Err(err).Str("operation", operation).Msg("user operation failed")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"INTERNAL_ERROR",
			"Operation failed",
			"",
		))
	}
}
