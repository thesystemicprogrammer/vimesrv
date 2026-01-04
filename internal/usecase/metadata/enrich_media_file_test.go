package metadata

import (
	"context"
	"errors"
	"testing"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// Mock implementations

type mockFilenameParser struct {
	parseFunc func(filename string) *ports.ParsedFilename
}

func (m *mockFilenameParser) Parse(filename string) *ports.ParsedFilename {
	if m.parseFunc != nil {
		return m.parseFunc(filename)
	}
	return &ports.ParsedFilename{Title: "Test", CleanTitle: "test"}
}

type mockTMDBClient struct {
	searchMovieFunc       func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error)
	searchTVFunc          func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error)
	searchMultiFunc       func(ctx context.Context, query string, language string) ([]ports.TMDBSearchResult, error)
	getMovieDetailsFunc   func(ctx context.Context, movieID int, language string) (*ports.TMDBMovieDetails, error)
	getSeriesDetailsFunc  func(ctx context.Context, seriesID int, language string) (*ports.TMDBSeriesDetails, error)
	getSeasonDetailsFunc  func(ctx context.Context, seriesID, seasonNumber int, language string) (*ports.TMDBSeasonDetails, error)
	getEpisodeDetailsFunc func(ctx context.Context, seriesID, seasonNumber, episodeNumber int, language string) (*ports.TMDBEpisodeDetails, error)
}

func (m *mockTMDBClient) SearchMovie(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
	if m.searchMovieFunc != nil {
		return m.searchMovieFunc(ctx, query, year, language)
	}
	return nil, nil
}

func (m *mockTMDBClient) SearchTV(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
	if m.searchTVFunc != nil {
		return m.searchTVFunc(ctx, query, year, language)
	}
	return nil, nil
}

func (m *mockTMDBClient) SearchMulti(ctx context.Context, query string, language string) ([]ports.TMDBSearchResult, error) {
	if m.searchMultiFunc != nil {
		return m.searchMultiFunc(ctx, query, language)
	}
	return nil, nil
}

func (m *mockTMDBClient) GetMovieDetails(ctx context.Context, movieID int, language string) (*ports.TMDBMovieDetails, error) {
	if m.getMovieDetailsFunc != nil {
		return m.getMovieDetailsFunc(ctx, movieID, language)
	}
	return nil, nil
}

func (m *mockTMDBClient) GetSeriesDetails(ctx context.Context, seriesID int, language string) (*ports.TMDBSeriesDetails, error) {
	if m.getSeriesDetailsFunc != nil {
		return m.getSeriesDetailsFunc(ctx, seriesID, language)
	}
	return nil, nil
}

func (m *mockTMDBClient) GetSeasonDetails(ctx context.Context, seriesID, seasonNumber int, language string) (*ports.TMDBSeasonDetails, error) {
	if m.getSeasonDetailsFunc != nil {
		return m.getSeasonDetailsFunc(ctx, seriesID, seasonNumber, language)
	}
	return nil, nil
}

func (m *mockTMDBClient) GetEpisodeDetails(ctx context.Context, seriesID, seasonNumber, episodeNumber int, language string) (*ports.TMDBEpisodeDetails, error) {
	if m.getEpisodeDetailsFunc != nil {
		return m.getEpisodeDetailsFunc(ctx, seriesID, seasonNumber, episodeNumber, language)
	}
	return nil, nil
}

func (m *mockTMDBClient) GetImageURL(path string, size string) string {
	return "https://image.tmdb.org/t/p/" + size + path
}

func (m *mockTMDBClient) GetMovieCredits(ctx context.Context, movieID int) (*ports.TMDBMovieCredits, error) {
	return nil, nil
}

func (m *mockTMDBClient) GetSeriesAggregateCredits(ctx context.Context, seriesID int) (*ports.TMDBSeriesAggregateCredits, error) {
	return nil, nil
}

func (m *mockTMDBClient) GetMovieReleaseDates(ctx context.Context, movieID int) (*ports.TMDBReleaseDatesResponse, error) {
	return nil, nil
}

func (m *mockTMDBClient) GetSimilarMovies(ctx context.Context, movieID int, language string) (*ports.TMDBSimilarMoviesResponse, error) {
	return nil, nil
}

func (m *mockTMDBClient) GetSimilarSeries(ctx context.Context, seriesID int, language string) (*ports.TMDBSimilarSeriesResponse, error) {
	return nil, nil
}

func (m *mockTMDBClient) GetCollectionDetails(ctx context.Context, collectionID int, language string) (*ports.TMDBCollectionDetails, error) {
	return nil, nil
}

func (m *mockTMDBClient) GetCollectionTranslations(ctx context.Context, collectionID int) (*ports.TMDBCollectionTranslationsResponse, error) {
	return nil, nil
}

func (m *mockTMDBClient) GetMovieTranslations(ctx context.Context, movieID int) (*ports.TMDBMovieTranslationsResponse, error) {
	return nil, nil
}

func (m *mockTMDBClient) GetSeriesTranslations(ctx context.Context, seriesID int) (*ports.TMDBSeriesTranslationsResponse, error) {
	return nil, nil
}

