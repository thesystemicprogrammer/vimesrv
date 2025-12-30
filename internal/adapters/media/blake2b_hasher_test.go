package media

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlake2bHasher(t *testing.T) {
	hasher := NewBlake2bHasher()
	assert.NotNil(t, hasher)
}

func TestBlake2bHasher_HashFile_SmallFile(t *testing.T) {
	// Create temp file less than 100MB
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "small.txt")

	content := []byte("Hello, World! This is a test file.")
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)

	hasher := NewBlake2bHasher()
	hash, err := hasher.HashFile(filePath)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 128) // BLAKE2b-512 produces 64 bytes = 128 hex chars
}

func TestBlake2bHasher_HashFile_LargeFile(t *testing.T) {
	// Create temp file larger than 100MB
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "large.bin")

	// Create a 150MB file
	file, err := os.Create(filePath)
	require.NoError(t, err)

	// Write 150MB of data
	chunk := make([]byte, 1024*1024) // 1MB chunks
	for i := 0; i < 150; i++ {
		// Fill with varied data to avoid compression tricks
		for j := range chunk {
			chunk[j] = byte(i + j)
		}
		_, err := file.Write(chunk)
		require.NoError(t, err)
	}
	file.Close()

	hasher := NewBlake2bHasher()
	hash, err := hasher.HashFile(filePath)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 128) // BLAKE2b-512 produces 64 bytes = 128 hex chars
}

func TestBlake2bHasher_HashFile_ExactlyThreshold(t *testing.T) {
	// Create temp file exactly 100MB
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "exact100mb.bin")

	file, err := os.Create(filePath)
	require.NoError(t, err)

	// Write exactly 100MB
	chunk := make([]byte, 1024*1024) // 1MB chunks
	for i := 0; i < 100; i++ {
		for j := range chunk {
			chunk[j] = byte(i + j)
		}
		_, err := file.Write(chunk)
		require.NoError(t, err)
	}
	file.Close()

	hasher := NewBlake2bHasher()
	hash, err := hasher.HashFile(filePath)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 128)
}

func TestBlake2bHasher_HashFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "empty.txt")

	err := os.WriteFile(filePath, []byte{}, 0644)
	require.NoError(t, err)

	hasher := NewBlake2bHasher()
	hash, err := hasher.HashFile(filePath)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 128)
}

func TestBlake2bHasher_HashFile_SameContent_SameHash(t *testing.T) {
	tmpDir := t.TempDir()

	content := []byte("Identical content for testing")

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	err := os.WriteFile(file1, content, 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, content, 0644)
	require.NoError(t, err)

	hasher := NewBlake2bHasher()

	hash1, err := hasher.HashFile(file1)
	require.NoError(t, err)

	hash2, err := hasher.HashFile(file2)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "Same content should produce same hash")
}

func TestBlake2bHasher_HashFile_DifferentContent_DifferentHash(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	err := os.WriteFile(file1, []byte("Content A"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("Content B"), 0644)
	require.NoError(t, err)

	hasher := NewBlake2bHasher()

	hash1, err := hasher.HashFile(file1)
	require.NoError(t, err)

	hash2, err := hasher.HashFile(file2)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "Different content should produce different hash")
}

func TestBlake2bHasher_HashFile_FileNotExists(t *testing.T) {
	hasher := NewBlake2bHasher()
	hash, err := hasher.HashFile("/nonexistent/file.txt")

	assert.Error(t, err)
	assert.Empty(t, hash)
	assert.Contains(t, err.Error(), "failed to open file")
}

func TestBlake2bHasher_HashFile_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	hasher := NewBlake2bHasher()
	hash, err := hasher.HashFile(tmpDir)

	assert.Error(t, err)
	assert.Empty(t, hash)
}

func TestBlake2bHasher_HashFile_Deterministic(t *testing.T) {
	// Hash the same file multiple times - should always get same result
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	content := []byte("Deterministic test content")
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)

	hasher := NewBlake2bHasher()

	// Hash multiple times
	hashes := make([]string, 5)
	for i := 0; i < 5; i++ {
		hash, err := hasher.HashFile(filePath)
		require.NoError(t, err)
		hashes[i] = hash
	}

	// All hashes should be identical
	for i := 1; i < len(hashes); i++ {
		assert.Equal(t, hashes[0], hashes[i], "Hash should be deterministic")
	}
}

func TestBlake2bHasher_HashFile_LargeFile_Collision(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	tmpDir := t.TempDir()

	file1Path := filepath.Join(tmpDir, "large1.bin")
	file2Path := filepath.Join(tmpDir, "large2.bin")

	// Create first file: 10MB (A) + 80MB (B) + 10MB (A) = 100MB total
	file1, err := os.Create(file1Path)
	require.NoError(t, err)

	chunkA := make([]byte, 10*1024*1024) // 10MB
	chunkB := make([]byte, 1024*1024)    // 1MB

	for i := range chunkA {
		chunkA[i] = byte(i % 256)
	}
	for i := range chunkB {
		chunkB[i] = byte((i + 100) % 256)
	}

	file1.Write(chunkA)       // First 10MB
	for j := 0; j < 80; j++ { // Middle 80MB
		file1.Write(chunkB)
	}
	file1.Write(chunkA) // Last 10MB
	file1.Close()

	// Create second file: 10MB (A) + 80MB (C) + 10MB (A) = 100MB total
	file2, err := os.Create(file2Path)
	require.NoError(t, err)

	chunkC := make([]byte, 1024*1024) // 1MB
	for i := range chunkC {
		chunkC[i] = byte((i + 200) % 256)
	}

	file2.Write(chunkA)       // First 10MB (same as file1)
	for j := 0; j < 80; j++ { // Middle 80MB (different from file1)
		file2.Write(chunkC)
	}
	file2.Write(chunkA) // Last 10MB (same as file1)
	file2.Close()

	hasher := NewBlake2bHasher()

	hash1, err := hasher.HashFile(file1Path)
	require.NoError(t, err)

	hash2, err := hasher.HashFile(file2Path)
	require.NoError(t, err)

	// Files have same first 10MB and last 10MB and same size (100MB)
	// With partial hashing (>= 100MB), we only hash first 10MB + last 10MB + size
	// So these two files will produce the SAME hash, demonstrating the collision
	assert.Equal(t, hash1, hash2, "Partial hashing limitation: same first/last 10MB and size produces same hash")
}
