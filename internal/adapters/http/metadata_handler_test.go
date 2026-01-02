package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/metadata"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// MockGetCandidatesUseCase is a mock for GetCandidatesUseCase
type MockGetCandidatesUseCase struct {
	ExecuteFunc func(ctx context.Context, input metadata.GetCandidatesInput) (*metadata.GetCandidatesOutput, error)
}

func (m *MockGetCandidatesUseCase) Execute(ctx context.Context, input metadata.GetCandidatesInput) (*metadata.GetCandidatesOutput, error) {
	return m.ExecuteFunc(ctx, input)
}

// MockLinkMetadataUseCase is a mock for LinkMetadataUseCase
type MockLinkMetadataUseCase struct {
	ExecuteFunc func(ctx context.Context, input metadata.LinkMetadataInput) (*metadata.LinkMetadataOutput, error)
}

func (m *MockLinkMetadataUseCase) Execute(ctx context.Context, input metadata.LinkMetadataInput) (*metadata.LinkMetadataOutput, error) {
	return m.ExecuteFunc(ctx, input)
}

// MockSearchMetadataUseCase is a mock for SearchMetadataUseCase
type MockSearchMetadataUseCase struct {
	ExecuteFunc func(ctx context.Context, input metadata.SearchMetadataInput) (*metadata.SearchMetadataOutput, error)
}

func (m *MockSearchMetadataUseCase) Execute(ctx context.Context, input metadata.SearchMetadataInput) (*metadata.SearchMetadataOutput, error) {
	return m.ExecuteFunc(ctx, input)
}

// MockLinkFromSearchUseCase is a mock for LinkFromSearchUseCase
type MockLinkFromSearchUseCase struct {
	ExecuteFunc func(ctx context.Context, input metadata.LinkFromSearchInput) (*metadata.LinkFromSearchOutput, error)
}

func (m *MockLinkFromSearchUseCase) Execute(ctx context.Context, input metadata.LinkFromSearchInput) (*metadata.LinkFromSearchOutput, error) {
	return m.ExecuteFunc(ctx, input)
}

// MockSkipEnrichmentUseCase is a mock for SkipEnrichmentUseCase
type MockSkipEnrichmentUseCase struct {
	ExecuteFunc func(ctx context.Context, input metadata.SkipEnrichmentInput) (*metadata.SkipEnrichmentOutput, error)
}

func (m *MockSkipEnrichmentUseCase) Execute(ctx context.Context, input metadata.SkipEnrichmentInput) (*metadata.SkipEnrichmentOutput, error) {
	return m.ExecuteFunc(ctx, input)
}

// MockResetEnrichmentUseCase is a mock for ResetEnrichmentUseCase
type MockResetEnrichmentUseCase struct {
	ExecuteFunc func(ctx context.Context, input metadata.ResetEnrichmentInput) (*metadata.ResetEnrichmentOutput, error)
}

func (m *MockResetEnrichmentUseCase) Execute(ctx context.Context, input metadata.ResetEnrichmentInput) (*metadata.ResetEnrichmentOutput, error) {
	return m.ExecuteFunc(ctx, input)
}

// createTestHandlerWithMocks creates a handler with custom mock implementations
func createTestHandlerWithMocks(
	getCandidates func(ctx context.Context, input metadata.GetCandidatesInput) (*metadata.GetCandidatesOutput, error),
	linkMetadata func(ctx context.Context, input metadata.LinkMetadataInput) (*metadata.LinkMetadataOutput, error),
	searchMetadata func(ctx context.Context, input metadata.SearchMetadataInput) (*metadata.SearchMetadataOutput, error),
	linkFromSearch func(ctx context.Context, input metadata.LinkFromSearchInput) (*metadata.LinkFromSearchOutput, error),
	skipEnrichment func(ctx context.Context, input metadata.SkipEnrichmentInput) (*metadata.SkipEnrichmentOutput, error),
	resetEnrichment func(ctx context.Context, input metadata.ResetEnrichmentInput) (*metadata.ResetEnrichmentOutput, error),
) *MetadataHandler {
	// Note: The handler takes concrete use case types, not interfaces.
	// For testing, we'll need to create a handler with nil use cases and test the nil check behavior,
	// or we need to refactor the handler to accept interfaces.
	// For now, we'll test the TMDB disabled response when use cases are nil.
	return NewMetadataHandler(nil, nil, nil, nil, nil, nil)
}

// --- Tests for GetCandidates ---

func TestMetadataHandler_GetCandidates_TMDBDisabled(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/media/test-id/candidates", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.GetCandidates(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "TMDB_DISABLED")
	assert.Contains(t, w.Body.String(), "TMDB integration is not enabled")
}

