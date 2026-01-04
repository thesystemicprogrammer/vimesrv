package metadata

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/repository"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/metadata/linker"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ===========================
// Test Infrastructure
// ===========================

// setupTestDatabase creates an in-memory SQLite database with migrations
func setupIntegrationTestDatabase(t *testing.T) (*database.DB, *sql.DB) {
	t.Helper()

	cfg := database.Config{
		Path:            "file::memory:?cache=shared",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	db, err := database.New(cfg)
	require.NoError(t, err, "failed to create test database")

	// Run migrations
	migration := database.NewDatabaseMigration(db.DB)
	err = migration.Migrate()
	require.NoError(t, err, "failed to run migrations")

	t.Cleanup(func() {
		db.Close()
	})

	return db, db.DB
}

// testTMDBClient is a mock TMDB client for integration tests
type integrationTestTMDBClient struct {
	searchMovieResults []ports.TMDBSearchResult
	searchTVResults    []ports.TMDBSearchResult
	searchMultiResults []ports.TMDBSearchResult
	movieDetails       *ports.TMDBMovieDetails
	seriesDetails      *ports.TMDBSeriesDetails
	seasonDetails      *ports.TMDBSeasonDetails
	episodeDetails     *ports.TMDBEpisodeDetails
}

func (m *integrationTestTMDBClient) SearchMovie(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
	return m.searchMovieResults, nil
}

func (m *integrationTestTMDBClient) SearchTV(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
	return m.searchTVResults, nil
}

func (m *integrationTestTMDBClient) SearchMulti(ctx context.Context, query string, language string) ([]ports.TMDBSearchResult, error) {
	return m.searchMultiResults, nil
}

func (m *integrationTestTMDBClient) GetMovieDetails(ctx context.Context, tmdbID int, language string) (*ports.TMDBMovieDetails, error) {
	return m.movieDetails, nil
}

func (m *integrationTestTMDBClient) GetSeriesDetails(ctx context.Context, tmdbID int, language string) (*ports.TMDBSeriesDetails, error) {
	return m.seriesDetails, nil
}

func (m *integrationTestTMDBClient) GetSeasonDetails(ctx context.Context, tmdbID int, seasonNumber int, language string) (*ports.TMDBSeasonDetails, error) {
	return m.seasonDetails, nil
}

func (m *integrationTestTMDBClient) GetEpisodeDetails(ctx context.Context, tmdbID int, seasonNumber int, episodeNumber int, language string) (*ports.TMDBEpisodeDetails, error) {
	return m.episodeDetails, nil
}

func (m *integrationTestTMDBClient) GetImageURL(path, size string) string {
	if path == "" {
		return ""
	}
	return "https://image.tmdb.org/t/p/" + size + path
}

func (m *integrationTestTMDBClient) GetMovieCredits(ctx context.Context, tmdbID int) (*ports.TMDBMovieCredits, error) {
	return nil, nil
}

func (m *integrationTestTMDBClient) GetSeriesAggregateCredits(ctx context.Context, tmdbID int) (*ports.TMDBSeriesAggregateCredits, error) {
	return nil, nil
}

func (m *integrationTestTMDBClient) GetMovieReleaseDates(ctx context.Context, tmdbID int) (*ports.TMDBReleaseDatesResponse, error) {
	return nil, nil
}

func (m *integrationTestTMDBClient) GetSimilarMovies(ctx context.Context, tmdbID int, language string) (*ports.TMDBSimilarMoviesResponse, error) {
	return nil, nil
}

func (m *integrationTestTMDBClient) GetSimilarSeries(ctx context.Context, tmdbID int, language string) (*ports.TMDBSimilarSeriesResponse, error) {
	return nil, nil
}

func (m *integrationTestTMDBClient) GetCollectionDetails(ctx context.Context, collectionID int, language string) (*ports.TMDBCollectionDetails, error) {
	return nil, nil
}

func (m *integrationTestTMDBClient) GetCollectionTranslations(ctx context.Context, collectionID int) (*ports.TMDBCollectionTranslationsResponse, error) {
	return nil, nil
}

func (m *integrationTestTMDBClient) GetMovieTranslations(ctx context.Context, tmdbID int) (*ports.TMDBMovieTranslationsResponse, error) {
	return nil, nil
}

func (m *integrationTestTMDBClient) GetSeriesTranslations(ctx context.Context, tmdbID int) (*ports.TMDBSeriesTranslationsResponse, error) {
	return nil, nil
}

// integrationTestFilenameParser is a mock filename parser
type integrationTestFilenameParser struct {
	parsedResult *ports.ParsedFilename
}

func (p *integrationTestFilenameParser) Parse(filename string) *ports.ParsedFilename {
	if p.parsedResult != nil {
		return p.parsedResult
	}
	// Default movie parse
	return &ports.ParsedFilename{
		Title:      "The Matrix",
		CleanTitle: "the matrix",
		Year:       1999,
		IsSeries:   false,
	}
}

// integrationTestImageDownloader is a mock image downloader
type integrationTestImageDownloader struct{}

func (d *integrationTestImageDownloader) DownloadImage(ctx context.Context, path string, imageType string, tmdbID int) (string, error) {
	return "/cache/images/" + path, nil
}

func (d *integrationTestImageDownloader) DownloadSeasonImage(ctx context.Context, path string, seriesID, seasonNumber int) (string, error) {
	return "/cache/images/seasons/" + path, nil
}

func (d *integrationTestImageDownloader) DownloadEpisodeImage(ctx context.Context, path string, seriesID, seasonNumber, episodeNumber int) (string, error) {
	return "/cache/images/episodes/" + path, nil
}

func (d *integrationTestImageDownloader) GetLocalPath(imageType string, id int) string {
	return "/cache/images/local"
}

func (d *integrationTestImageDownloader) ImageExists(imageType string, id int) bool {
	return false
}

// ===========================
// Integration Tests
// ===========================

func TestEnrichMediaFile_Integration_AutoLinkMovie(t *testing.T) {
	ctx := context.Background()
	db, sqlDB := setupIntegrationTestDatabase(t)

	// Create repositories
	mediaRepo := repository.NewMediaRepository(db)
	movieMetadataRepo := repository.NewSQLiteMovieMetadataRepository(db.DB)
	seriesMetadataRepo := repository.NewSQLiteSeriesMetadataRepository(db.DB)
	seasonMetadataRepo := repository.NewSQLiteSeasonMetadataRepository(db.DB)
	episodeMetadataRepo := repository.NewSQLiteEpisodeMetadataRepository(db.DB)
	candidateRepo := repository.NewSQLiteMetadataCandidateRepository(db.DB)

	// Create a media file
	media := &domain.MediaFile{
		ID:               "test-media-1",
		Fingerprint:      "fingerprint-1",
		FilePath:         "/movies/The.Matrix.1999.1080p.mp4",
		Filename:         "The.Matrix.1999.1080p.mp4",
		Status:           domain.MediaStatusReady,
		EnrichmentStatus: domain.EnrichmentStatusPending,
	}
	err := mediaRepo.Create(ctx, media)
	require.NoError(t, err)

	// Create mock TMDB client with high-confidence match
	tmdbClient := &integrationTestTMDBClient{
		searchMovieResults: []ports.TMDBSearchResult{
			{
				ID:            603,
				MediaType:     "movie",
				Title:         "The Matrix",
				OriginalTitle: "The Matrix",
				ReleaseDate:   "1999-03-30",
				Overview:      "A computer hacker learns about the true nature of reality.",
				PosterPath:    "/poster.jpg",
				VoteAverage:   8.5,
				Popularity:    50.0,
			},
		},
		movieDetails: &ports.TMDBMovieDetails{
			ID:            603,
			Title:         "The Matrix",
			OriginalTitle: "The Matrix",
			ReleaseDate:   "1999-03-30",
			Overview:      "A computer hacker learns about the true nature of reality.",
			PosterPath:    "/poster.jpg",
			BackdropPath:  "/backdrop.jpg",
			VoteAverage:   8.5,
			VoteCount:     20000,
			Popularity:    50.0,
			Runtime:       136,
			Status:        "Released",
			OriginalLang:  "en",
			Genres: []ports.TMDBGenre{
				{ID: 28, Name: "Action"},
				{ID: 878, Name: "Science Fiction"},
			},
		},
	}

	// Create config with auto-link threshold
	cfg := config.TMDBConfig{
		Enabled:           true,
		Language:          "en",
		AutoSearch:        true,
		AutoLinkThreshold: 70,
		MaxCandidates:     5,
		PosterSize:        "w500",
	}

	// Create use case (correct argument order: filenameParser, tmdbClient, ...)
	useCase := NewEnrichMediaFileUseCase(
		cfg,
		&integrationTestFilenameParser{},
		tmdbClient,
		&integrationTestImageDownloader{},
		mediaRepo,
		movieMetadataRepo,
		seriesMetadataRepo,
		seasonMetadataRepo,
		episodeMetadataRepo,
		candidateRepo,
		nil,
		nil,
		nil,
	)

	// Execute enrichment
	output, err := useCase.Execute(ctx, EnrichMediaFileInput{
		MediaID: "test-media-1",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.EnrichmentStatusAutoLinked, output.EnrichmentStatus)
	assert.Equal(t, domain.MetadataTypeMovie, output.MetadataType)

	// Verify media file was updated in database
	var enrichmentStatus, metadataType string
	var movieMetadataID sql.NullInt64
	err = sqlDB.QueryRow(`
		SELECT enrichment_status, metadata_type, movie_metadata_id 
		FROM media_files WHERE id = ?
	`, "test-media-1").Scan(&enrichmentStatus, &metadataType, &movieMetadataID)
	require.NoError(t, err)
	assert.Equal(t, domain.EnrichmentStatusAutoLinked, enrichmentStatus)
	assert.Equal(t, domain.MetadataTypeMovie, metadataType)
	assert.True(t, movieMetadataID.Valid, "movie_metadata_id should be set")

	// Verify movie metadata was created
	var tmdbID int
	var originalTitle string
	err = sqlDB.QueryRow(`
		SELECT tmdb_id, original_title FROM movie_metadata WHERE id = ?
	`, movieMetadataID.Int64).Scan(&tmdbID, &originalTitle)
	require.NoError(t, err)
	assert.Equal(t, 603, tmdbID)
	assert.Equal(t, "The Matrix", originalTitle)

	// Verify movie translation was created
	var translationTitle, overview string
	err = sqlDB.QueryRow(`
		SELECT title, overview FROM movie_metadata_translations 
		WHERE movie_metadata_id = ? AND language = ?
	`, movieMetadataID.Int64, "en").Scan(&translationTitle, &overview)
	require.NoError(t, err)
	assert.Equal(t, "The Matrix", translationTitle)
	assert.Contains(t, overview, "computer hacker")
}

func TestEnrichMediaFile_Integration_CandidatesFound(t *testing.T) {
	ctx := context.Background()
	db, sqlDB := setupIntegrationTestDatabase(t)

	// Create repositories
	mediaRepo := repository.NewMediaRepository(db)
	movieMetadataRepo := repository.NewSQLiteMovieMetadataRepository(db.DB)
	seriesMetadataRepo := repository.NewSQLiteSeriesMetadataRepository(db.DB)
	seasonMetadataRepo := repository.NewSQLiteSeasonMetadataRepository(db.DB)
	episodeMetadataRepo := repository.NewSQLiteEpisodeMetadataRepository(db.DB)
	candidateRepo := repository.NewSQLiteMetadataCandidateRepository(db.DB)

	// Create a media file
	media := &domain.MediaFile{
		ID:               "test-media-2",
		Fingerprint:      "fingerprint-2",
		FilePath:         "/movies/Matrix.mp4",
		Filename:         "Matrix.mp4",
		Status:           domain.MediaStatusReady,
		EnrichmentStatus: domain.EnrichmentStatusPending,
	}
	err := mediaRepo.Create(ctx, media)
	require.NoError(t, err)

	// Create mock TMDB client with multiple results (ambiguous)
	tmdbClient := &integrationTestTMDBClient{
		searchMovieResults: []ports.TMDBSearchResult{
			{
				ID:          603,
				MediaType:   "movie",
				Title:       "The Matrix",
				ReleaseDate: "1999-03-30",
				PosterPath:  "/matrix1.jpg",
				VoteAverage: 8.5,
			},
			{
				ID:          604,
				MediaType:   "movie",
				Title:       "The Matrix Reloaded",
				ReleaseDate: "2003-05-15",
				PosterPath:  "/matrix2.jpg",
				VoteAverage: 7.2,
			},
			{
				ID:          605,
				MediaType:   "movie",
				Title:       "The Matrix Revolutions",
				ReleaseDate: "2003-11-05",
				PosterPath:  "/matrix3.jpg",
				VoteAverage: 6.7,
			},
		},
	}

	// Create config with high auto-link threshold (should create candidates instead)
	cfg := config.TMDBConfig{
		Enabled:           true,
		Language:          "en",
		AutoSearch:        true,
		AutoLinkThreshold: 100, // Very high, so no auto-link
		MaxCandidates:     5,
		PosterSize:        "w500",
	}

	// Create use case
	useCase := NewEnrichMediaFileUseCase(
		cfg,
		&integrationTestFilenameParser{},
		tmdbClient,
		&integrationTestImageDownloader{},
		mediaRepo,
		movieMetadataRepo,
		seriesMetadataRepo,
		seasonMetadataRepo,
		episodeMetadataRepo,
		candidateRepo,
		nil,
		nil,
		nil,
	)

	// Execute enrichment
	output, err := useCase.Execute(ctx, EnrichMediaFileInput{
		MediaID: "test-media-2",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.EnrichmentStatusCandidatesFound, output.EnrichmentStatus)

	// Verify candidates were created in database
	var candidateCount int
	err = sqlDB.QueryRow(`
		SELECT COUNT(*) FROM metadata_candidates WHERE media_file_id = ?
	`, "test-media-2").Scan(&candidateCount)
	require.NoError(t, err)
	assert.Equal(t, 3, candidateCount, "should have created 3 candidates")

	// Verify candidates are ordered by confidence score
	rows, err := sqlDB.Query(`
		SELECT tmdb_id, title, confidence_score FROM metadata_candidates 
		WHERE media_file_id = ? ORDER BY confidence_score DESC
	`, "test-media-2")
	require.NoError(t, err)
	defer rows.Close()

	var lastScore int = 1000 // Start high
	for rows.Next() {
		var tmdbID int
		var title string
		var score int
		err = rows.Scan(&tmdbID, &title, &score)
		require.NoError(t, err)
		assert.LessOrEqual(t, score, lastScore, "candidates should be ordered by confidence descending")
		lastScore = score
	}
}

func TestLinkMetadata_Integration_LinkFromCandidate(t *testing.T) {
	ctx := context.Background()
	db, sqlDB := setupIntegrationTestDatabase(t)

	// Create repositories
	mediaRepo := repository.NewMediaRepository(db)
	movieMetadataRepo := repository.NewSQLiteMovieMetadataRepository(db.DB)
	seriesMetadataRepo := repository.NewSQLiteSeriesMetadataRepository(db.DB)
	seasonMetadataRepo := repository.NewSQLiteSeasonMetadataRepository(db.DB)
	episodeMetadataRepo := repository.NewSQLiteEpisodeMetadataRepository(db.DB)
	candidateRepo := repository.NewSQLiteMetadataCandidateRepository(db.DB)

	// Create a media file with candidates_found status
	media := &domain.MediaFile{
		ID:               "test-media-3",
		Fingerprint:      "fingerprint-3",
		FilePath:         "/movies/Matrix.mp4",
		Filename:         "Matrix.mp4",
		Status:           domain.MediaStatusReady,
		EnrichmentStatus: domain.EnrichmentStatusCandidatesFound,
	}
	err := mediaRepo.Create(ctx, media)
	require.NoError(t, err)

	// Create a candidate manually
	candidate := &domain.MetadataCandidate{
		MediaFileID:     "test-media-3",
		TMDBID:          603,
		CandidateType:   domain.MetadataTypeMovie,
		Title:           "The Matrix",
		ReleaseDate:     "1999-03-30",
		Overview:        "A computer hacker learns about reality.",
		PosterPath:      "/poster.jpg",
		ConfidenceScore: 95,
		Status:          "pending",
	}
	err = candidateRepo.Create(ctx, candidate)
	require.NoError(t, err)

	// Create mock TMDB client
	tmdbClient := &integrationTestTMDBClient{
		movieDetails: &ports.TMDBMovieDetails{
			ID:            603,
			Title:         "The Matrix",
			OriginalTitle: "The Matrix",
			ReleaseDate:   "1999-03-30",
			Overview:      "A computer hacker learns about reality.",
			PosterPath:    "/poster.jpg",
			Runtime:       136,
			Status:        "Released",
			OriginalLang:  "en",
		},
	}

	// Create config
	cfg := config.TMDBConfig{
		Enabled:    true,
		Language:   "en",
		PosterSize: "w500",
	}

	// Create credit and certification repositories for movie linker
	creditRepo := repository.NewSQLiteMovieCreditRepository(db.DB)
	certRepo := repository.NewSQLiteMovieCertificationRepository(db.DB)

	// Create linkers
	movieLinker := linker.NewMovieLinker(
		cfg,
		tmdbClient,
		&integrationTestImageDownloader{},
		movieMetadataRepo,
		creditRepo,
		certRepo,
		nil, // searchRepository - not needed for this test
	)
	episodeLinker := linker.NewEpisodeLinker(
		cfg,
		tmdbClient,
		&integrationTestImageDownloader{},
		seriesMetadataRepo,
		seasonMetadataRepo,
		episodeMetadataRepo,
		nil, // seriesCreditRepository - not needed for this test
		nil, // searchRepository - not needed for this test
	)

	// Create use case
	useCase := NewLinkMetadataUseCase(
		movieLinker,
		episodeLinker,
		mediaRepo,
		candidateRepo,
		nil,
		nil,
		nil,
	)

	// Execute link
	output, err := useCase.Execute(ctx, LinkMetadataInput{
		MediaID:     "test-media-3",
		CandidateID: candidate.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, "test-media-3", output.MediaID)
	assert.Equal(t, domain.MetadataTypeMovie, output.MetadataType)
	assert.Equal(t, "The Matrix", output.Title)

	// Verify media file was updated
	var enrichmentStatus, metadataType string
	var movieMetadataID sql.NullInt64
	err = sqlDB.QueryRow(`
		SELECT enrichment_status, metadata_type, movie_metadata_id 
		FROM media_files WHERE id = ?
	`, "test-media-3").Scan(&enrichmentStatus, &metadataType, &movieMetadataID)
	require.NoError(t, err)
	assert.Equal(t, domain.EnrichmentStatusLinked, enrichmentStatus)
	assert.Equal(t, domain.MetadataTypeMovie, metadataType)
	assert.True(t, movieMetadataID.Valid)

	// Verify candidate status was updated
	var candidateStatus string
	err = sqlDB.QueryRow(`
		SELECT status FROM metadata_candidates WHERE id = ?
	`, candidate.ID).Scan(&candidateStatus)
	require.NoError(t, err)
	assert.Equal(t, "selected", candidateStatus)
}

func TestSkipEnrichment_Integration(t *testing.T) {
	ctx := context.Background()
	db, sqlDB := setupIntegrationTestDatabase(t)

	// Create repositories
	mediaRepo := repository.NewMediaRepository(db)
	candidateRepo := repository.NewSQLiteMetadataCandidateRepository(db.DB)

	// Create a media file with candidates
	media := &domain.MediaFile{
		ID:               "test-media-4",
		Fingerprint:      "fingerprint-4",
		FilePath:         "/movies/Unknown.mp4",
		Filename:         "Unknown.mp4",
		Status:           domain.MediaStatusReady,
		EnrichmentStatus: domain.EnrichmentStatusCandidatesFound,
	}
	err := mediaRepo.Create(ctx, media)
	require.NoError(t, err)

	// Create candidates
	for i := 0; i < 3; i++ {
		candidate := &domain.MetadataCandidate{
			MediaFileID:   "test-media-4",
			TMDBID:        100 + i,
			CandidateType: domain.MetadataTypeMovie,
			Title:         "Unknown Movie",
			Status:        "pending",
		}
		err = candidateRepo.Create(ctx, candidate)
		require.NoError(t, err)
	}

	// Create use case
	useCase := NewSkipEnrichmentUseCase(mediaRepo, candidateRepo)

	// Execute skip
	output, err := useCase.Execute(ctx, SkipEnrichmentInput{
		MediaID: "test-media-4",
	})
	require.NoError(t, err)
	assert.Equal(t, "test-media-4", output.MediaID)

	// Verify media was marked as skipped
	var enrichmentStatus string
	err = sqlDB.QueryRow(`
		SELECT enrichment_status FROM media_files WHERE id = ?
	`, "test-media-4").Scan(&enrichmentStatus)
	require.NoError(t, err)
	assert.Equal(t, domain.EnrichmentStatusSkipped, enrichmentStatus)

	// Verify candidates were rejected
	var pendingCount int
	err = sqlDB.QueryRow(`
		SELECT COUNT(*) FROM metadata_candidates WHERE media_file_id = ? AND status = 'pending'
	`, "test-media-4").Scan(&pendingCount)
	require.NoError(t, err)
	assert.Equal(t, 0, pendingCount, "all candidates should be rejected")
}

func TestResetEnrichment_Integration(t *testing.T) {
	ctx := context.Background()
	db, sqlDB := setupIntegrationTestDatabase(t)

	// Create repositories
	mediaRepo := repository.NewMediaRepository(db)
	movieMetadataRepo := repository.NewSQLiteMovieMetadataRepository(db.DB)
	candidateRepo := repository.NewSQLiteMetadataCandidateRepository(db.DB)

	// Create movie metadata
	movie := domain.NewMovieMetadata(603, "The Matrix")
	err := movieMetadataRepo.Create(ctx, movie)
	require.NoError(t, err)

	// Create a linked media file
	media := &domain.MediaFile{
		ID:               "test-media-5",
		Fingerprint:      "fingerprint-5",
		FilePath:         "/movies/Matrix.mp4",
		Filename:         "Matrix.mp4",
		Status:           domain.MediaStatusReady,
		EnrichmentStatus: domain.EnrichmentStatusLinked,
		MetadataType:     domain.MetadataTypeMovie,
		MovieMetadataID:  &movie.ID,
	}
	err = mediaRepo.Create(ctx, media)
	require.NoError(t, err)

	// Create use case
	useCase := NewResetEnrichmentUseCase(mediaRepo, candidateRepo)

	// Execute reset
	output, err := useCase.Execute(ctx, ResetEnrichmentInput{
		MediaID: "test-media-5",
	})
	require.NoError(t, err)
	assert.Equal(t, "test-media-5", output.MediaID)

	// Verify media was reset
	var enrichmentStatus, metadataType string
	var movieMetadataID sql.NullInt64
	err = sqlDB.QueryRow(`
		SELECT enrichment_status, metadata_type, movie_metadata_id 
		FROM media_files WHERE id = ?
	`, "test-media-5").Scan(&enrichmentStatus, &metadataType, &movieMetadataID)
	require.NoError(t, err)
	assert.Equal(t, domain.EnrichmentStatusPending, enrichmentStatus)
	assert.Equal(t, "", metadataType)
	assert.False(t, movieMetadataID.Valid, "movie_metadata_id should be cleared")
}

func TestGetCandidates_Integration(t *testing.T) {
	ctx := context.Background()
	db, _ := setupIntegrationTestDatabase(t)

	// Create repositories
	mediaRepo := repository.NewMediaRepository(db)
	candidateRepo := repository.NewSQLiteMetadataCandidateRepository(db.DB)

	// Create a media file
	media := &domain.MediaFile{
		ID:               "test-media-6",
		Fingerprint:      "fingerprint-6",
		FilePath:         "/movies/Matrix.mp4",
		Filename:         "Matrix.mp4",
		Status:           domain.MediaStatusReady,
		EnrichmentStatus: domain.EnrichmentStatusCandidatesFound,
	}
	err := mediaRepo.Create(ctx, media)
	require.NoError(t, err)

	// Create candidates with different statuses
	candidates := []struct {
		tmdbID int
		title  string
		status string
		score  int
	}{
		{603, "The Matrix", "pending", 95},
		{604, "The Matrix Reloaded", "pending", 80},
		{605, "The Matrix Revolutions", "rejected", 70},
	}

	for _, c := range candidates {
		candidate := &domain.MetadataCandidate{
			MediaFileID:     "test-media-6",
			TMDBID:          c.tmdbID,
			CandidateType:   domain.MetadataTypeMovie,
			Title:           c.title,
			Status:          c.status,
			ConfidenceScore: c.score,
			PosterPath:      "/poster.jpg",
		}
		err = candidateRepo.Create(ctx, candidate)
		require.NoError(t, err)
	}

	// Create mock TMDB client
	tmdbClient := &integrationTestTMDBClient{}

	// Create config
	cfg := config.TMDBConfig{
		Enabled:    true,
		PosterSize: "w500",
	}

	// Create use case
	useCase := NewGetCandidatesUseCase(cfg, tmdbClient, mediaRepo, candidateRepo)

	// Test getting all candidates
	t.Run("all candidates", func(t *testing.T) {
		output, err := useCase.Execute(ctx, GetCandidatesInput{
			MediaID:     "test-media-6",
			PendingOnly: false,
		})
		require.NoError(t, err)
		assert.Equal(t, 3, output.Count)
		assert.Len(t, output.Candidates, 3)
	})

	// Test getting only pending candidates
	t.Run("pending only", func(t *testing.T) {
		output, err := useCase.Execute(ctx, GetCandidatesInput{
			MediaID:     "test-media-6",
			PendingOnly: true,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, output.Count)
		assert.Len(t, output.Candidates, 2)
		for _, c := range output.Candidates {
			assert.Equal(t, "pending", c.Status)
		}
	})
}
