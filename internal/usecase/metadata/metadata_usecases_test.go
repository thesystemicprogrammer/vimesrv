package metadata

import (
	"context"
	"errors"
	"testing"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// Extended mock for MetadataCandidateRepository with full functionality
type mockCandidateRepoFull struct {
	getFunc                      func(ctx context.Context, id int64) (*domain.MetadataCandidate, error)
	listByMediaFileIDFunc        func(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error)
	listPendingByMediaFileIDFunc func(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error)
	markSelectedFunc             func(ctx context.Context, candidateID int64) error
	rejectAllFunc                func(ctx context.Context, mediaFileID string) error
	deleteByMediaFileIDFunc      func(ctx context.Context, mediaFileID string) error
	createBatchFunc              func(ctx context.Context, candidates []domain.MetadataCandidate) error
}

func (m *mockCandidateRepoFull) Create(ctx context.Context, candidate *domain.MetadataCandidate) error {
	return nil
}

func (m *mockCandidateRepoFull) Get(ctx context.Context, id int64) (*domain.MetadataCandidate, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockCandidateRepoFull) ListByMediaFileID(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error) {
	if m.listByMediaFileIDFunc != nil {
		return m.listByMediaFileIDFunc(ctx, mediaFileID)
	}
	return nil, nil
}

func (m *mockCandidateRepoFull) ListPendingByMediaFileID(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error) {
	if m.listPendingByMediaFileIDFunc != nil {
		return m.listPendingByMediaFileIDFunc(ctx, mediaFileID)
	}
	return nil, nil
}

func (m *mockCandidateRepoFull) Update(ctx context.Context, candidate *domain.MetadataCandidate) error {
	return nil
}

func (m *mockCandidateRepoFull) Delete(ctx context.Context, id int64) error {
	return nil
}

func (m *mockCandidateRepoFull) DeleteByMediaFileID(ctx context.Context, mediaFileID string) error {
	if m.deleteByMediaFileIDFunc != nil {
		return m.deleteByMediaFileIDFunc(ctx, mediaFileID)
	}
	return nil
}

func (m *mockCandidateRepoFull) MarkSelected(ctx context.Context, candidateID int64) error {
	if m.markSelectedFunc != nil {
		return m.markSelectedFunc(ctx, candidateID)
	}
	return nil
}

func (m *mockCandidateRepoFull) RejectAll(ctx context.Context, mediaFileID string) error {
	if m.rejectAllFunc != nil {
		return m.rejectAllFunc(ctx, mediaFileID)
	}
	return nil
}

func (m *mockCandidateRepoFull) CreateBatch(ctx context.Context, candidates []domain.MetadataCandidate) error {
	if m.createBatchFunc != nil {
		return m.createBatchFunc(ctx, candidates)
	}
	return nil
}

// ============================================================================
// GetCandidates Tests
// ============================================================================

func TestGetCandidates_MediaNotFound(t *testing.T) {
	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return nil, nil
		},
	}

	uc := NewGetCandidatesUseCase(
		testConfig(),
		&mockTMDBClient{},
		mediaRepo,
		&mockCandidateRepoFull{},
	)

	_, err := uc.Execute(context.Background(), GetCandidatesInput{MediaID: "not-found"})
	if err == nil {
		t.Error("Expected error for media not found")
	}
}

