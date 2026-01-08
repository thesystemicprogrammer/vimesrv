package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

// WorkerConfigRepository provides access to worker configurations
type WorkerConfigRepository struct {
	db *sql.DB
}

// NewWorkerConfigRepository creates a new WorkerConfigRepository
func NewWorkerConfigRepository(db *database.DB) *WorkerConfigRepository {
	return &WorkerConfigRepository{db: db.DB}
}

// scanWorkerConfig scans a row into a WorkerConfig struct
func (r *WorkerConfigRepository) scanWorkerConfig(s scanner) (*domain.WorkerConfig, error) {
	var cfg domain.WorkerConfig
	var workerType string
	err := s.Scan(
		&cfg.ID,
		&cfg.Name,
		&workerType,
		&cfg.AcceptsVideo,
		&cfg.AcceptsAudio,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	cfg.WorkerType = domain.WorkerType(workerType)
	return &cfg, nil
}

// Get retrieves a worker config by name
func (r *WorkerConfigRepository) Get(ctx context.Context, name string) (*domain.WorkerConfig, error) {
	const query = `
	SELECT id, name, worker_type, accepts_video, accepts_audio, created_at, updated_at
	FROM worker_configs
	WHERE name = ?
	`
	cfg, err := r.scanWorkerConfig(r.db.QueryRowContext(ctx, query, name))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		logger.Error().Err(err).Str("name", name).Msg("failed to get worker config")
		return nil, err
	}
	return cfg, nil
}

// Create inserts a new worker config
func (r *WorkerConfigRepository) Create(ctx context.Context, cfg *domain.WorkerConfig) error {
	const command = `
	INSERT INTO worker_configs (name, worker_type, accepts_video, accepts_audio, created_at, updated_at)
	VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	result, err := r.db.ExecContext(ctx, command, cfg.Name, string(cfg.WorkerType), cfg.AcceptsVideo, cfg.AcceptsAudio)
	if err != nil {
		logger.Error().Err(err).Str("name", cfg.Name).Msg("failed to create worker config")
		return err
	}
	id, _ := result.LastInsertId()
	cfg.ID = id
	return nil
}

// Update updates an existing worker config
func (r *WorkerConfigRepository) Update(ctx context.Context, cfg *domain.WorkerConfig) error {
	const command = `
	UPDATE worker_configs
	SET accepts_video = ?, accepts_audio = ?, updated_at = CURRENT_TIMESTAMP
	WHERE name = ?
	`
	result, err := r.db.ExecContext(ctx, command, cfg.AcceptsVideo, cfg.AcceptsAudio, cfg.Name)
	if err != nil {
		logger.Error().Err(err).Str("name", cfg.Name).Msg("failed to update worker config")
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("worker config not found: %s", cfg.Name)
	}
	return nil
}

// Upsert creates or updates a worker config
func (r *WorkerConfigRepository) Upsert(ctx context.Context, cfg *domain.WorkerConfig) error {
	const command = `
	INSERT INTO worker_configs (name, worker_type, accepts_video, accepts_audio, created_at, updated_at)
	VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	ON CONFLICT(name) DO UPDATE SET
		accepts_video = excluded.accepts_video,
		accepts_audio = excluded.accepts_audio,
		updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, command, cfg.Name, string(cfg.WorkerType), cfg.AcceptsVideo, cfg.AcceptsAudio)
	if err != nil {
		logger.Error().Err(err).Str("name", cfg.Name).Msg("failed to upsert worker config")
		return err
	}
	return nil
}

// Delete removes a worker config by name
func (r *WorkerConfigRepository) Delete(ctx context.Context, name string) error {
	const command = `DELETE FROM worker_configs WHERE name = ?`
	_, err := r.db.ExecContext(ctx, command, name)
	if err != nil {
		logger.Error().Err(err).Str("name", name).Msg("failed to delete worker config")
		return err
	}
	return nil
}

