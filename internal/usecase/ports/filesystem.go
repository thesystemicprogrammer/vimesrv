package ports

import (
	"os"
	"path/filepath"
)

// CopyProgressCallback reports file copy progress.
// written: bytes written so far
// total: total file size in bytes
// percentComplete: percentage complete (0-100)
type CopyProgressCallback func(written, total int64, percentComplete float64)

// FileSystemService provides filesystem operations
type FileSystemService interface {
	// WalkDir walks the file tree rooted at root, calling walkFn for each file or directory
	WalkDir(root string, walkFn filepath.WalkFunc) error

	// CopyFile copies a file from src to dst
	// Creates parent directories if they don't exist
	CopyFile(src, dst string) error

	// CopyFileWithProgress copies a file from src to dst with progress reporting.
	// Creates parent directories if they don't exist.
	// The callback is called periodically for files >= 100MB.
	// If callback is nil, behaves the same as CopyFile.
	CopyFileWithProgress(src, dst string, callback CopyProgressCallback) error

	// DeleteFile deletes a file
	DeleteFile(path string) error

	// CreateDir creates a directory and all parent directories
	CreateDir(path string) error

	// RemoveEmptyDirs removes empty directories recursively starting from root
	RemoveEmptyDirs(root string) error

	// FileExists checks if a file exists
	FileExists(path string) bool

	// GetFileSize returns the size of a file in bytes
	GetFileSize(path string) (int64, error)

	// WriteFile writes data to a file
	// Creates parent directories if they don't exist
	WriteFile(path string, data []byte) error

	// ReadFile reads the contents of a file
	ReadFile(path string) ([]byte, error)

	// ReadDir reads a directory and returns its entries
	ReadDir(path string) ([]os.DirEntry, error)

	// ListFiles returns files matching a glob pattern within a directory
	ListFiles(dir, pattern string) ([]string, error)

	// Rename renames (moves) a file from oldPath to newPath
	Rename(oldPath, newPath string) error
}
