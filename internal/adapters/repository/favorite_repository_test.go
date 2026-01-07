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

// TestNewFavoriteRepository tests the repository constructor
func TestNewFavoriteRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFavoriteRepository(db)

	if repo == nil {
		t.Fatal("Expected repository to be non-nil")
	}
}

// TestFavoriteRepository_AddFavorite_Movie tests adding a movie to favorites
func TestFavoriteRepository_AddFavorite_Movie(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFavoriteRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	// Create movie metadata
	movieID := createTestMovieMetadata(t, db, "Inception", 2010)

	// Add movie to favorites
	favorite := &domain.Favorite{
		ID:              uuid.New().String(),
		UserID:          userID,
		MediaType:       "movie",
		MovieMetadataID: sql.NullInt64{Int64: movieID, Valid: true},
		AddedAt:         time.Now().UTC(),
	}
	err := repo.AddFavorite(ctx, favorite)
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}

	// Verify it's favorited
	isFavorited, err := repo.IsFavorited(ctx, userID, "movie", movieID)
	if err != nil {
		t.Fatalf("IsFavorited failed: %v", err)
	}

	if !isFavorited {
		t.Error("Expected movie to be favorited")
	}
}

// TestFavoriteRepository_AddFavorite_Series tests adding a series to favorites
func TestFavoriteRepository_AddFavorite_Series(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFavoriteRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	// Create series metadata
	seriesID := createTestSeriesMetadata(t, db, "Breaking Bad", 2008)

	// Add series to favorites
	favorite := &domain.Favorite{
		ID:               uuid.New().String(),
		UserID:           userID,
		MediaType:        "series",
		SeriesMetadataID: sql.NullInt64{Int64: seriesID, Valid: true},
		AddedAt:          time.Now().UTC(),
	}
	err := repo.AddFavorite(ctx, favorite)
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}

	// Verify it's favorited
	isFavorited, err := repo.IsFavorited(ctx, userID, "series", seriesID)
	if err != nil {
		t.Fatalf("IsFavorited failed: %v", err)
	}

	if !isFavorited {
		t.Error("Expected series to be favorited")
	}
}

// TestFavoriteRepository_AddFavorite_Duplicate tests adding same favorite twice
func TestFavoriteRepository_AddFavorite_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFavoriteRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	movieID := createTestMovieMetadata(t, db, "The Matrix", 1999)

	// Add first time
	favorite := &domain.Favorite{
		ID:              uuid.New().String(),
		UserID:          userID,
		MediaType:       "movie",
		MovieMetadataID: sql.NullInt64{Int64: movieID, Valid: true},
		AddedAt:         time.Now().UTC(),
	}
	err := repo.AddFavorite(ctx, favorite)
	if err != nil {
		t.Fatalf("First AddFavorite failed: %v", err)
	}

	// Add second time - should fail with unique constraint
	duplicate := &domain.Favorite{
		ID:              uuid.New().String(),
		UserID:          userID,
		MediaType:       "movie",
		MovieMetadataID: sql.NullInt64{Int64: movieID, Valid: true},
		AddedAt:         time.Now().UTC(),
	}
	err = repo.AddFavorite(ctx, duplicate)
	if err == nil {
		t.Error("Expected error when adding duplicate favorite, got nil")
	}
}

// TestFavoriteRepository_RemoveFavorite tests removing a favorite
func TestFavoriteRepository_RemoveFavorite(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFavoriteRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	movieID := createTestMovieMetadata(t, db, "Interstellar", 2014)

	// Add favorite
	favorite := &domain.Favorite{
		ID:              uuid.New().String(),
		UserID:          userID,
		MediaType:       "movie",
		MovieMetadataID: sql.NullInt64{Int64: movieID, Valid: true},
		AddedAt:         time.Now().UTC(),
	}
	err := repo.AddFavorite(ctx, favorite)
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}

	// Remove favorite
	err = repo.RemoveFavorite(ctx, userID, "movie", movieID)
	if err != nil {
		t.Fatalf("RemoveFavorite failed: %v", err)
	}

	// Verify it's no longer favorited
	isFavorited, err := repo.IsFavorited(ctx, userID, "movie", movieID)
	if err != nil {
		t.Fatalf("IsFavorited failed: %v", err)
	}

	if isFavorited {
		t.Error("Expected movie to not be favorited after removal")
	}
}

