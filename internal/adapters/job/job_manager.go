package job

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	usecasejob "github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SettingsKeyMaxParallelJobs is the database key for the max parallel jobs setting
const SettingsKeyMaxParallelJobs = "max_parallel_jobs"

type JobManagerInput struct {
	Config                  config.JobConfig
	ProcessNextJobUseCase   *usecasejob.ProcessNextJobUseCase
	SchedulerTickUseCase    *usecasejob.SchedulerTickUseCase
	RecoverStuckJobsUseCase *usecasejob.RecoverStuckJobsUseCase
	JobRepository           ports.JobRepository
	ScheduleRepository      ports.ScheduleRepository
	SettingsRepository      ports.SettingsRepository
	WorkerConfigRepository  ports.WorkerConfigRepository
	Handlers                ports.HandlerResolver
	BackoffStrategy         ports.BackoffStrategy
	Cron                    ports.CronParser
	Clock                   ports.Clock
}

// JobManager manages job processing with a dynamic pool of workers.
// It uses a semaphore pattern to limit the number of concurrent jobs.
type JobManager struct {
	JobManagerInput
	stopCh     chan struct{}
	cancelFunc context.CancelFunc
	ctx        context.Context
	started    int32
	stopped    int32
	wg         sync.WaitGroup

	// Concurrency control using semaphore pattern
	maxParallelJobs int32          // atomic: max concurrent jobs allowed
	activeJobs      int32          // atomic: currently running jobs
	dispatcherWg    sync.WaitGroup // tracks dispatcher goroutine

	// Worker configs for job filtering
	workerCfgs   map[string]*domain.WorkerConfig // worker name -> config
	workerCfgsMu sync.RWMutex                    // protects workerCfgs

	// Reconfiguration
	reconfigureMu sync.Mutex // prevents concurrent reconfiguration
}

func NewJobManager(input JobManagerInput) *JobManager {
	if input.Clock == nil {
		input.Clock = ports.RealClock{}
	}

	return &JobManager{
		JobManagerInput: input,
		stopCh:          make(chan struct{}),
		workerCfgs:      make(map[string]*domain.WorkerConfig),
	}
}

func (manager *JobManager) Start() error {
	if !atomic.CompareAndSwapInt32(&manager.started, 0, 1) {
		return fmt.Errorf("manager already started")
	}

	// Create a cancellable context for all workers
	ctx, cancel := context.WithCancel(context.Background())
	manager.cancelFunc = cancel
	manager.ctx = ctx

	// Load max parallel jobs from database
	maxJobs, err := manager.loadMaxParallelJobs(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to load max parallel jobs from database, defaulting to 1")
		maxJobs = 1
	}
	atomic.StoreInt32(&manager.maxParallelJobs, int32(maxJobs))

	// Load worker configs from database
	if err := manager.loadWorkerConfigs(ctx); err != nil {
		logger.Error().Err(err).Msg("failed to load worker configs from database")
		// Continue anyway - workers will process all job types
	}

	// Start the job dispatcher
	manager.dispatcherWg.Add(1)
	go func() {
		defer manager.dispatcherWg.Done()
		manager.dispatcherLoop(ctx)
	}()

	// Scheduler
	manager.wg.Add(1)
	go func() {
		defer manager.wg.Done()
		manager.schedulerLoop(ctx)
	}()

	// Stuck job recovery
	manager.wg.Add(1)
	go func() {
		defer manager.wg.Done()
		manager.stuckJobRecoveryLoop(ctx)
	}()

	logger.Info().Int("maxParallelJobs", maxJobs).Msg("job manager started")
	return nil
}

// dispatcherLoop is the main job dispatcher that manages concurrent job execution.
// It uses a semaphore pattern: checks if we're under the limit, then spawns a job processor.
func (manager *JobManager) dispatcherLoop(ctx context.Context) {
	duration := time.Duration(manager.Config.PollingIntervalInSeconds) * time.Second
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-manager.stopCh:
			return
		case <-ctx.Done():
			logger.Info().Msg("dispatcher context cancelled, shutting down")
			return
		default:
		}

		// Check if we can start a new job
		maxJobs := atomic.LoadInt32(&manager.maxParallelJobs)
		currentJobs := atomic.LoadInt32(&manager.activeJobs)

		if currentJobs >= maxJobs {
			// At capacity, wait before checking again
			select {
			case <-ticker.C:
			case <-manager.stopCh:
				return
			case <-ctx.Done():
				return
			}
			continue
		}

		// Try to claim and process a job
		// Use a worker name based on a slot number for config lookup
		slotNumber := currentJobs + 1
		workerName := fmt.Sprintf("server-worker-%d", slotNumber)

		// Get allowed job types for this worker slot
		excludeTypes := manager.getExcludedTypesForWorker(workerName)

		// Increment active jobs before trying to claim
		atomic.AddInt32(&manager.activeJobs, 1)

		go func() {
			defer atomic.AddInt32(&manager.activeJobs, -1)

			found, err := manager.ProcessNextJobUseCase.ExecuteWithExclusions(ctx, workerName, excludeTypes)
			if err != nil {
				logger.Error().Err(err).Str("worker", workerName).Msg("Use Case Processing Next Job failed")
				time.Sleep(shared.ErrorBackoffDuration)
				return
			}

			if !found {
				logger.Debug().Str("worker", workerName).Msg("no job found for worker")
				time.Sleep(100 * time.Millisecond)
				return
			}

			// Job processed successfully
		}()
	}
}

