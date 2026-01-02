package user

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"golang.org/x/crypto/bcrypt"
)

// ResetPasswordUseCase handles admin resetting a user's password
type ResetPasswordUseCase struct {
	userRepo ports.UserRepository
}

func NewResetPasswordUseCase(userRepo ports.UserRepository) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{userRepo: userRepo}
}

type ResetPasswordInput struct {
	UserID      string
	NewPassword string
}

func (uc *ResetPasswordUseCase) Execute(ctx context.Context, input ResetPasswordInput) (*domain.User, error) {
	// Validate password
	if len(input.NewPassword) < MinPasswordLength {
		return nil, ErrPasswordTooShort
	}

	// Get the user
	user, err := uc.userRepo.Get(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), BcryptCost)
	if err != nil {
		return nil, err
	}

	// Update password and set must_change_password flag
	user.PasswordHash = string(hashedPassword)
	user.MustChangePassword = true

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