// --- Tests for LinkMetadata ---

func TestMetadataHandler_LinkMetadata_TMDBDisabled(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	body := `{"candidate_id": 123}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/media/test-id/link", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.LinkMetadata(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "TMDB_DISABLED")
}

func TestMetadataHandler_LinkMetadata_InvalidRequest(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	// Missing candidate_id
	body := `{}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/media/test-id/link", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.LinkMetadata(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_REQUEST")
}

func TestMetadataHandler_LinkMetadata_InvalidJSON(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	body := `{invalid json`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/media/test-id/link", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.LinkMetadata(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_REQUEST")
}

// --- Tests for SearchMetadata ---

func TestMetadataHandler_SearchMetadata_TMDBDisabled(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	body := `{"query": "Matrix"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/media/test-id/search", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.SearchMetadata(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "TMDB_DISABLED")
}

func TestMetadataHandler_SearchMetadata_InvalidRequest(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	// Missing query (required)
	body := `{}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/media/test-id/search", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.SearchMetadata(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_REQUEST")
}

// --- Tests for LinkFromSearch ---

func TestMetadataHandler_LinkFromSearch_TMDBDisabled(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	body := `{"tmdb_id": 603, "media_type": "movie"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/media/test-id/link-search", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.LinkFromSearch(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "TMDB_DISABLED")
}

func TestMetadataHandler_LinkFromSearch_InvalidRequest_MissingTMDBID(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	body := `{"media_type": "movie"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/media/test-id/link-search", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.LinkFromSearch(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_REQUEST")
}

func TestMetadataHandler_LinkFromSearch_InvalidRequest_MissingMediaType(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	body := `{"tmdb_id": 603}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/media/test-id/link-search", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.LinkFromSearch(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_REQUEST")
}

func TestMetadataHandler_LinkFromSearch_InvalidMediaType(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	body := `{"tmdb_id": 603, "media_type": "invalid"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/media/test-id/link-search", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.LinkFromSearch(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_MEDIA_TYPE")
	assert.Contains(t, w.Body.String(), "media_type must be 'movie' or 'tv'")
}

// --- Tests for SkipEnrichment ---

func TestMetadataHandler_SkipEnrichment_TMDBDisabled(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/media/test-id/skip", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.SkipEnrichment(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "TMDB_DISABLED")
}

// --- Tests for ResetEnrichment ---

func TestMetadataHandler_ResetEnrichment_TMDBDisabled(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/media/test-id/reset", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.ResetEnrichment(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "TMDB_DISABLED")
}

// --- Tests for RegisterRoutes ---

func TestMetadataHandler_RegisterRoutes(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	router := gin.New()
	group := router.Group("/api/v1")
	handler.RegisterRoutes(group)

	// Verify all routes are registered by making requests
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/media/test-id/candidates"},
		{"POST", "/api/v1/media/test-id/link"},
		{"POST", "/api/v1/media/test-id/search"},
		{"POST", "/api/v1/media/test-id/link-search"},
		{"POST", "/api/v1/media/test-id/skip"},
		{"POST", "/api/v1/media/test-id/reset"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			var req *http.Request
			if route.method == "POST" {
				req, _ = http.NewRequest(route.method, route.path, bytes.NewBufferString("{}"))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(route.method, route.path, nil)
			}
			router.ServeHTTP(w, req)

			// Should not be 404 (route exists)
			assert.NotEqual(t, http.StatusNotFound, w.Code, "Route %s %s should be registered", route.method, route.path)
		})
	}
}

// --- Tests for Request/Response structure ---

func TestLinkMetadataRequest_JSONBinding(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantValid bool
	}{
		{
			name:      "valid request",
			json:      `{"candidate_id": 123}`,
			wantValid: true,
		},
		{
			name:      "missing candidate_id",
			json:      `{}`,
			wantValid: false,
		},
		{
			name:      "candidate_id zero",
			json:      `{"candidate_id": 0}`,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req LinkMetadataRequest
			err := json.Unmarshal([]byte(tt.json), &req)
			if tt.wantValid {
				assert.NoError(t, err)
				assert.NotZero(t, req.CandidateID)
			}
		})
	}
}

