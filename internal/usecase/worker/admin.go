// Package worker contains use cases for worker management.
package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/worker"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SettingsKeyMaxParallelJobs is the database key for max parallel jobs setting
const SettingsKeyMaxParallelJobs = "max_parallel_jobs"

// WorkerConfigWithStatus extends WorkerConfig with runtime status information
type WorkerConfigWithStatus struct {
	*domain.WorkerConfig
	Online     bool       `json:"online"`
	LastSeen   *time.Time `json:"last_seen,omitempty"`
	ActiveJobs int        `json:"active_jobs"`
}

// ListWorkerConfigsUseCase lists all worker configurations
type ListWorkerConfigsUseCase struct {
	workerConfigRepo ports.WorkerConfigRepository
	workerRegistry   *worker.Registry // Optional: for online status
}

// NewListWorkerConfigsUseCase creates a new ListWorkerConfigsUseCase
func NewListWorkerConfigsUseCase(workerConfigRepo ports.WorkerConfigRepository) *ListWorkerConfigsUseCase {
	return &ListWorkerConfigsUseCase{
		workerConfigRepo: workerConfigRepo,
	}
}

// WithRegistry adds a worker registry for online status information
func (uc *ListWorkerConfigsUseCase) WithRegistry(registry *worker.Registry) *ListWorkerConfigsUseCase {
	uc.workerRegistry = registry
	return uc
}

// ListWorkerConfigsOutput is the output of ListWorkerConfigsUseCase
type ListWorkerConfigsOutput struct {
	LocalWorkers       []*WorkerConfigWithStatus `json:"local_workers"`
	DistributedWorkers []*WorkerConfigWithStatus `json:"distributed_workers"`
}

// Execute lists all worker configurations grouped by type
func (uc *ListWorkerConfigsUseCase) Execute(ctx context.Context) (*ListWorkerConfigsOutput, error) {
	localWorkers, err := uc.workerConfigRepo.ListByType(ctx, domain.WorkerTypeLocal)
	if err != nil {
		return nil, fmt.Errorf("failed to list local workers: %w", err)
	}

	distributedWorkers, err := uc.workerConfigRepo.ListByType(ctx, domain.WorkerTypeDistributed)
	if err != nil {
		return nil, fmt.Errorf("failed to list distributed workers: %w", err)
	}

	return &ListWorkerConfigsOutput{
		LocalWorkers:       uc.enrichWithStatus(localWorkers),
		DistributedWorkers: uc.enrichWithStatus(distributedWorkers),
	}, nil
}

// enrichWithStatus adds runtime status to worker configs if registry is available
func (uc *ListWorkerConfigsUseCase) enrichWithStatus(configs []*domain.WorkerConfig) []*WorkerConfigWithStatus {
	result := make([]*WorkerConfigWithStatus, len(configs))
	for i, cfg := range configs {
		ws := &WorkerConfigWithStatus{
			WorkerConfig: cfg,
			Online:       false,
			ActiveJobs:   0,
		}

		// Enrich with registry data if available
		if uc.workerRegistry != nil {
			state := uc.workerRegistry.GetWorker(cfg.Name)
			if state != nil {
				ws.Online = uc.workerRegistry.IsWorkerAlive(cfg.Name)
				ws.LastSeen = &state.LastSeen
				ws.ActiveJobs = state.ActiveJobs
			}
		}

		result[i] = ws
	}
	return result
}

// GetLocalWorkerCountUseCase gets the configured local worker count
type GetLocalWorkerCountUseCase struct {
	settingsRepo ports.SettingsRepository
}

// NewGetLocalWorkerCountUseCase creates a new GetLocalWorkerCountUseCase
func NewGetLocalWorkerCountUseCase(settingsRepo ports.SettingsRepository) *GetLocalWorkerCountUseCase {
	return &GetLocalWorkerCountUseCase{
		settingsRepo: settingsRepo,
	}
}

// Execute returns the current local worker count
func (uc *GetLocalWorkerCountUseCase) Execute(ctx context.Context) (int, error) {
	count, err := uc.settingsRepo.GetInt(ctx, SettingsKeyMaxParallelJobs)
	if err != nil {
		return 0, fmt.Errorf("failed to get local worker count: %w", err)
	}
	if count < 1 {
		return 1, nil // Default to 1
	}
	return count, nil
}

// SetLocalWorkerCountUseCase sets the local worker count
type SetLocalWorkerCountUseCase struct {
	settingsRepo     ports.SettingsRepository
	workerConfigRepo ports.WorkerConfigRepository
}

// NewSetLocalWorkerCountUseCase creates a new SetLocalWorkerCountUseCase
func NewSetLocalWorkerCountUseCase(
	settingsRepo ports.SettingsRepository,
	workerConfigRepo ports.WorkerConfigRepository,
) *SetLocalWorkerCountUseCase {
	return &SetLocalWorkerCountUseCase{
		settingsRepo:     settingsRepo,
		workerConfigRepo: workerConfigRepo,
	}
}

