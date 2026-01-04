package transcode

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/transcode"
)

// MockTranscodeRepository for testing
type MockTranscodeRepository struct {
	GetFn            func(ctx context.Context, id string) (*domain.Transcode, error)
	MarkProcessingFn func(ctx context.Context, id string, outputPath string) error
	MarkCompletedFn  func(ctx context.Context, id string, outputPath string) error
	MarkFailedFn     func(ctx context.Context, id string) error
}

func (m *MockTranscodeRepository) Create(ctx context.Context, transcode *domain.Transcode) error {
	return nil
}

func (m *MockTranscodeRepository) Get(ctx context.Context, id string) (*domain.Transcode, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, id)
	}
	return &domain.Transcode{
		ID:         id,
		MediaID:    "media123",
		Quality:    "360p",
		TrackType:  domain.TrackTypeVideo,
		TrackIndex: 0,
		Status:     domain.TranscodePending,
	}, nil
}

func (m *MockTranscodeRepository) GetByMediaID(ctx context.Context, mediaID string) ([]*domain.Transcode, error) {
	return nil, nil
}

func (m *MockTranscodeRepository) UpdateStatus(ctx context.Context, id string, status domain.TranscodeStatus) error {
	return nil
}

func (m *MockTranscodeRepository) MarkProcessing(ctx context.Context, id string, outputPath string) error {
	if m.MarkProcessingFn != nil {
		return m.MarkProcessingFn(ctx, id, outputPath)
	}
	return nil
}

func (m *MockTranscodeRepository) MarkCompleted(ctx context.Context, id string, outputPath string) error {
	if m.MarkCompletedFn != nil {
		return m.MarkCompletedFn(ctx, id, outputPath)
	}
	return nil
}

func (m *MockTranscodeRepository) MarkFailed(ctx context.Context, id string) error {
	if m.MarkFailedFn != nil {
		return m.MarkFailedFn(ctx, id)
	}
	return nil
}

func (m *MockTranscodeRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *MockTranscodeRepository) ListPending(ctx context.Context, limit int) ([]*domain.Transcode, error) {
	return nil, nil
}

func (m *MockTranscodeRepository) GetProcessingByMediaID(ctx context.Context, mediaID string) ([]*domain.Transcode, error) {
	return nil, nil
}

// MockMediaRepository for testing
type MockMediaRepository struct {
	GetFn func(ctx context.Context, id string) (*domain.MediaFile, error)
}

func (m *MockMediaRepository) Create(ctx context.Context, media *domain.MediaFile) error {
	return nil
}

func (m *MockMediaRepository) FindByFingerprint(ctx context.Context, fingerprint string) (*domain.MediaFile, error) {
	return nil, nil
}

func (m *MockMediaRepository) ExistsByFingerprint(ctx context.Context, fingerprint string) (bool, error) {
	return false, nil
}

func (m *MockMediaRepository) Update(ctx context.Context, media *domain.MediaFile) error {
	return nil
}

func (m *MockMediaRepository) Get(ctx context.Context, id string) (*domain.MediaFile, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, id)
	}
	return &domain.MediaFile{
		ID:       id,
		FilePath: "/media/test.mp4",
	}, nil
}

func (m *MockMediaRepository) List(ctx context.Context, page, perPage int) ([]*domain.MediaFile, int, error) {
	return nil, 0, nil
}

func (m *MockMediaRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *MockMediaRepository) FindByEpisodeMetadataIDs(ctx context.Context, episodeMetadataIDs []int64) ([]*domain.MediaFile, error) {
	return nil, nil
}

// MockTranscoder for testing
type MockTranscoder struct {
	IsAvailableFn           func() error
	TranscodeVideoFn        func(ctx context.Context, opts ports.TranscodeOptions, progress ports.ProgressCallback) error
	TranscodeAudioFn        func(ctx context.Context, opts ports.TranscodeOptions, progress ports.ProgressCallback) error
	ExtractSubtitleFn       func(ctx context.Context, opts ports.TranscodeOptions) error
	ProbeSegmentDurationsFn func(ctx context.Context, segmentPath string) ([]ports.SegmentInfo, error)
}

