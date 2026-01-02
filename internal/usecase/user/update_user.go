package user

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// UpdateUserUseCase handles updating a user's role
type UpdateUserUseCase struct {
	userRepo ports.UserRepository
}

func NewUpdateUserUseCase(userRepo ports.UserRepository) *UpdateUserUseCase {
	return &UpdateUserUseCase{userRepo: userRepo}
}

type UpdateUserInput struct {
	UserID  string
	Role    shared.UserRole
	ActorID string // ID of the user performing the update
}

func (uc *UpdateUserUseCase) Execute(ctx context.Context, input UpdateUserInput) (*domain.User, error) {
	// Validate role
	if !input.Role.IsValid() {
		return nil, ErrInvalidRole
	}

	// Get the user to update
	user, err := uc.userRepo.Get(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// If demoting from admin, ensure this is not the last admin
	if user.Role == shared.RoleAdmin && input.Role != shared.RoleAdmin {
		adminCount, err := uc.countAdmins(ctx)
		if err != nil {
			return nil, err
		}
		if adminCount <= 1 {
			return nil, ErrLastAdmin
		}
	}

	// Update the role
	user.Role = input.Role

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (uc *UpdateUserUseCase) countAdmins(ctx context.Context) (int, error) {
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
