package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
)

// TestNewWatchProgressRepository tests the repository constructor
func TestNewWatchProgressRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWatchProgressRepository(db)

	if repo == nil {
		t.Fatal("Expected repository to be non-nil")
	}
}

// TestWatchProgressRepository_SaveProgress_Movie tests saving progress for a movie
func TestWatchProgressRepository_SaveProgress_Movie(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWatchProgressRepository(db)
	ctx := context.Background()

	// Create test user
	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	// Create test media file
	media := createTestMediaFile()
	mediaRepo := NewMediaRepository(db)
	if err := mediaRepo.Create(ctx, media); err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	// Save progress
	progress := &domain.WatchProgress{
		UserID:          userID,
		MediaID:         sql.NullString{String: media.ID, Valid: true},
		PositionSeconds: 300,
		DurationSeconds: 3600,
		ProgressPercent: 8.33,
		Completed:       false,
		LastWatchedAt:   time.Now().UTC(),
	}

	err := repo.SaveProgress(ctx, progress)
	if err != nil {
		t.Fatalf("SaveProgress failed: %v", err)
	}

	// Verify progress was saved
	saved, err := repo.GetProgress(ctx, userID, media.ID, nil)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	if saved == nil {
		t.Fatal("Expected progress to be found")
	}

	if saved.PositionSeconds != 300 {
		t.Errorf("Expected position 300, got %d", saved.PositionSeconds)
	}

	if saved.Completed {
		t.Error("Expected completed to be false")
	}
}

// TestWatchProgressRepository_SaveProgress_Update tests updating existing progress
func TestWatchProgressRepository_SaveProgress_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWatchProgressRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	media := createTestMediaFile()
	mediaRepo := NewMediaRepository(db)
	if err := mediaRepo.Create(ctx, media); err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	// Save initial progress
	progress1 := &domain.WatchProgress{
		UserID:            userID,
		MediaID:           sql.NullString{String: media.ID, Valid: true},
		EpisodeMetadataID: sql.NullInt64{Valid: false}, // NULL for movies
		PositionSeconds:   300,
		DurationSeconds:   3600,
		ProgressPercent:   8.33,
		Completed:         false,
		LastWatchedAt:     time.Now().UTC(),
	}

	if err := repo.SaveProgress(ctx, progress1); err != nil {
		t.Fatalf("First SaveProgress failed: %v", err)
	}

	// Update progress - should trigger ON CONFLICT with partial unique index
	time.Sleep(10 * time.Millisecond)
	progress2 := &domain.WatchProgress{
		UserID:            userID,
		MediaID:           sql.NullString{String: media.ID, Valid: true},
		EpisodeMetadataID: sql.NullInt64{Valid: false},
		PositionSeconds:   600,
		DurationSeconds:   3600,
		ProgressPercent:   16.67,
		Completed:         false,
		LastWatchedAt:     time.Now().UTC(),
	}

	if err := repo.SaveProgress(ctx, progress2); err != nil {
		t.Fatalf("Second SaveProgress failed: %v", err)
	}

	// Verify only 1 record exists (ON CONFLICT should have updated, not inserted)
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM watch_progress WHERE user_id = ?", userID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count records: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 record after update (ON CONFLICT), got %d", count)
	}

	// Verify progress was updated to latest values
	saved, err := repo.GetProgress(ctx, userID, media.ID, nil)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	if saved == nil {
		t.Fatal("Expected progress to exist")
	}

	if saved.PositionSeconds != 600 {
		t.Errorf("Expected position 600 after update, got %d", saved.PositionSeconds)
	}

	if saved.ProgressPercent != 16.67 {
		t.Errorf("Expected progress 16.67%% after update, got %.2f%%", saved.ProgressPercent)
	}
}

// TestWatchProgressRepository_SaveProgress_Completed tests marking as completed at 95%
func TestWatchProgressRepository_SaveProgress_Completed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWatchProgressRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	media := createTestMediaFile()
	media.Duration = 3600 // 1 hour
	mediaRepo := NewMediaRepository(db)
	if err := mediaRepo.Create(ctx, media); err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	// Save progress at 96% (should be completed)
	progress := &domain.WatchProgress{
		UserID:          userID,
		MediaID:         sql.NullString{String: media.ID, Valid: true},
		PositionSeconds: 3456, // 96%
		DurationSeconds: 3600,
		ProgressPercent: 96.0,
		Completed:       true,
		LastWatchedAt:   time.Now().UTC(),
	}

	if err := repo.SaveProgress(ctx, progress); err != nil {
		t.Fatalf("SaveProgress failed: %v", err)
	}

	saved, err := repo.GetProgress(ctx, userID, media.ID, nil)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	if !saved.Completed {
		t.Error("Expected completed to be true at 96%")
	}
}

// TestWatchProgressRepository_GetProgress_NotFound tests getting non-existent progress
func TestWatchProgressRepository_GetProgress_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWatchProgressRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	mediaID := uuid.New().String()

	progress, err := repo.GetProgress(ctx, userID, mediaID, nil)
	if err != nil {
		t.Fatalf("GetProgress should not error on not found: %v", err)
	}

	if progress != nil {
		t.Error("Expected nil progress for non-existent media")
	}
}