func TestGetCandidates_Success(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mp4", "file.mp4", "file.mp4")
	media.EnrichmentStatus = domain.EnrichmentStatusCandidatesFound

	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return media, nil
		},
	}

	season := 1
	episode := 1
	candidateRepo := &mockCandidateRepoFull{
		listByMediaFileIDFunc: func(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error) {
			return []domain.MetadataCandidate{
				{
					ID:              1,
					MediaFileID:     "test-id",
					CandidateType:   domain.CandidateTypeMovie,
					TMDBID:          603,
					Title:           "The Matrix",
					ConfidenceScore: 95,
					PosterPath:      "/poster.jpg",
					Status:          domain.CandidateStatusPending,
				},
				{
					ID:              2,
					MediaFileID:     "test-id",
					CandidateType:   domain.CandidateTypeSeries,
					TMDBID:          1396,
					Title:           "Breaking Bad",
					ConfidenceScore: 80,
					SeasonNumber:    &season,
					EpisodeNumber:   &episode,
					Status:          domain.CandidateStatusPending,
				},
			}, nil
		},
	}

	uc := NewGetCandidatesUseCase(
		testConfig(),
		&mockTMDBClient{},
		mediaRepo,
		candidateRepo,
	)

	output, err := uc.Execute(context.Background(), GetCandidatesInput{MediaID: "test-id"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output.Count != 2 {
		t.Errorf("Expected 2 candidates, got %d", output.Count)
	}
	if output.EnrichmentStatus != domain.EnrichmentStatusCandidatesFound {
		t.Errorf("Expected enrichment status %s, got %s", domain.EnrichmentStatusCandidatesFound, output.EnrichmentStatus)
	}
	if len(output.Candidates) != 2 {
		t.Fatalf("Expected 2 candidates in output")
	}
	if output.Candidates[0].Title != "The Matrix" {
		t.Errorf("Expected first candidate title 'The Matrix', got %s", output.Candidates[0].Title)
	}
	if output.Candidates[0].PosterURL == "" {
		t.Error("Expected poster URL to be generated")
	}
}

func TestGetCandidates_PendingOnly(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mp4", "file.mp4", "file.mp4")

	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return media, nil
		},
	}

	pendingOnlyCalled := false
	candidateRepo := &mockCandidateRepoFull{
		listPendingByMediaFileIDFunc: func(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error) {
			pendingOnlyCalled = true
			return []domain.MetadataCandidate{}, nil
		},
	}

	uc := NewGetCandidatesUseCase(
		testConfig(),
		&mockTMDBClient{},
		mediaRepo,
		candidateRepo,
	)

	_, err := uc.Execute(context.Background(), GetCandidatesInput{MediaID: "test-id", PendingOnly: true})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !pendingOnlyCalled {
		t.Error("Expected ListPendingByMediaFileID to be called")
	}
}

// ============================================================================
// LinkMetadata Tests
// ============================================================================

func TestLinkMetadata_MediaNotFound(t *testing.T) {
	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return nil, nil
		},
	}

	uc := NewLinkMetadataUseCase(
		testConfig(),
		&mockTMDBClient{},
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		&mockCandidateRepoFull{},
	)

	_, err := uc.Execute(context.Background(), LinkMetadataInput{MediaID: "not-found", CandidateID: 1})
	if err == nil {
		t.Error("Expected error for media not found")
	}
}

func TestLinkMetadata_CandidateNotFound(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mp4", "file.mp4", "file.mp4")

	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return media, nil
		},
	}

	candidateRepo := &mockCandidateRepoFull{
		getFunc: func(ctx context.Context, id int64) (*domain.MetadataCandidate, error) {
			return nil, nil
		},
	}

	uc := NewLinkMetadataUseCase(
		testConfig(),
		&mockTMDBClient{},
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		candidateRepo,
	)

	_, err := uc.Execute(context.Background(), LinkMetadataInput{MediaID: "test-id", CandidateID: 999})
	if err == nil {
		t.Error("Expected error for candidate not found")
	}
}

func TestLinkMetadata_CandidateWrongMedia(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mp4", "file.mp4", "file.mp4")

	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return media, nil
		},
	}

	candidateRepo := &mockCandidateRepoFull{
		getFunc: func(ctx context.Context, id int64) (*domain.MetadataCandidate, error) {
			return &domain.MetadataCandidate{
				ID:          1,
				MediaFileID: "other-media-id", // Wrong media
				TMDBID:      603,
			}, nil
		},
	}

	uc := NewLinkMetadataUseCase(
		testConfig(),
		&mockTMDBClient{},
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		candidateRepo,
	)

	_, err := uc.Execute(context.Background(), LinkMetadataInput{MediaID: "test-id", CandidateID: 1})
	if err == nil {
		t.Error("Expected error for candidate belonging to wrong media")
	}
}

