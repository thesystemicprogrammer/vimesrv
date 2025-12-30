package ports

import "path/filepath"

// FileSystemService provides filesystem operations
type FileSystemService interface {
	// WalkDir walks the file tree rooted at root, calling walkFn for each file or directory
	WalkDir(root string, walkFn filepath.WalkFunc) error

	// CopyFile copies a file from src to dst
	// Creates parent directories if they don't exist
	CopyFile(src, dst string) error

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
}
