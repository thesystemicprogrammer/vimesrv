package user

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"golang.org/x/crypto/bcrypt"
)

// ChangePasswordUseCase handles a user changing their own password
type ChangePasswordUseCase struct {
	userRepo ports.UserRepository
}

func NewChangePasswordUseCase(userRepo ports.UserRepository) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{userRepo: userRepo}
}

type ChangePasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

func (uc *ChangePasswordUseCase) Execute(ctx context.Context, input ChangePasswordInput) (*domain.User, error) {
	// Validate new password
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

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.CurrentPassword)); err != nil {
		return nil, ErrWrongPassword
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), BcryptCost)
	if err != nil {
		return nil, err
	}

	// Update password and clear must_change_password flag
	user.PasswordHash = string(hashedPassword)
	user.MustChangePassword = false

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