// SetLocalWorkerCountInput is the input for SetLocalWorkerCountUseCase
type SetLocalWorkerCountInput struct {
	Count int `json:"count"`
}

// Execute sets the local worker count and ensures worker configs exist
func (uc *SetLocalWorkerCountUseCase) Execute(ctx context.Context, input SetLocalWorkerCountInput) error {
	if input.Count < 1 {
		return fmt.Errorf("worker count must be at least 1")
	}
	if input.Count > 16 {
		return fmt.Errorf("worker count cannot exceed 16")
	}

	// Update the setting
	if err := uc.settingsRepo.SetInt(ctx, SettingsKeyMaxParallelJobs, input.Count); err != nil {
		return fmt.Errorf("failed to set local worker count: %w", err)
	}

	// Ensure worker configs exist for all workers up to count
	if err := uc.workerConfigRepo.EnsureLocalWorkersExist(ctx, input.Count); err != nil {
		return fmt.Errorf("failed to ensure worker configs exist: %w", err)
	}

	// Delete worker configs above the count
	if err := uc.workerConfigRepo.DeleteLocalWorkersAbove(ctx, input.Count); err != nil {
		return fmt.Errorf("failed to delete excess worker configs: %w", err)
	}

	return nil
}

// GetWorkerConfigUseCase gets a specific worker configuration
type GetWorkerConfigUseCase struct {
	workerConfigRepo ports.WorkerConfigRepository
}

// NewGetWorkerConfigUseCase creates a new GetWorkerConfigUseCase
func NewGetWorkerConfigUseCase(workerConfigRepo ports.WorkerConfigRepository) *GetWorkerConfigUseCase {
	return &GetWorkerConfigUseCase{
		workerConfigRepo: workerConfigRepo,
	}
}

// Execute returns a worker configuration by name
func (uc *GetWorkerConfigUseCase) Execute(ctx context.Context, name string) (*domain.WorkerConfig, error) {
	config, err := uc.workerConfigRepo.Get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker config: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("worker config not found: %s", name)
	}
	return config, nil
}

// UpdateWorkerConfigUseCase updates a worker configuration
type UpdateWorkerConfigUseCase struct {
	workerConfigRepo ports.WorkerConfigRepository
}

// NewUpdateWorkerConfigUseCase creates a new UpdateWorkerConfigUseCase
func NewUpdateWorkerConfigUseCase(workerConfigRepo ports.WorkerConfigRepository) *UpdateWorkerConfigUseCase {
	return &UpdateWorkerConfigUseCase{
		workerConfigRepo: workerConfigRepo,
	}
}

// UpdateWorkerConfigInput is the input for UpdateWorkerConfigUseCase
type UpdateWorkerConfigInput struct {
	Name         string `json:"-"` // From URL path
	AcceptsVideo *bool  `json:"accepts_video,omitempty"`
	AcceptsAudio *bool  `json:"accepts_audio,omitempty"`
}

// Execute updates a worker configuration
func (uc *UpdateWorkerConfigUseCase) Execute(ctx context.Context, input UpdateWorkerConfigInput) (*domain.WorkerConfig, error) {
	// Get existing config
	config, err := uc.workerConfigRepo.Get(ctx, input.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker config: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("worker config not found: %s", input.Name)
	}

	// Apply updates
	if input.AcceptsVideo != nil {
		config.AcceptsVideo = *input.AcceptsVideo
	}
	if input.AcceptsAudio != nil {
		config.AcceptsAudio = *input.AcceptsAudio
	}

	// Save updates
	if err := uc.workerConfigRepo.Update(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to update worker config: %w", err)
	}

	return config, nil
}

// DeleteWorkerConfigUseCase deletes a distributed worker configuration
type DeleteWorkerConfigUseCase struct {
	workerConfigRepo ports.WorkerConfigRepository
}

// NewDeleteWorkerConfigUseCase creates a new DeleteWorkerConfigUseCase
func NewDeleteWorkerConfigUseCase(workerConfigRepo ports.WorkerConfigRepository) *DeleteWorkerConfigUseCase {
	return &DeleteWorkerConfigUseCase{
		workerConfigRepo: workerConfigRepo,
	}
}

// Execute deletes a distributed worker configuration by name.
// Only distributed workers can be deleted; local workers are managed via SetLocalWorkerCount.
func (uc *DeleteWorkerConfigUseCase) Execute(ctx context.Context, name string) error {
	// Get existing config to verify it exists and is a distributed worker
	config, err := uc.workerConfigRepo.Get(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get worker config: %w", err)
	}
	if config == nil {
		return fmt.Errorf("worker config not found: %s", name)
	}

	// Only allow deleting distributed workers
	if config.WorkerType != domain.WorkerTypeDistributed {
		return fmt.Errorf("cannot delete local worker config; use worker count to manage local workers")
	}

	// Delete the config
	if err := uc.workerConfigRepo.Delete(ctx, name); err != nil {
		return fmt.Errorf("failed to delete worker config: %w", err)
	}

	return nil
}
