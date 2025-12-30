package media

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewOSFileSystem tests the constructor
func TestNewOSFileSystem(t *testing.T) {
	fs := NewOSFileSystem()
	if fs == nil {
		t.Fatal("Expected filesystem to be non-nil")
	}

	_, ok := fs.(*OSFileSystem)
	if !ok {
		t.Fatal("Expected filesystem to be *OSFileSystem")
	}
}

// TestOSFileSystem_WalkDir tests walking a directory tree
func TestOSFileSystem_WalkDir(t *testing.T) {
	// Create test directory structure
	tempDir := t.TempDir()

	// Create subdirectories and files
	dirs := []string{
		"dir1",
		"dir1/subdir1",
		"dir2",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	files := []string{
		"file1.txt",
		"dir1/file2.txt",
		"dir1/subdir1/file3.txt",
		"dir2/file4.txt",
	}
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	// Walk directory and collect paths
	fs := NewOSFileSystem()
	var foundPaths []string
	err := fs.WalkDir(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Store relative path for easier verification
		relPath, _ := filepath.Rel(tempDir, path)
		if relPath != "." {
			foundPaths = append(foundPaths, relPath)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}

	// Verify all directories and files were found
	expectedItems := append(dirs, files...)
	if len(foundPaths) != len(expectedItems) {
		t.Errorf("Expected %d items, found %d", len(expectedItems), len(foundPaths))
	}

	// Check that all expected items exist in found paths
	foundMap := make(map[string]bool)
	for _, path := range foundPaths {
		foundMap[path] = true
	}

	for _, expected := range expectedItems {
		if !foundMap[expected] {
			t.Errorf("Expected to find '%s', but it was not in walked paths", expected)
		}
	}
}

// TestOSFileSystem_WalkDir_EmptyDirectory tests walking an empty directory
func TestOSFileSystem_WalkDir_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	fs := NewOSFileSystem()
	var foundPaths []string
	err := fs.WalkDir(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		foundPaths = append(foundPaths, path)
		return nil
	})

	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}

	// Should only find the root directory itself
	if len(foundPaths) != 1 {
		t.Errorf("Expected 1 path (root), found %d", len(foundPaths))
	}
}

// TestOSFileSystem_WalkDir_NonExistentDirectory tests walking a non-existent directory
func TestOSFileSystem_WalkDir_NonExistentDirectory(t *testing.T) {
	fs := NewOSFileSystem()
	err := fs.WalkDir("/path/that/does/not/exist", func(path string, info os.FileInfo, err error) error {
		return err
	})

	if err == nil {
		t.Error("Expected error when walking non-existent directory, got nil")
	}
}

// TestOSFileSystem_CopyFile_SmallFile tests copying a small file
func TestOSFileSystem_CopyFile_SmallFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create source file (100 bytes - well below 1GB threshold)
	srcPath := filepath.Join(tempDir, "source.txt")
	content := []byte("This is a test file with some content for testing file copy functionality.")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	dstPath := filepath.Join(tempDir, "subdir", "destination.txt")
	fs := NewOSFileSystem()
	err := fs.CopyFile(srcPath, dstPath)

	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination file exists and has same content
	copiedContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copiedContent) != string(content) {
		t.Errorf("Copied content doesn't match. Expected: %s, Got: %s", content, copiedContent)
	}

	// Verify file size
	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)
	if srcInfo.Size() != dstInfo.Size() {
		t.Errorf("File sizes don't match. Src: %d, Dst: %d", srcInfo.Size(), dstInfo.Size())
	}
}

// TestOSFileSystem_CopyFile_LargeFile tests copying a large file (with progress logging)
func TestOSFileSystem_CopyFile_LargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	tempDir := t.TempDir()

	// Create a large file (10MB - enough to test the logic, but not 1GB to keep test fast)
	// Note: The actual progress logging threshold is 1GB, so this won't trigger progress logs
	// but we can test that large files copy correctly
	srcPath := filepath.Join(tempDir, "large_source.bin")
	fileSize := int64(10 * 1024 * 1024) // 10MB

	// Create large file efficiently
	file, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}
	if err := file.Truncate(fileSize); err != nil {
		file.Close()
		t.Fatalf("Failed to truncate file: %v", err)
	}
	file.Close()

	// Copy file
	dstPath := filepath.Join(tempDir, "large_destination.bin")
	fs := NewOSFileSystem()
	err = fs.CopyFile(srcPath, dstPath)

	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination file exists and has same size
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		t.Fatalf("Failed to stat source file: %v", err)
	}

	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("Failed to stat destination file: %v", err)
	}

	if srcInfo.Size() != dstInfo.Size() {
		t.Errorf("File sizes don't match. Src: %d, Dst: %d", srcInfo.Size(), dstInfo.Size())
	}
}