type mockImageDownloader struct {
	downloadImageFunc        func(ctx context.Context, tmdbPath string, imageType string, id int) (string, error)
	downloadSeasonImageFunc  func(ctx context.Context, tmdbPath string, seriesID int, seasonNumber int) (string, error)
	downloadEpisodeImageFunc func(ctx context.Context, tmdbPath string, seriesID int, seasonNumber int, episodeNumber int) (string, error)
}

func (m *mockImageDownloader) DownloadImage(ctx context.Context, tmdbPath string, imageType string, id int) (string, error) {
	if m.downloadImageFunc != nil {
		return m.downloadImageFunc(ctx, tmdbPath, imageType, id)
	}
	return "/path/to/image.jpg", nil
}

func (m *mockImageDownloader) DownloadSeasonImage(ctx context.Context, tmdbPath string, seriesID int, seasonNumber int) (string, error) {
	if m.downloadSeasonImageFunc != nil {
		return m.downloadSeasonImageFunc(ctx, tmdbPath, seriesID, seasonNumber)
	}
	return "/path/to/season.jpg", nil
}

func (m *mockImageDownloader) DownloadEpisodeImage(ctx context.Context, tmdbPath string, seriesID int, seasonNumber int, episodeNumber int) (string, error) {
	if m.downloadEpisodeImageFunc != nil {
		return m.downloadEpisodeImageFunc(ctx, tmdbPath, seriesID, seasonNumber, episodeNumber)
	}
	return "/path/to/episode.jpg", nil
}

func (m *mockImageDownloader) GetLocalPath(imageType string, id int) string {
	return "/path/to/local"
}

func (m *mockImageDownloader) ImageExists(imageType string, id int) bool {
	return false
}

type mockMediaRepository struct {
	getFunc                 func(ctx context.Context, id string) (*domain.MediaFile, error)
	updateFunc              func(ctx context.Context, media *domain.MediaFile) error
	createFunc              func(ctx context.Context, media *domain.MediaFile) error
	findByFingerprintFunc   func(ctx context.Context, fingerprint string) (*domain.MediaFile, error)
	existsByFingerprintFunc func(ctx context.Context, fingerprint string) (bool, error)
	listFunc                func(ctx context.Context, page, perPage int) ([]*domain.MediaFile, int, error)
}

func (m *mockMediaRepository) Get(ctx context.Context, id string) (*domain.MediaFile, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockMediaRepository) Update(ctx context.Context, media *domain.MediaFile) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, media)
	}
	return nil
}

func (m *mockMediaRepository) Create(ctx context.Context, media *domain.MediaFile) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, media)
	}
	return nil
}

func (m *mockMediaRepository) FindByFingerprint(ctx context.Context, fingerprint string) (*domain.MediaFile, error) {
	if m.findByFingerprintFunc != nil {
		return m.findByFingerprintFunc(ctx, fingerprint)
	}
	return nil, nil
}

func (m *mockMediaRepository) ExistsByFingerprint(ctx context.Context, fingerprint string) (bool, error) {
	if m.existsByFingerprintFunc != nil {
		return m.existsByFingerprintFunc(ctx, fingerprint)
	}
	return false, nil
}

func (m *mockMediaRepository) List(ctx context.Context, page, perPage int) ([]*domain.MediaFile, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, page, perPage)
	}
	return nil, 0, nil
}

type mockMovieMetadataRepository struct {
	getByTMDBIDFunc       func(ctx context.Context, tmdbID int) (*domain.MovieMetadata, error)
	createFunc            func(ctx context.Context, metadata *domain.MovieMetadata) error
	createTranslationFunc func(ctx context.Context, translation *domain.MovieMetadataTranslation) error
}

func (m *mockMovieMetadataRepository) Get(ctx context.Context, id int64) (*domain.MovieMetadata, error) {
	return nil, nil
}

func (m *mockMovieMetadataRepository) GetByTMDBID(ctx context.Context, tmdbID int) (*domain.MovieMetadata, error) {
	if m.getByTMDBIDFunc != nil {
		return m.getByTMDBIDFunc(ctx, tmdbID)
	}
	return nil, nil
}

func (m *mockMovieMetadataRepository) Create(ctx context.Context, metadata *domain.MovieMetadata) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, metadata)
	}
	metadata.ID = 1
	return nil
}

func (m *mockMovieMetadataRepository) Update(ctx context.Context, metadata *domain.MovieMetadata) error {
	return nil
}

func (m *mockMovieMetadataRepository) Delete(ctx context.Context, id int64) error {
	return nil
}

func (m *mockMovieMetadataRepository) ExistsByTMDBID(ctx context.Context, tmdbID int) (bool, error) {
	return false, nil
}

func (m *mockMovieMetadataRepository) CreateTranslation(ctx context.Context, translation *domain.MovieMetadataTranslation) error {
	if m.createTranslationFunc != nil {
		return m.createTranslationFunc(ctx, translation)
	}
	return nil
}