// loadMaxParallelJobs reads the max parallel jobs setting from the settings table
func (manager *JobManager) loadMaxParallelJobs(ctx context.Context) (int, error) {
	if manager.SettingsRepository == nil {
		return 1, nil // Default to 1 if no settings repository
	}
	count, err := manager.SettingsRepository.GetInt(ctx, SettingsKeyMaxParallelJobs)
	if err != nil {
		return 1, err
	}
	if count < 1 {
		return 1, nil
	}
	return count, nil
}

// loadWorkerConfigs loads all local worker configs from the database
func (manager *JobManager) loadWorkerConfigs(ctx context.Context) error {
	if manager.WorkerConfigRepository == nil {
		return nil
	}

	configs, err := manager.WorkerConfigRepository.ListByType(ctx, domain.WorkerTypeLocal)
	if err != nil {
		return err
	}

	manager.workerCfgsMu.Lock()
	defer manager.workerCfgsMu.Unlock()

	// Clear existing configs and reload
	manager.workerCfgs = make(map[string]*domain.WorkerConfig)
	for _, cfg := range configs {
		manager.workerCfgs[cfg.Name] = cfg
	}

	return nil
}

// GetWorkerConfig returns the worker config for a given worker name
func (manager *JobManager) GetWorkerConfig(workerName string) *domain.WorkerConfig {
	manager.workerCfgsMu.RLock()
	defer manager.workerCfgsMu.RUnlock()
	if cfg, ok := manager.workerCfgs[workerName]; ok {
		return cfg.Copy()
	}
	return nil
}

// ReloadWorkerConfigs reloads worker configurations from the database.
// This allows config changes (video/audio settings) to take effect without restart.
func (manager *JobManager) ReloadWorkerConfigs(ctx context.Context) error {
	return manager.loadWorkerConfigs(ctx)
}

// Reconfigure dynamically adjusts the max parallel jobs based on database settings.
func (manager *JobManager) Reconfigure(ctx context.Context) error {
	manager.reconfigureMu.Lock()
	defer manager.reconfigureMu.Unlock()

	if atomic.LoadInt32(&manager.stopped) == 1 {
		return fmt.Errorf("manager is stopped")
	}

	// Load new max parallel jobs from database
	newMax, err := manager.loadMaxParallelJobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to load max parallel jobs: %w", err)
	}

	oldMax := atomic.LoadInt32(&manager.maxParallelJobs)
	atomic.StoreInt32(&manager.maxParallelJobs, int32(newMax))

	// Reload worker configs
	if err := manager.loadWorkerConfigs(ctx); err != nil {
		logger.Error().Err(err).Msg("failed to reload worker configs during reconfigure")
		// Continue anyway - use existing configs
	}

	logger.Info().
		Int("from", int(oldMax)).
		Int("to", newMax).
		Msg("reconfigured max parallel jobs")

	return nil
}

// GetActiveJobCount returns the number of currently active jobs
func (manager *JobManager) GetActiveJobCount() int {
	return int(atomic.LoadInt32(&manager.activeJobs))
}

// GetMaxParallelJobs returns the current max parallel jobs setting
func (manager *JobManager) GetMaxParallelJobs() int {
	return int(atomic.LoadInt32(&manager.maxParallelJobs))
}

func (manager *JobManager) Stop(ctx context.Context) error {
	if atomic.CompareAndSwapInt32(&manager.stopped, 0, 1) {
		// Cancel the worker context to interrupt ongoing jobs
		if manager.cancelFunc != nil {
			manager.cancelFunc()
		}
		close(manager.stopCh)
	}
	done := make(chan struct{})
	go func() {
		manager.dispatcherWg.Wait()
		manager.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		logger.Info().Msg("job manager stopped")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// getExcludedTypesForWorker returns the job types that should be excluded for a specific worker
func (manager *JobManager) getExcludedTypesForWorker(workerName string) []string {
	cfg := manager.GetWorkerConfig(workerName)
	if cfg == nil {
		// No config found - accept all job types (default for backwards compatibility)
		return nil
	}

	var excludeTypes []string

	if !cfg.AcceptsVideo {
		excludeTypes = append(excludeTypes, "transcode_video")
	}
	if !cfg.AcceptsAudio {
		excludeTypes = append(excludeTypes, "transcode_audio")
	}

	return excludeTypes
}

func (manager *JobManager) schedulerLoop(ctx context.Context) {
	duration := time.Duration(manager.Config.SchedulerIntervalInSeconds) * time.Second
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-manager.stopCh:
			return
		case <-ctx.Done():
			logger.Info().Msg("scheduler context cancelled, shutting down")
			return
		case <-ticker.C:
			_ = manager.SchedulerTickUseCase.Execute(ctx)
		}
	}
}

func (manager *JobManager) stuckJobRecoveryLoop(ctx context.Context) {
	duration := time.Duration(manager.Config.StuckJobCheckIntervalMinutes) * time.Minute
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-manager.stopCh:
			return
		case <-ctx.Done():
			logger.Info().Msg("stuck job recovery context cancelled, shutting down")
			return
		case <-ticker.C:
			_ = manager.RecoverStuckJobsUseCase.Execute(ctx)
		}
	}
}
