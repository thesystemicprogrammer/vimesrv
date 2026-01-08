package worker

import (
	"context"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/worker"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// RegisterWorkerInput contains the input for registering a worker
type RegisterWorkerInput struct {
	WorkerID string
	Name     string
	Capacity int // Max concurrent jobs (from worker's perspective)
}

// RegisterWorkerOutput contains the response for worker registration
type RegisterWorkerOutput struct {
	// Config contains the server-side configuration for this worker
	Config *WorkerConfigInfo
}

// WorkerConfigInfo contains the configuration info to return to a worker
type WorkerConfigInfo struct {
	AcceptsVideo bool `json:"accepts_video"`
	AcceptsAudio bool `json:"accepts_audio"`
}

// RegisterWorkerUseCase handles worker registration
type RegisterWorkerUseCase struct {
	workerRegistry   *worker.Registry
	workerConfigRepo ports.WorkerConfigRepository
}

// NewRegisterWorkerUseCase creates a new RegisterWorkerUseCase
func NewRegisterWorkerUseCase(
	workerRegistry *worker.Registry,
	workerConfigRepo ports.WorkerConfigRepository,
) *RegisterWorkerUseCase {
	return &RegisterWorkerUseCase{
		workerRegistry:   workerRegistry,
		workerConfigRepo: workerConfigRepo,
	}
}

// Execute registers a worker with the server and ensures a config exists in the database
func (uc *RegisterWorkerUseCase) Execute(ctx context.Context, input RegisterWorkerInput) (*RegisterWorkerOutput, error) {
	// Default capacity to 1 if not specified
	capacity := input.Capacity
	if capacity <= 0 {
		capacity = 1
	}

	// Register in the in-memory registry for tracking online status
	uc.workerRegistry.Register(input.WorkerID, input.Name, capacity)

	// Ensure a worker config exists in the database
	var cfg *domain.WorkerConfig
	var err error

	if uc.workerConfigRepo != nil {
		cfg, err = uc.workerConfigRepo.Get(ctx, input.Name)
		if err != nil {
			logger.Error().Err(err).Str("name", input.Name).Msg("failed to get worker config")
			// Continue anyway - worker can still function without persistent config
		}

		if cfg == nil {
			// Create new config with defaults: disabled (no video, no audio)
			cfg = &domain.WorkerConfig{
				Name:         input.Name,
				WorkerType:   domain.WorkerTypeDistributed,
				AcceptsVideo: false,
				AcceptsAudio: false,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			if err := uc.workerConfigRepo.Create(ctx, cfg); err != nil {
				logger.Error().Err(err).Str("name", input.Name).Msg("failed to create worker config")
				// Continue anyway
			} else {
				logger.Info().
					Str("name", input.Name).
					Msg("Created new distributed worker config with defaults")
			}
		}
	}

	logger.Info().
		Str("worker_id", input.WorkerID).
		Str("name", input.Name).
		Int("capacity", capacity).
		Msg("Worker registered")

	// Build output with config info
	output := &RegisterWorkerOutput{}
	if cfg != nil {
		output.Config = &WorkerConfigInfo{
			AcceptsVideo: cfg.AcceptsVideo,
			AcceptsAudio: cfg.AcceptsAudio,
		}
	}

	return output, nil
}

// HeartbeatInput contains the input for worker heartbeat
type HeartbeatInput struct {
	WorkerID   string
	Name       string
	ActiveJobs int
	Capacity   int
}

// HeartbeatOutput contains the response for worker heartbeat
type HeartbeatOutput struct {
	OK         bool              `json:"ok"`
	ServerTime int64             `json:"server_time"`
	QueuedJobs int               `json:"queued_jobs"`
	Config     *WorkerConfigInfo `json:"config,omitempty"`
}

// HeartbeatUseCase handles worker heartbeats
type HeartbeatUseCase struct {
	workerRegistry   *worker.Registry
	jobRepo          ports.JobRepository
	workerConfigRepo ports.WorkerConfigRepository
}

// NewHeartbeatUseCase creates a new HeartbeatUseCase
func NewHeartbeatUseCase(
	workerRegistry *worker.Registry,
	jobRepo ports.JobRepository,
	workerConfigRepo ports.WorkerConfigRepository,
) *HeartbeatUseCase {
	return &HeartbeatUseCase{
		workerRegistry:   workerRegistry,
		jobRepo:          jobRepo,
		workerConfigRepo: workerConfigRepo,
	}
}

// Execute processes a worker heartbeat and returns server status including config
func (uc *HeartbeatUseCase) Execute(ctx context.Context, input HeartbeatInput) (*HeartbeatOutput, error) {
	// Update worker state
	if uc.workerRegistry.Touch(input.WorkerID) {
		// Update active jobs count
		uc.workerRegistry.SetActiveJobs(input.WorkerID, input.ActiveJobs)
	} else {
		// Worker not registered, register it now
		uc.workerRegistry.Register(input.WorkerID, input.Name, input.Capacity)
	}

	// Get queued job count for informational purposes
	queuedJobs, err := uc.jobRepo.CountQueuedTranscodeJobs(ctx)
	if err != nil {
		// Non-fatal, just log and continue
		logger.Warn().Err(err).Msg("failed to count queued transcode jobs")
		queuedJobs = 0
	}

	// Get worker config to return to the worker
	var configInfo *WorkerConfigInfo
	if uc.workerConfigRepo != nil {
		cfg, err := uc.workerConfigRepo.Get(ctx, input.Name)
		if err != nil {
			logger.Warn().Err(err).Str("name", input.Name).Msg("failed to get worker config for heartbeat")
		} else if cfg != nil {
			configInfo = &WorkerConfigInfo{
				AcceptsVideo: cfg.AcceptsVideo,
				AcceptsAudio: cfg.AcceptsAudio,
			}
		}
	}

	return &HeartbeatOutput{
		OK:         true,
		ServerTime: time.Now().Unix(),
		QueuedJobs: queuedJobs,
		Config:     configInfo,
	}, nil
}
