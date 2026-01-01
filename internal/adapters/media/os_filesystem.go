package media

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

const (
	// progressThreshold is the file size threshold for logging copy progress
	progressThreshold = 1 * 1024 * 1024 * 1024 // 1GB

	// progressInterval is the percentage interval for progress logging
	progressInterval = 10 // Log every 10%

	// minProgressLogInterval is the minimum time between progress logs
	minProgressLogInterval = 30 * time.Second
)

// OSFileSystem implements filesystem operations using OS primitives
type OSFileSystem struct{}

// NewOSFileSystem creates a new OSFileSystem instance
func NewOSFileSystem() ports.FileSystemService {
	return &OSFileSystem{}
}

// WalkDir walks the file tree rooted at root, calling walkFn for each file or directory
func (fs *OSFileSystem) WalkDir(root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(root, walkFn)
}

// CopyFile copies a file from src to dst with progress logging for large files
func (fs *OSFileSystem) CopyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	fileSize := srcInfo.Size()

	// Create parent directories for destination
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy with or without progress logging based on file size
	if fileSize >= progressThreshold {
		return fs.copyWithProgress(srcFile, dstFile, fileSize, src, dst)
	}

	// Simple copy for small files
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// copyWithProgress copies a file with progress logging
func (fs *OSFileSystem) copyWithProgress(src io.Reader, dst io.Writer, totalSize int64, srcPath, dstPath string) error {
	buffer := make([]byte, 1024*1024) // 1MB buffer
	var written int64
	lastLogTime := time.Now()
	lastLoggedPercent := 0

	logger.Info().
		Str("src", srcPath).
		Str("dst", dstPath).
		Int64("size_bytes", totalSize).
		Msg("Starting large file copy")

	for {
		nr, readErr := src.Read(buffer)
		if nr > 0 {
			nw, writeErr := dst.Write(buffer[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if writeErr != nil {
				return fmt.Errorf("failed to write to destination: %w", writeErr)
			}
			if nr != nw {
				return fmt.Errorf("short write: wrote %d bytes, expected %d", nw, nr)
			}

			// Check if we should log progress
			currentPercent := int((written * 100) / totalSize)
			timeSinceLastLog := time.Since(lastLogTime)

			if currentPercent >= lastLoggedPercent+progressInterval && timeSinceLastLog >= minProgressLogInterval {
				logger.Info().
					Str("src", srcPath).
					Str("dst", dstPath).
					Int64("written_bytes", written).
					Int64("total_bytes", totalSize).
					Int("percent", currentPercent).
					Msg("File copy progress")

				lastLoggedPercent = currentPercent
				lastLogTime = time.Now()
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("failed to read from source: %w", readErr)
		}
	}

	logger.Info().
		Str("src", srcPath).
		Str("dst", dstPath).
		Int64("size_bytes", written).
		Msg("Large file copy completed")

	return nil
}

// DeleteFile deletes a file
func (fs *OSFileSystem) DeleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// CreateDir creates a directory and all parent directories
func (fs *OSFileSystem) CreateDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

// RemoveEmptyDirs removes empty directories recursively starting from root
func (fs *OSFileSystem) RemoveEmptyDirs(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Skip root directory
		if path == root {
			return nil
		}

		// Try to remove directory (will only succeed if empty)
		if err := os.Remove(path); err != nil {
			// Ignore errors (directory not empty or other issues)
			return nil
		}

		logger.Debug().Str("path", path).Msg("Removed empty directory")
		return nil
	})
}

// FileExists checks if a file exists
func (fs *OSFileSystem) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetFileSize returns the size of a file in bytes
func (fs *OSFileSystem) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}
	return info.Size(), nil
}

// WriteFile writes data to a file
func (fs *OSFileSystem) WriteFile(path string, data []byte) error {
	// Create parent directories for the file
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
