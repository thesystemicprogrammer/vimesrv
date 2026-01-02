package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUsernameExists   = errors.New("username already exists")
	ErrInvalidRole      = errors.New("invalid role")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrCannotDeleteSelf = errors.New("cannot delete yourself")
	ErrLastAdmin        = errors.New("cannot delete the last admin user")
	ErrWrongPassword    = errors.New("current password is incorrect")
	ErrPasswordTooShort = errors.New("password must be at least 6 characters")
)

const (
	MinPasswordLength = 6
	BcryptCost        = 10
)

// CreateUserUseCase handles creating new users
type CreateUserUseCase struct {
	userRepo ports.UserRepository
}

func NewCreateUserUseCase(userRepo ports.UserRepository) *CreateUserUseCase {
	return &CreateUserUseCase{userRepo: userRepo}
}

type CreateUserInput struct {
	Username  string
	Password  string
	Role      shared.UserRole
	CreatedBy string // ID of the user creating this user
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, input CreateUserInput) (*domain.User, error) {
	// Validate role
	if !input.Role.IsValid() {
		return nil, ErrInvalidRole
	}

	// Validate password
	if len(input.Password) < MinPasswordLength {
		return nil, ErrPasswordTooShort
	}

	// Check if username already exists
	exists, err := uc.userRepo.ExistsByUsername(ctx, input.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), BcryptCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:                 uuid.New().String(),
		Username:           input.Username,
		PasswordHash:       string(hashedPassword),
		Role:               input.Role,
		MustChangePassword: true, // New users must change password on first login
	}

	if input.CreatedBy != "" {
		user.CreatedBy.Valid = true
		user.CreatedBy.String = input.CreatedBy
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
