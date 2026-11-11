package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/apierror"
	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/ranker"
)

type MockSearXNGClient struct {
	Results []ranker.ScoredResult
	Error   error
}

func (m *MockSearXNGClient) Search(query string, opts *SearchOptions) ([]ranker.ScoredResult, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Results, nil
}

func setupTestServer() *Server {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Ranking: config.RankingConfig{
			Weights: config.WeightConfig{
				TitleMatch:      20,
				URLMatch:        10,
				EngineAgreement: 5,
				TrancoTop10k:    15,
				TrancoTop100k:   10,
				TrancoTop1M:     5,
				Freshness:       5,
			},
			Penalties: config.PenaltyConfig{
				Blocklist:       -50,
				ContentFarm:     -30,
				ExcessiveParams: -10,
				ParamThreshold:  5,
			},
		},
		Blocklists: config.BlocklistConfig{},
	}

	blManager := blocklist.NewManager(&cfg.Blocklists, nil)
	rankEngine := ranker.New(&cfg.Ranking, blManager)
	rankEngine.SetRelevanceFloor(ranker.RelevanceFloorConfig{})

	return New(cfg, nil, rankEngine, blManager)
}

func TestHandler_Search_Success(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
	rec := httptest.NewRecorder()

	server.handleSearch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp SearchResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "test", resp.Query)
}

func TestHandler_Search_MissingQuery(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rec := httptest.NewRecorder()

	server.handleSearch(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Search_EmptyQuery(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/search?q=", nil)
	rec := httptest.NewRecorder()

	server.handleSearch(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Search_WithPagination(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&page=2", nil)
	rec := httptest.NewRecorder()

	server.handleSearch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_Search_WithDebug(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&debug=true", nil)
	rec := httptest.NewRecorder()

	server.handleSearch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_Health_Success(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.handleHealth(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp HealthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestHandler_BlocklistReload_Success(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodPost, "/blocklist/reload", nil)
	rec := httptest.NewRecorder()

	server.handleBlocklistReload(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_BlocklistReload_WrongMethod(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/blocklist/reload", nil)
	rec := httptest.NewRecorder()

	server.handleBlocklistReload(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestHandler_NotFound(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSearchResponse_JSON(t *testing.T) {
	resp := SearchResponse{
		Query: "test query",
		Results: []ranker.ScoredResult{
			{
				URL:           "https://example.com",
				Title:         "Example",
				Snippet:       "Example snippet",
				Engines:       []string{"google"},
				OriginalScore: 100,
				AdjustedScore: 120,
				Badges: ranker.Badges{
					Reputable: true,
				},
			},
		},
		Meta: SearchMeta{
			Total:             1,
			BlocklistFiltered: 0,
			ProcessingTimeMs:  50,
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded SearchResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "test query", decoded.Query)
	assert.Len(t, decoded.Results, 1)
	assert.Equal(t, "https://example.com", decoded.Results[0].URL)
	assert.True(t, decoded.Results[0].Badges.Reputable)
}

func TestHandler_Search_CORS_Preflight(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodOptions, "/search", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestHandler_Search_InvalidPage(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&page=invalid", nil)
	rec := httptest.NewRecorder()

	server.handleSearch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_Search_NegativePage(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&page=-1", nil)
	rec := httptest.NewRecorder()

	server.handleSearch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_SecurityHeaders(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
}

func TestServer_WriteError(t *testing.T) {
	server := setupTestServer()

	rec := httptest.NewRecorder()
	server.writeError(rec, http.StatusBadRequest, apierror.CodeMissingQuery, "test error message")

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var env apierror.Envelope
	err := json.Unmarshal(rec.Body.Bytes(), &env)
	require.NoError(t, err)
	assert.Equal(t, apierror.CodeMissingQuery, env.Error.Code)
	assert.Equal(t, "test error message", env.Error.Message)
}

func TestServer_MissingQueryReturnsCodedEnvelope(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rec := httptest.NewRecorder()

	server.handleSearch(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var env apierror.Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, apierror.CodeMissingQuery, env.Error.Code)
	assert.NotEmpty(t, env.Error.Message)
}
