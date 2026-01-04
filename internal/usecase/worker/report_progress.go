package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/worker"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ReportProgressInput contains the input for reporting job progress
type ReportProgressInput struct {
	JobID      int64
	WorkerID   string
	Percentage float64
	Speed      string
	ETASeconds int
}

// ReportProgressUseCase handles progress reports from workers
type ReportProgressUseCase struct {
	jobRepo        ports.JobRepository
	workerRegistry *worker.Registry
	jobNotifier    ports.JobNotifier
}

// NewReportProgressUseCase creates a new ReportProgressUseCase
func NewReportProgressUseCase(
	jobRepo ports.JobRepository,
	workerRegistry *worker.Registry,
	jobNotifier ports.JobNotifier,
) *ReportProgressUseCase {
	return &ReportProgressUseCase{
		jobRepo:        jobRepo,
		workerRegistry: workerRegistry,
		jobNotifier:    jobNotifier,
	}
}

// Execute reports progress for a job and updates worker heartbeat
func (uc *ReportProgressUseCase) Execute(ctx context.Context, input ReportProgressInput) error {
	// 1. Touch worker timestamp (progress doubles as heartbeat)
	if !uc.workerRegistry.Touch(input.WorkerID) {
		return fmt.Errorf("worker not registered: %s", input.WorkerID)
	}

	// 2. Get job and verify ownership
	job, err := uc.jobRepo.Get(ctx, input.JobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}
	if !job.WorkerID.Valid || job.WorkerID.String != input.WorkerID {
		return fmt.Errorf("job not owned by worker %s", input.WorkerID)
	}

	// 3. Parse payload to get transcode info for the message
	var payload TranscodeJobPayload
	var transcodeID string
	if err := json.Unmarshal(job.Payload, &payload); err == nil {
		transcodeID = payload.TranscodeID
	}

	// 4. Broadcast progress via WebSocket
	uc.jobNotifier.NotifyJobProgress(job.ID, job.Type, ports.JobProgress{
		Percentage: input.Percentage,
		Speed:      input.Speed,
		Message:    fmt.Sprintf("Transcoding %s - %.1f%%", transcodeID, input.Percentage),
	})

	return nil
}