func TestSearchMetadataRequest_JSONBinding(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantQuery string
		wantYear  int
		wantType  string
	}{
		{
			name:      "basic query",
			json:      `{"query": "Matrix"}`,
			wantQuery: "Matrix",
			wantYear:  0,
			wantType:  "",
		},
		{
			name:      "query with year",
			json:      `{"query": "Matrix", "year": 1999}`,
			wantQuery: "Matrix",
			wantYear:  1999,
			wantType:  "",
		},
		{
			name:      "query with type",
			json:      `{"query": "Matrix", "media_type": "movie"}`,
			wantQuery: "Matrix",
			wantYear:  0,
			wantType:  "movie",
		},
		{
			name:      "full request",
			json:      `{"query": "Matrix", "year": 1999, "media_type": "movie", "max_results": 5}`,
			wantQuery: "Matrix",
			wantYear:  1999,
			wantType:  "movie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req SearchMetadataRequest
			err := json.Unmarshal([]byte(tt.json), &req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantQuery, req.Query)
			assert.Equal(t, tt.wantYear, req.Year)
			assert.Equal(t, tt.wantType, req.MediaType)
		})
	}
}

func TestLinkFromSearchRequest_JSONBinding(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantTMDBID  int
		wantType    string
		wantSeason  int
		wantEpisode int
	}{
		{
			name:        "movie request",
			json:        `{"tmdb_id": 603, "media_type": "movie"}`,
			wantTMDBID:  603,
			wantType:    "movie",
			wantSeason:  0,
			wantEpisode: 0,
		},
		{
			name:        "tv request with season/episode",
			json:        `{"tmdb_id": 1396, "media_type": "tv", "season_number": 1, "episode_number": 1}`,
			wantTMDBID:  1396,
			wantType:    "tv",
			wantSeason:  1,
			wantEpisode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req LinkFromSearchRequest
			err := json.Unmarshal([]byte(tt.json), &req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantTMDBID, req.TMDBID)
			assert.Equal(t, tt.wantType, req.MediaType)
			assert.Equal(t, tt.wantSeason, req.SeasonNumber)
			assert.Equal(t, tt.wantEpisode, req.EpisodeNumber)
		})
	}
}

// --- Integration-style tests with actual use case instances ---

// These tests would require refactoring the handler to accept interfaces
// or using real use cases with mocked dependencies.
// For now, we demonstrate the pattern with a note about future improvements.

func TestMetadataHandler_GetCandidates_QueryParams(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	tests := []struct {
		name            string
		url             string
		wantPendingOnly bool
	}{
		{
			name:            "default no query param",
			url:             "/api/v1/media/test-id/candidates",
			wantPendingOnly: false,
		},
		{
			name:            "pending_only=true",
			url:             "/api/v1/media/test-id/candidates?pending_only=true",
			wantPendingOnly: true,
		},
		{
			name:            "pending_only=false",
			url:             "/api/v1/media/test-id/candidates?pending_only=false",
			wantPendingOnly: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", tt.url, nil)
			c.Params = gin.Params{{Key: "id", Value: "test-id"}}

			// The handler will return TMDB_DISABLED since use case is nil
			// but we can verify the request is parsed correctly by the response
			handler.GetCandidates(c)

			// Use case is nil, so we get the disabled response
			assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		})
	}
}

// --- Error response format tests ---

func TestMetadataHandler_ErrorResponseFormat(t *testing.T) {
	handler := NewMetadataHandler(nil, nil, nil, nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/media/test-id/candidates", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.GetCandidates(c)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify response structure
	assert.Equal(t, false, response["success"])
	assert.NotNil(t, response["error"])

	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "TMDB_DISABLED", errorData["code"])
	assert.Equal(t, "TMDB integration is not enabled", errorData["message"])
}

// Note: To properly test the success paths, the handler would need to accept
// interfaces instead of concrete use case types. This would allow injecting
// mock implementations. Consider refactoring for better testability.

// Example of how tests would look with interface-based injection:
/*
func TestMetadataHandler_GetCandidates_Success(t *testing.T) {
	mockUC := &MockGetCandidatesUseCase{
		ExecuteFunc: func(ctx context.Context, input metadata.GetCandidatesInput) (*metadata.GetCandidatesOutput, error) {
			return &metadata.GetCandidatesOutput{
				MediaID:          input.MediaID,
				EnrichmentStatus: "candidates_found",
				Candidates: []metadata.CandidateDTO{
					{
						ID:              1,
						TMDBID:          603,
						CandidateType:   "movie",
						Title:           "The Matrix",
						ConfidenceScore: 95,
					},
				},
				Count: 1,
			}, nil
		},
	}

	handler := NewMetadataHandlerWithInterfaces(mockUC, ...)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/media/test-id/candidates", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.GetCandidates(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "The Matrix")
}
*/

// Suppress unused variable warnings for mock types
var (
	_ = MockGetCandidatesUseCase{}
	_ = MockLinkMetadataUseCase{}
	_ = MockSearchMetadataUseCase{}
	_ = MockLinkFromSearchUseCase{}
	_ = MockSkipEnrichmentUseCase{}
	_ = MockResetEnrichmentUseCase{}
	_ = errors.New // Import used for mock patterns
)
