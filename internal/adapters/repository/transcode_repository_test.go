package repository

import (
	"context"
	"testing"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

func TestNewTranscodeRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTranscodeRepository(db)

	if repo == nil {
		t.Fatal("Expected repository to be non-nil")
	}
}

func TestTranscodeRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTranscodeRepository(db)
	ctx := context.Background()

	// First create a media file to reference
	mediaRepo := NewMediaRepository(db)
	media := createTestMediaFile()
	err := mediaRepo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	transcode := domain.NewTranscode("test-360p-video", media.ID, "360p", domain.TrackTypeVideo, 0)

	err = repo.Create(ctx, transcode)
	if err != nil {
		t.Fatalf("Failed to create transcode: %v", err)
	}

	// Verify it was created
	retrieved, err := repo.Get(ctx, transcode.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve transcode: %v", err)
	}

	if retrieved.ID != transcode.ID {
		t.Errorf("Expected ID %s, got %s", transcode.ID, retrieved.ID)
	}
	if retrieved.MediaID != transcode.MediaID {
		t.Errorf("Expected MediaID %s, got %s", transcode.MediaID, retrieved.MediaID)
	}
	if retrieved.Quality != transcode.Quality {
		t.Errorf("Expected Quality %s, got %s", transcode.Quality, retrieved.Quality)
	}
	if retrieved.TrackType != transcode.TrackType {
		t.Errorf("Expected TrackType %s, got %s", transcode.TrackType, retrieved.TrackType)
	}
	if retrieved.Status != domain.TranscodePending {
		t.Errorf("Expected Status %s, got %s", domain.TranscodePending, retrieved.Status)
	}
}

func TestTranscodeRepository_GetByMediaID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTranscodeRepository(db)
	mediaRepo := NewMediaRepository(db)
	ctx := context.Background()

	// Create media file
	media := createTestMediaFile()
	err := mediaRepo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	// Create multiple transcodes for this media
	transcodes := []*domain.Transcode{
		domain.NewTranscode("test-360p-video", media.ID, "360p", domain.TrackTypeVideo, 0),
		domain.NewTranscode("test-720p-video", media.ID, "720p", domain.TrackTypeVideo, 0),
		domain.NewTranscode("test-audio-0", media.ID, "", domain.TrackTypeAudio, 0),
		domain.NewTranscode("test-audio-1", media.ID, "", domain.TrackTypeAudio, 1),
		domain.NewTranscode("test-subtitle-0", media.ID, "", domain.TrackTypeSubtitle, 0),
	}

	for _, transcode := range transcodes {
		err = repo.Create(ctx, transcode)
		if err != nil {
			t.Fatalf("Failed to create transcode: %v", err)
		}
	}

	// Retrieve all transcodes for this media
	retrieved, err := repo.GetByMediaID(ctx, media.ID)
	if err != nil {
		t.Fatalf("Failed to get transcodes by media ID: %v", err)
	}

	if len(retrieved) != len(transcodes) {
		t.Errorf("Expected %d transcodes, got %d", len(transcodes), len(retrieved))
	}

	// Verify ordering (by quality, track_type, track_index)
	if len(retrieved) >= 2 {
		if retrieved[0].Quality != "" && retrieved[1].Quality != "" {
			// Video tracks should be ordered by quality
			if retrieved[0].Quality > retrieved[1].Quality {
				t.Error("Expected transcodes to be ordered by quality")
			}
		}
	}
}

func TestTranscodeRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTranscodeRepository(db)
	mediaRepo := NewMediaRepository(db)
	ctx := context.Background()

	// Create media and transcode
	media := createTestMediaFile()
	err := mediaRepo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	transcode := domain.NewTranscode("test-360p-video", media.ID, "360p", domain.TrackTypeVideo, 0)
	err = repo.Create(ctx, transcode)
	if err != nil {
		t.Fatalf("Failed to create transcode: %v", err)
	}

	// Update status
	err = repo.UpdateStatus(ctx, transcode.ID, domain.TranscodeProcessing)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Verify
	retrieved, err := repo.Get(ctx, transcode.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve transcode: %v", err)
	}

	if retrieved.Status != domain.TranscodeProcessing {
		t.Errorf("Expected Status %s, got %s", domain.TranscodeProcessing, retrieved.Status)
	}
}

func TestTranscodeRepository_MarkProcessing(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTranscodeRepository(db)
	mediaRepo := NewMediaRepository(db)
	ctx := context.Background()

	media := createTestMediaFile()
	err := mediaRepo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	transcode := domain.NewTranscode("test-360p-video", media.ID, "360p", domain.TrackTypeVideo, 0)
	err = repo.Create(ctx, transcode)
	if err != nil {
		t.Fatalf("Failed to create transcode: %v", err)
	}

	err = repo.MarkProcessing(ctx, transcode.ID)
	if err != nil {
		t.Fatalf("Failed to mark as processing: %v", err)
	}

	retrieved, err := repo.Get(ctx, transcode.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve transcode: %v", err)
	}

	if retrieved.Status != domain.TranscodeProcessing {
		t.Errorf("Expected Status %s, got %s", domain.TranscodeProcessing, retrieved.Status)
	}
}