func (m *MockTranscoder) IsAvailable() error {
	if m.IsAvailableFn != nil {
		return m.IsAvailableFn()
	}
	return nil
}

func (m *MockTranscoder) TranscodeVideo(ctx context.Context, opts ports.TranscodeOptions, progress ports.ProgressCallback) error {
	if m.TranscodeVideoFn != nil {
		return m.TranscodeVideoFn(ctx, opts, progress)
	}
	return nil
}

func (m *MockTranscoder) TranscodeAudio(ctx context.Context, opts ports.TranscodeOptions, progress ports.ProgressCallback) error {
	if m.TranscodeAudioFn != nil {
		return m.TranscodeAudioFn(ctx, opts, progress)
	}
	return nil
}

func (m *MockTranscoder) ExtractSubtitle(ctx context.Context, opts ports.TranscodeOptions) error {
	if m.ExtractSubtitleFn != nil {
		return m.ExtractSubtitleFn(ctx, opts)
	}
	return nil
}

func (m *MockTranscoder) ProbeSegmentDurations(ctx context.Context, segmentPath string) ([]ports.SegmentInfo, error) {
	if m.ProbeSegmentDurationsFn != nil {
		return m.ProbeSegmentDurationsFn(ctx, segmentPath)
	}
	return []ports.SegmentInfo{}, nil
}

// MockFileSystemService for testing
type MockFileSystemService struct {
	WriteFileFn func(path string, data []byte) error
}

func (m *MockFileSystemService) WalkDir(root string, walkFn filepath.WalkFunc) error {
	return nil
}

func (m *MockFileSystemService) CopyFile(src, dst string) error {
	return nil
}

func (m *MockFileSystemService) DeleteFile(path string) error {
	return nil
}

func (m *MockFileSystemService) CreateDir(path string) error {
	return nil
}

func (m *MockFileSystemService) RemoveEmptyDirs(root string) error {
	return nil
}

func (m *MockFileSystemService) FileExists(path string) bool {
	return true
}

func (m *MockFileSystemService) GetFileSize(path string) (int64, error) {
	return 0, nil
}

func (m *MockFileSystemService) WriteFile(path string, data []byte) error {
	if m.WriteFileFn != nil {
		return m.WriteFileFn(path, data)
	}
	return nil
}

func (m *MockFileSystemService) ReadFile(path string) ([]byte, error) {
	return nil, nil
}

func (m *MockFileSystemService) ReadDir(path string) ([]os.DirEntry, error) {
	return nil, nil
}

func (m *MockFileSystemService) ListFiles(dir, pattern string) ([]string, error) {
	return nil, nil
}

func (m *MockFileSystemService) Rename(oldPath, newPath string) error {
	return nil
}

func (m *MockFileSystemService) CopyFileWithProgress(src, dst string, callback ports.CopyProgressCallback) error {
	return nil
}

func (m *MockFileSystemService) RemoveDir(path string) error {
	return nil
}

// MockJobNotifier for testing
type MockJobNotifier struct{}

func (m *MockJobNotifier) NotifyJobStarted(job *domain.Job) {}
func (m *MockJobNotifier) NotifyJobProgress(jobID int64, jobType string, progress ports.JobProgress) {
}
func (m *MockJobNotifier) NotifyJobCompleted(job *domain.Job)                              {}
func (m *MockJobNotifier) NotifyJobFailed(job *domain.Job, errorMessage string)            {}
func (m *MockJobNotifier) NotifyJobRetrying(job *domain.Job, attempt int, maxAttempts int) {}
func (m *MockJobNotifier) NotifyJobQueued(job *domain.Job)                                 {}

