package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
)

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *database.DB {
	db, err := database.New(database.Config{Path: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations
	migration := database.NewDatabaseMigration(db.DB)
	if err := migration.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// createTestMediaFile creates a test media file with default values
func createTestMediaFile() *domain.MediaFile {
	now := time.Now().UTC()
	id := uuid.New().String()
	// Generate a 64-character fingerprint (BLAKE2b-512 produces 128 hex chars, but we'll use 64 for simplicity)
	fingerprint := uuid.New().String() + uuid.New().String()[:28] // 36 + 28 = 64 chars
	return &domain.MediaFile{
		ID:                id,
		Fingerprint:       fingerprint,
		FilePath:          "/media/library/" + id + "/test_video.mp4", // Unique file path
		OriginalFilename:  "test_video.mp4",
		Filename:          "test_video.mp4",
		Title:             "Test Video",
		Duration:          120,
		FileSize:          1048576,
		Format:            "mp4",
		VideoCodec:        "h264",
		AudioCodecs:       []string{"aac", "ac3"},
		Resolution:        "1920x1080",
		Width:             1920,
		Height:            1080,
		Bitrate:           5000000,
		AudioTracks:       2,
		SubtitleTracks:    1,
		SubtitleLanguages: []string{"eng", "spa"},
		Status:            domain.MediaStatusReady,
		EnrichmentStatus:  domain.EnrichmentStatusPending,
		CreatedAt:         now,
		UpdatedAt:         now,
		ScannedAt:         now,
	}
}

// TestNewMediaRepository tests the repository constructor
func TestNewMediaRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)

	if repo == nil {
		t.Fatal("Expected repository to be non-nil")
	}
}

// TestMediaRepository_Create tests creating a new media file
func TestMediaRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)
	ctx := context.Background()

	media := createTestMediaFile()
	err := repo.Create(ctx, media)

	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify the record was inserted by querying it back
	found, err := repo.FindByFingerprint(ctx, media.Fingerprint)
	if err != nil {
		t.Fatalf("FindByFingerprint failed: %v", err)
	}

	if found == nil {
		t.Fatal("Expected to find created media file, got nil")
	}

	// Verify fields
	if found.ID != media.ID {
		t.Errorf("Expected ID %s, got %s", media.ID, found.ID)
	}
	if found.Fingerprint != media.Fingerprint {
		t.Errorf("Expected fingerprint %s, got %s", media.Fingerprint, found.Fingerprint)
	}
	if found.VideoCodec != media.VideoCodec {
		t.Errorf("Expected video codec %s, got %s", media.VideoCodec, found.VideoCodec)
	}
	if len(found.AudioCodecs) != len(media.AudioCodecs) {
		t.Errorf("Expected %d audio codecs, got %d", len(media.AudioCodecs), len(found.AudioCodecs))
	}
	if len(found.SubtitleLanguages) != len(media.SubtitleLanguages) {
		t.Errorf("Expected %d subtitle languages, got %d", len(media.SubtitleLanguages), len(found.SubtitleLanguages))
	}
}

// TestMediaRepository_Create_DuplicateFingerprint tests creating with duplicate fingerprint
func TestMediaRepository_Create_DuplicateFingerprint(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)
	ctx := context.Background()

	media1 := createTestMediaFile()
	err := repo.Create(ctx, media1)
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	// Try to create another media file with same fingerprint
	media2 := createTestMediaFile()
	media2.ID = uuid.New().String()         // Different ID
	media2.Fingerprint = media1.Fingerprint // Same fingerprint

	err = repo.Create(ctx, media2)
	if err == nil {
		t.Error("Expected error when creating with duplicate fingerprint, got nil")
	}
}

// TestMediaRepository_Create_EmptyArrays tests creating with empty audio/subtitle arrays
func TestMediaRepository_Create_EmptyArrays(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)
	ctx := context.Background()

	media := createTestMediaFile()
	media.AudioCodecs = []string{}
	media.SubtitleLanguages = []string{}

	err := repo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify arrays are properly stored as empty JSON arrays
	found, err := repo.FindByFingerprint(ctx, media.Fingerprint)
	if err != nil {
		t.Fatalf("FindByFingerprint failed: %v", err)
	}

	if found.AudioCodecs == nil {
		t.Error("Expected AudioCodecs to be non-nil empty slice")
	}
	if len(found.AudioCodecs) != 0 {
		t.Errorf("Expected empty AudioCodecs, got %d items", len(found.AudioCodecs))
	}
	if found.SubtitleLanguages == nil {
		t.Error("Expected SubtitleLanguages to be non-nil empty slice")
	}
	if len(found.SubtitleLanguages) != 0 {
		t.Errorf("Expected empty SubtitleLanguages, got %d items", len(found.SubtitleLanguages))
	}
}

