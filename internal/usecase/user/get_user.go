package user

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetUserUseCase handles retrieving a user by ID
type GetUserUseCase struct {
	userRepo ports.UserRepository
}

func NewGetUserUseCase(userRepo ports.UserRepository) *GetUserUseCase {
	return &GetUserUseCase{userRepo: userRepo}
}

func (uc *GetUserUseCase) Execute(ctx context.Context, userID string) (*domain.User, error) {
	user, err := uc.userRepo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}
