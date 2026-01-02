package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// UserRepository provides persistence operations for users
type UserRepository interface {
	// Create inserts a new user
	Create(ctx context.Context, user *domain.User) error

	// Get retrieves a user by ID
	// Returns nil if not found
	Get(ctx context.Context, id string) (*domain.User, error)

	// GetByUsername retrieves a user by username (case-insensitive)
	// Returns nil if not found
	GetByUsername(ctx context.Context, username string) (*domain.User, error)

	// List retrieves all users
	List(ctx context.Context) ([]*domain.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *domain.User) error

	// Delete removes a user by ID
	Delete(ctx context.Context, id string) error

	// Count returns the total number of users
	Count(ctx context.Context) (int, error)

	// ExistsByUsername checks if a user with the given username exists
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}