func (m *mockMovieMetadataRepository) GetTranslation(ctx context.Context, movieMetadataID int64, language string) (*domain.MovieMetadataTranslation, error) {
	return nil, nil
}

func (m *mockMovieMetadataRepository) GetTranslations(ctx context.Context, movieMetadataID int64) ([]domain.MovieMetadataTranslation, error) {
	return nil, nil
}

func (m *mockMovieMetadataRepository) UpsertTranslation(ctx context.Context, translation *domain.MovieMetadataTranslation) error {
	return nil
}

func (m *mockMovieMetadataRepository) ListIDsWithoutTranslation(ctx context.Context, language string) ([]ports.MovieMetadataForTranslation, error) {
	return nil, nil
}

func (m *mockMovieMetadataRepository) SetFullCreditsFetched(ctx context.Context, movieMetadataID int64) error {
	return nil
}

func (m *mockMovieMetadataRepository) HasFullCreditsFetched(ctx context.Context, movieMetadataID int64) (bool, error) {
	return false, nil
}

func (m *mockMovieMetadataRepository) GetTMDBIDByID(ctx context.Context, movieMetadataID int64) (int, error) {
	return 0, nil
}

type mockSeriesMetadataRepository struct {
	getByTMDBIDFunc       func(ctx context.Context, tmdbID int) (*domain.SeriesMetadata, error)
	createFunc            func(ctx context.Context, metadata *domain.SeriesMetadata) error
	createTranslationFunc func(ctx context.Context, translation *domain.SeriesMetadataTranslation) error
}

func (m *mockSeriesMetadataRepository) Get(ctx context.Context, id int64) (*domain.SeriesMetadata, error) {
	return nil, nil
}

func (m *mockSeriesMetadataRepository) GetByTMDBID(ctx context.Context, tmdbID int) (*domain.SeriesMetadata, error) {
	if m.getByTMDBIDFunc != nil {
		return m.getByTMDBIDFunc(ctx, tmdbID)
	}
	return nil, nil
}

func (m *mockSeriesMetadataRepository) GetWithSeasons(ctx context.Context, id int64) (*domain.SeriesMetadata, error) {
	return nil, nil
}

func (m *mockSeriesMetadataRepository) Create(ctx context.Context, metadata *domain.SeriesMetadata) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, metadata)
	}
	metadata.ID = 1
	return nil
}

func (m *mockSeriesMetadataRepository) Update(ctx context.Context, metadata *domain.SeriesMetadata) error {
	return nil
}

func (m *mockSeriesMetadataRepository) Delete(ctx context.Context, id int64) error {
	return nil
}

func (m *mockSeriesMetadataRepository) ExistsByTMDBID(ctx context.Context, tmdbID int) (bool, error) {
	return false, nil
}

func (m *mockSeriesMetadataRepository) CreateTranslation(ctx context.Context, translation *domain.SeriesMetadataTranslation) error {
	if m.createTranslationFunc != nil {
		return m.createTranslationFunc(ctx, translation)
	}
	return nil
}

func (m *mockSeriesMetadataRepository) GetTranslation(ctx context.Context, seriesMetadataID int64, language string) (*domain.SeriesMetadataTranslation, error) {
	return nil, nil
}

func (m *mockSeriesMetadataRepository) GetTranslations(ctx context.Context, seriesMetadataID int64) ([]domain.SeriesMetadataTranslation, error) {
	return nil, nil
}

func (m *mockSeriesMetadataRepository) UpsertTranslation(ctx context.Context, translation *domain.SeriesMetadataTranslation) error {
	return nil
}

func (m *mockSeriesMetadataRepository) ListIDsWithoutTranslation(ctx context.Context, language string) ([]ports.SeriesMetadataForTranslation, error) {
	return nil, nil
}

func (m *mockSeriesMetadataRepository) SetFullCreditsFetched(ctx context.Context, seriesMetadataID int64) error {
	return nil
}

func (m *mockSeriesMetadataRepository) HasFullCreditsFetched(ctx context.Context, seriesMetadataID int64) (bool, error) {
	return false, nil
}

func (m *mockSeriesMetadataRepository) GetTMDBIDByID(ctx context.Context, seriesMetadataID int64) (int, error) {
	return 0, nil
}

type mockSeasonMetadataRepository struct {
	getBySeriesAndNumberFunc func(ctx context.Context, seriesID int64, seasonNumber int) (*domain.SeasonMetadata, error)
	createFunc               func(ctx context.Context, metadata *domain.SeasonMetadata) error
	createTranslationFunc    func(ctx context.Context, translation *domain.SeasonMetadataTranslation) error
}

func (m *mockSeasonMetadataRepository) Get(ctx context.Context, id int64) (*domain.SeasonMetadata, error) {
	return nil, nil
}

func (m *mockSeasonMetadataRepository) GetBySeriesAndNumber(ctx context.Context, seriesID int64, seasonNumber int) (*domain.SeasonMetadata, error) {
	if m.getBySeriesAndNumberFunc != nil {
		return m.getBySeriesAndNumberFunc(ctx, seriesID, seasonNumber)
	}
	return nil, nil
}

