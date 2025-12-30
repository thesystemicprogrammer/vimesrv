package library

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
)

// MockScanLibraryRepository is a mock implementation of ports.ScanLibraryRepository
type MockScanLibraryRepository struct {
	mock.Mock
}

func (m *MockScanLibraryRepository) Scan(libraryPath string) error {
	args := m.Called(libraryPath)
	return args.Error(0)
}

// TestScanLibraryUseCase_Execute_Success tests successful library scan
func TestScanLibraryUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(MockScanLibraryRepository)

	cfg := config.MediaConfig{
		LibraryPath:      "/media/library",
		TrashPath:        "/media/trash",
		SupportedFormats: []string{".mp4", ".mkv", ".avi"},
	}

	mockRepo.On("Scan", "/media/library").Return(nil)

	uc := NewScanLibraryUseCase(cfg, mockRepo)

	err := uc.Execute()

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestScanLibraryUseCase_Execute_ScanError tests error during scan
func TestScanLibraryUseCase_Execute_ScanError(t *testing.T) {
	mockRepo := new(MockScanLibraryRepository)

	cfg := config.MediaConfig{
		LibraryPath: "/media/library",
	}

	expectedErr := errors.New("scan failed: directory not found")
	mockRepo.On("Scan", "/media/library").Return(expectedErr)

	uc := NewScanLibraryUseCase(cfg, mockRepo)

	err := uc.Execute()

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockRepo.AssertExpectations(t)
}

// TestScanLibraryUseCase_Execute_EmptyLibraryPath tests scan with empty path
func TestScanLibraryUseCase_Execute_EmptyLibraryPath(t *testing.T) {
	mockRepo := new(MockScanLibraryRepository)

	cfg := config.MediaConfig{
		LibraryPath: "",
	}

	mockRepo.On("Scan", "").Return(nil)

	uc := NewScanLibraryUseCase(cfg, mockRepo)

	err := uc.Execute()

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestScanLibraryUseCase_Execute_DifferentPaths tests that correct path is used
func TestScanLibraryUseCase_Execute_DifferentPaths(t *testing.T) {
	testCases := []struct {
		name        string
		libraryPath string
	}{
		{
			name:        "Unix absolute path",
			libraryPath: "/home/user/videos",
		},
		{
			name:        "Windows path",
			libraryPath: "C:\\Videos\\Library",
		},
		{
			name:        "Relative path",
			libraryPath: "./media/library",
		},
		{
			name:        "Path with spaces",
			libraryPath: "/media/my library/videos",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(MockScanLibraryRepository)

			cfg := config.MediaConfig{
				LibraryPath: tc.libraryPath,
			}

			mockRepo.On("Scan", tc.libraryPath).Return(nil)

			uc := NewScanLibraryUseCase(cfg, mockRepo)

			err := uc.Execute()

			require.NoError(t, err)
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestScanLibraryUseCase_Execute_PermissionError tests permission denied error
func TestScanLibraryUseCase_Execute_PermissionError(t *testing.T) {
	mockRepo := new(MockScanLibraryRepository)

	cfg := config.MediaConfig{
		LibraryPath: "/root/restricted",
	}

	expectedErr := errors.New("permission denied")
	mockRepo.On("Scan", "/root/restricted").Return(expectedErr)

	uc := NewScanLibraryUseCase(cfg, mockRepo)

	err := uc.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
	mockRepo.AssertExpectations(t)
}
