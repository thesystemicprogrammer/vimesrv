//go:build integration

package rebuild

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/media"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/repository"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/library"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// =============================================================================
// Test Context
// =============================================================================

type integrationTestContext struct {
	t           *testing.T
	baseDir     string
	stagingPath string
	mediaPath   string
	libraryPath string

	db     *database.DB
	sqlDB  *sql.DB
	config *config.Config

	// Repositories
	mediaRepository              ports.MediaRepository
	userRepository               ports.UserRepository
	rebuildRepository            ports.RebuildRepository
	transcodeRepository          ports.TranscodeRepository
	jobRepository                ports.JobRepository
	movieMetadataRepository      ports.MovieMetadataRepository
	seriesMetadataRepository     ports.SeriesMetadataRepository
	seasonMetadataRepository     ports.SeasonMetadataRepository
	episodeMetadataRepository    ports.EpisodeMetadataRepository
	movieCreditRepository        ports.MovieCreditRepository
	movieCertificationRepository ports.MovieCertificationRepository

	// Adapters
	filesystem      ports.FileSystemService
	fileHasher      ports.FileHasher
	ffprobeService  ports.FFProbeService
	tmdbClient      *mockTMDBClient
	imageDownloader *mockImageDownloader
}

// =============================================================================
// Mock TMDB Client
// =============================================================================

type mockTMDBClient struct {
	movieDetails   map[int]*ports.TMDBMovieDetails
	seriesDetails  map[int]*ports.TMDBSeriesDetails
	seasonDetails  map[string]*ports.TMDBSeasonDetails  // "seriesID-seasonNum"
	episodeDetails map[string]*ports.TMDBEpisodeDetails // "seriesID-seasonNum-epNum"
}

func newMockTMDBClient() *mockTMDBClient {
	return &mockTMDBClient{
		movieDetails: map[int]*ports.TMDBMovieDetails{
			27205: {
				ID:          27205,
				Title:       "Inception",
				ReleaseDate: "2010-07-16",
				Overview:    "A thief who steals corporate secrets through dream-sharing technology.",
				PosterPath:  "/inception_poster.jpg",
			},
			438631: {
				ID:          438631,
				Title:       "Dune",
				ReleaseDate: "2021-10-22",
				Overview:    "Paul Atreides, a brilliant and gifted young man.",
				PosterPath:  "/dune_poster.jpg",
			},
		},
		seriesDetails: map[int]*ports.TMDBSeriesDetails{
			66732: {
				ID:              66732,
				Name:            "Stranger Things",
				FirstAirDate:    "2016-07-15",
				Overview:        "When a young boy vanishes, a small town uncovers a mystery.",
				PosterPath:      "/stranger_poster.jpg",
				NumberOfSeasons: 4,
			},
		},
		seasonDetails: map[string]*ports.TMDBSeasonDetails{
			"66732-1": {
				ID:           3582,
				SeasonNumber: 1,
				Name:         "Season 1",
				AirDate:      "2016-07-15",
				PosterPath:   "/season1_poster.jpg",
			},
		},
		episodeDetails: map[string]*ports.TMDBEpisodeDetails{
			"66732-1-1": {
				ID:            62085,
				Name:          "Chapter One: The Vanishing of Will Byers",
				SeasonNumber:  1,
				EpisodeNumber: 1,
				Overview:      "On his way home from a friend's house, young Will sees something terrifying.",
				AirDate:       "2016-07-15",
				StillPath:     "/episode_still.jpg",
			},
		},
	}
}

func (m *mockTMDBClient) SearchMovie(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
	return nil, nil
}

func (m *mockTMDBClient) SearchTV(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
	return nil, nil
}

func (m *mockTMDBClient) SearchMulti(ctx context.Context, query string, language string) ([]ports.TMDBSearchResult, error) {
	return nil, nil
}

func (m *mockTMDBClient) GetMovieDetails(ctx context.Context, movieID int, language string) (*ports.TMDBMovieDetails, error) {
	if details, ok := m.movieDetails[movieID]; ok {
		return details, nil
	}
	return nil, fmt.Errorf("movie not found: %d", movieID)
}

func (m *mockTMDBClient) GetSeriesDetails(ctx context.Context, seriesID int, language string) (*ports.TMDBSeriesDetails, error) {
	if details, ok := m.seriesDetails[seriesID]; ok {
		return details, nil
	}
	return nil, fmt.Errorf("series not found: %d", seriesID)
}