func (m *mockSeasonMetadataRepository) GetWithEpisodes(ctx context.Context, id int64) (*domain.SeasonMetadata, error) {
	return nil, nil
}

func (m *mockSeasonMetadataRepository) ListBySeriesID(ctx context.Context, seriesID int64) ([]domain.SeasonMetadata, error) {
	return nil, nil
}

func (m *mockSeasonMetadataRepository) Create(ctx context.Context, metadata *domain.SeasonMetadata) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, metadata)
	}
	metadata.ID = 1
	return nil
}

func (m *mockSeasonMetadataRepository) Update(ctx context.Context, metadata *domain.SeasonMetadata) error {
	return nil
}

func (m *mockSeasonMetadataRepository) Delete(ctx context.Context, id int64) error {
	return nil
}

func (m *mockSeasonMetadataRepository) CreateTranslation(ctx context.Context, translation *domain.SeasonMetadataTranslation) error {
	if m.createTranslationFunc != nil {
		return m.createTranslationFunc(ctx, translation)
	}
	return nil
}

func (m *mockSeasonMetadataRepository) GetTranslation(ctx context.Context, seasonMetadataID int64, language string) (*domain.SeasonMetadataTranslation, error) {
	return nil, nil
}

func (m *mockSeasonMetadataRepository) GetTranslations(ctx context.Context, seasonMetadataID int64) ([]domain.SeasonMetadataTranslation, error) {
	return nil, nil
}

func (m *mockSeasonMetadataRepository) UpsertTranslation(ctx context.Context, translation *domain.SeasonMetadataTranslation) error {
	return nil
}

func (m *mockSeasonMetadataRepository) ListIDsWithoutTranslation(ctx context.Context, language string) ([]ports.SeasonMetadataForTranslation, error) {
	return nil, nil
}

type mockEpisodeMetadataRepository struct {
	getBySeasonAndNumberFunc func(ctx context.Context, seasonID int64, episodeNumber int) (*domain.EpisodeMetadata, error)
	createFunc               func(ctx context.Context, metadata *domain.EpisodeMetadata) error
	createTranslationFunc    func(ctx context.Context, translation *domain.EpisodeMetadataTranslation) error
}

func (m *mockEpisodeMetadataRepository) Get(ctx context.Context, id int64) (*domain.EpisodeMetadata, error) {
	return nil, nil
}

func (m *mockEpisodeMetadataRepository) GetBySeasonAndNumber(ctx context.Context, seasonID int64, episodeNumber int) (*domain.EpisodeMetadata, error) {
	if m.getBySeasonAndNumberFunc != nil {
		return m.getBySeasonAndNumberFunc(ctx, seasonID, episodeNumber)
	}
	return nil, nil
}

func (m *mockEpisodeMetadataRepository) ListBySeasonID(ctx context.Context, seasonID int64) ([]domain.EpisodeMetadata, error) {
	return nil, nil
}

func (m *mockEpisodeMetadataRepository) Create(ctx context.Context, metadata *domain.EpisodeMetadata) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, metadata)
	}
	metadata.ID = 1
	return nil
}

func (m *mockEpisodeMetadataRepository) Update(ctx context.Context, metadata *domain.EpisodeMetadata) error {
	return nil
}

func (m *mockEpisodeMetadataRepository) Delete(ctx context.Context, id int64) error {
	return nil
}

func (m *mockEpisodeMetadataRepository) CreateTranslation(ctx context.Context, translation *domain.EpisodeMetadataTranslation) error {
	if m.createTranslationFunc != nil {
		return m.createTranslationFunc(ctx, translation)
	}
	return nil
}

func (m *mockEpisodeMetadataRepository) GetTranslation(ctx context.Context, episodeMetadataID int64, language string) (*domain.EpisodeMetadataTranslation, error) {
	return nil, nil
}

func (m *mockEpisodeMetadataRepository) GetTranslations(ctx context.Context, episodeMetadataID int64) ([]domain.EpisodeMetadataTranslation, error) {
	return nil, nil
}

func (m *mockEpisodeMetadataRepository) UpsertTranslation(ctx context.Context, translation *domain.EpisodeMetadataTranslation) error {
	return nil
}

func (m *mockEpisodeMetadataRepository) ListIDsWithoutTranslation(ctx context.Context, language string) ([]ports.EpisodeMetadataForTranslation, error) {
	return nil, nil
}

type mockMetadataCandidateRepository struct {
	createBatchFunc         func(ctx context.Context, candidates []domain.MetadataCandidate) error
	deleteByMediaFileIDFunc func(ctx context.Context, mediaFileID string) error
}

func (m *mockMetadataCandidateRepository) Create(ctx context.Context, candidate *domain.MetadataCandidate) error {
	return nil
}

func (m *mockMetadataCandidateRepository) Get(ctx context.Context, id int64) (*domain.MetadataCandidate, error) {
	return nil, nil
}

func (m *mockMetadataCandidateRepository) ListByMediaFileID(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error) {
	return nil, nil
}

