package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// SettingsRepository provides access to the settings key-value store
type SettingsRepository interface {
	// Get retrieves a setting value by key. Returns empty string if not found.
	Get(ctx context.Context, key string) (string, error)
	// Set stores or updates a setting value
	Set(ctx context.Context, key, value string) error
	// GetInt retrieves a setting value as an integer. Returns 0 if not found.
	GetInt(ctx context.Context, key string) (int, error)
	// SetInt stores or updates a setting value as an integer
	SetInt(ctx context.Context, key string, value int) error
	// Delete removes a setting by key
	Delete(ctx context.Context, key string) error
}

// WorkerConfigRepository provides access to worker configurations
type WorkerConfigRepository interface {
	// Get retrieves a worker config by name
	Get(ctx context.Context, name string) (*domain.WorkerConfig, error)
	// GetByID retrieves a worker config by ID
	GetByID(ctx context.Context, id int64) (*domain.WorkerConfig, error)
	// Create inserts a new worker config
	Create(ctx context.Context, cfg *domain.WorkerConfig) error
	// Update updates an existing worker config
	Update(ctx context.Context, cfg *domain.WorkerConfig) error
	// Upsert creates or updates a worker config
	Upsert(ctx context.Context, cfg *domain.WorkerConfig) error
	// Delete removes a worker config by name
	Delete(ctx context.Context, name string) error
	// ListByType returns all worker configs of a specific type
	ListByType(ctx context.Context, workerType domain.WorkerType) ([]*domain.WorkerConfig, error)
	// ListAll returns all worker configs
	ListAll(ctx context.Context) ([]*domain.WorkerConfig, error)
	// CountByType returns the count of worker configs of a specific type
	CountByType(ctx context.Context, workerType domain.WorkerType) (int, error)
	// EnsureLocalWorkersExist ensures that local worker configs exist for workers 1 through count
	EnsureLocalWorkersExist(ctx context.Context, count int) error
	// DeleteLocalWorkersAbove deletes local worker configs with index above the given count
	DeleteLocalWorkersAbove(ctx context.Context, count int) error
}
