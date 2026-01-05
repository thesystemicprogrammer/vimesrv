package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/worker"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// --- Mock implementations ---

type mockJobRepository struct {
	job *domain.Job
	err error
}

func (m *mockJobRepository) Get(ctx context.Context, jobID int64) (*domain.Job, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.job, nil
}

func (m *mockJobRepository) MarkSuccess(ctx context.Context, jobID int64) error {
	return nil
}

// Implement other JobRepository methods as no-ops
func (m *mockJobRepository) Enqueue(ctx context.Context, job *domain.Job) (int64, error) {
	return 0, nil
}
func (m *mockJobRepository) ClaimNextJobDue(ctx context.Context, workerID string) (*domain.Job, bool, error) {
	return nil, false, nil
}
func (m *mockJobRepository) ClaimNextJobDueExcludingTypes(ctx context.Context, workerID string, excludeTypes []string) (*domain.Job, bool, error) {
	return nil, false, nil
}
func (m *mockJobRepository) Reschedule(ctx context.Context, jobID int64, runAt time.Time, lastErr string) error {
	return nil
}
func (m *mockJobRepository) MarkDead(ctx context.Context, jobID int64, lastErr string) error {
	return nil
}
func (m *mockJobRepository) FindStuckJobs(ctx context.Context, threshold time.Duration) ([]*domain.Job, error) {
	return nil, nil
}
func (m *mockJobRepository) ResetStuckJob(ctx context.Context, jobID int64) error { return nil }
func (m *mockJobRepository) ExistsPendingJobByType(ctx context.Context, jobType string, language string) (bool, error) {
	return false, nil
}
func (m *mockJobRepository) ListJobs(ctx context.Context, filter ports.JobListFilter) (*ports.JobListResult, error) {
	return nil, nil
}
func (m *mockJobRepository) ClaimNextTranscodeJob(ctx context.Context, workerID string) (*domain.Job, error) {
	return nil, nil
}
func (m *mockJobRepository) CountQueuedTranscodeJobs(ctx context.Context) (int, error) {
	return 0, nil
}

type mockTranscodeRepository struct {
	transcode *domain.Transcode
	err       error
}

func (m *mockTranscodeRepository) Get(ctx context.Context, id string) (*domain.Transcode, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.transcode, nil
}

func (m *mockTranscodeRepository) MarkCompleted(ctx context.Context, id string, outputPath string) error {
	return nil
}

// Implement other TranscodeRepository methods as no-ops
func (m *mockTranscodeRepository) Create(ctx context.Context, transcode *domain.Transcode) error {
	return nil
}
func (m *mockTranscodeRepository) GetByMediaID(ctx context.Context, mediaID string) ([]*domain.Transcode, error) {
	return nil, nil
}
func (m *mockTranscodeRepository) GetProcessingByMediaID(ctx context.Context, mediaID string) ([]*domain.Transcode, error) {
	return nil, nil
}
func (m *mockTranscodeRepository) UpdateStatus(ctx context.Context, id string, status domain.TranscodeStatus) error {
	return nil
}
func (m *mockTranscodeRepository) MarkProcessing(ctx context.Context, id string, outputPath string) error {
	return nil
}
func (m *mockTranscodeRepository) MarkFailed(ctx context.Context, id string) error { return nil }
func (m *mockTranscodeRepository) Delete(ctx context.Context, id string) error     { return nil }
func (m *mockTranscodeRepository) ListPending(ctx context.Context, limit int) ([]*domain.Transcode, error) {
	return nil, nil
}

// --- Tests ---

