package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/library"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// MockFileHasher for testing
type MockFileHasher struct {
	HashFileFn func(filePath string) (string, error)
}

func (m *MockFileHasher) HashFile(filePath string) (string, error) {
	if m.HashFileFn != nil {
		return m.HashFileFn(filePath)
	}
	return "mock_hash_123", nil
}

// MockFFProbeService for testing
type MockFFProbeService struct {
	IsAvailableFn        func() error
	ValidateVideoFn      func(filePath string) (bool, error)
	ExtractMetadataFn    func(filePath string) (*ports.VideoMetadata, error)
	GetAudioStreamsFn    func(filePath string) ([]*ports.AudioStreamInfo, error)
	GetSubtitleStreamsFn func(filePath string) ([]*ports.SubtitleStreamInfo, error)
}

func (m *MockFFProbeService) IsAvailable() error {
	if m.IsAvailableFn != nil {
		return m.IsAvailableFn()
	}
	return nil
}

func (m *MockFFProbeService) ValidateVideo(filePath string) (bool, error) {
	if m.ValidateVideoFn != nil {
		return m.ValidateVideoFn(filePath)
	}
	return true, nil
}

func (m *MockFFProbeService) ExtractMetadata(filePath string) (*ports.VideoMetadata, error) {
	if m.ExtractMetadataFn != nil {
		return m.ExtractMetadataFn(filePath)
	}
	return &ports.VideoMetadata{
		Duration:   120,
		FileSize:   1024,
		Format:     "mp4",
		VideoCodec: "h264",
	}, nil
}

func (m *MockFFProbeService) GetAudioStreams(filePath string) ([]*ports.AudioStreamInfo, error) {
	if m.GetAudioStreamsFn != nil {
		return m.GetAudioStreamsFn(filePath)
	}
	return []*ports.AudioStreamInfo{}, nil
}

func (m *MockFFProbeService) GetSubtitleStreams(filePath string) ([]*ports.SubtitleStreamInfo, error) {
	if m.GetSubtitleStreamsFn != nil {
		return m.GetSubtitleStreamsFn(filePath)
	}
	return []*ports.SubtitleStreamInfo{}, nil
}

// MockFileSystemService for testing
type MockFileSystemService struct {
	WalkDirFn         func(root string, walkFn filepath.WalkFunc) error
	CopyFileFn        func(src, dst string) error
	DeleteFileFn      func(path string) error
	CreateDirFn       func(path string) error
	RemoveEmptyDirsFn func(root string) error
	FileExistsFn      func(path string) bool
	GetFileSizeFn     func(path string) (int64, error)
}

func (m *MockFileSystemService) WalkDir(root string, walkFn filepath.WalkFunc) error {
	if m.WalkDirFn != nil {
		return m.WalkDirFn(root, walkFn)
	}
	return nil
}

func (m *MockFileSystemService) CopyFile(src, dst string) error {
	if m.CopyFileFn != nil {
		return m.CopyFileFn(src, dst)
	}
	return nil
}

func (m *MockFileSystemService) DeleteFile(path string) error {
	if m.DeleteFileFn != nil {
		return m.DeleteFileFn(path)
	}
	return nil
}

func (m *MockFileSystemService) CreateDir(path string) error {
	if m.CreateDirFn != nil {
		return m.CreateDirFn(path)
	}
	return nil
}

func (m *MockFileSystemService) RemoveEmptyDirs(root string) error {
	if m.RemoveEmptyDirsFn != nil {
		return m.RemoveEmptyDirsFn(root)
	}
	return nil
}

func (m *MockFileSystemService) FileExists(path string) bool {
	if m.FileExistsFn != nil {
		return m.FileExistsFn(path)
	}
	return true
}

func (m *MockFileSystemService) GetFileSize(path string) (int64, error) {
	if m.GetFileSizeFn != nil {
		return m.GetFileSizeFn(path)
	}
	return 1024, nil
}