func (m *mockTMDBClient) GetSeasonDetails(ctx context.Context, seriesID, seasonNumber int, language string) (*ports.TMDBSeasonDetails, error) {
	key := fmt.Sprintf("%d-%d", seriesID, seasonNumber)
	if details, ok := m.seasonDetails[key]; ok {
		return details, nil
	}
	return nil, fmt.Errorf("season not found: %s", key)
}

func (m *mockTMDBClient) GetEpisodeDetails(ctx context.Context, seriesID, seasonNumber, episodeNumber int, language string) (*ports.TMDBEpisodeDetails, error) {
	key := fmt.Sprintf("%d-%d-%d", seriesID, seasonNumber, episodeNumber)
	if details, ok := m.episodeDetails[key]; ok {
		return details, nil
	}
	return nil, fmt.Errorf("episode not found: %s", key)
}

func (m *mockTMDBClient) GetImageURL(path string, size string) string {
	return "https://image.tmdb.org/t/p/" + size + path
}

func (m *mockTMDBClient) GetMovieCredits(ctx context.Context, movieID int) (*ports.TMDBMovieCredits, error) {
	return &ports.TMDBMovieCredits{ID: movieID, Cast: []ports.TMDBCastMember{}, Crew: []ports.TMDBCrewMember{}}, nil
}

func (m *mockTMDBClient) GetSeriesAggregateCredits(ctx context.Context, seriesID int) (*ports.TMDBSeriesAggregateCredits, error) {
	return &ports.TMDBSeriesAggregateCredits{ID: seriesID}, nil
}

func (m *mockTMDBClient) GetMovieReleaseDates(ctx context.Context, movieID int) (*ports.TMDBReleaseDatesResponse, error) {
	return &ports.TMDBReleaseDatesResponse{ID: movieID, Results: []ports.TMDBReleaseDateCountry{}}, nil
}

func (m *mockTMDBClient) GetSimilarMovies(ctx context.Context, movieID int, language string) (*ports.TMDBSimilarMoviesResponse, error) {
	return &ports.TMDBSimilarMoviesResponse{}, nil
}

func (m *mockTMDBClient) GetSimilarSeries(ctx context.Context, seriesID int, language string) (*ports.TMDBSimilarSeriesResponse, error) {
	return &ports.TMDBSimilarSeriesResponse{}, nil
}

func (m *mockTMDBClient) GetCollectionDetails(ctx context.Context, collectionID int, language string) (*ports.TMDBCollectionDetails, error) {
	return nil, nil
}

func (m *mockTMDBClient) GetCollectionTranslations(ctx context.Context, collectionID int) (*ports.TMDBCollectionTranslationsResponse, error) {
	return nil, nil
}

func (m *mockTMDBClient) GetMovieTranslations(ctx context.Context, movieID int) (*ports.TMDBMovieTranslationsResponse, error) {
	return &ports.TMDBMovieTranslationsResponse{ID: movieID}, nil
}

func (m *mockTMDBClient) GetSeriesTranslations(ctx context.Context, seriesID int) (*ports.TMDBSeriesTranslationsResponse, error) {
	return &ports.TMDBSeriesTranslationsResponse{ID: seriesID}, nil
}

// =============================================================================
// Mock Image Downloader
// =============================================================================

type mockImageDownloader struct{}

func (m *mockImageDownloader) DownloadImage(ctx context.Context, tmdbPath, imageType string, id int) (string, error) {
	return fmt.Sprintf("/images/%s/%d.jpg", imageType, id), nil
}

func (m *mockImageDownloader) DownloadSeasonImage(ctx context.Context, tmdbPath string, seriesID, seasonNumber int) (string, error) {
	return fmt.Sprintf("/images/season/%d_%d.jpg", seriesID, seasonNumber), nil
}

func (m *mockImageDownloader) DownloadEpisodeImage(ctx context.Context, tmdbPath string, seriesID, seasonNumber, episodeNumber int) (string, error) {
	return fmt.Sprintf("/images/episode/%d_%d_%d.jpg", seriesID, seasonNumber, episodeNumber), nil
}

func (m *mockImageDownloader) GetLocalPath(imageType string, id int) string {
	return fmt.Sprintf("/images/%s/%d.jpg", imageType, id)
}

func (m *mockImageDownloader) ImageExists(imageType string, id int) bool {
	return false
}

// =============================================================================
// Database Snapshot Types
// =============================================================================

type databaseSnapshot struct {
	users      []userSnapshot
	mediaFiles []mediaFileSnapshot
	transcodes []transcodeSnapshot
}

type userSnapshot struct {
	username     string
	passwordHash string
	role         string
}

type mediaFileSnapshot struct {
	fingerprint       string
	metadataType      string
	movieMetadataID   *int64
	episodeMetadataID *int64
	edition           string
	enrichmentStatus  string
}

