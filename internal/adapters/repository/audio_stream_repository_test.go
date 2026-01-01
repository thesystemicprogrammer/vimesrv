package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// TestNewAudioStreamRepository tests the repository constructor
func TestNewAudioStreamRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAudioStreamRepository(db)

	if repo == nil {
		t.Fatal("Expected repository to be non-nil")
	}
}

// TestAudioStreamRepository_Create tests creating a new audio stream
func TestAudioStreamRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAudioStreamRepository(db)
	ctx := context.Background()

	// Create a test media file first (foreign key requirement)
	mediaRepo := NewMediaRepository(db)
	media := createTestMediaFile()
	if err := mediaRepo.Create(ctx, media); err != nil {
		t.Fatalf("Failed to create test media file: %v", err)
	}

	// Create audio stream
	audioStream := domain.NewAudioStream(media.ID, 0, "aac", "eng", 2, "stereo", 48000, "English Audio")

	err := repo.Create(ctx, audioStream)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify ID was assigned
	if audioStream.ID == 0 {
		t.Error("Expected ID to be assigned after Create")
	}
}

// TestAudioStreamRepository_GetByMediaID tests retrieving audio streams by media ID
func TestAudioStreamRepository_GetByMediaID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAudioStreamRepository(db)
	ctx := context.Background()

	// Create a test media file first
	mediaRepo := NewMediaRepository(db)
	media := createTestMediaFile()
	if err := mediaRepo.Create(ctx, media); err != nil {
		t.Fatalf("Failed to create test media file: %v", err)
	}

	// Create multiple audio streams
	stream1 := domain.NewAudioStream(media.ID, 0, "aac", "eng", 2, "stereo", 48000, "English")
	stream2 := domain.NewAudioStream(media.ID, 1, "ac3", "spa", 6, "5.1", 48000, "Spanish")

	if err := repo.Create(ctx, stream1); err != nil {
		t.Fatalf("Failed to create stream1: %v", err)
	}
	if err := repo.Create(ctx, stream2); err != nil {
		t.Fatalf("Failed to create stream2: %v", err)
	}

	// Retrieve streams
	streams, err := repo.GetByMediaID(ctx, media.ID)
	if err != nil {
		t.Fatalf("GetByMediaID failed: %v", err)
	}

	// Verify results
	if len(streams) != 2 {
		t.Fatalf("Expected 2 streams, got %d", len(streams))
	}

	// Verify order (should be by stream_index)
	if streams[0].StreamIndex != 0 {
		t.Errorf("Expected first stream index 0, got %d", streams[0].StreamIndex)
	}
	if streams[1].StreamIndex != 1 {
		t.Errorf("Expected second stream index 1, got %d", streams[1].StreamIndex)
	}

	// Verify fields
	if streams[0].Codec != "aac" {
		t.Errorf("Expected codec 'aac', got '%s'", streams[0].Codec)
	}
	if streams[0].Language != "eng" {
		t.Errorf("Expected language 'eng', got '%s'", streams[0].Language)
	}
	if streams[0].Channels != 2 {
		t.Errorf("Expected channels 2, got %d", streams[0].Channels)
	}
}

// TestAudioStreamRepository_GetByMediaID_NotFound tests querying non-existent media
func TestAudioStreamRepository_GetByMediaID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAudioStreamRepository(db)
	ctx := context.Background()

	nonExistentID := uuid.New().String()
	streams, err := repo.GetByMediaID(ctx, nonExistentID)

	if err != nil {
		t.Fatalf("Expected no error for non-existent media, got: %v", err)
	}

	if len(streams) != 0 {
		t.Errorf("Expected 0 streams for non-existent media, got %d", len(streams))
	}
}

// TestAudioStreamRepository_DeleteByMediaID tests deleting audio streams
func TestAudioStreamRepository_DeleteByMediaID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAudioStreamRepository(db)
	ctx := context.Background()

	// Create a test media file
	mediaRepo := NewMediaRepository(db)
	media := createTestMediaFile()
	if err := mediaRepo.Create(ctx, media); err != nil {
		t.Fatalf("Failed to create test media file: %v", err)
	}

	// Create audio streams
	stream1 := domain.NewAudioStream(media.ID, 0, "aac", "eng", 2, "stereo", 48000, "English")
	stream2 := domain.NewAudioStream(media.ID, 1, "ac3", "spa", 6, "5.1", 48000, "Spanish")

	if err := repo.Create(ctx, stream1); err != nil {
		t.Fatalf("Failed to create stream1: %v", err)
	}
	if err := repo.Create(ctx, stream2); err != nil {
		t.Fatalf("Failed to create stream2: %v", err)
	}

	// Delete all streams for this media
	err := repo.DeleteByMediaID(ctx, media.ID)
	if err != nil {
		t.Fatalf("DeleteByMediaID failed: %v", err)
	}

	// Verify deletion
	streams, err := repo.GetByMediaID(ctx, media.ID)
	if err != nil {
		t.Fatalf("GetByMediaID failed: %v", err)
	}

	if len(streams) != 0 {
		t.Errorf("Expected 0 streams after deletion, got %d", len(streams))
	}
}