func TestLinkMetadata_MovieSuccess(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mp4", "file.mp4", "file.mp4")

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

	markSelectedCalled := false
	candidateRepo := &mockCandidateRepoFull{
		getFunc: func(ctx context.Context, id int64) (*domain.MetadataCandidate, error) {
			return &domain.MetadataCandidate{
				ID:            1,
				MediaFileID:   "test-id",
				CandidateType: domain.CandidateTypeMovie,
				TMDBID:        603,
				Title:         "The Matrix",
			}, nil
		},
		markSelectedFunc: func(ctx context.Context, candidateID int64) error {
			markSelectedCalled = true
			return nil
		},
	}

	tmdbClient := &mockTMDBClient{
		getMovieDetailsFunc: func(ctx context.Context, movieID int, language string) (*ports.TMDBMovieDetails, error) {
			return &ports.TMDBMovieDetails{
				ID:            603,
				Title:         "The Matrix",
				OriginalTitle: "The Matrix",
				ReleaseDate:   "1999-03-30",
			}, nil
		},
	}

	movieRepo := &mockMovieMetadataRepository{
		getByTMDBIDFunc: func(ctx context.Context, tmdbID int) (*domain.MovieMetadata, error) {
			return nil, nil // Not existing
		},
		createFunc: func(ctx context.Context, metadata *domain.MovieMetadata) error {
			metadata.ID = 1
			return nil
		},
	}

	uc := NewLinkMetadataUseCase(
		testConfig(),
		tmdbClient,
		nil,
		mediaRepo,
		movieRepo,
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		candidateRepo,
	)

	output, err := uc.Execute(context.Background(), LinkMetadataInput{MediaID: "test-id", CandidateID: 1})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !markSelectedCalled {
		t.Error("Expected MarkSelected to be called")
	}
	if output.MetadataType != domain.MetadataTypeMovie {
		t.Errorf("Expected metadata type movie, got %s", output.MetadataType)
	}
	if updatedMedia == nil || updatedMedia.EnrichmentStatus != domain.EnrichmentStatusLinked {
		t.Error("Expected media to be updated with linked status")
	}
	if updatedMedia.MovieMetadataID == nil || *updatedMedia.MovieMetadataID != 1 {
		t.Error("Expected media to be linked to movie metadata ID 1")
	}
}

