package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// Mock implementations
type MockFileHasher struct {
	mock.Mock
}

func (m *MockFileHasher) HashFile(filePath string) (string, error) {
	args := m.Called(filePath)
	return args.String(0), args.Error(1)
}

type MockFFProbeService struct {
	mock.Mock
}

func (m *MockFFProbeService) IsAvailable() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockFFProbeService) ValidateVideo(filePath string) (bool, error) {
	args := m.Called(filePath)
	return args.Bool(0), args.Error(1)
}

func (m *MockFFProbeService) ExtractMetadata(filePath string) (*ports.VideoMetadata, error) {
	args := m.Called(filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.VideoMetadata), args.Error(1)
}

func (m *MockFFProbeService) GetAudioStreams(filePath string) ([]*ports.AudioStreamInfo, error) {
	args := m.Called(filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ports.AudioStreamInfo), args.Error(1)
}

func (m *MockFFProbeService) GetSubtitleStreams(filePath string) ([]*ports.SubtitleStreamInfo, error) {
	args := m.Called(filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ports.SubtitleStreamInfo), args.Error(1)
}

type MockFileSystemService struct {
	mock.Mock
}

func (m *MockFileSystemService) WalkDir(root string, walkFn filepath.WalkFunc) error {
	args := m.Called(root, walkFn)
	return args.Error(0)
}

func (m *MockFileSystemService) CopyFile(src, dst string) error {
	args := m.Called(src, dst)
	return args.Error(0)
}

func (m *MockFileSystemService) DeleteFile(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockFileSystemService) CreateDir(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockFileSystemService) RemoveEmptyDirs(root string) error {
	args := m.Called(root)
	return args.Error(0)
}

func (m *MockFileSystemService) FileExists(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

func (m *MockFileSystemService) GetFileSize(path string) (int64, error) {
	args := m.Called(path)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFileSystemService) WriteFile(path string, data []byte) error {
	args := m.Called(path, data)
	return args.Error(0)
}

func (m *MockFileSystemService) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileSystemService) ReadDir(path string) ([]os.DirEntry, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]os.DirEntry), args.Error(1)
}

func (m *MockFileSystemService) ListFiles(dir, pattern string) ([]string, error) {
	args := m.Called(dir, pattern)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockFileSystemService) Rename(oldPath, newPath string) error {
	args := m.Called(oldPath, newPath)
	return args.Error(0)
}

func (m *MockFileSystemService) CopyFileWithProgress(src, dst string, callback ports.CopyProgressCallback) error {
	args := m.Called(src, dst, callback)
	return args.Error(0)
}

func (m *MockFileSystemService) RemoveDir(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

type MockMediaRepository struct {
	mock.Mock
}

func (m *MockMediaRepository) Create(ctx context.Context, media *domain.MediaFile) error {
	args := m.Called(ctx, media)
	return args.Error(0)
}

func (m *MockMediaRepository) FindByFingerprint(ctx context.Context, fingerprint string) (*domain.MediaFile, error) {
	args := m.Called(ctx, fingerprint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MediaFile), args.Error(1)
}

func (m *MockMediaRepository) ExistsByFingerprint(ctx context.Context, fingerprint string) (bool, error) {
	args := m.Called(ctx, fingerprint)
	return args.Bool(0), args.Error(1)
}

func (m *MockMediaRepository) Update(ctx context.Context, media *domain.MediaFile) error {
	args := m.Called(ctx, media)
	return args.Error(0)
}

func (m *MockMediaRepository) Get(ctx context.Context, id string) (*domain.MediaFile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MediaFile), args.Error(1)
}

func (m *MockMediaRepository) List(ctx context.Context, page, perPage int) ([]*domain.MediaFile, int, error) {
	args := m.Called(ctx, page, perPage)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.MediaFile), args.Int(1), args.Error(2)
}

func (m *MockMediaRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMediaRepository) FindByEpisodeMetadataIDs(ctx context.Context, episodeMetadataIDs []int64) ([]*domain.MediaFile, error) {
	args := m.Called(ctx, episodeMetadataIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.MediaFile), args.Error(1)
}

// Helper to create test config
func testConfig() config.MediaConfig {
	return config.MediaConfig{
		LibraryPath:           "/library",
		MediaPath:             "/library/media",
		StagingPath:           "/library/staging",
		TrashPath:             "/library/trash",
		SupportedFormats:      []string{".mp4", ".mkv", ".avi"},
		FFProbeTimeoutSeconds: 30,
	}
}

