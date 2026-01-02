package user

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ListUsersUseCase handles listing all users
type ListUsersUseCase struct {
	userRepo ports.UserRepository
}

func NewListUsersUseCase(userRepo ports.UserRepository) *ListUsersUseCase {
	return &ListUsersUseCase{userRepo: userRepo}
}

func (uc *ListUsersUseCase) Execute(ctx context.Context) ([]*domain.User, error) {
	return uc.userRepo.List(ctx)
}