// TestNewTranscodeVideoJobHandler tests the handler constructor
func TestNewTranscodeVideoJobHandler(t *testing.T) {
	cfg := &config.Config{
		Media: config.MediaConfig{
			MediaPath:              "/media",
			TranscodeOutputPattern: "{media_path}/{media_id}/transcoded",
		},
		Transcoding: config.TranscodingConfig{
			SegmentDuration: 4,
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k", AudioBitrate: "96k"},
			},
		},
	}

	useCase := transcode.NewProcessTranscodeUseCase(
		&MockTranscodeRepository{},
		&MockMediaRepository{},
		&MockTranscoder{},
		&MockFileSystemService{},
		&MockJobNotifier{},
		cfg,
	)

	handler := NewTranscodeVideoJobHandler(useCase)

	assert.NotNil(t, handler, "Handler should not be nil")
}

// TestTranscodeVideoJobHandler_Execute_Success tests successful job execution
func TestTranscodeVideoJobHandler_Execute_Success(t *testing.T) {
	cfg := &config.Config{
		Media: config.MediaConfig{
			MediaPath:              "/media",
			TranscodeOutputPattern: "{media_path}/{media_id}/transcoded",
		},
		Transcoding: config.TranscodingConfig{
			SegmentDuration: 4,
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k", AudioBitrate: "96k"},
			},
		},
	}

	mockTranscodeRepo := &MockTranscodeRepository{
		GetFn: func(ctx context.Context, id string) (*domain.Transcode, error) {
			if id == "transcode123" {
				return &domain.Transcode{
					ID:         "transcode123",
					MediaID:    "media123",
					Quality:    "360p",
					TrackType:  domain.TrackTypeVideo,
					TrackIndex: 0,
					Status:     domain.TranscodePending,
				}, nil
			}
			return nil, errors.New("transcode not found")
		},
	}

	mockMediaRepo := &MockMediaRepository{
		GetFn: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			if id == "media123" {
				return &domain.MediaFile{
					ID:       "media123",
					FilePath: "/media/test.mp4",
				}, nil
			}
			return nil, errors.New("media not found")
		},
	}

	mockTranscoder := &MockTranscoder{
		TranscodeVideoFn: func(ctx context.Context, opts ports.TranscodeOptions, progress ports.ProgressCallback) error {
			return nil
		},
		ProbeSegmentDurationsFn: func(ctx context.Context, segmentPath string) ([]ports.SegmentInfo, error) {
			return []ports.SegmentInfo{
				{Number: 0, Duration: 4000},
				{Number: 1, Duration: 4000},
			}, nil
		},
	}

	useCase := transcode.NewProcessTranscodeUseCase(
		mockTranscodeRepo,
		mockMediaRepo,
		mockTranscoder,
		&MockFileSystemService{},
		&MockJobNotifier{},
		cfg,
	)

	handler := NewTranscodeVideoJobHandler(useCase)

	ctx := context.Background()
	payload := TranscodeJobPayload{
		TranscodeID: "transcode123",
	}
	payloadBytes, _ := json.Marshal(payload)

	job := &domain.Job{
		ID:       1,
		Type:     "transcode_video",
		Payload:  payloadBytes,
		Status:   "queued",
		Priority: 0,
	}

	err := handler(ctx, job)

	assert.NoError(t, err, "Handler should execute successfully")
}

// TestTranscodeVideoJobHandler_Execute_InvalidPayload tests handling of invalid payload
func TestTranscodeVideoJobHandler_Execute_InvalidPayload(t *testing.T) {
	cfg := &config.Config{
		Media: config.MediaConfig{
			MediaPath:              "/media",
			TranscodeOutputPattern: "{media_path}/{media_id}/transcoded",
		},
		Transcoding: config.TranscodingConfig{
			SegmentDuration: 4,
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k", AudioBitrate: "96k"},
			},
		},
	}

	useCase := transcode.NewProcessTranscodeUseCase(
		&MockTranscodeRepository{},
		&MockMediaRepository{},
		&MockTranscoder{},
		&MockFileSystemService{},
		&MockJobNotifier{},
		cfg,
	)

	handler := NewTranscodeVideoJobHandler(useCase)

	ctx := context.Background()
	job := &domain.Job{
		ID:       2,
		Type:     "transcode_video",
		Payload:  []byte("invalid json"),
		Status:   "queued",
		Priority: 0,
	}

	err := handler(ctx, job)

	assert.Error(t, err, "Handler should fail with invalid JSON payload")
	assert.Contains(t, err.Error(), "failed to parse transcode job payload")
}

