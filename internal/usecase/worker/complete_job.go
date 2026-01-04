package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/worker"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// CompleteJobInput contains the input for completing a worker job
type CompleteJobInput struct {
	JobID        int64
	WorkerID     string
	SegmentCount int      // Number of segments created
	OutputFiles  []string // List of output files (optional, for validation)
}

// CompleteWorkerJobUseCase handles marking a worker job as completed
type CompleteWorkerJobUseCase struct {
	jobRepo        ports.JobRepository
	transcodeRepo  ports.TranscodeRepository
	workerRegistry *worker.Registry
	jobNotifier    ports.JobNotifier
	transcoder     ports.Transcoder
	filesystem     ports.FileSystemService
}

// NewCompleteWorkerJobUseCase creates a new CompleteWorkerJobUseCase
func NewCompleteWorkerJobUseCase(
	jobRepo ports.JobRepository,
	transcodeRepo ports.TranscodeRepository,
	workerRegistry *worker.Registry,
	jobNotifier ports.JobNotifier,
	transcoder ports.Transcoder,
	filesystem ports.FileSystemService,
) *CompleteWorkerJobUseCase {
	return &CompleteWorkerJobUseCase{
		jobRepo:        jobRepo,
		transcodeRepo:  transcodeRepo,
		workerRegistry: workerRegistry,
		jobNotifier:    jobNotifier,
		transcoder:     transcoder,
		filesystem:     filesystem,
	}
}

// Execute marks a job as completed after validating the output
func (uc *CompleteWorkerJobUseCase) Execute(ctx context.Context, input CompleteJobInput) error {
	// 1. Touch worker timestamp
	uc.workerRegistry.Touch(input.WorkerID)

	// 2. Get job and verify ownership
	job, err := uc.jobRepo.Get(ctx, input.JobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}
	if !job.WorkerID.Valid || job.WorkerID.String != input.WorkerID {
		return fmt.Errorf("job not owned by worker %s", input.WorkerID)
	}

	// 3. Parse payload to get transcode ID
	var payload TranscodeJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("invalid job payload: %w", err)
	}

	// 4. Get transcode record
	transcode, err := uc.transcodeRepo.Get(ctx, payload.TranscodeID)
	if err != nil {
		return fmt.Errorf("transcode not found: %w", err)
	}

	// 5. Validate output exists
	if err := uc.validateOutput(transcode.OutputPath, transcode.TrackType); err != nil {
		return fmt.Errorf("output validation failed: %w", err)
	}

	// 6. For video/audio: probe segment durations and save segments.json
	if transcode.TrackType == domain.TrackTypeVideo || transcode.TrackType == domain.TrackTypeAudio {
		segments, err := uc.transcoder.ProbeSegmentDurations(ctx, transcode.OutputPath)
		if err != nil {
			logger.Warn().Err(err).Str("transcode_id", transcode.ID).Msg("failed to probe segment durations")
		} else {
			if saveErr := uc.saveSegmentsJSON(transcode.OutputPath, segments); saveErr != nil {
				logger.Warn().Err(saveErr).Str("transcode_id", transcode.ID).Msg("failed to save segments.json")
			}
		}
	}

	// 7. Mark transcode as completed
	if err := uc.transcodeRepo.MarkCompleted(ctx, transcode.ID, transcode.OutputPath); err != nil {
		return fmt.Errorf("failed to mark transcode as completed: %w", err)
	}

	// 8. Mark job as succeeded
	if err := uc.jobRepo.MarkSuccess(ctx, job.ID); err != nil {
		return fmt.Errorf("failed to mark job as succeeded: %w", err)
	}

	// 9. Decrement worker's active job count
	uc.workerRegistry.DecrementActiveJobs(input.WorkerID)

	// 10. Notify via WebSocket
	job.FinishedAt = sql.NullTime{Time: time.Now(), Valid: true}
	uc.jobNotifier.NotifyJobCompleted(job)

	logger.Info().
		Int64("job_id", job.ID).
		Str("transcode_id", transcode.ID).
		Str("worker_id", input.WorkerID).
		Str("track_type", string(transcode.TrackType)).
		Msg("Worker completed transcode job")

	return nil
}

// validateOutput checks that the expected output files exist
func (uc *CompleteWorkerJobUseCase) validateOutput(outputPath string, trackType domain.TrackType) error {
	switch trackType {
	case domain.TrackTypeVideo, domain.TrackTypeAudio:
		// Check init.mp4 exists
		initPath := filepath.Join(outputPath, "init.mp4")
		if _, err := os.Stat(initPath); os.IsNotExist(err) {
			return fmt.Errorf("init.mp4 not found at %s", initPath)
		}

		// Check at least one segment exists
		pattern := filepath.Join(outputPath, "chunk-*.m4s")
		segments, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("failed to glob segments: %w", err)
		}
		if len(segments) == 0 {
			return fmt.Errorf("no segment files found matching %s", pattern)
		}

	case domain.TrackTypeSubtitle:
		// outputPath is the .vtt file itself
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			return fmt.Errorf("subtitle file not found at %s", outputPath)
		}
	}

	return nil
}

// saveSegmentsJSON saves segment timing data to segments.json
func (uc *CompleteWorkerJobUseCase) saveSegmentsJSON(outputPath string, segments []ports.SegmentInfo) error {
	data := struct {
		Segments []ports.SegmentInfo `json:"segments"`
	}{
		Segments: segments,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal segment timings: %w", err)
	}

	timingFilePath := filepath.Join(outputPath, "segments.json")
	return uc.filesystem.WriteFile(timingFilePath, jsonData)
}