func TestLinkMetadata_SeriesSuccess(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mp4", "file.mp4", "file.mp4")

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

	season := 1
	episode := 1
	candidateRepo := &mockCandidateRepoFull{
		getFunc: func(ctx context.Context, id int64) (*domain.MetadataCandidate, error) {
			return &domain.MetadataCandidate{
				ID:            1,
				MediaFileID:   "test-id",
				CandidateType: domain.CandidateTypeSeries,
				TMDBID:        1396,
				Title:         "Breaking Bad",
				SeasonNumber:  &season,
				EpisodeNumber: &episode,
			}, nil
		},
		markSelectedFunc: func(ctx context.Context, candidateID int64) error {
			return nil
		},
	}

	tmdbClient := &mockTMDBClient{
		getSeriesDetailsFunc: func(ctx context.Context, seriesID int, language string) (*ports.TMDBSeriesDetails, error) {
			return &ports.TMDBSeriesDetails{
				ID:           1396,
				Name:         "Breaking Bad",
				OriginalName: "Breaking Bad",
			}, nil
		},
		getSeasonDetailsFunc: func(ctx context.Context, seriesID, seasonNumber int, language string) (*ports.TMDBSeasonDetails, error) {
			return &ports.TMDBSeasonDetails{
				ID:           1,
				SeasonNumber: 1,
				Name:         "Season 1",
			}, nil
		},
		getEpisodeDetailsFunc: func(ctx context.Context, seriesID, seasonNumber, episodeNumber int, language string) (*ports.TMDBEpisodeDetails, error) {
			return &ports.TMDBEpisodeDetails{
				ID:            1,
				EpisodeNumber: 1,
				Name:          "Pilot",
			}, nil
		},
	}

	seriesRepo := &mockSeriesMetadataRepository{
		getByTMDBIDFunc: func(ctx context.Context, tmdbID int) (*domain.SeriesMetadata, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, metadata *domain.SeriesMetadata) error {
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

	uc := NewLinkMetadataUseCase(
		testConfig(),
		tmdbClient,
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		seriesRepo,
		seasonRepo,
		episodeRepo,
		candidateRepo,
	)

	output, err := uc.Execute(context.Background(), LinkMetadataInput{MediaID: "test-id", CandidateID: 1})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output.MetadataType != domain.MetadataTypeEpisode {
		t.Errorf("Expected metadata type episode, got %s", output.MetadataType)
	}
	if updatedMedia == nil || updatedMedia.EnrichmentStatus != domain.EnrichmentStatusLinked {
		t.Error("Expected media to be updated with linked status")
	}
	if updatedMedia.EpisodeMetadataID == nil || *updatedMedia.EpisodeMetadataID != 1 {
		t.Error("Expected media to be linked to episode metadata ID 1")
	}
}

// ============================================================================
// SearchMetadata Tests
// ============================================================================

func TestSearchMetadata_EmptyQuery(t *testing.T) {
	uc := NewSearchMetadataUseCase(
		testConfig(),
		&mockTMDBClient{},
	)

	_, err := uc.Execute(context.Background(), SearchMetadataInput{MediaID: "test-id", Query: ""})
	if err == nil {
		t.Error("Expected error for empty query")
	}
}

func TestSearchMetadata_MovieSearch(t *testing.T) {
	tmdbClient := &mockTMDBClient{
		searchMovieFunc: func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
			if query != "Matrix" {
				t.Errorf("Expected query 'Matrix', got '%s'", query)
			}
			if year != 1999 {
				t.Errorf("Expected year 1999, got %d", year)
			}
			return []ports.TMDBSearchResult{
				{
					ID:          603,
					Title:       "The Matrix",
					ReleaseDate: "1999-03-30",
					PosterPath:  "/poster.jpg",
				},
			}, nil
		},
	}

	uc := NewSearchMetadataUseCase(testConfig(), tmdbClient)

	output, err := uc.Execute(context.Background(), SearchMetadataInput{
		MediaID:   "test-id",
		Query:     "Matrix",
		Year:      1999,
		MediaType: "movie",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output.Count != 1 {
		t.Errorf("Expected 1 result, got %d", output.Count)
	}
	if len(output.Results) != 1 {
		t.Fatalf("Expected 1 result in output")
	}
	if output.Results[0].Title != "The Matrix" {
		t.Errorf("Expected title 'The Matrix', got '%s'", output.Results[0].Title)
	}
	if output.Results[0].MediaType != "movie" {
		t.Errorf("Expected media type 'movie', got '%s'", output.Results[0].MediaType)
	}
}

func TestSearchMetadata_TVSearch(t *testing.T) {
	tmdbClient := &mockTMDBClient{
		searchTVFunc: func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
			return []ports.TMDBSearchResult{
				{
					ID:          1396,
					Title:       "Breaking Bad",
					ReleaseDate: "2008-01-20",
				},
			}, nil
		},
	}

	uc := NewSearchMetadataUseCase(testConfig(), tmdbClient)

	output, err := uc.Execute(context.Background(), SearchMetadataInput{
		MediaID:   "test-id",
		Query:     "Breaking Bad",
		MediaType: "tv",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output.Count != 1 {
		t.Errorf("Expected 1 result, got %d", output.Count)
	}
	if output.Results[0].MediaType != "tv" {
		t.Errorf("Expected media type 'tv', got '%s'", output.Results[0].MediaType)
	}
}

func TestSearchMetadata_MultiSearch(t *testing.T) {
	tmdbClient := &mockTMDBClient{
		searchMultiFunc: func(ctx context.Context, query string, language string) ([]ports.TMDBSearchResult, error) {
			return []ports.TMDBSearchResult{
				{ID: 603, Title: "The Matrix", MediaType: "movie"},
				{ID: 1396, Title: "Breaking Bad", MediaType: "tv"},
				{ID: 999, Title: "Some Person", MediaType: "person"}, // Should be filtered out
			}, nil
		},
	}

	uc := NewSearchMetadataUseCase(testConfig(), tmdbClient)

	output, err := uc.Execute(context.Background(), SearchMetadataInput{
		MediaID:   "test-id",
		Query:     "test",
		MediaType: "", // Empty = multi search
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output.Count != 2 {
		t.Errorf("Expected 2 results (person filtered), got %d", output.Count)
	}
}

func TestSearchMetadata_MaxResults(t *testing.T) {
	tmdbClient := &mockTMDBClient{
		searchMovieFunc: func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
			results := make([]ports.TMDBSearchResult, 20)
			for i := 0; i < 20; i++ {
				results[i] = ports.TMDBSearchResult{ID: i, Title: "Movie"}
			}
			return results, nil
		},
	}

	uc := NewSearchMetadataUseCase(testConfig(), tmdbClient)

	output, err := uc.Execute(context.Background(), SearchMetadataInput{
		MediaID:    "test-id",
		Query:      "test",
		MediaType:  "movie",
		MaxResults: 5,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output.Count != 5 {
		t.Errorf("Expected 5 results (limited), got %d", output.Count)
	}
}

func TestSearchMetadata_TMDBError(t *testing.T) {
	tmdbClient := &mockTMDBClient{
		searchMovieFunc: func(ctx context.Context, query string, year int, language string) ([]ports.TMDBSearchResult, error) {
			return nil, errors.New("API error")
		},
	}

	uc := NewSearchMetadataUseCase(testConfig(), tmdbClient)

	_, err := uc.Execute(context.Background(), SearchMetadataInput{
		MediaID:   "test-id",
		Query:     "test",
		MediaType: "movie",
	})
	if err == nil {
		t.Error("Expected error for TMDB failure")
	}
}

// ============================================================================
// SkipEnrichment Tests
// ============================================================================

func TestSkipEnrichment_MediaNotFound(t *testing.T) {
	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return nil, nil
		},
	}

	uc := NewSkipEnrichmentUseCase(mediaRepo, &mockCandidateRepoFull{})

	_, err := uc.Execute(context.Background(), SkipEnrichmentInput{MediaID: "not-found"})
	if err == nil {
		t.Error("Expected error for media not found")
	}
}

func TestSkipEnrichment_Success(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mp4", "file.mp4", "file.mp4")

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

	rejectAllCalled := false
	candidateRepo := &mockCandidateRepoFull{
		rejectAllFunc: func(ctx context.Context, mediaFileID string) error {
			rejectAllCalled = true
			return nil
		},
	}

	uc := NewSkipEnrichmentUseCase(mediaRepo, candidateRepo)

	output, err := uc.Execute(context.Background(), SkipEnrichmentInput{MediaID: "test-id"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !rejectAllCalled {
		t.Error("Expected RejectAll to be called")
	}
	if updatedMedia == nil || updatedMedia.EnrichmentStatus != domain.EnrichmentStatusSkipped {
		t.Error("Expected media to be updated with skipped status")
	}
	if output.Message != "Enrichment skipped" {
		t.Errorf("Unexpected message: %s", output.Message)
	}
}

// ============================================================================
// ResetEnrichment Tests
// ============================================================================

func TestResetEnrichment_MediaNotFound(t *testing.T) {
	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return nil, nil
		},
	}

	uc := NewResetEnrichmentUseCase(mediaRepo, &mockCandidateRepoFull{})

	_, err := uc.Execute(context.Background(), ResetEnrichmentInput{MediaID: "not-found"})
	if err == nil {
		t.Error("Expected error for media not found")
	}
}

func TestResetEnrichment_Success(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mp4", "file.mp4", "file.mp4")
	media.EnrichmentStatus = domain.EnrichmentStatusLinked
	movieID := int64(1)
	media.MovieMetadataID = &movieID
	media.MetadataType = domain.MetadataTypeMovie

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

	deleteCandidatesCalled := false
	candidateRepo := &mockCandidateRepoFull{
		deleteByMediaFileIDFunc: func(ctx context.Context, mediaFileID string) error {
			deleteCandidatesCalled = true
			return nil
		},
	}

	uc := NewResetEnrichmentUseCase(mediaRepo, candidateRepo)

	output, err := uc.Execute(context.Background(), ResetEnrichmentInput{MediaID: "test-id"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !deleteCandidatesCalled {
		t.Error("Expected DeleteByMediaFileID to be called")
	}
	if updatedMedia == nil || updatedMedia.EnrichmentStatus != domain.EnrichmentStatusPending {
		t.Error("Expected media to be reset to pending status")
	}
	if updatedMedia.MovieMetadataID != nil {
		t.Error("Expected MovieMetadataID to be cleared")
	}
	if updatedMedia.MetadataType != domain.MetadataTypeNone {
		t.Error("Expected MetadataType to be cleared")
	}
	if output.Message != "Enrichment reset to pending" {
		t.Errorf("Unexpected message: %s", output.Message)
	}
}

// ============================================================================
// LinkFromSearch Tests
// ============================================================================

func TestLinkFromSearch_MediaNotFound(t *testing.T) {
	mediaRepo := &mockMediaRepository{
		getFunc: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return nil, nil
		},
	}

	uc := NewLinkFromSearchUseCase(
		testConfig(),
		&mockTMDBClient{},
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		&mockCandidateRepoFull{},
	)

	_, err := uc.Execute(context.Background(), LinkFromSearchInput{
		MediaID:   "not-found",
		TMDBID:    603,
		MediaType: "movie",
	})
	if err == nil {
		t.Error("Expected error for media not found")
	}
}

func TestLinkFromSearch_MovieSuccess(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mp4", "file.mp4", "file.mp4")

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

	rejectAllCalled := false
	candidateRepo := &mockCandidateRepoFull{
		rejectAllFunc: func(ctx context.Context, mediaFileID string) error {
			rejectAllCalled = true
			return nil
		},
	}

	tmdbClient := &mockTMDBClient{
		getMovieDetailsFunc: func(ctx context.Context, movieID int, language string) (*ports.TMDBMovieDetails, error) {
			return &ports.TMDBMovieDetails{
				ID:            603,
				Title:         "The Matrix",
				OriginalTitle: "The Matrix",
			}, nil
		},
	}

	movieRepo := &mockMovieMetadataRepository{
		getByTMDBIDFunc: func(ctx context.Context, tmdbID int) (*domain.MovieMetadata, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, metadata *domain.MovieMetadata) error {
			metadata.ID = 1
			return nil
		},
	}

	uc := NewLinkFromSearchUseCase(
		testConfig(),
		tmdbClient,
		nil,
		mediaRepo,
		movieRepo,
		&mockSeriesMetadataRepository{},
		&mockSeasonMetadataRepository{},
		&mockEpisodeMetadataRepository{},
		candidateRepo,
	)

	output, err := uc.Execute(context.Background(), LinkFromSearchInput{
		MediaID:   "test-id",
		TMDBID:    603,
		MediaType: "movie",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !rejectAllCalled {
		t.Error("Expected RejectAll to be called to clean up existing candidates")
	}
	if output.MetadataType != domain.MetadataTypeMovie {
		t.Errorf("Expected metadata type movie, got %s", output.MetadataType)
	}
	if updatedMedia == nil || updatedMedia.EnrichmentStatus != domain.EnrichmentStatusLinked {
		t.Error("Expected media to be updated with linked status")
	}
}

func TestLinkFromSearch_SeriesSuccess(t *testing.T) {
	media := domain.NewMediaFile("test-id", "fingerprint", "/path/to/file.mp4", "file.mp4", "file.mp4")

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

	candidateRepo := &mockCandidateRepoFull{
		rejectAllFunc: func(ctx context.Context, mediaFileID string) error {
			return nil
		},
	}

	tmdbClient := &mockTMDBClient{
		getSeriesDetailsFunc: func(ctx context.Context, seriesID int, language string) (*ports.TMDBSeriesDetails, error) {
			return &ports.TMDBSeriesDetails{
				ID:           1396,
				Name:         "Breaking Bad",
				OriginalName: "Breaking Bad",
			}, nil
		},
		getSeasonDetailsFunc: func(ctx context.Context, seriesID, seasonNumber int, language string) (*ports.TMDBSeasonDetails, error) {
			return &ports.TMDBSeasonDetails{
				ID:           1,
				SeasonNumber: 2,
				Name:         "Season 2",
			}, nil
		},
		getEpisodeDetailsFunc: func(ctx context.Context, seriesID, seasonNumber, episodeNumber int, language string) (*ports.TMDBEpisodeDetails, error) {
			return &ports.TMDBEpisodeDetails{
				ID:            1,
				EpisodeNumber: 5,
				Name:          "Episode 5",
			}, nil
		},
	}

	seriesRepo := &mockSeriesMetadataRepository{
		getByTMDBIDFunc: func(ctx context.Context, tmdbID int) (*domain.SeriesMetadata, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, metadata *domain.SeriesMetadata) error {
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

	uc := NewLinkFromSearchUseCase(
		testConfig(),
		tmdbClient,
		nil,
		mediaRepo,
		&mockMovieMetadataRepository{},
		seriesRepo,
		seasonRepo,
		episodeRepo,
		candidateRepo,
	)

	output, err := uc.Execute(context.Background(), LinkFromSearchInput{
		MediaID:       "test-id",
		TMDBID:        1396,
		MediaType:     "tv",
		SeasonNumber:  2,
		EpisodeNumber: 5,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output.MetadataType != domain.MetadataTypeEpisode {
		t.Errorf("Expected metadata type episode, got %s", output.MetadataType)
	}
	if updatedMedia == nil || updatedMedia.EpisodeMetadataID == nil {
		t.Error("Expected media to be linked to episode")
	}
}