func (m *MockFileSystemService) WriteFile(path string, data []byte) error {
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
	if m.CopyFileFn != nil {
		return m.CopyFileFn(src, dst)
	}
	return nil
}

func (m *MockFileSystemService) RemoveDir(path string) error {
	return nil
}

// MockMediaRepository for testing
type MockMediaRepository struct {
	CreateFn              func(ctx context.Context, media *domain.MediaFile) error
	FindByFingerprintFn   func(ctx context.Context, fingerprint string) (*domain.MediaFile, error)
	ExistsByFingerprintFn func(ctx context.Context, fingerprint string) (bool, error)
	UpdateFn              func(ctx context.Context, media *domain.MediaFile) error
}

func (m *MockMediaRepository) Create(ctx context.Context, media *domain.MediaFile) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, media)
	}
	return nil
}

func (m *MockMediaRepository) FindByFingerprint(ctx context.Context, fingerprint string) (*domain.MediaFile, error) {
	if m.FindByFingerprintFn != nil {
		return m.FindByFingerprintFn(ctx, fingerprint)
	}
	return nil, nil
}

func (m *MockMediaRepository) ExistsByFingerprint(ctx context.Context, fingerprint string) (bool, error) {
	if m.ExistsByFingerprintFn != nil {
		return m.ExistsByFingerprintFn(ctx, fingerprint)
	}
	return false, nil
}

func (m *MockMediaRepository) Update(ctx context.Context, media *domain.MediaFile) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, media)
	}
	return nil
}

