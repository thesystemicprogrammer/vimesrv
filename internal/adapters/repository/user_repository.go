package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

type SQLiteUserRepository struct {
	db *sql.DB
}

func NewSQLiteUserRepository(db *database.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db.DB}
}

func (r *SQLiteUserRepository) scanUserRow(s scanner) (*domain.User, error) {
	var user domain.User
	var roleStr string
	var mustChangePassword int

	err := s.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&roleStr,
		&mustChangePassword,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	user.Role = shared.UserRole(roleStr)
	user.MustChangePassword = mustChangePassword == 1

	return &user, nil
}

func (r *SQLiteUserRepository) Create(ctx context.Context, user *domain.User) error {
	const query = `
		INSERT INTO users (id, username, password_hash, role, must_change_password, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	mustChangePassword := 0
	if user.MustChangePassword {
		mustChangePassword = 1
	}

	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.PasswordHash,
		string(user.Role),
		mustChangePassword,
		user.CreatedAt,
		user.UpdatedAt,
		user.CreatedBy,
	)
	if err != nil {
		logger.Error().Err(err).Str("username", user.Username).Msg("failed to create user")
		return err
	}

	return nil
}

func (r *SQLiteUserRepository) Get(ctx context.Context, id string) (*domain.User, error) {
	const query = `
		SELECT id, username, password_hash, role, must_change_password, created_at, updated_at, created_by
		FROM users
		WHERE id = ?
	`

	user, err := r.scanUserRow(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		logger.Error().Err(err).Str("id", id).Msg("failed to get user by id")
		return nil, err
	}

	return user, nil
}

func (r *SQLiteUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	const query = `
		SELECT id, username, password_hash, role, must_change_password, created_at, updated_at, created_by
		FROM users
		WHERE username = ? COLLATE NOCASE
	`

	user, err := r.scanUserRow(r.db.QueryRowContext(ctx, query, username))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		logger.Error().Err(err).Str("username", username).Msg("failed to get user by username")
		return nil, err
	}

	return user, nil
}

func (r *SQLiteUserRepository) List(ctx context.Context) ([]*domain.User, error) {
	const query = `
		SELECT id, username, password_hash, role, must_change_password, created_at, updated_at, created_by
		FROM users
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		logger.Error().Err(err).Msg("failed to list users")
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user, err := r.scanUserRow(rows)
		if err != nil {
			logger.Error().Err(err).Msg("failed to scan user row")
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("error iterating user rows")
		return nil, err
	}

	return users, nil
}

func (r *SQLiteUserRepository) Update(ctx context.Context, user *domain.User) error {
	const query = `
		UPDATE users
		SET username = ?, password_hash = ?, role = ?, must_change_password = ?, updated_at = ?
		WHERE id = ?
	`

	mustChangePassword := 0
	if user.MustChangePassword {
		mustChangePassword = 1
	}

	user.UpdatedAt = time.Now().UTC()

	result, err := r.db.ExecContext(ctx, query,
		user.Username,
		user.PasswordHash,
		string(user.Role),
		mustChangePassword,
		user.UpdatedAt,
		user.ID,
	)
	if err != nil {
		logger.Error().Err(err).Str("id", user.ID).Msg("failed to update user")
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logger.Warn().Str("id", user.ID).Msg("no user found to update")
	}

	return nil
}

func (r *SQLiteUserRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM users WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		logger.Error().Err(err).Str("id", id).Msg("failed to delete user")
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logger.Warn().Str("id", id).Msg("no user found to delete")
	}

	return nil
}

func (r *SQLiteUserRepository) Count(ctx context.Context) (int, error) {
	const query = `SELECT COUNT(*) FROM users`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		logger.Error().Err(err).Msg("failed to count users")
		return 0, err
	}

	return count, nil
}

func (r *SQLiteUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	const query = `SELECT 1 FROM users WHERE username = ? COLLATE NOCASE LIMIT 1`

	var exists int
	err := r.db.QueryRowContext(ctx, query, username).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		logger.Error().Err(err).Str("username", username).Msg("failed to check if user exists")
		return false, err
	}

	return true, nil
}