// TestScanLibraryUseCase_Execute_StagingPathNotExists tests when staging path doesn't exist
func TestScanLibraryUseCase_Execute_StagingPathNotExists(t *testing.T) {
	mockHasher := new(MockFileHasher)
	mockFFProbe := new(MockFFProbeService)
	mockFS := new(MockFileSystemService)
	mockRepo := new(MockMediaRepository)

	cfg := testConfig()

	mockFS.On("FileExists", cfg.StagingPath).Return(false)

	uc := NewScanLibraryUseCase(cfg, mockHasher, mockFFProbe, mockFS, mockRepo, nil, nil)

	err := uc.Execute(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "staging path does not exist")
	mockFS.AssertExpectations(t)
}

// TestScanLibraryUseCase_Execute_EmptyDirectory tests scanning empty directory
func TestScanLibraryUseCase_Execute_EmptyDirectory(t *testing.T) {
	mockHasher := new(MockFileHasher)
	mockFFProbe := new(MockFFProbeService)
	mockFS := new(MockFileSystemService)
	mockRepo := new(MockMediaRepository)

	cfg := testConfig()

	mockFS.On("FileExists", cfg.StagingPath).Return(true)
	mockFS.On("WalkDir", cfg.StagingPath, mock.Anything).Return(nil)
	// RemoveEmptyDirs is NOT called when there are no files to process

	uc := NewScanLibraryUseCase(cfg, mockHasher, mockFFProbe, mockFS, mockRepo, nil, nil)

	err := uc.Execute(context.Background())

	require.NoError(t, err)
	mockFS.AssertExpectations(t)
}

// TestScanLibraryUseCase_Execute_UnsupportedFormat tests skipping unsupported file formats
func TestScanLibraryUseCase_Execute_UnsupportedFormat(t *testing.T) {
	mockHasher := new(MockFileHasher)
	mockFFProbe := new(MockFFProbeService)
	mockFS := new(MockFileSystemService)
	mockRepo := new(MockMediaRepository)

	cfg := testConfig()
	ctx := context.Background()

	mockFS.On("FileExists", cfg.StagingPath).Return(true)

	// Simulate WalkDir finding an unsupported file
	mockFS.On("WalkDir", cfg.StagingPath, mock.Anything).Run(func(args mock.Arguments) {
		walkFn := args.Get(1).(filepath.WalkFunc)
		// Call with .txt file (unsupported)
		walkFn("/library/staging/document.txt", &mockFileInfo{name: "document.txt", isDir: false}, nil)
	}).Return(nil)

	// RemoveEmptyDirs is NOT called when there are no supported files to process

	uc := NewScanLibraryUseCase(cfg, mockHasher, mockFFProbe, mockFS, mockRepo, nil, nil)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockFS.AssertExpectations(t)
	// Should not call ffprobe, hash, or repo for unsupported format
	mockFFProbe.AssertNotCalled(t, "ValidateVideo")
	mockHasher.AssertNotCalled(t, "HashFile")
}

// TestScanLibraryUseCase_Execute_InvalidVideo tests handling invalid video file
func TestScanLibraryUseCase_Execute_InvalidVideo(t *testing.T) {
	mockHasher := new(MockFileHasher)
	mockFFProbe := new(MockFFProbeService)
	mockFS := new(MockFileSystemService)
	mockRepo := new(MockMediaRepository)

	cfg := testConfig()
	ctx := context.Background()
	filePath := "/library/staging/corrupted.mp4"

	mockFS.On("FileExists", cfg.StagingPath).Return(true)

	mockFS.On("WalkDir", cfg.StagingPath, mock.Anything).Run(func(args mock.Arguments) {
		walkFn := args.Get(1).(filepath.WalkFunc)
		walkFn(filePath, &mockFileInfo{name: "corrupted.mp4", isDir: false}, nil)
	}).Return(nil)

	mockFFProbe.On("ValidateVideo", filePath).Return(false, nil)
	mockFS.On("DeleteFile", filePath).Return(nil)
	mockFS.On("RemoveEmptyDirs", cfg.StagingPath).Return(nil)

	uc := NewScanLibraryUseCase(cfg, mockHasher, mockFFProbe, mockFS, mockRepo, nil, nil)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockFFProbe.AssertExpectations(t)
	mockFS.AssertExpectations(t)
	// Should not proceed to hashing
	mockHasher.AssertNotCalled(t, "HashFile")
}