func TestCompleteWorkerJobUseCase_Execute_VideoJob(t *testing.T) {
	// Setup temp directory with mock output files
	tempDir := t.TempDir()
	initPath := filepath.Join(tempDir, "init.mp4")
	segmentPath := filepath.Join(tempDir, "chunk-001.m4s")
	if err := os.WriteFile(initPath, []byte("init"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(segmentPath, []byte("segment"), 0644); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(TranscodeJobPayload{
		TranscodeID: "transcode-123",
	})

	job := &domain.Job{
		ID:       1,
		Type:     "transcode:video",
		Payload:  payload,
		Status:   shared.StatusRunning,
		WorkerID: sql.NullString{String: "worker-1", Valid: true},
	}

	transcode := &domain.Transcode{
		ID:         "transcode-123",
		MediaID:    "media-456",
		TrackType:  domain.TrackTypeVideo,
		OutputPath: tempDir,
		Status:     domain.TranscodeProcessing,
	}

	uc := NewCompleteWorkerJobUseCase(
		&mockJobRepository{job: job},
		&mockTranscodeRepository{transcode: transcode},
		worker.NewRegistry(60*time.Second),
		&ports.NoOpJobNotifier{},
	)

	input := CompleteJobInput{
		JobID:        1,
		WorkerID:     "worker-1",
		SegmentCount: 1,
	}

	err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestCompleteWorkerJobUseCase_Execute_SubtitleJob(t *testing.T) {
	tempDir := t.TempDir()
	vttPath := filepath.Join(tempDir, "subtitle.vtt")
	if err := os.WriteFile(vttPath, []byte("WEBVTT\n\n00:00:00.000 --> 00:00:05.000\nHello"), 0644); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(TranscodeJobPayload{
		TranscodeID: "transcode-sub-123",
	})

	job := &domain.Job{
		ID:       2,
		Type:     "transcode:subtitle",
		Payload:  payload,
		Status:   shared.StatusRunning,
		WorkerID: sql.NullString{String: "worker-1", Valid: true},
	}

	transcode := &domain.Transcode{
		ID:         "transcode-sub-123",
		MediaID:    "media-456",
		TrackType:  domain.TrackTypeSubtitle,
		OutputPath: vttPath,
		Status:     domain.TranscodeProcessing,
	}

	uc := NewCompleteWorkerJobUseCase(
		&mockJobRepository{job: job},
		&mockTranscodeRepository{transcode: transcode},
		worker.NewRegistry(60*time.Second),
		&ports.NoOpJobNotifier{},
	)

	input := CompleteJobInput{
		JobID:        2,
		WorkerID:     "worker-1",
		SegmentCount: 1,
	}

	err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestCompleteWorkerJobUseCase_Execute_JobNotOwnedByWorker(t *testing.T) {
	payload, _ := json.Marshal(TranscodeJobPayload{
		TranscodeID: "transcode-123",
	})

	job := &domain.Job{
		ID:       1,
		Type:     "transcode:video",
		Payload:  payload,
		Status:   shared.StatusRunning,
		WorkerID: sql.NullString{String: "other-worker", Valid: true},
	}

	uc := NewCompleteWorkerJobUseCase(
		&mockJobRepository{job: job},
		&mockTranscodeRepository{},
		worker.NewRegistry(60*time.Second),
		&ports.NoOpJobNotifier{},
	)

	input := CompleteJobInput{
		JobID:    1,
		WorkerID: "worker-1", // Different worker
	}

	err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("Expected error for job not owned by worker")
	}
	if err.Error() != "job not owned by worker worker-1" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCompleteWorkerJobUseCase_Execute_OutputValidationFails(t *testing.T) {
	// Empty temp directory - no output files
	tempDir := t.TempDir()

	payload, _ := json.Marshal(TranscodeJobPayload{
		TranscodeID: "transcode-123",
	})

	job := &domain.Job{
		ID:       1,
		Type:     "transcode:video",
		Payload:  payload,
		Status:   shared.StatusRunning,
		WorkerID: sql.NullString{String: "worker-1", Valid: true},
	}

	transcode := &domain.Transcode{
		ID:         "transcode-123",
		MediaID:    "media-456",
		TrackType:  domain.TrackTypeVideo,
		OutputPath: tempDir, // No init.mp4 or segments
		Status:     domain.TranscodeProcessing,
	}

	uc := NewCompleteWorkerJobUseCase(
		&mockJobRepository{job: job},
		&mockTranscodeRepository{transcode: transcode},
		worker.NewRegistry(60*time.Second),
		&ports.NoOpJobNotifier{},
	)

	input := CompleteJobInput{
		JobID:        1,
		WorkerID:     "worker-1",
		SegmentCount: 1,
	}

	err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("Expected error for missing output files")
	}
	// Error should mention init.mp4 not found
	if !contains(err.Error(), "init.mp4 not found") {
		t.Errorf("Expected error about init.mp4, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
