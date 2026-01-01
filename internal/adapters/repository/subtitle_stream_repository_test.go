package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// TestNewSubtitleStreamRepository tests the repository constructor
func TestNewSubtitleStreamRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubtitleStreamRepository(db)

	if repo == nil {
		t.Fatal("Expected repository to be non-nil")
	}
}

// TestSubtitleStreamRepository_Create tests creating a new subtitle stream
func TestSubtitleStreamRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubtitleStreamRepository(db)
	ctx := context.Background()

	// Create a test media file first (foreign key requirement)
	mediaRepo := NewMediaRepository(db)
	media := createTestMediaFile()
	if err := mediaRepo.Create(ctx, media); err != nil {
		t.Fatalf("Failed to create test media file: %v", err)
	}

	// Create subtitle stream
	subtitleStream := domain.NewSubtitleStream(media.ID, 0, "subrip", "eng", "English Subtitles", false)

	err := repo.Create(ctx, subtitleStream)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify ID was assigned
	if subtitleStream.ID == 0 {
		t.Error("Expected ID to be assigned after Create")
	}
}

// TestSubtitleStreamRepository_GetByMediaID tests retrieving subtitle streams by media ID
func TestSubtitleStreamRepository_GetByMediaID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubtitleStreamRepository(db)
	ctx := context.Background()

	// Create a test media file first
	mediaRepo := NewMediaRepository(db)
	media := createTestMediaFile()
	if err := mediaRepo.Create(ctx, media); err != nil {
		t.Fatalf("Failed to create test media file: %v", err)
	}

	// Create multiple subtitle streams
	stream1 := domain.NewSubtitleStream(media.ID, 0, "subrip", "eng", "English", false)
	stream2 := domain.NewSubtitleStream(media.ID, 1, "ass", "spa", "Spanish", false)
	stream3 := domain.NewSubtitleStream(media.ID, 2, "subrip", "eng", "English (Forced)", true)

	if err := repo.Create(ctx, stream1); err != nil {
		t.Fatalf("Failed to create stream1: %v", err)
	}
	if err := repo.Create(ctx, stream2); err != nil {
		t.Fatalf("Failed to create stream2: %v", err)
	}
	if err := repo.Create(ctx, stream3); err != nil {
		t.Fatalf("Failed to create stream3: %v", err)
	}

	// Retrieve streams
	streams, err := repo.GetByMediaID(ctx, media.ID)
	if err != nil {
		t.Fatalf("GetByMediaID failed: %v", err)
	}

	// Verify results
	if len(streams) != 3 {
		t.Fatalf("Expected 3 streams, got %d", len(streams))
	}

	// Verify order (should be by stream_index)
	if streams[0].StreamIndex != 0 {
		t.Errorf("Expected first stream index 0, got %d", streams[0].StreamIndex)
	}
	if streams[1].StreamIndex != 1 {
		t.Errorf("Expected second stream index 1, got %d", streams[1].StreamIndex)
	}
	if streams[2].StreamIndex != 2 {
		t.Errorf("Expected third stream index 2, got %d", streams[2].StreamIndex)
	}

	// Verify fields
	if streams[0].Codec != "subrip" {
		t.Errorf("Expected codec 'subrip', got '%s'", streams[0].Codec)
	}
	if streams[0].Language != "eng" {
		t.Errorf("Expected language 'eng', got '%s'", streams[0].Language)
	}
	if streams[0].Forced != false {
		t.Errorf("Expected forced false, got %v", streams[0].Forced)
	}

	// Verify forced flag
	if streams[2].Forced != true {
		t.Errorf("Expected third stream to be forced, got %v", streams[2].Forced)
	}
}

// TestSubtitleStreamRepository_GetByMediaID_NotFound tests querying non-existent media
func TestSubtitleStreamRepository_GetByMediaID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubtitleStreamRepository(db)
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

// TestSubtitleStreamRepository_DeleteByMediaID tests deleting subtitle streams
func TestSubtitleStreamRepository_DeleteByMediaID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubtitleStreamRepository(db)
	ctx := context.Background()

	// Create a test media file
	mediaRepo := NewMediaRepository(db)
	media := createTestMediaFile()
	if err := mediaRepo.Create(ctx, media); err != nil {
		t.Fatalf("Failed to create test media file: %v", err)
	}

	// Create subtitle streams
	stream1 := domain.NewSubtitleStream(media.ID, 0, "subrip", "eng", "English", false)
	stream2 := domain.NewSubtitleStream(media.ID, 1, "ass", "spa", "Spanish", false)

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
