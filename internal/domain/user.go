package domain

import (
	"database/sql"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
)

// User represents a user in the system
type User struct {
	ID                 string
	Username           string
	PasswordHash       string
	Role               shared.UserRole
	MustChangePassword bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CreatedBy          sql.NullString
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == shared.RoleAdmin
}

// IsManager returns true if the user has manager role
func (u *User) IsManager() bool {
	return u.Role == shared.RoleManager
}

// CanManageUsers returns true if the user can manage other users (admin only)
func (u *User) CanManageUsers() bool {
	return u.IsAdmin()
}

// CanManageLibrary returns true if the user can manage the library (admin or manager)
func (u *User) CanManageLibrary() bool {
	return u.IsAdmin() || u.IsManager()
}