// TestFavoriteRepository_GetUserFavorites tests retrieving user's favorites
func TestFavoriteRepository_GetUserFavorites(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFavoriteRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	// Create and favorite 3 movies
	movie1ID := createTestMovieMetadata(t, db, "Movie 1", 2020)
	movie2ID := createTestMovieMetadata(t, db, "Movie 2", 2021)
	movie3ID := createTestMovieMetadata(t, db, "Movie 3", 2022)

	time.Sleep(10 * time.Millisecond)
	favorite1 := &domain.Favorite{
		ID:              uuid.New().String(),
		UserID:          userID,
		MediaType:       "movie",
		MovieMetadataID: sql.NullInt64{Int64: movie1ID, Valid: true},
		AddedAt:         time.Now().UTC(),
	}
	err := repo.AddFavorite(ctx, favorite1)
	if err != nil {
		t.Fatalf("AddFavorite movie1 failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	favorite2 := &domain.Favorite{
		ID:              uuid.New().String(),
		UserID:          userID,
		MediaType:       "movie",
		MovieMetadataID: sql.NullInt64{Int64: movie2ID, Valid: true},
		AddedAt:         time.Now().UTC(),
	}
	err = repo.AddFavorite(ctx, favorite2)
	if err != nil {
		t.Fatalf("AddFavorite movie2 failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	favorite3 := &domain.Favorite{
		ID:              uuid.New().String(),
		UserID:          userID,
		MediaType:       "movie",
		MovieMetadataID: sql.NullInt64{Int64: movie3ID, Valid: true},
		AddedAt:         time.Now().UTC(),
	}
	err = repo.AddFavorite(ctx, favorite3)
	if err != nil {
		t.Fatalf("AddFavorite movie3 failed: %v", err)
	}

	// Create and favorite 1 series
	seriesID := createTestSeriesMetadata(t, db, "Series 1", 2019)
	time.Sleep(10 * time.Millisecond)
	favorite4 := &domain.Favorite{
		ID:               uuid.New().String(),
		UserID:           userID,
		MediaType:        "series",
		SeriesMetadataID: sql.NullInt64{Int64: seriesID, Valid: true},
		AddedAt:          time.Now().UTC(),
	}
	err = repo.AddFavorite(ctx, favorite4)
	if err != nil {
		t.Fatalf("AddFavorite series failed: %v", err)
	}

	// Get all favorites
	favorites, err := repo.GetUserFavorites(ctx, userID, 10)
	if err != nil {
		t.Fatalf("GetUserFavorites failed: %v", err)
	}

	// Should have 4 favorites total
	if len(favorites) != 4 {
		t.Fatalf("Expected 4 favorites, got %d", len(favorites))
	}

	// Should be ordered by added_at DESC (most recent first)
	if favorites[0].Title != "Series 1" {
		t.Errorf("Expected first favorite to be 'Series 1', got '%s'", favorites[0].Title)
	}

	// Count movies and series
	movieCount := 0
	seriesCount := 0
	for _, fav := range favorites {
		if fav.MediaType == "movie" {
			movieCount++
		} else if fav.MediaType == "series" {
			seriesCount++
		}
	}

	if movieCount != 3 {
		t.Errorf("Expected 3 movies, got %d", movieCount)
	}

	if seriesCount != 1 {
		t.Errorf("Expected 1 series, got %d", seriesCount)
	}
}

// TestFavoriteRepository_GetUserFavorites_Empty tests empty favorites list
func TestFavoriteRepository_GetUserFavorites_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFavoriteRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	favorites, err := repo.GetUserFavorites(ctx, userID, 10)
	if err != nil {
		t.Fatalf("GetUserFavorites failed: %v", err)
	}

	if len(favorites) != 0 {
		t.Errorf("Expected 0 favorites, got %d", len(favorites))
	}
}

// TestFavoriteRepository_GetUserFavorites_Limit tests pagination
func TestFavoriteRepository_GetUserFavorites_Limit(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFavoriteRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	// Add 5 favorites
	for i := 1; i <= 5; i++ {
		movieID := createTestMovieMetadata(t, db, "Movie", 2020+i)
		time.Sleep(10 * time.Millisecond)
		favorite := &domain.Favorite{
			ID:              uuid.New().String(),
			UserID:          userID,
			MediaType:       "movie",
			MovieMetadataID: sql.NullInt64{Int64: movieID, Valid: true},
			AddedAt:         time.Now().UTC(),
		}
		err := repo.AddFavorite(ctx, favorite)
		if err != nil {
			t.Fatalf("AddFavorite failed: %v", err)
		}
	}

	// Get only 3
	favorites, err := repo.GetUserFavorites(ctx, userID, 3)
	if err != nil {
		t.Fatalf("GetUserFavorites failed: %v", err)
	}

	if len(favorites) != 3 {
		t.Errorf("Expected 3 favorites with limit, got %d", len(favorites))
	}
}

// TestFavoriteRepository_IsFavorited_NotFavorited tests checking non-favorited item
func TestFavoriteRepository_IsFavorited_NotFavorited(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFavoriteRepository(db)
	ctx := context.Background()

	userID := uuid.New().String()
	createTestUser(t, db, userID, "testuser")

	movieID := createTestMovieMetadata(t, db, "Some Movie", 2023)

	isFavorited, err := repo.IsFavorited(ctx, userID, "movie", movieID)
	if err != nil {
		t.Fatalf("IsFavorited failed: %v", err)
	}

	if isFavorited {
		t.Error("Expected movie to not be favorited")
	}
}

// Helper functions

// createTestMovieMetadata creates test movie metadata and returns the ID
func createTestMovieMetadata(t *testing.T, db *database.DB, title string, year int) int64 {
	// Generate unique tmdb_id based on title and year to avoid UNIQUE constraint violations
	tmdbID := int64(len(title)*1000 + year)

	result, err := db.DB.Exec(`
		INSERT INTO movie_metadata (tmdb_id, original_title, release_date, runtime, vote_average, vote_count, popularity, status, original_lang, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tmdbID, title, time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), 120, 8.5, 1000, 50.0, "Released", "en", time.Now().UTC(), time.Now().UTC())
	if err != nil {
		t.Fatalf("Failed to create movie metadata: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert ID: %v", err)
	}

	return id
}

// createTestSeriesMetadata creates test series metadata and returns the ID
func createTestSeriesMetadata(t *testing.T, db *database.DB, title string, year int) int64 {
	// Generate unique tmdb_id based on title and year to avoid UNIQUE constraint violations
	tmdbID := int64(len(title)*10000 + year)

	result, err := db.DB.Exec(`
		INSERT INTO series_metadata (tmdb_id, original_name, first_air_date, number_of_seasons, number_of_episodes, status, vote_average, vote_count, popularity, original_lang, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tmdbID, title, time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), 5, 50, "Ended", 9.0, 5000, 80.0, "en", time.Now().UTC(), time.Now().UTC())
	if err != nil {
		t.Fatalf("Failed to create series metadata: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert ID: %v", err)
	}

	return id
}