// TestOSFileSystem_CopyFile_SourceNotExists tests copying from non-existent source
func TestOSFileSystem_CopyFile_SourceNotExists(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "non_existent.txt")
	dstPath := filepath.Join(tempDir, "destination.txt")

	fs := NewOSFileSystem()
	err := fs.CopyFile(srcPath, dstPath)

	if err == nil {
		t.Error("Expected error when copying from non-existent source, got nil")
	}
}

// TestOSFileSystem_CopyFile_CreatesParentDirectories tests that parent dirs are created
func TestOSFileSystem_CopyFile_CreatesParentDirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(srcPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy to nested destination (parent dirs don't exist)
	dstPath := filepath.Join(tempDir, "level1", "level2", "level3", "destination.txt")
	fs := NewOSFileSystem()
	err := fs.CopyFile(srcPath, dstPath)

	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination file exists
	if _, err := os.Stat(dstPath); err != nil {
		t.Errorf("Destination file was not created: %v", err)
	}
}

// TestOSFileSystem_DeleteFile tests deleting a file
func TestOSFileSystem_DeleteFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create file to delete
	filePath := filepath.Join(tempDir, "to_delete.txt")
	if err := os.WriteFile(filePath, []byte("delete me"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("File should exist before deletion: %v", err)
	}

	// Delete file
	fs := NewOSFileSystem()
	err := fs.DeleteFile(filePath)

	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	// Verify file no longer exists
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("File should not exist after deletion")
	}
}

// TestOSFileSystem_DeleteFile_NonExistent tests deleting a non-existent file
func TestOSFileSystem_DeleteFile_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "non_existent.txt")

	fs := NewOSFileSystem()
	err := fs.DeleteFile(filePath)

	if err == nil {
		t.Error("Expected error when deleting non-existent file, got nil")
	}
}

// TestOSFileSystem_CreateDir tests creating a directory
func TestOSFileSystem_CreateDir(t *testing.T) {
	tempDir := t.TempDir()
	dirPath := filepath.Join(tempDir, "new_directory")

	fs := NewOSFileSystem()
	err := fs.CreateDir(dirPath)

	if err != nil {
		t.Fatalf("CreateDir failed: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("Directory should exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("Created path should be a directory")
	}
}

// TestOSFileSystem_CreateDir_Nested tests creating nested directories
func TestOSFileSystem_CreateDir_Nested(t *testing.T) {
	tempDir := t.TempDir()
	dirPath := filepath.Join(tempDir, "level1", "level2", "level3")

	fs := NewOSFileSystem()
	err := fs.CreateDir(dirPath)

	if err != nil {
		t.Fatalf("CreateDir failed: %v", err)
	}

	// Verify all levels exist
	info, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("Directory should exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("Created path should be a directory")
	}
}

// TestOSFileSystem_CreateDir_AlreadyExists tests creating a directory that already exists
func TestOSFileSystem_CreateDir_AlreadyExists(t *testing.T) {
	tempDir := t.TempDir()
	dirPath := filepath.Join(tempDir, "existing_dir")

	// Create directory first
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create initial directory: %v", err)
	}

	// Try to create it again
	fs := NewOSFileSystem()
	err := fs.CreateDir(dirPath)

	// Should not error (MkdirAll is idempotent)
	if err != nil {
		t.Errorf("CreateDir should not error on existing directory: %v", err)
	}
}

