package user

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// DeleteUserUseCase handles deleting a user
type DeleteUserUseCase struct {
	userRepo ports.UserRepository
}

func NewDeleteUserUseCase(userRepo ports.UserRepository) *DeleteUserUseCase {
	return &DeleteUserUseCase{userRepo: userRepo}
}

type DeleteUserInput struct {
	UserID  string
	ActorID string // ID of the user performing the deletion
}

func (uc *DeleteUserUseCase) Execute(ctx context.Context, input DeleteUserInput) error {
	// Prevent self-deletion
	if input.UserID == input.ActorID {
		return ErrCannotDeleteSelf
	}

	// Get the user to delete
	user, err := uc.userRepo.Get(ctx, input.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// If deleting an admin, ensure this is not the last admin
	if user.Role == shared.RoleAdmin {
		adminCount, err := uc.countAdmins(ctx)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return ErrLastAdmin
		}
	}

	return uc.userRepo.Delete(ctx, input.UserID)
}

func (uc *DeleteUserUseCase) countAdmins(ctx context.Context) (int, error) {
	users, err := uc.userRepo.List(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, u := range users {
		if u.Role == shared.RoleAdmin {
			count++
		}
	}
	return count, nil
}
