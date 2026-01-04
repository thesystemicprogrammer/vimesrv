package worker

import (
	"context"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/worker"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// RegisterWorkerInput contains the input for registering a worker
type RegisterWorkerInput struct {
	WorkerID string
	Name     string
	Capacity int // Max concurrent jobs
}

// RegisterWorkerUseCase handles worker registration
type RegisterWorkerUseCase struct {
	workerRegistry *worker.Registry
}

// NewRegisterWorkerUseCase creates a new RegisterWorkerUseCase
func NewRegisterWorkerUseCase(workerRegistry *worker.Registry) *RegisterWorkerUseCase {
	return &RegisterWorkerUseCase{
		workerRegistry: workerRegistry,
	}
}

// Execute registers a worker with the server
func (uc *RegisterWorkerUseCase) Execute(ctx context.Context, input RegisterWorkerInput) error {
	// Default capacity to 1 if not specified
	capacity := input.Capacity
	if capacity <= 0 {
		capacity = 1
	}

	uc.workerRegistry.Register(input.WorkerID, input.Name, capacity)

	logger.Info().
		Str("worker_id", input.WorkerID).
		Str("name", input.Name).
		Int("capacity", capacity).
		Msg("Worker registered")

	return nil
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
	OK         bool
	ServerTime int64
	QueuedJobs int
}

// HeartbeatUseCase handles worker heartbeats
type HeartbeatUseCase struct {
	workerRegistry *worker.Registry
	jobRepo        ports.JobRepository
}

// NewHeartbeatUseCase creates a new HeartbeatUseCase
func NewHeartbeatUseCase(
	workerRegistry *worker.Registry,
	jobRepo ports.JobRepository,
) *HeartbeatUseCase {
	return &HeartbeatUseCase{
		workerRegistry: workerRegistry,
		jobRepo:        jobRepo,
	}
}

// Execute processes a worker heartbeat and returns server status
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

	return &HeartbeatOutput{
		OK:         true,
		ServerTime: time.Now().Unix(),
		QueuedJobs: queuedJobs,
	}, nil
}