// ListByType returns all worker configs of a specific type
func (r *WorkerConfigRepository) ListByType(ctx context.Context, workerType domain.WorkerType) ([]*domain.WorkerConfig, error) {
	const query = `
	SELECT id, name, worker_type, accepts_video, accepts_audio, created_at, updated_at
	FROM worker_configs
	WHERE worker_type = ?
	ORDER BY name
	`
	rows, err := r.db.QueryContext(ctx, query, string(workerType))
	if err != nil {
		logger.Error().Err(err).Str("workerType", string(workerType)).Msg("failed to list worker configs by type")
		return nil, err
	}
	defer rows.Close()

	var configs []*domain.WorkerConfig
	for rows.Next() {
		cfg, err := r.scanWorkerConfig(rows)
		if err != nil {
			logger.Error().Err(err).Msg("failed to scan worker config row")
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, rows.Err()
}

// ListAll returns all worker configs
func (r *WorkerConfigRepository) ListAll(ctx context.Context) ([]*domain.WorkerConfig, error) {
	const query = `
	SELECT id, name, worker_type, accepts_video, accepts_audio, created_at, updated_at
	FROM worker_configs
	ORDER BY worker_type, name
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		logger.Error().Err(err).Msg("failed to list all worker configs")
		return nil, err
	}
	defer rows.Close()

	var configs []*domain.WorkerConfig
	for rows.Next() {
		cfg, err := r.scanWorkerConfig(rows)
		if err != nil {
			logger.Error().Err(err).Msg("failed to scan worker config row")
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, rows.Err()
}

// GetByID retrieves a worker config by ID
func (r *WorkerConfigRepository) GetByID(ctx context.Context, id int64) (*domain.WorkerConfig, error) {
	const query = `
	SELECT id, name, worker_type, accepts_video, accepts_audio, created_at, updated_at
	FROM worker_configs
	WHERE id = ?
	`
	cfg, err := r.scanWorkerConfig(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		logger.Error().Err(err).Int64("id", id).Msg("failed to get worker config by ID")
		return nil, err
	}
	return cfg, nil
}

// CountByType returns the count of worker configs of a specific type
func (r *WorkerConfigRepository) CountByType(ctx context.Context, workerType domain.WorkerType) (int, error) {
	const query = `SELECT COUNT(*) FROM worker_configs WHERE worker_type = ?`
	var count int
	err := r.db.QueryRowContext(ctx, query, string(workerType)).Scan(&count)
	if err != nil {
		logger.Error().Err(err).Str("workerType", string(workerType)).Msg("failed to count worker configs by type")
		return 0, err
	}
	return count, nil
}

// EnsureLocalWorkersExist ensures that local worker configs exist for workers 1 through count.
// Creates new configs with accepts_video=false, accepts_audio=false for new workers (except first worker).
func (r *WorkerConfigRepository) EnsureLocalWorkersExist(ctx context.Context, count int) error {
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("server-worker-%d", i)
		existing, err := r.Get(ctx, name)
		if err != nil {
			return err
		}
		if existing == nil {
			// Create new worker config
			// First worker gets video+audio enabled, subsequent workers are disabled
			cfg := &domain.WorkerConfig{
				Name:         name,
				WorkerType:   domain.WorkerTypeLocal,
				AcceptsVideo: i == 1,
				AcceptsAudio: i == 1,
			}
			if err := r.Create(ctx, cfg); err != nil {
				return err
			}
			logger.Info().Str("name", name).Bool("enabled", i == 1).Msg("created local worker config")
		}
	}
	return nil
}

// DeleteLocalWorkersAbove deletes local worker configs with index above the given count
func (r *WorkerConfigRepository) DeleteLocalWorkersAbove(ctx context.Context, count int) error {
	// Get all local workers
	workers, err := r.ListByType(ctx, domain.WorkerTypeLocal)
	if err != nil {
		return err
	}

	for _, w := range workers {
		var idx int
		if _, err := fmt.Sscanf(w.Name, "server-worker-%d", &idx); err == nil {
			if idx > count {
				if err := r.Delete(ctx, w.Name); err != nil {
					return err
				}
				logger.Info().Str("name", w.Name).Msg("deleted local worker config")
			}
		}
	}
	return nil
}