func (m *mockMetadataCandidateRepository) ListPendingByMediaFileID(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error) {
	return nil, nil
}

func (m *mockMetadataCandidateRepository) Update(ctx context.Context, candidate *domain.MetadataCandidate) error {
	return nil
}

func (m *mockMetadataCandidateRepository) Delete(ctx context.Context, id int64) error {
	return nil
}

func (m *mockMetadataCandidateRepository) DeleteByMediaFileID(ctx context.Context, mediaFileID string) error {
	if m.deleteByMediaFileIDFunc != nil {
		return m.deleteByMediaFileIDFunc(ctx, mediaFileID)
	}
	return nil
}

func (m *mockMetadataCandidateRepository) MarkSelected(ctx context.Context, candidateID int64) error {
	return nil
}

func (m *mockMetadataCandidateRepository) RejectAll(ctx context.Context, mediaFileID string) error {
	return nil
}

func (m *mockMetadataCandidateRepository) CreateBatch(ctx context.Context, candidates []domain.MetadataCandidate) error {
	if m.createBatchFunc != nil {
		return m.createBatchFunc(ctx, candidates)
	}
	return nil
}

// Helper to create test config
func testConfig() config.TMDBConfig {
	return config.TMDBConfig{
		Enabled:           true,
		APIKey:            "test-api-key",
		Language:          "en-US",
		AutoSearch:        true,
		AutoLinkThreshold: 70,
		MaxCandidates:     5,
		ImageCachePath:    "/tmp/images",
		DownloadImages:    false,
		PosterSize:        "w500",
		BackdropSize:      "w1280",
		RequestsPer10s:    35,
	}
}

// Tests