// TestOSFileSystem_RemoveEmptyDirs tests removing empty directories
func TestOSFileSystem_RemoveEmptyDirs(t *testing.T) {
	tempDir := t.TempDir()

	// Create directory structure with some empty dirs
	dirs := []string{
		"empty1",
		"empty2",
		"nonempty",
		"nonempty/subdir",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Add a file to make one directory non-empty
	filePath := filepath.Join(tempDir, "nonempty", "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Remove empty directories
	fs := NewOSFileSystem()
	err := fs.RemoveEmptyDirs(tempDir)

	if err != nil {
		t.Fatalf("RemoveEmptyDirs failed: %v", err)
	}

	// Verify empty directories were removed
	if _, err := os.Stat(filepath.Join(tempDir, "empty1")); !os.IsNotExist(err) {
		t.Error("empty1 should have been removed")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "empty2")); !os.IsNotExist(err) {
		t.Error("empty2 should have been removed")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "nonempty", "subdir")); !os.IsNotExist(err) {
		t.Error("nonempty/subdir should have been removed")
	}

	// Verify non-empty directory still exists
	if _, err := os.Stat(filepath.Join(tempDir, "nonempty")); err != nil {
		t.Error("nonempty directory should still exist")
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Error("file in nonempty directory should still exist")
	}

	// Verify root directory still exists
	if _, err := os.Stat(tempDir); err != nil {
		t.Error("root directory should still exist")
	}
}

// TestOSFileSystem_RemoveEmptyDirs_AllEmpty tests removing all empty directories
func TestOSFileSystem_RemoveEmptyDirs_AllEmpty(t *testing.T) {
	tempDir := t.TempDir()

	// Create only empty directories at the same level (no nesting)
	// Note: Nested empty directories require multiple passes to fully remove
	// because filepath.Walk visits parents before children
	dirs := []string{
		"empty1",
		"empty2",
		"empty3",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Remove empty directories
	fs := NewOSFileSystem()
	err := fs.RemoveEmptyDirs(tempDir)

	if err != nil {
		t.Fatalf("RemoveEmptyDirs failed: %v", err)
	}

	// Verify all subdirectories were removed
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected all subdirectories to be removed, found %d entries", len(entries))
	}
}

// TestOSFileSystem_FileExists tests checking if a file exists
func TestOSFileSystem_FileExists(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file
	filePath := filepath.Join(tempDir, "exists.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	fs := NewOSFileSystem()

	// Test existing file
	if !fs.FileExists(filePath) {
		t.Error("FileExists should return true for existing file")
	}

	// Test non-existent file
	nonExistentPath := filepath.Join(tempDir, "does_not_exist.txt")
	if fs.FileExists(nonExistentPath) {
		t.Error("FileExists should return false for non-existent file")
	}
}

// TestOSFileSystem_FileExists_Directory tests FileExists with a directory
func TestOSFileSystem_FileExists_Directory(t *testing.T) {
	tempDir := t.TempDir()

	// Create a subdirectory
	dirPath := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	fs := NewOSFileSystem()

	// FileExists should return true for directories too (os.Stat works on dirs)
	if !fs.FileExists(dirPath) {
		t.Error("FileExists should return true for existing directory")
	}
}

// TestOSFileSystem_GetFileSize tests getting file size
func TestOSFileSystem_GetFileSize(t *testing.T) {
	tempDir := t.TempDir()

	// Create files with known sizes
	testCases := []struct {
		name    string
		content string
		size    int64
	}{
		{"empty.txt", "", 0},
		{"small.txt", "Hello", 5},
		{"medium.txt", "This is a test file with more content", 37},
	}

	fs := NewOSFileSystem()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tc.name)
			if err := os.WriteFile(filePath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			size, err := fs.GetFileSize(filePath)
			if err != nil {
				t.Fatalf("GetFileSize failed: %v", err)
			}

			if size != tc.size {
				t.Errorf("Expected size %d, got %d", tc.size, size)
			}
		})
	}
}

// TestOSFileSystem_GetFileSize_NonExistent tests getting size of non-existent file
func TestOSFileSystem_GetFileSize_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "non_existent.txt")

	fs := NewOSFileSystem()
	_, err := fs.GetFileSize(filePath)

	if err == nil {
		t.Error("Expected error when getting size of non-existent file, got nil")
	}
}

// TestOSFileSystem_GetFileSize_Directory tests getting size of a directory
func TestOSFileSystem_GetFileSize_Directory(t *testing.T) {
	tempDir := t.TempDir()

	// Create a subdirectory
	dirPath := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	fs := NewOSFileSystem()
	size, err := fs.GetFileSize(dirPath)

	// Should not error (os.Stat works on directories)
	if err != nil {
		t.Errorf("GetFileSize should not error on directory: %v", err)
	}

	// Directory size varies by OS, but should be non-negative
	if size < 0 {
		t.Errorf("Directory size should be non-negative, got %d", size)
	}
}
