package repository

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

// SettingsRepository provides access to the settings key-value store
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository creates a new SettingsRepository
func NewSettingsRepository(db *database.DB) *SettingsRepository {
	return &SettingsRepository{db: db.DB}
}

// Get retrieves a setting value by key. Returns empty string if not found.
func (r *SettingsRepository) Get(ctx context.Context, key string) (string, error) {
	const query = `SELECT value FROM settings WHERE key = ?`
	var value string
	err := r.db.QueryRowContext(ctx, query, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		logger.Error().Err(err).Str("key", key).Msg("failed to get setting")
		return "", err
	}
	return value, nil
}

// Set stores or updates a setting value
func (r *SettingsRepository) Set(ctx context.Context, key, value string) error {
	const command = `
	INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, command, key, value)
	if err != nil {
		logger.Error().Err(err).Str("key", key).Msg("failed to set setting")
		return err
	}
	return nil
}

// GetInt retrieves a setting value as an integer. Returns 0 if not found.
func (r *SettingsRepository) GetInt(ctx context.Context, key string) (int, error) {
	value, err := r.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	if value == "" {
		return 0, nil
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		logger.Error().Err(err).Str("key", key).Str("value", value).Msg("failed to parse setting as int")
		return 0, err
	}
	return intValue, nil
}

// SetInt stores or updates a setting value as an integer
func (r *SettingsRepository) SetInt(ctx context.Context, key string, value int) error {
	return r.Set(ctx, key, strconv.Itoa(value))
}

// Delete removes a setting by key
func (r *SettingsRepository) Delete(ctx context.Context, key string) error {
	const command = `DELETE FROM settings WHERE key = ?`
	_, err := r.db.ExecContext(ctx, command, key)
	if err != nil {
		logger.Error().Err(err).Str("key", key).Msg("failed to delete setting")
		return err
	}
	return nil
}

// Setting keys for worker management
const (
	SettingMaxParallelJobs = "max_parallel_jobs"
)