// TestScanLibraryUseCase_Execute_DuplicateFile tests duplicate detection
func TestScanLibraryUseCase_Execute_DuplicateFile(t *testing.T) {
	mockHasher := new(MockFileHasher)
	mockFFProbe := new(MockFFProbeService)
	mockFS := new(MockFileSystemService)
	mockRepo := new(MockMediaRepository)

	cfg := testConfig()
	ctx := context.Background()
	filePath := "/library/staging/video.mp4"
	fingerprint := "abc123def456"

	mockFS.On("FileExists", cfg.StagingPath).Return(true)

	mockFS.On("WalkDir", cfg.StagingPath, mock.Anything).Run(func(args mock.Arguments) {
		walkFn := args.Get(1).(filepath.WalkFunc)
		walkFn(filePath, &mockFileInfo{name: "video.mp4", isDir: false}, nil)
	}).Return(nil)

	mockFFProbe.On("ValidateVideo", filePath).Return(true, nil)
	mockHasher.On("HashFile", filePath).Return(fingerprint, nil)
	mockRepo.On("ExistsByFingerprint", ctx, fingerprint).Return(true, nil)
	mockFS.On("DeleteFile", filePath).Return(nil)
	mockFS.On("RemoveEmptyDirs", cfg.StagingPath).Return(nil)

	uc := NewScanLibraryUseCase(cfg, mockHasher, mockFFProbe, mockFS, mockRepo, nil, nil)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockFFProbe.AssertExpectations(t)
	mockHasher.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockFS.AssertExpectations(t)
	// Should not call ExtractMetadata or Create for duplicate
	mockFFProbe.AssertNotCalled(t, "ExtractMetadata")
	mockRepo.AssertNotCalled(t, "Create")
}

// TestScanLibraryUseCase_Execute_SuccessfulImport tests successful file import
func TestScanLibraryUseCase_Execute_SuccessfulImport(t *testing.T) {
	mockHasher := new(MockFileHasher)
	mockFFProbe := new(MockFFProbeService)
	mockFS := new(MockFileSystemService)
	mockRepo := new(MockMediaRepository)

	cfg := testConfig()
	ctx := context.Background()
	filePath := "/library/staging/video.mp4"
	// Fingerprint must be at least 32 chars for DeriveIDFromFingerprint to produce a 36-char UUID
	fingerprint := "abc123def456789012345678901234567890"

	metadata := &ports.VideoMetadata{
		Duration:          120,
		FileSize:          1024000,
		Format:            "mp4",
		VideoCodec:        "h264",
		AudioCodecs:       []string{"aac"},
		Resolution:        "1920x1080",
		Width:             1920,
		Height:            1080,
		Bitrate:           5000000,
		AudioTracks:       1,
		SubtitleTracks:    0,
		SubtitleLanguages: []string{},
	}

	mockFS.On("FileExists", cfg.StagingPath).Return(true)

	mockFS.On("WalkDir", cfg.StagingPath, mock.Anything).Run(func(args mock.Arguments) {
		walkFn := args.Get(1).(filepath.WalkFunc)
		walkFn(filePath, &mockFileInfo{name: "video.mp4", isDir: false}, nil)
	}).Return(nil)

	mockFFProbe.On("ValidateVideo", filePath).Return(true, nil)
	mockHasher.On("HashFile", filePath).Return(fingerprint, nil)
	mockRepo.On("ExistsByFingerprint", ctx, fingerprint).Return(false, nil)
	mockFFProbe.On("ExtractMetadata", filePath).Return(metadata, nil)

	// Expect CreateDir with UUID-based path (any valid UUID pattern)
	mockFS.On("CreateDir", mock.MatchedBy(func(path string) bool {
		// Path should be /library/media/{uuid}
		return filepath.Dir(path) == cfg.MediaPath && len(filepath.Base(path)) == 36
	})).Return(nil)

	// Expect CopyFileWithProgress with UUID-based path
	mockFS.On("CopyFileWithProgress", filePath, mock.MatchedBy(func(path string) bool {
		// Path should be /library/media/{uuid}/video.mp4
		return filepath.Base(path) == "video.mp4" &&
			filepath.Dir(filepath.Dir(path)) == cfg.MediaPath &&
			len(filepath.Base(filepath.Dir(path))) == 36
	}), mock.Anything).Return(nil)

	mockRepo.On("Create", ctx, mock.MatchedBy(func(m *domain.MediaFile) bool {
		return m.Fingerprint == fingerprint &&
			m.OriginalFilename == "video.mp4" &&
			m.Duration == 120 &&
			m.FileSize == 1024000 &&
			len(m.ID) == 36 // UUID length
	})).Return(nil)
	mockFS.On("DeleteFile", filePath).Return(nil)
	mockFS.On("RemoveEmptyDirs", cfg.StagingPath).Return(nil)

	uc := NewScanLibraryUseCase(cfg, mockHasher, mockFFProbe, mockFS, mockRepo, nil, nil)

	err := uc.Execute(ctx)

	require.NoError(t, err)
	mockFFProbe.AssertExpectations(t)
	mockHasher.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockFS.AssertExpectations(t)
}