// TestTranscodeVideoJobHandler_Execute_MissingTranscodeID tests handling of missing transcode_id
func TestTranscodeVideoJobHandler_Execute_MissingTranscodeID(t *testing.T) {
	cfg := &config.Config{
		Media: config.MediaConfig{
			MediaPath:              "/media",
			TranscodeOutputPattern: "{media_path}/{media_id}/transcoded",
		},
		Transcoding: config.TranscodingConfig{
			SegmentDuration: 4,
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k", AudioBitrate: "96k"},
			},
		},
	}

	useCase := transcode.NewProcessTranscodeUseCase(
		&MockTranscodeRepository{},
		&MockMediaRepository{},
		&MockTranscoder{},
		&MockFileSystemService{},
		&MockJobNotifier{},
		cfg,
	)

	handler := NewTranscodeVideoJobHandler(useCase)

	ctx := context.Background()
	payload := TranscodeJobPayload{
		TranscodeID: "", // Empty transcode ID
	}
	payloadBytes, _ := json.Marshal(payload)

	job := &domain.Job{
		ID:       3,
		Type:     "transcode_video",
		Payload:  payloadBytes,
		Status:   "queued",
		Priority: 0,
	}

	err := handler(ctx, job)

	assert.Error(t, err, "Handler should fail with missing transcode_id")
	assert.Contains(t, err.Error(), "transcode_id is required")
}

// TestTranscodeVideoJobHandler_Execute_UseCaseError tests error propagation from use case
func TestTranscodeVideoJobHandler_Execute_UseCaseError(t *testing.T) {
	cfg := &config.Config{
		Media: config.MediaConfig{
			MediaPath:              "/media",
			TranscodeOutputPattern: "{media_path}/{media_id}/transcoded",
		},
		Transcoding: config.TranscodingConfig{
			SegmentDuration: 4,
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k", AudioBitrate: "96k"},
			},
		},
	}

	mockTranscodeRepo := &MockTranscodeRepository{
		GetFn: func(ctx context.Context, id string) (*domain.Transcode, error) {
			return nil, errors.New("transcode not found")
		},
	}

	useCase := transcode.NewProcessTranscodeUseCase(
		mockTranscodeRepo,
		&MockMediaRepository{},
		&MockTranscoder{},
		&MockFileSystemService{},
		&MockJobNotifier{},
		cfg,
	)

	handler := NewTranscodeVideoJobHandler(useCase)

	ctx := context.Background()
	payload := TranscodeJobPayload{
		TranscodeID: "nonexistent",
	}
	payloadBytes, _ := json.Marshal(payload)

	job := &domain.Job{
		ID:       4,
		Type:     "transcode_video",
		Payload:  payloadBytes,
		Status:   "queued",
		Priority: 0,
	}

	err := handler(ctx, job)

	assert.Error(t, err, "Handler should propagate use case error")
	assert.Contains(t, err.Error(), "failed to process transcode")
}

// TestTranscodeVideoJobHandler_ImplementsJobHandlerInterface tests that handler matches interface
func TestTranscodeVideoJobHandler_ImplementsJobHandlerInterface(t *testing.T) {
	cfg := &config.Config{
		Media: config.MediaConfig{
			MediaPath:              "/media",
			TranscodeOutputPattern: "{media_path}/{media_id}/transcoded",
		},
		Transcoding: config.TranscodingConfig{
			SegmentDuration: 4,
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k", AudioBitrate: "96k"},
			},
		},
	}

	useCase := transcode.NewProcessTranscodeUseCase(
		&MockTranscodeRepository{},
		&MockMediaRepository{},
		&MockTranscoder{},
		&MockFileSystemService{},
		&MockJobNotifier{},
		cfg,
	)

	handler := NewTranscodeVideoJobHandler(useCase)

	// Verify handler matches the JobHandler function signature
	var _ ports.JobHandler = handler
}