func TestTranscodeRepository_MarkCompleted(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTranscodeRepository(db)
	mediaRepo := NewMediaRepository(db)
	ctx := context.Background()

	media := createTestMediaFile()
	err := mediaRepo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	transcode := domain.NewTranscode("test-360p-video", media.ID, "360p", domain.TrackTypeVideo, 0)
	err = repo.Create(ctx, transcode)
	if err != nil {
		t.Fatalf("Failed to create transcode: %v", err)
	}

	outputPath := "/path/to/output/360p/video"
	err = repo.MarkCompleted(ctx, transcode.ID, outputPath)
	if err != nil {
		t.Fatalf("Failed to mark as completed: %v", err)
	}

	retrieved, err := repo.Get(ctx, transcode.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve transcode: %v", err)
	}

	if retrieved.Status != domain.TranscodeCompleted {
		t.Errorf("Expected Status %s, got %s", domain.TranscodeCompleted, retrieved.Status)
	}
	if retrieved.OutputPath != outputPath {
		t.Errorf("Expected OutputPath %s, got %s", outputPath, retrieved.OutputPath)
	}
}

func TestTranscodeRepository_MarkFailed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTranscodeRepository(db)
	mediaRepo := NewMediaRepository(db)
	ctx := context.Background()

	media := createTestMediaFile()
	err := mediaRepo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	transcode := domain.NewTranscode("test-360p-video", media.ID, "360p", domain.TrackTypeVideo, 0)
	err = repo.Create(ctx, transcode)
	if err != nil {
		t.Fatalf("Failed to create transcode: %v", err)
	}

	err = repo.MarkFailed(ctx, transcode.ID)
	if err != nil {
		t.Fatalf("Failed to mark as failed: %v", err)
	}

	retrieved, err := repo.Get(ctx, transcode.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve transcode: %v", err)
	}

	if retrieved.Status != domain.TranscodeFailed {
		t.Errorf("Expected Status %s, got %s", domain.TranscodeFailed, retrieved.Status)
	}
}

func TestTranscodeRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTranscodeRepository(db)
	mediaRepo := NewMediaRepository(db)
	ctx := context.Background()

	media := createTestMediaFile()
	err := mediaRepo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	transcode := domain.NewTranscode("test-360p-video", media.ID, "360p", domain.TrackTypeVideo, 0)
	err = repo.Create(ctx, transcode)
	if err != nil {
		t.Fatalf("Failed to create transcode: %v", err)
	}

	err = repo.Delete(ctx, transcode.ID)
	if err != nil {
		t.Fatalf("Failed to delete transcode: %v", err)
	}

	// Verify it's deleted
	_, err = repo.Get(ctx, transcode.ID)
	if err == nil {
		t.Error("Expected error when getting deleted transcode, got nil")
	}
}

func TestTranscodeRepository_ListPending(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTranscodeRepository(db)
	mediaRepo := NewMediaRepository(db)
	ctx := context.Background()

	media := createTestMediaFile()
	err := mediaRepo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	// Create transcodes with different statuses
	pendingTranscode1 := domain.NewTranscode("test-360p-video", media.ID, "360p", domain.TrackTypeVideo, 0)
	pendingTranscode2 := domain.NewTranscode("test-720p-video", media.ID, "720p", domain.TrackTypeVideo, 0)
	processingTranscode := domain.NewTranscode("test-audio-0", media.ID, "", domain.TrackTypeAudio, 0)

	err = repo.Create(ctx, pendingTranscode1)
	if err != nil {
		t.Fatalf("Failed to create transcode: %v", err)
	}

	err = repo.Create(ctx, pendingTranscode2)
	if err != nil {
		t.Fatalf("Failed to create transcode: %v", err)
	}

	err = repo.Create(ctx, processingTranscode)
	if err != nil {
		t.Fatalf("Failed to create transcode: %v", err)
	}

	// Mark one as processing
	err = repo.MarkProcessing(ctx, processingTranscode.ID)
	if err != nil {
		t.Fatalf("Failed to mark as processing: %v", err)
	}

	// List pending transcodes
	pending, err := repo.ListPending(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to list pending transcodes: %v", err)
	}

	// Should only get the 2 pending ones
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending transcodes, got %d", len(pending))
	}

	// Verify they're all pending
	for _, transcode := range pending {
		if transcode.Status != domain.TranscodePending {
			t.Errorf("Expected Status %s, got %s", domain.TranscodePending, transcode.Status)
		}
	}
}