func TestEnrichMediaFile_MediaNotFound(t *testing.T) {
	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return nil, nil
		},
	}

	uc := NewEnrichMediaFileUseCase(
		testConfig(),
		&mockFilenameParser{},
		&mockTMDBClient{},
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		&mockMetadataCandidateRepository{},
		nil,
		nil,
		nil,
	)

	_, err := uc.Execute(context.Background(), EnrichMediaFileInput{MediaID: "not-found"})
	if err == nil {
		t.Error("Expected error for media not found")
	}
	if err.Error() != "media file not found: not-found" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestEnrichMediaFile_AlreadyProcessed(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mp4", "file.mp4", "file.mp4")
	media.EnrichmentStatus = domain.EnrichmentStatusAutoLinked

	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return media, nil
		},
	}

	uc := NewEnrichMediaFileUseCase(
		testConfig(),
		&mockFilenameParser{},
		&mockTMDBClient{},
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		&mockMetadataCandidateRepository{},
		nil,
		nil,
		nil,
	)

	output, err := uc.Execute(context.Background(), EnrichMediaFileInput{MediaID: "test-id"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if output.EnrichmentStatus != domain.EnrichmentStatusAutoLinked {
		t.Errorf("Expected status %s, got %s", domain.EnrichmentStatusAutoLinked, output.EnrichmentStatus)
	}
	if output.Message != "Already processed" {
		t.Errorf("Unexpected message: %s", output.Message)
	}
}

func TestEnrichMediaFile_MovieNoResults(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/The.Matrix.1999.mkv", "The.Matrix.1999.mkv", "The.Matrix.1999.mkv")

	updateCalled := false
	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return media, nil
		},
		updateFunc: func(ctx context.Context, m *domain.MediaFile) error {
			updateCalled = true
			if m.EnrichmentStatus != domain.EnrichmentStatusManualRequired {
				t.Errorf("Expected status %s, got %s", domain.EnrichmentStatusManualRequired, m.EnrichmentStatus)
			}
			return nil
		},
	}

	parser := &mockFilenameParser{
		parseFunc: func(filename string) *ports.ParsedFilename {
			return &ports.ParsedFilename{
				Title:      "The Matrix",
				CleanTitle: "the matrix",
				Year:       1999,
				IsSeries:   false,
			}
		},
	}

	tmdbClient := &mockTMDBClient{
		searchMovieFunc: func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
			return []ports.TMDBSearchResult{}, nil
		},
	}

	uc := NewEnrichMediaFileUseCase(
		testConfig(),
		parser,
		tmdbClient,
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		&mockMetadataCandidateRepository{},
		nil,
		nil,
		nil,
	)

	output, err := uc.Execute(context.Background(), EnrichMediaFileInput{MediaID: "test-id"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !updateCalled {
		t.Error("Expected media repository update to be called")
	}
	if output.EnrichmentStatus != domain.EnrichmentStatusManualRequired {
		t.Errorf("Expected status %s, got %s", domain.EnrichmentStatusManualRequired, output.EnrichmentStatus)
	}
}

func TestEnrichMediaFile_MovieAutoLink(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/The.Matrix.1999.mkv", "The.Matrix.1999.mkv", "The.Matrix.1999.mkv")

	var updatedMedia *domain.MediaFile
	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return media, nil
		},
		updateFunc: func(ctx context.Context, m *domain.MediaFile) error {
			updatedMedia = m
			return nil
		},
	}

	parser := &mockFilenameParser{
		parseFunc: func(filename string) *ports.ParsedFilename {
			return &ports.ParsedFilename{
				Title:      "The Matrix",
				CleanTitle: "matrix",
				Year:       1999,
				IsSeries:   false,
			}
		},
	}

	tmdbClient := &mockTMDBClient{
		searchMovieFunc: func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
			return []ports.TMDBSearchResult{
				{
					ID:            603,
					Title:         "The Matrix",
					OriginalTitle: "The Matrix",
					ReleaseDate:   "1999-03-30",
					Overview:      "A computer hacker learns...",
					Popularity:    100.0,
					VoteCount:     10000,
				},
			}, nil
		},
		getMovieDetailsFunc: func(ctx context.Context, movieID int, language string) (*ports.TMDBMovieDetails, error) {
			return &ports.TMDBMovieDetails{
				ID:            603,
				Title:         "The Matrix",
				OriginalTitle: "The Matrix",
				ReleaseDate:   "1999-03-30",
				Runtime:       136,
				Overview:      "A computer hacker learns...",
				Genres:        []ports.TMDBGenre{{ID: 28, Name: "Action"}, {ID: 878, Name: "Science Fiction"}},
			}, nil
		},
	}

	movieMetadataCreated := false
	movieMetadataRepo := &mockMovieMetadataRepository{
		getByTMDBIDFunc: func(ctx context.Context, tmdbID int) (*domain.MovieMetadata, error) {
			return nil, nil // Not existing
		},
		createFunc: func(ctx context.Context, metadata *domain.MovieMetadata) error {
			movieMetadataCreated = true
			metadata.ID = 1
			return nil
		},
	}

	uc := NewEnrichMediaFileUseCase(
		testConfig(),
		parser,
		tmdbClient,
		nil,
		mediaRepo,
		movieMetadataRepo,
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		&mockMetadataCandidateRepository{},
		nil,
		nil,
		nil,
	)

	output, err := uc.Execute(context.Background(), EnrichMediaFileInput{MediaID: "test-id"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !movieMetadataCreated {
		t.Error("Expected movie metadata to be created")
	}
	if output.EnrichmentStatus != domain.EnrichmentStatusAutoLinked {
		t.Errorf("Expected status %s, got %s", domain.EnrichmentStatusAutoLinked, output.EnrichmentStatus)
	}
	if output.MetadataType != domain.MetadataTypeMovie {
		t.Errorf("Expected metadata type %s, got %s", domain.MetadataTypeMovie, output.MetadataType)
	}
	if updatedMedia == nil || updatedMedia.MovieMetadataID == nil || *updatedMedia.MovieMetadataID != 1 {
		t.Error("Expected media to be linked to movie metadata ID 1")
	}
}

func TestEnrichMediaFile_MovieCandidatesFound(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/Dune.2021.mkv", "Dune.2021.mkv", "Dune.2021.mkv")

	var updatedMedia *domain.MediaFile
	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return media, nil
		},
		updateFunc: func(ctx context.Context, m *domain.MediaFile) error {
			updatedMedia = m
			return nil
		},
	}

	parser := &mockFilenameParser{
		parseFunc: func(filename string) *ports.ParsedFilename {
			return &ports.ParsedFilename{
				Title:      "Dune",
				CleanTitle: "dune",
				Year:       2021,
				IsSeries:   false,
			}
		},
	}

	// Return multiple results with lower confidence scores
	tmdbClient := &mockTMDBClient{
		searchMovieFunc: func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
			return []ports.TMDBSearchResult{
				{
					ID:            438631,
					Title:         "Dune",
					OriginalTitle: "Dune",
					ReleaseDate:   "2021-09-15",
					Overview:      "Paul Atreides...",
					Popularity:    50.0,
					VoteCount:     5000,
				},
				{
					ID:            841,
					Title:         "Dune",
					OriginalTitle: "Dune",
					ReleaseDate:   "1984-12-14",
					Overview:      "In the year 10191...",
					Popularity:    20.0,
					VoteCount:     2000,
				},
			}, nil
		},
	}

	// Set threshold very high so no auto-link occurs
	cfg := testConfig()
	cfg.AutoLinkThreshold = 100

	var storedCandidates []domain.MetadataCandidate
	candidateRepo := &mockMetadataCandidateRepository{
		createBatchFunc: func(ctx context.Context, candidates []domain.MetadataCandidate) error {
			storedCandidates = candidates
			return nil
		},
	}

	uc := NewEnrichMediaFileUseCase(
		cfg,
		parser,
		tmdbClient,
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		candidateRepo,
		nil,
		nil,
		nil,
	)

	output, err := uc.Execute(context.Background(), EnrichMediaFileInput{MediaID: "test-id"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output.EnrichmentStatus != domain.EnrichmentStatusCandidatesFound {
		t.Errorf("Expected status %s, got %s", domain.EnrichmentStatusCandidatesFound, output.EnrichmentStatus)
	}
	if len(storedCandidates) != 2 {
		t.Errorf("Expected 2 candidates, got %d", len(storedCandidates))
	}
	if updatedMedia == nil || updatedMedia.EnrichmentStatus != domain.EnrichmentStatusCandidatesFound {
		t.Error("Expected media to have candidates_found status")
	}
}

func TestEnrichMediaFile_SeriesAutoLink(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/Breaking.Bad.S01E01.720p.mkv", "Breaking.Bad.S01E01.720p.mkv", "Breaking.Bad.S01E01.720p.mkv")

	var updatedMedia *domain.MediaFile
	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return media, nil
		},
		updateFunc: func(ctx context.Context, m *domain.MediaFile) error {
			updatedMedia = m
			return nil
		},
	}

	parser := &mockFilenameParser{
		parseFunc: func(filename string) *ports.ParsedFilename {
			return &ports.ParsedFilename{
				Title:         "Breaking Bad",
				CleanTitle:    "breaking bad",
				Year:          0,
				IsSeries:      true,
				SeasonNumber:  1,
				EpisodeNumber: 1,
			}
		},
	}

	tmdbClient := &mockTMDBClient{
		searchTVFunc: func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
			return []ports.TMDBSearchResult{
				{
					ID:            1396,
					Title:         "Breaking Bad",
					OriginalTitle: "Breaking Bad",
					ReleaseDate:   "2008-01-20",
					Overview:      "A high school chemistry teacher...",
					Popularity:    200.0,
					VoteCount:     15000,
				},
			}, nil
		},
		getSeriesDetailsFunc: func(ctx context.Context, seriesID int, language string) (*ports.TMDBSeriesDetails, error) {
			return &ports.TMDBSeriesDetails{
				ID:              1396,
				Name:            "Breaking Bad",
				OriginalName:    "Breaking Bad",
				FirstAirDate:    "2008-01-20",
				NumberOfSeasons: 5,
				Genres:          []ports.TMDBGenre{{ID: 18, Name: "Drama"}},
			}, nil
		},
		getSeasonDetailsFunc: func(ctx context.Context, seriesID, seasonNumber int, language string) (*ports.TMDBSeasonDetails, error) {
			return &ports.TMDBSeasonDetails{
				ID:           3572,
				SeasonNumber: 1,
				Name:         "Season 1",
				Episodes: []ports.TMDBEpisodeSummary{
					{ID: 62085, EpisodeNumber: 1, Name: "Pilot"},
				},
			}, nil
		},
		getEpisodeDetailsFunc: func(ctx context.Context, seriesID, seasonNumber, episodeNumber int, language string) (*ports.TMDBEpisodeDetails, error) {
			return &ports.TMDBEpisodeDetails{
				ID:            62085,
				EpisodeNumber: 1,
				SeasonNumber:  1,
				Name:          "Pilot",
				Overview:      "Walter White, a chemistry teacher...",
				Runtime:       58,
			}, nil
		},
	}

	seriesCreated := false
	seriesRepo := &mockSeriesMetadataRepository{
		getByTMDBIDFunc: func(ctx context.Context, tmdbID int) (*domain.SeriesMetadata, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, metadata *domain.SeriesMetadata) error {
			seriesCreated = true
			metadata.ID = 1
			return nil
		},
	}

	seasonRepo := &mockSeasonMetadataRepository{
		getBySeriesAndNumberFunc: func(ctx context.Context, seriesID int64, seasonNumber int) (*domain.SeasonMetadata, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, metadata *domain.SeasonMetadata) error {
			metadata.ID = 1
			return nil
		},
	}

	episodeRepo := &mockEpisodeMetadataRepository{
		getBySeasonAndNumberFunc: func(ctx context.Context, seasonID int64, episodeNumber int) (*domain.EpisodeMetadata, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, metadata *domain.EpisodeMetadata) error {
			metadata.ID = 1
			return nil
		},
	}

	uc := NewEnrichMediaFileUseCase(
		testConfig(),
		parser,
		tmdbClient,
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		seriesRepo,
		seasonRepo,
		episodeRepo,
		&mockMetadataCandidateRepository{},
		nil,
		nil,
		nil,
	)

	output, err := uc.Execute(context.Background(), EnrichMediaFileInput{MediaID: "test-id"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !seriesCreated {
		t.Error("Expected series metadata to be created")
	}
	if output.EnrichmentStatus != domain.EnrichmentStatusAutoLinked {
		t.Errorf("Expected status %s, got %s", domain.EnrichmentStatusAutoLinked, output.EnrichmentStatus)
	}
	if output.MetadataType != domain.MetadataTypeEpisode {
		t.Errorf("Expected metadata type %s, got %s", domain.MetadataTypeEpisode, output.MetadataType)
	}
	if updatedMedia == nil || updatedMedia.EpisodeMetadataID == nil || *updatedMedia.EpisodeMetadataID != 1 {
		t.Error("Expected media to be linked to episode metadata ID 1")
	}
}

func TestEnrichMediaFile_TMDBSearchError(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mkv", "file.mkv", "file.mkv")

	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return media, nil
		},
	}

	parser := &mockFilenameParser{
		parseFunc: func(filename string) *ports.ParsedFilename {
			return &ports.ParsedFilename{
				Title:      "Test",
				CleanTitle: "test",
				IsSeries:   false,
			}
		},
	}

	tmdbClient := &mockTMDBClient{
		searchMovieFunc: func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
			return nil, errors.New("API rate limit exceeded")
		},
	}

	uc := NewEnrichMediaFileUseCase(
		testConfig(),
		parser,
		tmdbClient,
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		&mockMetadataCandidateRepository{},
		nil,
		nil,
		nil,
	)

	_, err := uc.Execute(context.Background(), EnrichMediaFileInput{MediaID: "test-id"})
	if err == nil {
		t.Error("Expected error for TMDB search failure")
	}
	if !errors.Is(err, err) || err.Error() != "failed to search TMDB: API rate limit exceeded" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestEnrichMediaFile_ReuseExistingMovieMetadata(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/The.Matrix.1999.mkv", "The.Matrix.1999.mkv", "The.Matrix.1999.mkv")

	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return media, nil
		},
		updateFunc: func(ctx context.Context, m *domain.MediaFile) error {
			return nil
		},
	}

	parser := &mockFilenameParser{
		parseFunc: func(filename string) *ports.ParsedFilename {
			return &ports.ParsedFilename{
				Title:      "The Matrix",
				CleanTitle: "matrix",
				Year:       1999,
				IsSeries:   false,
			}
		},
	}

	tmdbClient := &mockTMDBClient{
		searchMovieFunc: func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
			return []ports.TMDBSearchResult{
				{
					ID:            603,
					Title:         "The Matrix",
					OriginalTitle: "The Matrix",
					ReleaseDate:   "1999-03-30",
					Popularity:    100.0,
					VoteCount:     10000,
				},
			}, nil
		},
		getMovieDetailsFunc: func(ctx context.Context, movieID int, language string) (*ports.TMDBMovieDetails, error) {
			return &ports.TMDBMovieDetails{
				ID:            603,
				Title:         "The Matrix",
				OriginalTitle: "The Matrix",
				ReleaseDate:   "1999-03-30",
			}, nil
		},
	}

	existingMovie := &domain.MovieMetadata{
		ID:            42,
		TMDBID:        603,
		OriginalTitle: "The Matrix",
	}

	movieMetadataCreated := false
	movieMetadataRepo := &mockMovieMetadataRepository{
		getByTMDBIDFunc: func(ctx context.Context, tmdbID int) (*domain.MovieMetadata, error) {
			return existingMovie, nil // Already exists
		},
		createFunc: func(ctx context.Context, metadata *domain.MovieMetadata) error {
			movieMetadataCreated = true
			return nil
		},
	}

	uc := NewEnrichMediaFileUseCase(
		testConfig(),
		parser,
		tmdbClient,
		nil,
		mediaRepo,
		movieMetadataRepo,
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		&mockMetadataCandidateRepository{},
		nil,
		nil,
		nil,
	)

	output, err := uc.Execute(context.Background(), EnrichMediaFileInput{MediaID: "test-id"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if movieMetadataCreated {
		t.Error("Expected to reuse existing movie metadata, not create new one")
	}
	if output.EnrichmentStatus != domain.EnrichmentStatusAutoLinked {
		t.Errorf("Expected status %s, got %s", domain.EnrichmentStatusAutoLinked, output.EnrichmentStatus)
	}
}

// Test title similarity calculation
func TestCalculateTitleSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		parsed   string
		tmdb     string
		minScore float64
		maxScore float64
	}{
		{"exact match", "the matrix", "The Matrix", 1.0, 1.0},
		{"case insensitive", "MATRIX", "matrix", 1.0, 1.0},
		{"article removal", "the godfather", "Godfather", 0.8, 1.0},
		{"partial match", "avatar", "Avatar: The Way of Water", 0.2, 0.5},
		{"completely different", "star wars", "The Matrix", 0.0, 0.3},
		{"with punctuation", "spider man", "Spider-Man", 0.9, 1.0},
		{"sequel number", "john wick 2", "John Wick: Chapter 2", 0.4, 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateTitleSimilarity(tt.parsed, tt.tmdb)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Score %f not in expected range [%f, %f]", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"The Matrix", "matrix"},
		{"A Beautiful Mind", "beautiful mind"},
		{"An American Werewolf in London", "american werewolf in london"},
		{"Spider-Man", "spider man"},
		{"Lord of the Rings: The Fellowship", "lord of the rings the fellowship"},
		{"Rock & Roll", "rock and roll"},
		{"What's Up, Doc?", "whats up doc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeTitle(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"1999-03-30", 1999},
		{"2021-10-22", 2021},
		{"1984-12-14", 1984},
		{"", 0},
		{"invalid", 0},
		{"2000", 2000},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractYear(tt.input)
			if result != tt.expected {
				t.Errorf("extractYear(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			result := levenshteinDistance(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, result, tt.expected)
			}
		})
	}
}