// TestWatchProgressRepository_GetContinueWatching tests fetching continue watching list
func TestWatchProgressRepository_GetContinueWatching(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWatchProgressRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	mediaRepo := NewMediaRepository(db)

	// Create 3 media files with different progress
	media1 := createTestMediaFile()
	media1.Title = "Movie 1"
	if err := mediaRepo.Create(ctx, media1); err != nil {
		t.Fatalf("Failed to create media1: %v", err)
	}

	media2 := createTestMediaFile()
	media2.Title = "Movie 2"
	if err := mediaRepo.Create(ctx, media2); err != nil {
		t.Fatalf("Failed to create media2: %v", err)
	}

	media3 := createTestMediaFile()
	media3.Title = "Movie 3"
	if err := mediaRepo.Create(ctx, media3); err != nil {
		t.Fatalf("Failed to create media3: %v", err)
	}

	// Save progress for media1 (in progress)
	time.Sleep(10 * time.Millisecond)
	progress1 := &domain.WatchProgress{
		UserID:          userID,
		MediaID:         sql.NullString{String: media1.ID, Valid: true},
		PositionSeconds: 300,
		DurationSeconds: 3600,
		ProgressPercent: 8.33,
		Completed:       false,
		LastWatchedAt:   time.Now().UTC(),
	}
	if err := repo.SaveProgress(ctx, progress1); err != nil {
		t.Fatalf("SaveProgress media1 failed: %v", err)
	}

	// Save progress for media2 (completed - should not appear)
	time.Sleep(10 * time.Millisecond)
	progress2 := &domain.WatchProgress{
		UserID:          userID,
		MediaID:         sql.NullString{String: media2.ID, Valid: true},
		PositionSeconds: 3500,
		DurationSeconds: 3600,
		ProgressPercent: 97.22,
		Completed:       true,
		LastWatchedAt:   time.Now().UTC(),
	}
	if err := repo.SaveProgress(ctx, progress2); err != nil {
		t.Fatalf("SaveProgress media2 failed: %v", err)
	}

	// Save progress for media3 (in progress, most recent)
	time.Sleep(10 * time.Millisecond)
	progress3 := &domain.WatchProgress{
		UserID:          userID,
		MediaID:         sql.NullString{String: media3.ID, Valid: true},
		PositionSeconds: 1800,
		DurationSeconds: 3600,
		ProgressPercent: 50.0,
		Completed:       false,
		LastWatchedAt:   time.Now().UTC(),
	}
	if err := repo.SaveProgress(ctx, progress3); err != nil {
		t.Fatalf("SaveProgress media3 failed: %v", err)
	}

	// Get continue watching
	items, err := repo.GetContinueWatching(ctx, userID, 10)
	if err != nil {
		t.Fatalf("GetContinueWatching failed: %v", err)
	}

	// Should only return 2 items (completed items excluded)
	if len(items) != 2 {
		t.Fatalf("Expected 2 continue watching items, got %d", len(items))
	}

	// Most recent should be first (media3)
	if items[0].Title != "Movie 3" {
		t.Errorf("Expected first item to be 'Movie 3', got '%s'", items[0].Title)
	}

	if items[1].Title != "Movie 1" {
		t.Errorf("Expected second item to be 'Movie 1', got '%s'", items[1].Title)
	}
}

// TestWatchProgressRepository_RemoveFromContinueWatching tests manual removal
func TestWatchProgressRepository_RemoveFromContinueWatching(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWatchProgressRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	media := createTestMediaFile()
	mediaRepo := NewMediaRepository(db)
	if err := mediaRepo.Create(ctx, media); err != nil {
		t.Fatalf("Failed to create test media: %v", err)
	}

	// Save progress
	progress := &domain.WatchProgress{
		UserID:          userID,
		MediaID:         sql.NullString{String: media.ID, Valid: true},
		PositionSeconds: 300,
		DurationSeconds: 3600,
		ProgressPercent: 8.33,
		Completed:       false,
		LastWatchedAt:   time.Now().UTC(),
	}
	if err := repo.SaveProgress(ctx, progress); err != nil {
		t.Fatalf("SaveProgress failed: %v", err)
	}

	// Remove from continue watching
	err := repo.RemoveFromContinueWatching(ctx, userID, media.ID, nil)
	if err != nil {
		t.Fatalf("RemoveFromContinueWatching failed: %v", err)
	}

	// Verify progress is marked as manually removed (not deleted)
	saved, err := repo.GetProgress(ctx, userID, media.ID, nil)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	if saved == nil {
		t.Fatal("Expected progress to still exist")
	}

	if !saved.ManuallyRemoved {
		t.Error("Expected manually_removed to be true")
	}

	// Verify it doesn't appear in continue watching list
	items, err := repo.GetContinueWatching(ctx, userID, 10)
	if err != nil {
		t.Fatalf("GetContinueWatching failed: %v", err)
	}

	if len(items) != 0 {
		t.Error("Expected manually removed item not to appear in continue watching")
	}
}

// createTestUser creates a test user in the database
func createTestUser(t *testing.T, db *database.DB, userID, username string) {
	_, err := db.DB.Exec(`
		INSERT INTO users (id, username, password_hash, role, must_change_password, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, username, "hash", "user", false, time.Now().UTC(), time.Now().UTC())
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
}