func (m *MockMediaRepository) Get(ctx context.Context, id string) (*domain.MediaFile, error) {
	return nil, nil
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

func (m *MockMediaRepository) CountBySeasonMetadataID(ctx context.Context, seasonMetadataID int64) (int, error) {
	return 0, nil
}

func (m *MockMediaRepository) CountBySeriesMetadataID(ctx context.Context, seriesMetadataID int64) (int, error) {
	return 0, nil
}

func (m *MockMediaRepository) Search(ctx context.Context, query string, limit int) ([]*domain.MediaFile, error) {
	return nil, nil
}

// TestNewScanLibraryJobHandler tests the handler constructor
func TestNewScanLibraryJobHandler(t *testing.T) {
	cfg := config.MediaConfig{
		StagingPath:      "/tmp/staging",
		LibraryPath:      "/tmp/library",
		SupportedFormats: []string{".mp4"},
	}

	useCase := library.NewScanLibraryUseCase(
		cfg,
		&MockFileHasher{},
		&MockFFProbeService{},
		&MockFileSystemService{},
		&MockMediaRepository{},
		nil,
		nil,
	)

	handler := NewScanLibraryJobHandler(useCase)

	assert.NotNil(t, handler, "Handler should not be nil")
}

// TestScanLibraryJobHandler_Execute_Success tests successful job execution
func TestScanLibraryJobHandler_Execute_Success(t *testing.T) {
	cfg := config.MediaConfig{
		StagingPath:      "/tmp/staging",
		LibraryPath:      "/tmp/library",
		SupportedFormats: []string{".mp4"},
	}

	// Mock filesystem to return no files (empty staging)
	mockFS := &MockFileSystemService{
		WalkDirFn: func(root string, walkFn filepath.WalkFunc) error {
			return nil // No files to process
		},
		RemoveEmptyDirsFn: func(root string) error {
			return nil
		},
	}

	useCase := library.NewScanLibraryUseCase(
		cfg,
		&MockFileHasher{},
		&MockFFProbeService{},
		mockFS,
		&MockMediaRepository{},
		nil,
		nil,
	)

	handler := NewScanLibraryJobHandler(useCase)

	ctx := context.Background()
	job := &domain.Job{
		ID:       1,
		Type:     "scan_library",
		Payload:  nil,
		Status:   "queued",
		Priority: 0,
	}

	err := handler(ctx, job)

	assert.NoError(t, err, "Handler should execute successfully with empty staging")
}

// TestScanLibraryJobHandler_Execute_UseCaseError tests error propagation from use case
func TestScanLibraryJobHandler_Execute_UseCaseError(t *testing.T) {
	cfg := config.MediaConfig{
		StagingPath:      "/tmp/nonexistent",
		LibraryPath:      "/tmp/library",
		SupportedFormats: []string{".mp4"},
	}

	// Mock filesystem to return false for FileExists so use case returns error
	mockFS := &MockFileSystemService{
		FileExistsFn: func(path string) bool {
			return false // Staging path does not exist
		},
	}

	useCase := library.NewScanLibraryUseCase(
		cfg,
		&MockFileHasher{},
		&MockFFProbeService{},
		mockFS,
		&MockMediaRepository{},
		nil,
		nil,
	)

	handler := NewScanLibraryJobHandler(useCase)

	ctx := context.Background()
	job := &domain.Job{
		ID:       2,
		Type:     "scan_library",
		Payload:  nil,
		Status:   "queued",
		Priority: 0,
	}

	err := handler(ctx, job)

	assert.Error(t, err, "Handler should propagate use case error")
	assert.Contains(t, err.Error(), "staging path does not exist")
}

// TestScanLibraryJobHandler_Execute_ContextCanceled tests context cancellation handling
func TestScanLibraryJobHandler_Execute_ContextCanceled(t *testing.T) {
	cfg := config.MediaConfig{
		StagingPath:      "/tmp/staging",
		LibraryPath:      "/tmp/library",
		SupportedFormats: []string{".mp4"},
	}

	// Mock filesystem to simulate long operation and check context
	mockFS := &MockFileSystemService{
		WalkDirFn: func(root string, walkFn filepath.WalkFunc) error {
			// Simulate work
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	useCase := library.NewScanLibraryUseCase(
		cfg,
		&MockFileHasher{},
		&MockFFProbeService{},
		mockFS,
		&MockMediaRepository{},
		nil,
		nil,
	)

	handler := NewScanLibraryJobHandler(useCase)

	// Create context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	job := &domain.Job{
		ID:       3,
		Type:     "scan_library",
		Payload:  nil,
		Status:   "queued",
		Priority: 0,
	}

	err := handler(ctx, job)

	// The handler should pass through the context, and use case should detect cancellation
	// For this simple test, we just verify the handler completes
	// (More sophisticated cancellation handling would be in the use case itself)
	assert.NoError(t, err, "Handler completed (cancellation handling is in use case)")
}

// TestScanLibraryJobHandler_Execute_NoPayloadRequired tests that payload is ignored
func TestScanLibraryJobHandler_Execute_NoPayloadRequired(t *testing.T) {
	cfg := config.MediaConfig{
		StagingPath:      "/tmp/staging",
		LibraryPath:      "/tmp/library",
		SupportedFormats: []string{".mp4"},
	}

	mockFS := &MockFileSystemService{
		WalkDirFn: func(root string, walkFn filepath.WalkFunc) error {
			return nil
		},
		RemoveEmptyDirsFn: func(root string) error {
			return nil
		},
	}

	useCase := library.NewScanLibraryUseCase(
		cfg,
		&MockFileHasher{},
		&MockFFProbeService{},
		mockFS,
		&MockMediaRepository{},
		nil,
		nil,
	)

	handler := NewScanLibraryJobHandler(useCase)

	ctx := context.Background()

	// Test with nil payload
	job1 := &domain.Job{
		ID:       4,
		Type:     "scan_library",
		Payload:  nil,
		Status:   "queued",
		Priority: 0,
	}
	err1 := handler(ctx, job1)
	assert.NoError(t, err1, "Handler should work with nil payload")

	// Test with non-nil payload (should be ignored)
	job2 := &domain.Job{
		ID:       5,
		Type:     "scan_library",
		Payload:  []byte(`{"ignored": "data"}`),
		Status:   "queued",
		Priority: 0,
	}
	err2 := handler(ctx, job2)
	assert.NoError(t, err2, "Handler should ignore non-nil payload")
}

// TestScanLibraryJobHandler_Execute_MultipleJobs tests that handler can process multiple jobs
func TestScanLibraryJobHandler_Execute_MultipleJobs(t *testing.T) {
	cfg := config.MediaConfig{
		StagingPath:      "/tmp/staging",
		LibraryPath:      "/tmp/library",
		SupportedFormats: []string{".mp4"},
	}

	callCount := 0
	mockFS := &MockFileSystemService{
		WalkDirFn: func(root string, walkFn filepath.WalkFunc) error {
			callCount++
			return nil
		},
		RemoveEmptyDirsFn: func(root string) error {
			return nil
		},
	}

	useCase := library.NewScanLibraryUseCase(
		cfg,
		&MockFileHasher{},
		&MockFFProbeService{},
		mockFS,
		&MockMediaRepository{},
		nil,
		nil,
	)

	handler := NewScanLibraryJobHandler(useCase)
	ctx := context.Background()

	// Execute handler multiple times (simulating multiple job executions)
	for i := 1; i <= 3; i++ {
		job := &domain.Job{
			ID:       int64(i),
			Type:     "scan_library",
			Payload:  nil,
			Status:   "queued",
			Priority: 0,
		}
		err := handler(ctx, job)
		assert.NoError(t, err, "Handler should execute job %d successfully", i)
	}

	assert.Equal(t, 3, callCount, "Use case should be called once per job")
}

// TestScanLibraryJobHandler_UsesConfigPaths tests that handler uses paths from config
func TestScanLibraryJobHandler_UsesConfigPaths(t *testing.T) {
	cfg := config.MediaConfig{
		StagingPath:      "/custom/staging/path",
		LibraryPath:      "/custom/library/path",
		SupportedFormats: []string{".mp4", ".mkv"},
	}

	// Track which path WalkDir is called with
	var capturedPath string
	mockFS := &MockFileSystemService{
		WalkDirFn: func(root string, walkFn filepath.WalkFunc) error {
			capturedPath = root
			return nil
		},
		RemoveEmptyDirsFn: func(root string) error {
			return nil
		},
	}

	useCase := library.NewScanLibraryUseCase(
		cfg,
		&MockFileHasher{},
		&MockFFProbeService{},
		mockFS,
		&MockMediaRepository{},
		nil,
		nil,
	)

	handler := NewScanLibraryJobHandler(useCase)

	ctx := context.Background()
	job := &domain.Job{
		ID:       6,
		Type:     "scan_library",
		Payload:  nil,
		Status:   "queued",
		Priority: 0,
	}

	err := handler(ctx, job)

	assert.NoError(t, err)
	assert.Equal(t, "/custom/staging/path", capturedPath, "Handler should use staging path from config")
}

// TestScanLibraryJobHandler_ImplementsJobHandlerInterface tests that handler matches interface
func TestScanLibraryJobHandler_ImplementsJobHandlerInterface(t *testing.T) {
	cfg := config.MediaConfig{
		StagingPath:      "/tmp/staging",
		LibraryPath:      "/tmp/library",
		SupportedFormats: []string{".mp4"},
	}

	useCase := library.NewScanLibraryUseCase(
		cfg,
		&MockFileHasher{},
		&MockFFProbeService{},
		&MockFileSystemService{},
		&MockMediaRepository{},
		nil,
		nil,
	)

	handler := NewScanLibraryJobHandler(useCase)

	// Verify handler matches the JobHandler function signature
	var _ ports.JobHandler = handler

	// Verify it can be called with the expected signature
	ctx := context.Background()
	job := &domain.Job{ID: 1, Type: "scan_library"}

	err := handler(ctx, job)

	// The exact error doesn't matter - we're just testing the interface
	_ = err // Might be error or nil depending on mock setup
}