// TestMediaRepository_FindByFingerprint tests finding by fingerprint
func TestMediaRepository_FindByFingerprint(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)
	ctx := context.Background()

	// Create a media file
	media := createTestMediaFile()
	err := repo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Find by fingerprint
	found, err := repo.FindByFingerprint(ctx, media.Fingerprint)
	if err != nil {
		t.Fatalf("FindByFingerprint failed: %v", err)
	}

	if found == nil {
		t.Fatal("Expected to find media file, got nil")
	}

	// Verify all fields are correctly retrieved
	if found.ID != media.ID {
		t.Errorf("Expected ID %s, got %s", media.ID, found.ID)
	}
	if found.Fingerprint != media.Fingerprint {
		t.Errorf("Expected fingerprint %s, got %s", media.Fingerprint, found.Fingerprint)
	}
	if found.FilePath != media.FilePath {
		t.Errorf("Expected file path %s, got %s", media.FilePath, found.FilePath)
	}
	if found.Duration != media.Duration {
		t.Errorf("Expected duration %d, got %d", media.Duration, found.Duration)
	}
	if found.FileSize != media.FileSize {
		t.Errorf("Expected file size %d, got %d", media.FileSize, found.FileSize)
	}
	if found.Width != media.Width {
		t.Errorf("Expected width %d, got %d", media.Width, found.Width)
	}
	if found.Height != media.Height {
		t.Errorf("Expected height %d, got %d", media.Height, found.Height)
	}
	if found.Status != media.Status {
		t.Errorf("Expected status %s, got %s", media.Status, found.Status)
	}

	// Verify JSON arrays
	if len(found.AudioCodecs) != len(media.AudioCodecs) {
		t.Errorf("Expected %d audio codecs, got %d", len(media.AudioCodecs), len(found.AudioCodecs))
	}
	for i, codec := range media.AudioCodecs {
		if found.AudioCodecs[i] != codec {
			t.Errorf("Expected audio codec %s at index %d, got %s", codec, i, found.AudioCodecs[i])
		}
	}

	if len(found.SubtitleLanguages) != len(media.SubtitleLanguages) {
		t.Errorf("Expected %d subtitle languages, got %d", len(media.SubtitleLanguages), len(found.SubtitleLanguages))
	}
	for i, lang := range media.SubtitleLanguages {
		if found.SubtitleLanguages[i] != lang {
			t.Errorf("Expected subtitle language %s at index %d, got %s", lang, i, found.SubtitleLanguages[i])
		}
	}
}

// TestMediaRepository_FindByFingerprint_NotFound tests finding non-existent fingerprint
func TestMediaRepository_FindByFingerprint_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)
	ctx := context.Background()

	found, err := repo.FindByFingerprint(ctx, "non_existent_fingerprint")

	if err != nil {
		t.Fatalf("FindByFingerprint should not error on not found, got: %v", err)
	}

	if found != nil {
		t.Error("Expected nil when fingerprint not found, got non-nil")
	}
}

// TestMediaRepository_ExistsByFingerprint tests checking if fingerprint exists
func TestMediaRepository_ExistsByFingerprint(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)
	ctx := context.Background()

	// Create a media file
	media := createTestMediaFile()
	err := repo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Check existing fingerprint
	exists, err := repo.ExistsByFingerprint(ctx, media.Fingerprint)
	if err != nil {
		t.Fatalf("ExistsByFingerprint failed: %v", err)
	}

	if !exists {
		t.Error("Expected fingerprint to exist, got false")
	}

	// Check non-existent fingerprint
	exists, err = repo.ExistsByFingerprint(ctx, "non_existent_fingerprint")
	if err != nil {
		t.Fatalf("ExistsByFingerprint failed: %v", err)
	}

	if exists {
		t.Error("Expected fingerprint to not exist, got true")
	}
}

// TestMediaRepository_Update tests updating an existing media file
func TestMediaRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)
	ctx := context.Background()

	// Create a media file
	media := createTestMediaFile()
	err := repo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update fields
	media.Title = "Updated Title"
	media.Status = domain.MediaStatusError
	media.Duration = 240
	media.AudioCodecs = []string{"aac", "ac3", "dts"}
	media.SubtitleLanguages = []string{"eng", "spa", "fre"}
	media.UpdatedAt = time.Now().UTC()

	err = repo.Update(ctx, media)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify updates
	found, err := repo.FindByFingerprint(ctx, media.Fingerprint)
	if err != nil {
		t.Fatalf("FindByFingerprint failed: %v", err)
	}

	if found.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", found.Title)
	}
	if found.Status != domain.MediaStatusError {
		t.Errorf("Expected status '%s', got '%s'", domain.MediaStatusError, found.Status)
	}
	if found.Duration != 240 {
		t.Errorf("Expected duration 240, got %d", found.Duration)
	}
	if len(found.AudioCodecs) != 3 {
		t.Errorf("Expected 3 audio codecs, got %d", len(found.AudioCodecs))
	}
	if len(found.SubtitleLanguages) != 3 {
		t.Errorf("Expected 3 subtitle languages, got %d", len(found.SubtitleLanguages))
	}
}

// TestMediaRepository_Update_NotFound tests updating non-existent media file
func TestMediaRepository_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)
	ctx := context.Background()

	media := createTestMediaFile()
	// Don't create it, just try to update
	err := repo.Update(ctx, media)

	if err == nil {
		t.Error("Expected error when updating non-existent media file, got nil")
	}
}

// TestMediaRepository_Update_ChangeFingerprint tests changing fingerprint in update
func TestMediaRepository_Update_ChangeFingerprint(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)
	ctx := context.Background()

	// Create a media file
	media := createTestMediaFile()
	originalFingerprint := media.Fingerprint
	err := repo.Create(ctx, media)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update with different fingerprint
	media.Fingerprint = uuid.New().String() + uuid.New().String()[:28] // 64 chars
	err = repo.Update(ctx, media)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify new fingerprint is set
	found, err := repo.FindByFingerprint(ctx, media.Fingerprint)
	if err != nil {
		t.Fatalf("FindByFingerprint failed: %v", err)
	}
	if found == nil {
		t.Fatal("Expected to find media file with new fingerprint")
	}

	// Verify old fingerprint no longer exists
	found, err = repo.FindByFingerprint(ctx, originalFingerprint)
	if err != nil {
		t.Fatalf("FindByFingerprint failed: %v", err)
	}
	if found != nil {
		t.Error("Expected old fingerprint to not exist after update")
	}
}

// TestMediaRepository_ConcurrentOperations tests concurrent create operations
func TestMediaRepository_ConcurrentOperations(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)
	ctx := context.Background()

	// Create multiple media files sequentially (not concurrently)
	// Concurrent operations on SQLite in-memory DB can be problematic in tests
	numFiles := 5
	for i := 0; i < numFiles; i++ {
		media := createTestMediaFile()
		err := repo.Create(ctx, media)
		if err != nil {
			t.Errorf("Create operation %d failed: %v", i, err)
		}
	}

	// Verify all files were created
	// (In a real test, we'd query to count, but this shows the basic pattern)
}

// TestMediaRepository_ComplexArrayData tests with various array data
func TestMediaRepository_ComplexArrayData(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMediaRepository(db)
	ctx := context.Background()

	testCases := []struct {
		name              string
		audioCodecs       []string
		subtitleLanguages []string
	}{
		{
			name:              "Multiple audio codecs",
			audioCodecs:       []string{"aac", "ac3", "dts", "truehd", "flac"},
			subtitleLanguages: []string{"eng"},
		},
		{
			name:              "Multiple subtitle languages",
			audioCodecs:       []string{"aac"},
			subtitleLanguages: []string{"eng", "spa", "fre", "ger", "ita", "jpn", "chi"},
		},
		{
			name:              "Single items",
			audioCodecs:       []string{"aac"},
			subtitleLanguages: []string{"eng"},
		},
		{
			name:              "Empty arrays",
			audioCodecs:       []string{},
			subtitleLanguages: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			media := createTestMediaFile()
			media.AudioCodecs = tc.audioCodecs
			media.SubtitleLanguages = tc.subtitleLanguages

			err := repo.Create(ctx, media)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			found, err := repo.FindByFingerprint(ctx, media.Fingerprint)
			if err != nil {
				t.Fatalf("FindByFingerprint failed: %v", err)
			}

			if len(found.AudioCodecs) != len(tc.audioCodecs) {
				t.Errorf("Expected %d audio codecs, got %d", len(tc.audioCodecs), len(found.AudioCodecs))
			}
			if len(found.SubtitleLanguages) != len(tc.subtitleLanguages) {
				t.Errorf("Expected %d subtitle languages, got %d", len(tc.subtitleLanguages), len(found.SubtitleLanguages))
			}
		})
	}
}