// TestScanLibraryUseCase_Execute_DatabaseError_Rollback tests rollback on DB error
func TestScanLibraryUseCase_Execute_DatabaseError_Rollback(t *testing.T) {
	mockHasher := new(MockFileHasher)
	mockFFProbe := new(MockFFProbeService)
	mockFS := new(MockFileSystemService)
	mockRepo := new(MockMediaRepository)

	cfg := testConfig()
	ctx := context.Background()
	filePath := "/library/staging/video.mp4"
	// Fingerprint must be at least 32 chars for DeriveIDFromFingerprint to produce a 36-char UUID
	fingerprint := "abc123def456789012345678901234567890"

	metadata := &ports.VideoMetadata{
		Duration:   120,
		FileSize:   1024000,
		Format:     "mp4",
		VideoCodec: "h264",
	}

	mockFS.On("FileExists", cfg.StagingPath).Return(true)

	mockFS.On("WalkDir", cfg.StagingPath, mock.Anything).Run(func(args mock.Arguments) {
		walkFn := args.Get(1).(filepath.WalkFunc)
		walkFn(filePath, &mockFileInfo{name: "video.mp4", isDir: false}, nil)
	}).Return(nil)

	mockFFProbe.On("ValidateVideo", filePath).Return(true, nil)
	mockHasher.On("HashFile", filePath).Return(fingerprint, nil)
	mockRepo.On("ExistsByFingerprint", ctx, fingerprint).Return(false, nil)
	mockFFProbe.On("ExtractMetadata", filePath).Return(metadata, nil)

	// Expect CreateDir with UUID-based path
	mockFS.On("CreateDir", mock.MatchedBy(func(path string) bool {
		return filepath.Dir(path) == cfg.MediaPath && len(filepath.Base(path)) == 36
	})).Return(nil)

	// Expect CopyFileWithProgress with UUID-based path
	mockFS.On("CopyFileWithProgress", filePath, mock.MatchedBy(func(path string) bool {
		return filepath.Base(path) == "video.mp4" &&
			filepath.Dir(filepath.Dir(path)) == cfg.MediaPath &&
			len(filepath.Base(filepath.Dir(path))) == 36
	}), mock.Anything).Return(nil)

	// Database insert fails
	dbError := errors.New("database connection failed")
	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.MediaFile")).Return(dbError)

	// Should rollback by deleting copied file (with UUID-based path)
	mockFS.On("DeleteFile", mock.MatchedBy(func(path string) bool {
		return filepath.Base(path) == "video.mp4" &&
			filepath.Dir(filepath.Dir(path)) == cfg.MediaPath &&
			len(filepath.Base(filepath.Dir(path))) == 36
	})).Return(nil)
	mockFS.On("RemoveEmptyDirs", cfg.StagingPath).Return(nil)

	uc := NewScanLibraryUseCase(cfg, mockHasher, mockFFProbe, mockFS, mockRepo, nil, nil)

	err := uc.Execute(ctx)

	require.NoError(t, err) // Execute itself doesn't fail, but file processing does
	mockRepo.AssertExpectations(t)
	mockFS.AssertExpectations(t)
	// Verify rollback was called
	mockFS.AssertCalled(t, "DeleteFile", mock.MatchedBy(func(path string) bool {
		return filepath.Base(path) == "video.mp4"
	}))
}

// TestScanLibraryUseCase_Execute_ContextCanceled tests context cancellation
func TestScanLibraryUseCase_Execute_ContextCanceled(t *testing.T) {
	mockHasher := new(MockFileHasher)
	mockFFProbe := new(MockFFProbeService)
	mockFS := new(MockFileSystemService)
	mockRepo := new(MockMediaRepository)

	cfg := testConfig()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockFS.On("FileExists", cfg.StagingPath).Return(true)

	mockFS.On("WalkDir", cfg.StagingPath, mock.Anything).Run(func(args mock.Arguments) {
		walkFn := args.Get(1).(filepath.WalkFunc)
		// Simulate finding a file, but context is canceled
		walkFn("/library/staging/video.mp4", &mockFileInfo{name: "video.mp4", isDir: false}, nil)
	}).Return(context.Canceled)

	uc := NewScanLibraryUseCase(cfg, mockHasher, mockFFProbe, mockFS, mockRepo, nil, nil)

	err := uc.Execute(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	mockFS.AssertExpectations(t)
}

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name  string
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 1024 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }
