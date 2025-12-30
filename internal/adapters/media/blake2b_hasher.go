package media

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/blake2b"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

const (
	// hashThreshold is the file size threshold for using partial vs full hash
	hashThreshold = 100 * 1024 * 1024 // 100MB

	// partialHashSize is the amount to read from start and end for large files
	partialHashSize = 10 * 1024 * 1024 // 10MB
)

// Blake2bHasher implements file hashing using BLAKE2b-512
type Blake2bHasher struct{}

// NewBlake2bHasher creates a new Blake2bHasher instance
func NewBlake2bHasher() ports.FileHasher {
	return &Blake2bHasher{}
}

// HashFile generates a BLAKE2b-512 hash for the given file
// For files < 100MB: full file hash
// For files >= 100MB: hash of (first 10MB + last 10MB + file size)
func (h *Blake2bHasher) HashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}
	fileSize := stat.Size()

	// Create BLAKE2b-512 hasher
	hasher, err := blake2b.New512(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create hasher: %w", err)
	}

	// Choose strategy based on file size
	if fileSize < hashThreshold {
		// Full file hash for small files
		if _, err := io.Copy(hasher, file); err != nil {
			return "", fmt.Errorf("failed to hash file: %w", err)
		}
	} else {
		// Partial hash for large files
		if err := h.hashPartial(file, hasher, fileSize); err != nil {
			return "", fmt.Errorf("failed to hash large file: %w", err)
		}
	}

	// Return hexadecimal representation
	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash), nil
}

// hashPartial performs partial hashing: first 10MB + last 10MB + file size
func (h *Blake2bHasher) hashPartial(file *os.File, hasher io.Writer, fileSize int64) error {
	// Read first 10MB
	buffer := make([]byte, partialHashSize)
	n, err := io.ReadFull(file, buffer)
	if err != nil && err != io.ErrUnexpectedEOF {
		return fmt.Errorf("failed to read first chunk: %w", err)
	}
	if _, err := hasher.Write(buffer[:n]); err != nil {
		return fmt.Errorf("failed to write first chunk to hasher: %w", err)
	}

	// Seek to last 10MB
	seekPos := fileSize - partialHashSize
	if seekPos < 0 {
		seekPos = 0
	}
	if _, err := file.Seek(seekPos, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to end chunk: %w", err)
	}

	// Read last 10MB
	n, err = io.ReadFull(file, buffer)
	if err != nil && err != io.ErrUnexpectedEOF {
		return fmt.Errorf("failed to read last chunk: %w", err)
	}
	if _, err := hasher.Write(buffer[:n]); err != nil {
		return fmt.Errorf("failed to write last chunk to hasher: %w", err)
	}

	// Include file size in hash
	sizeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(sizeBytes, uint64(fileSize))
	if _, err := hasher.Write(sizeBytes); err != nil {
		return fmt.Errorf("failed to write file size to hasher: %w", err)
	}

	return nil
}