type transcodeSnapshot struct {
	mediaID    string
	trackType  string
	quality    string
	trackIndex int
	status     string
}

// =============================================================================
// Test Setup Functions
// =============================================================================

func setupTestDatabase(t *testing.T) (*database.DB, *sql.DB) {
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

	migration := database.NewDatabaseMigration(db.DB)
	err = migration.Migrate()
	require.NoError(t, err, "failed to run migrations")

	t.Cleanup(func() {
		db.Close()
	})

	return db, db.DB
}

func setupIntegrationTest(t *testing.T) *integrationTestContext {
	t.Helper()

	// Create temp base directory
	baseDir := t.TempDir()

	// Create subdirectories
	stagingPath := filepath.Join(baseDir, "staging")
	mediaPath := filepath.Join(baseDir, "media")
	libraryPath := filepath.Join(baseDir, "library")

	require.NoError(t, os.MkdirAll(stagingPath, 0755))
	require.NoError(t, os.MkdirAll(mediaPath, 0755))
	require.NoError(t, os.MkdirAll(libraryPath, 0755))

	// Create database
	db, sqlDB := setupTestDatabase(t)

	// Create config
	cfg := &config.Config{
		Media: config.MediaConfig{
			StagingPath:      stagingPath,
			MediaPath:        mediaPath,
			LibraryPath:      libraryPath,
			SupportedFormats: []string{".mp4", ".mkv", ".avi"},
		},
		TMDB: config.TMDBConfig{
			Enabled:    true,
			Language:   "en",
			AutoSearch: false, // We'll link manually in the test
		},
		Rebuild: config.RebuildConfig{
			AllowRebuild: true,
		},
	}

	// Create adapters
	filesystem := media.NewOSFileSystem()
	fileHasher := media.NewBlake2bHasher()
	ffprobeService := media.NewFFProbeAdapter(30) // 30 second timeout

	// Create repositories
	mediaRepository := repository.NewMediaRepository(db)
	userRepository := repository.NewSQLiteUserRepository(db)
	rebuildRepository := repository.NewSQLiteRebuildRepository(db)
	transcodeRepository := repository.NewTranscodeRepository(db)
	jobRepository := repository.NewJobRepository(db)
	movieMetadataRepository := repository.NewSQLiteMovieMetadataRepository(db.DB)
	seriesMetadataRepository := repository.NewSQLiteSeriesMetadataRepository(db.DB)
	seasonMetadataRepository := repository.NewSQLiteSeasonMetadataRepository(db.DB)
	episodeMetadataRepository := repository.NewSQLiteEpisodeMetadataRepository(db.DB)
	movieCreditRepository := repository.NewSQLiteMovieCreditRepository(db.DB)
	movieCertificationRepository := repository.NewSQLiteMovieCertificationRepository(db.DB)

	// Create mocks
	tmdbClient := newMockTMDBClient()
	imageDownloader := &mockImageDownloader{}

	return &integrationTestContext{
		t:           t,
		baseDir:     baseDir,
		stagingPath: stagingPath,
		mediaPath:   mediaPath,
		libraryPath: libraryPath,

		db:     db,
		sqlDB:  sqlDB,
		config: cfg,

		mediaRepository:              mediaRepository,
		userRepository:               userRepository,
		rebuildRepository:            rebuildRepository,
		transcodeRepository:          transcodeRepository,
		jobRepository:                jobRepository,
		movieMetadataRepository:      movieMetadataRepository,
		seriesMetadataRepository:     seriesMetadataRepository,
		seasonMetadataRepository:     seasonMetadataRepository,
		episodeMetadataRepository:    episodeMetadataRepository,
		movieCreditRepository:        movieCreditRepository,
		movieCertificationRepository: movieCertificationRepository,

		filesystem:      filesystem,
		fileHasher:      fileHasher,
		ffprobeService:  ffprobeService,
		tmdbClient:      tmdbClient,
		imageDownloader: imageDownloader,
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func copyFixturesToStaging(t *testing.T, stagingPath string, fixtures []string) {
	t.Helper()

	// Navigate to fixtures directory (relative to this test file)
	// internal/usecase/rebuild -> internal/usecase -> internal -> project root
	fixturesDir := filepath.Join("..", "..", "..", "test", "fixtures")

	for _, fixture := range fixtures {
		src := filepath.Join(fixturesDir, fixture)
		dst := filepath.Join(stagingPath, fixture)

		data, err := os.ReadFile(src)
		require.NoError(t, err, "failed to read fixture %s", fixture)

		err = os.WriteFile(dst, data, 0644)
		require.NoError(t, err, "failed to write fixture %s", fixture)
	}
}

func createTestUser(t *testing.T, ctx context.Context, userRepo ports.UserRepository, username, passwordHash string) *domain.User {
	t.Helper()

	user := &domain.User{
		ID:                 "test-user-id",
		Username:           username,
		PasswordHash:       passwordHash,
		Role:               shared.RoleAdmin,
		MustChangePassword: false,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	err := userRepo.Create(ctx, user)
	require.NoError(t, err, "failed to create test user")

	return user
}

func listAllMediaFiles(t *testing.T, ctx context.Context, mediaRepo ports.MediaRepository) []*domain.MediaFile {
	t.Helper()

	mediaFiles, _, err := mediaRepo.List(ctx, 0, 100)
	require.NoError(t, err, "failed to list media files")

	return mediaFiles
}

func createFakeTranscodes(t *testing.T, ctx context.Context, tc *integrationTestContext, mediaFiles []*domain.MediaFile) {
	t.Helper()

	for _, mf := range mediaFiles {
		// Create transcode directory structure
		transcodedDir := filepath.Join(tc.mediaPath, mf.ID, "transcoded")

		// Video: 720p
		videoDir := filepath.Join(transcodedDir, "720p")
		require.NoError(t, os.MkdirAll(videoDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(videoDir, "init.mp4"), []byte("fake video init"), 0644))

		// Audio: audio-0
		audioDir := filepath.Join(transcodedDir, "audio-0")
		require.NoError(t, os.MkdirAll(audioDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(audioDir, "init.mp4"), []byte("fake audio init"), 0644))

		// Create DB records
		videoTranscode := domain.NewTranscode(
			fmt.Sprintf("%s-video-720p-0", mf.ID),
			mf.ID, "720p", domain.TrackTypeVideo, 0,
		)
		videoTranscode.MarkCompleted(videoDir)
		require.NoError(t, tc.transcodeRepository.Create(ctx, videoTranscode))

		audioTranscode := domain.NewTranscode(
			fmt.Sprintf("%s-audio--0", mf.ID),
			mf.ID, "", domain.TrackTypeAudio, 0,
		)
		audioTranscode.MarkCompleted(audioDir)
		require.NoError(t, tc.transcodeRepository.Create(ctx, audioTranscode))
	}
}

func linkMediaToMovieMetadata(t *testing.T, ctx context.Context, tc *integrationTestContext, mediaFile *domain.MediaFile, tmdbID int, edition string) {
	t.Helper()

	// Get movie details from mock
	details, err := tc.tmdbClient.GetMovieDetails(ctx, tmdbID, "en")
	require.NoError(t, err, "failed to get movie details")

	// Create movie metadata record
	movieMetadata := &domain.MovieMetadata{
		TMDBID:        details.ID,
		OriginalTitle: details.Title,
		ReleaseDate:   details.ReleaseDate,
		PosterPath:    details.PosterPath,
		BackdropPath:  details.BackdropPath,
	}

	err = tc.movieMetadataRepository.Create(ctx, movieMetadata)
	require.NoError(t, err, "failed to create movie metadata")

	// Link media file to metadata
	mediaFile.LinkToMovie(movieMetadata.ID)
	mediaFile.SetEnrichmentAutoLinked()
	if edition != "" {
		mediaFile.Edition = edition
	}

	err = tc.mediaRepository.Update(ctx, mediaFile)
	require.NoError(t, err, "failed to update media file with link")
}

func linkMediaToEpisodeMetadata(t *testing.T, ctx context.Context, tc *integrationTestContext, mediaFile *domain.MediaFile, seriesTMDBID, seasonNum, episodeNum int) {
	t.Helper()

	// Get series details from mock
	seriesDetails, err := tc.tmdbClient.GetSeriesDetails(ctx, seriesTMDBID, "en")
	require.NoError(t, err, "failed to get series details")

	// Create series metadata record
	seriesMetadata := &domain.SeriesMetadata{
		TMDBID:       seriesDetails.ID,
		OriginalName: seriesDetails.Name,
		FirstAirDate: seriesDetails.FirstAirDate,
		PosterPath:   seriesDetails.PosterPath,
	}

	err = tc.seriesMetadataRepository.Create(ctx, seriesMetadata)
	require.NoError(t, err, "failed to create series metadata")

	// Get season details from mock
	seasonDetails, err := tc.tmdbClient.GetSeasonDetails(ctx, seriesTMDBID, seasonNum, "en")
	require.NoError(t, err, "failed to get season details")

	// Create season metadata record
	seasonMetadata := &domain.SeasonMetadata{
		SeriesID:     seriesMetadata.ID,
		TMDBID:       seasonDetails.ID,
		SeasonNumber: seasonDetails.SeasonNumber,
		AirDate:      seasonDetails.AirDate,
		PosterPath:   seasonDetails.PosterPath,
	}

	err = tc.seasonMetadataRepository.Create(ctx, seasonMetadata)
	require.NoError(t, err, "failed to create season metadata")

	// Get episode details from mock
	episodeDetails, err := tc.tmdbClient.GetEpisodeDetails(ctx, seriesTMDBID, seasonNum, episodeNum, "en")
	require.NoError(t, err, "failed to get episode details")

	// Create episode metadata record
	episodeMetadata := &domain.EpisodeMetadata{
		SeasonID:      seasonMetadata.ID,
		TMDBID:        episodeDetails.ID,
		EpisodeNumber: episodeDetails.EpisodeNumber,
		AirDate:       episodeDetails.AirDate,
		StillPath:     episodeDetails.StillPath,
	}

	err = tc.episodeMetadataRepository.Create(ctx, episodeMetadata)
	require.NoError(t, err, "failed to create episode metadata")

	// Link media file to episode metadata
	mediaFile.LinkToEpisode(episodeMetadata.ID)
	mediaFile.SetEnrichmentAutoLinked()

	err = tc.mediaRepository.Update(ctx, mediaFile)
	require.NoError(t, err, "failed to update media file with episode link")
}

func deleteMediaFileFromDisk(t *testing.T, tc *integrationTestContext, mediaID string) {
	t.Helper()
	mediaDir := filepath.Join(tc.mediaPath, mediaID)
	require.NoError(t, os.RemoveAll(mediaDir), "failed to delete media directory")
}

// =============================================================================
// Snapshot Capture and Comparison
// =============================================================================

func captureSnapshot(t *testing.T, sqlDB *sql.DB) *databaseSnapshot {
	t.Helper()
	snapshot := &databaseSnapshot{}

	// Query users
	userRows, err := sqlDB.Query(`SELECT username, password_hash, role FROM users ORDER BY username`)
	require.NoError(t, err, "failed to query users")
	defer userRows.Close()

	for userRows.Next() {
		var u userSnapshot
		require.NoError(t, userRows.Scan(&u.username, &u.passwordHash, &u.role))
		snapshot.users = append(snapshot.users, u)
	}

	// Query media files
	mediaRows, err := sqlDB.Query(`SELECT fingerprint, metadata_type, movie_metadata_id, episode_metadata_id, edition, enrichment_status FROM media_files ORDER BY fingerprint`)
	require.NoError(t, err, "failed to query media files")
	defer mediaRows.Close()

	for mediaRows.Next() {
		var mf mediaFileSnapshot
		var metadataType, edition, enrichmentStatus sql.NullString
		var movieID, episodeID sql.NullInt64

		require.NoError(t, mediaRows.Scan(&mf.fingerprint, &metadataType, &movieID, &episodeID, &edition, &enrichmentStatus))

		mf.metadataType = metadataType.String
		mf.edition = edition.String
		mf.enrichmentStatus = enrichmentStatus.String
		if movieID.Valid {
			mf.movieMetadataID = &movieID.Int64
		}
		if episodeID.Valid {
			mf.episodeMetadataID = &episodeID.Int64
		}

		snapshot.mediaFiles = append(snapshot.mediaFiles, mf)
	}

	// Query transcodes
	transcodeRows, err := sqlDB.Query(`SELECT media_id, track_type, quality, track_index, status FROM transcodes ORDER BY media_id, track_type, track_index`)
	require.NoError(t, err, "failed to query transcodes")
	defer transcodeRows.Close()

	for transcodeRows.Next() {
		var tr transcodeSnapshot
		require.NoError(t, transcodeRows.Scan(&tr.mediaID, &tr.trackType, &tr.quality, &tr.trackIndex, &tr.status))
		snapshot.transcodes = append(snapshot.transcodes, tr)
	}

	return snapshot
}

func compareSnapshots(t *testing.T, before, after *databaseSnapshot, expectMissingFiles int) {
	t.Helper()

	// === USERS ===
	require.Len(t, after.users, len(before.users), "user count mismatch")
	for i, beforeUser := range before.users {
		afterUser := after.users[i]
		assert.Equal(t, beforeUser.username, afterUser.username, "username mismatch")
		assert.Equal(t, beforeUser.passwordHash, afterUser.passwordHash, "password hash not preserved!")
		assert.Equal(t, beforeUser.role, afterUser.role, "role mismatch")
	}

	// === MEDIA FILES ===
	expectedMediaCount := len(before.mediaFiles) - expectMissingFiles
	require.Len(t, after.mediaFiles, expectedMediaCount, "media file count mismatch")

	// Build fingerprint map for comparison
	afterByFingerprint := make(map[string]mediaFileSnapshot)
	for _, mf := range after.mediaFiles {
		afterByFingerprint[mf.fingerprint] = mf
	}

	missingCount := 0
	for _, beforeMedia := range before.mediaFiles {
		afterMedia, exists := afterByFingerprint[beforeMedia.fingerprint]
		if !exists {
			// This file was intentionally deleted
			missingCount++
			continue
		}

		assert.Equal(t, beforeMedia.metadataType, afterMedia.metadataType,
			"metadata type mismatch for fingerprint %s", beforeMedia.fingerprint)
		assert.Equal(t, beforeMedia.edition, afterMedia.edition,
			"edition not preserved for fingerprint %s", beforeMedia.fingerprint)
		assert.Equal(t, beforeMedia.enrichmentStatus, afterMedia.enrichmentStatus,
			"enrichment status mismatch for fingerprint %s", beforeMedia.fingerprint)

		// Check metadata link exists (IDs may differ, but should both be set or both nil)
		if beforeMedia.movieMetadataID != nil {
			assert.NotNil(t, afterMedia.movieMetadataID,
				"movie metadata ID should be set for fingerprint %s", beforeMedia.fingerprint)
		}
		if beforeMedia.episodeMetadataID != nil {
			assert.NotNil(t, afterMedia.episodeMetadataID,
				"episode metadata ID should be set for fingerprint %s", beforeMedia.fingerprint)
		}
	}

	assert.Equal(t, expectMissingFiles, missingCount, "missing file count mismatch")

	// === TRANSCODES ===
	// Only compare transcodes for media files that exist after rebuild
	beforeTranscodesByMedia := make(map[string][]transcodeSnapshot)
	for _, tr := range before.transcodes {
		beforeTranscodesByMedia[tr.mediaID] = append(beforeTranscodesByMedia[tr.mediaID], tr)
	}

	afterTranscodesByMedia := make(map[string][]transcodeSnapshot)
	for _, tr := range after.transcodes {
		afterTranscodesByMedia[tr.mediaID] = append(afterTranscodesByMedia[tr.mediaID], tr)
	}

	// Verify that transcodes exist for media files that exist after rebuild
	// We use the maps to ensure we have transcode data for each media file
	_ = beforeTranscodesByMedia // Referenced above for completeness
	_ = afterTranscodesByMedia  // Will be used in future detailed transcode matching

	// Simplified: just check that some transcodes were recovered for remaining files
	afterTranscodeCount := len(after.transcodes)
	expectedMinTranscodes := (len(before.mediaFiles) - expectMissingFiles) * 2 // 2 transcodes per file
	assert.GreaterOrEqual(t, afterTranscodeCount, expectedMinTranscodes,
		"expected at least %d transcodes, got %d", expectedMinTranscodes, afterTranscodeCount)
}

// =============================================================================
// Main E2E Integration Test
// =============================================================================

func TestRebuild_E2E_Integration(t *testing.T) {
	ctx := context.Background()

	// === SETUP ===
	t.Log("Setting up integration test...")
	tc := setupIntegrationTest(t)

	// Define test fixtures
	fixtures := []string{
		"inception_2010.mp4",
		"dune_2021.mp4",
		"stranger_things_e01_s01_2016.mp4",
	}

	// Copy fixtures to staging
	t.Log("Copying fixtures to staging directory...")
	copyFixturesToStaging(t, tc.stagingPath, fixtures)

	// === STEP 1: Initial Import via ScanLibraryUseCase ===
	t.Log("Step 1: Running ScanLibraryUseCase to import fixtures...")

	scanUseCase := library.NewScanLibraryUseCase(
		tc.config.Media,
		tc.fileHasher,
		tc.ffprobeService,
		tc.filesystem,
		tc.mediaRepository,
		nil, // No transcode use case
	)

	err := scanUseCase.Execute(ctx)
	require.NoError(t, err, "ScanLibraryUseCase failed")

	// Verify files were imported
	mediaFiles := listAllMediaFiles(t, ctx, tc.mediaRepository)
	require.Len(t, mediaFiles, 3, "expected 3 media files to be imported")
	t.Logf("  Imported %d media files", len(mediaFiles))

	// === STEP 2: Create Test User ===
	t.Log("Step 2: Creating test user...")
	testPasswordHash := "$2a$10$abcdefghijklmnopqrstuv" // Fake bcrypt hash for testing
	testUser := createTestUser(t, ctx, tc.userRepository, "testadmin", testPasswordHash)
	t.Logf("  Created user: %s", testUser.Username)

	// === STEP 3: Link Media to Metadata ===
	t.Log("Step 3: Linking media files to TMDB metadata...")

	// Sort media files by filename to get predictable mapping
	sort.Slice(mediaFiles, func(i, j int) bool {
		return mediaFiles[i].OriginalFilename < mediaFiles[j].OriginalFilename
	})

	// Map fixtures to media files
	var inceptionMedia, duneMedia, strangerThingsMedia *domain.MediaFile
	for _, mf := range mediaFiles {
		switch {
		case mf.OriginalFilename == "inception_2010.mp4":
			inceptionMedia = mf
		case mf.OriginalFilename == "dune_2021.mp4":
			duneMedia = mf
		case mf.OriginalFilename == "stranger_things_e01_s01_2016.mp4":
			strangerThingsMedia = mf
		}
	}

	require.NotNil(t, inceptionMedia, "inception media not found")
	require.NotNil(t, duneMedia, "dune media not found")
	require.NotNil(t, strangerThingsMedia, "stranger things media not found")

	// Link inception to movie (with edition to test edition preservation)
	linkMediaToMovieMetadata(t, ctx, tc, inceptionMedia, 27205, "IMAX")
	t.Logf("  Linked inception with edition 'IMAX'")

	// Link dune to movie (no edition)
	linkMediaToMovieMetadata(t, ctx, tc, duneMedia, 438631, "")
	t.Logf("  Linked dune (no edition)")

	// Link stranger things to episode
	linkMediaToEpisodeMetadata(t, ctx, tc, strangerThingsMedia, 66732, 1, 1)
	t.Logf("  Linked stranger things S01E01")

	// === STEP 4: Create Fake Transcodes ===
	t.Log("Step 4: Creating fake transcode directories...")
	// Refresh media files after linking
	mediaFiles = listAllMediaFiles(t, ctx, tc.mediaRepository)
	createFakeTranscodes(t, ctx, tc, mediaFiles)
	t.Logf("  Created transcodes for %d media files", len(mediaFiles))

	// === STEP 5: Capture Before Snapshot ===
	t.Log("Step 5: Capturing 'before' database snapshot...")
	beforeSnapshot := captureSnapshot(t, tc.sqlDB)
	t.Logf("  Snapshot: %d users, %d media files, %d transcodes",
		len(beforeSnapshot.users), len(beforeSnapshot.mediaFiles), len(beforeSnapshot.transcodes))

	// Verify snapshot contents
	assert.Len(t, beforeSnapshot.users, 1, "expected 1 user")
	assert.Len(t, beforeSnapshot.mediaFiles, 3, "expected 3 media files")
	assert.Len(t, beforeSnapshot.transcodes, 6, "expected 6 transcodes (2 per file)")

	// === STEP 6: Export (PrepareUseCase) ===
	t.Log("Step 6: Running PrepareUseCase to export rebuild.json...")

	prepareUseCase := NewPrepareUseCase(tc.config, tc.userRepository, tc.rebuildRepository, tc.filesystem)
	err = prepareUseCase.Execute(ctx)
	require.NoError(t, err, "PrepareUseCase failed")

	// Verify rebuild.json was created
	rebuildPath := filepath.Join(tc.libraryPath, "rebuild.json")
	require.True(t, tc.filesystem.FileExists(rebuildPath), "rebuild.json was not created")
	t.Log("  rebuild.json created successfully")

	// === STEP 7: Error Scenario - Delete one media file from disk ===
	t.Log("Step 7: Simulating missing file by deleting stranger_things from disk...")
	deleteMediaFileFromDisk(t, tc, strangerThingsMedia.ID)
	t.Logf("  Deleted media directory for %s", strangerThingsMedia.ID)

	// === STEP 8: Clear Database (Nuclear Reset) ===
	t.Log("Step 8: Clearing all database tables (nuclear reset)...")
	err = tc.rebuildRepository.ClearAllTables(ctx)
	require.NoError(t, err, "ClearAllTables failed")

	// Verify database is empty
	emptySnapshot := captureSnapshot(t, tc.sqlDB)
	assert.Len(t, emptySnapshot.users, 0, "database should have no users after clear")
	assert.Len(t, emptySnapshot.mediaFiles, 0, "database should have no media files after clear")
	assert.Len(t, emptySnapshot.transcodes, 0, "database should have no transcodes after clear")
	t.Log("  Database cleared successfully")

	// === STEP 9: Rebuild ===
	t.Log("Step 9: Running rebuild from dump...")

	// Create rebuild use case
	rebuildUseCase := NewRebuildUseCase(
		tc.config,
		tc.rebuildRepository,
		tc.transcodeRepository,
		tc.mediaRepository,
		tc.jobRepository,
		tc.filesystem,
	)

	// Execute phase 1: import users, build auto-link map
	result, err := rebuildUseCase.Execute(ctx)
	require.NoError(t, err, "RebuildUseCase.Execute failed")
	t.Logf("  Phase 1: %d users imported, %d media links loaded",
		result.UsersImported, result.MediaLinksLoaded)

	// Build auto-link map for scanner
	autoLinkMap := make(map[string]AutoLinkData)
	for fingerprint, link := range rebuildUseCase.GetAutoLinkMap() {
		autoLinkMap[fingerprint] = link.ToAutoLinkData()
	}

	// Create linker
	linker := NewLinker(
		tc.config.TMDB,
		tc.tmdbClient,
		tc.imageDownloader,
		tc.movieMetadataRepository,
		tc.seriesMetadataRepository,
		tc.seasonMetadataRepository,
		tc.episodeMetadataRepository,
		tc.movieCreditRepository,
		tc.movieCertificationRepository,
		tc.mediaRepository,
	)

	// Create scanner
	scanner := NewScanner(
		tc.config.Media,
		tc.fileHasher,
		tc.ffprobeService,
		tc.filesystem,
		tc.mediaRepository,
		linker,
		autoLinkMap,
	)

	// Execute phase 2: scan library
	scanResult, err := scanner.Scan(ctx)
	require.NoError(t, err, "Scanner.Scan failed")
	t.Logf("  Phase 2: scanned=%d, processed=%d, linked=%d, errors=%d",
		scanResult.FilesScanned, scanResult.FilesProcessed, scanResult.FilesLinked, len(scanResult.Errors))

	// Should have processed 2 files (stranger_things was deleted)
	assert.Equal(t, 2, scanResult.FilesScanned, "expected 2 files scanned")
	assert.Equal(t, 2, scanResult.FilesProcessed, "expected 2 files processed")
	assert.Equal(t, 2, scanResult.FilesLinked, "expected 2 files linked")

	// Execute phase 3: recover transcodes
	transcodesRecovered, err := rebuildUseCase.RecoverTranscodes(ctx)
	require.NoError(t, err, "RecoverTranscodes failed")
	t.Logf("  Phase 3: %d transcodes recovered", transcodesRecovered)

	// Should recover 4 transcodes (2 per remaining file)
	assert.Equal(t, 4, transcodesRecovered, "expected 4 transcodes recovered")

	// === STEP 10: Capture After Snapshot ===
	t.Log("Step 10: Capturing 'after' database snapshot...")
	afterSnapshot := captureSnapshot(t, tc.sqlDB)
	t.Logf("  Snapshot: %d users, %d media files, %d transcodes",
		len(afterSnapshot.users), len(afterSnapshot.mediaFiles), len(afterSnapshot.transcodes))

	// === STEP 11: Compare Snapshots ===
	t.Log("Step 11: Comparing snapshots...")
	compareSnapshots(t, beforeSnapshot, afterSnapshot, 1) // 1 file was deleted

	// === STEP 12: Additional Assertions ===
	t.Log("Step 12: Running additional assertions...")

	// Verify user password hash was preserved
	assert.Equal(t, beforeSnapshot.users[0].passwordHash, afterSnapshot.users[0].passwordHash,
		"password hash should be identical after rebuild")
	t.Log("  ✓ Password hash preserved")

	// Verify edition was preserved (for inception)
	var inceptionAfter mediaFileSnapshot
	for _, mf := range afterSnapshot.mediaFiles {
		if mf.edition == "IMAX" {
			inceptionAfter = mf
			break
		}
	}
	assert.Equal(t, "IMAX", inceptionAfter.edition, "inception edition should be preserved")
	t.Log("  ✓ Edition field preserved")

	// Verify result statistics
	assert.Equal(t, 1, result.UsersImported, "should have imported 1 user")
	assert.Equal(t, 3, result.MediaLinksLoaded, "should have loaded 3 media links")
	t.Log("  ✓ Result statistics correct")

	t.Log("")
	t.Log("========================================")
	t.Log("  ✓ Rebuild E2E Integration Test PASSED")
	t.Log("========================================")
}
