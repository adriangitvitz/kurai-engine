package searxng

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/config"
)

func TestClient_Search_Success(t *testing.T) {
	expectedResponse := Response{
		Query:           "test query",
		NumberOfResults: 100,
		Results: []Result{
			{
				URL:     "https://example.com",
				Title:   "Example Title",
				Content: "Example content snippet",
				Engines: []string{"google", "duckduckgo"},
				Score:   85.5,
			},
		},
		Suggestions: []string{"test suggestion"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search", r.URL.Path)
		assert.Equal(t, "test query", r.URL.Query().Get("q"))
		assert.Equal(t, "json", r.URL.Query().Get("format"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewClient(&config.SearXNGConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}, nil)

	resp, err := client.Search(context.Background(), "test query", nil)
	require.NoError(t, err)

	assert.Equal(t, "test query", resp.Query)
	assert.Equal(t, int64(100), resp.NumberOfResults)
	assert.Len(t, resp.Results, 1)
	assert.Equal(t, "https://example.com", resp.Results[0].URL)
	assert.Equal(t, "Example Title", resp.Results[0].Title)
	assert.Equal(t, []string{"google", "duckduckgo"}, resp.Results[0].Engines)
	assert.Equal(t, 85.5, resp.Results[0].Score)
}

func TestClient_Search_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "3", r.URL.Query().Get("pageno"))
		json.NewEncoder(w).Encode(Response{Query: "paginated"})
	}))
	defer server.Close()

	client := NewClient(&config.SearXNGConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}, nil)

	opts := &SearchOptions{Page: 3}
	resp, err := client.Search(context.Background(), "test", opts)
	require.NoError(t, err)
	assert.Equal(t, "paginated", resp.Query)
}

func TestClient_Search_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "en", r.URL.Query().Get("language"))
		assert.Equal(t, "2", r.URL.Query().Get("safesearch"))
		assert.Equal(t, "week", r.URL.Query().Get("time_range"))
		assert.Equal(t, "general,images", r.URL.Query().Get("categories"))
		assert.Equal(t, "google,bing", r.URL.Query().Get("engines"))
		json.NewEncoder(w).Encode(Response{})
	}))
	defer server.Close()

	client := NewClient(&config.SearXNGConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}, nil)

	opts := &SearchOptions{
		Language:   "en",
		SafeSearch: 2,
		TimeRange:  "week",
		Categories: []string{"general", "images"},
		Engines:    []string{"google", "bing"},
	}
	_, err := client.Search(context.Background(), "test", opts)
	require.NoError(t, err)
}

func TestClient_Search_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode(Response{})
	}))
	defer server.Close()

	client := NewClient(&config.SearXNGConfig{
		URL:     server.URL,
		Timeout: 50 * time.Millisecond,
	}, nil)

	_, err := client.Search(context.Background(), "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestClient_Search_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		json.NewEncoder(w).Encode(Response{})
	}))
	defer server.Close()

	client := NewClient(&config.SearXNGConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Search(ctx, "test", nil)
	assert.Error(t, err)
}

func TestClient_Search_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient(&config.SearXNGConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}, nil)

	_, err := client.Search(context.Background(), "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestClient_Search_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client := NewClient(&config.SearXNGConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}, nil)

	_, err := client.Search(context.Background(), "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestClient_Search_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json {"))
	}))
	defer server.Close()

	client := NewClient(&config.SearXNGConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}, nil)

	_, err := client.Search(context.Background(), "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decoding response")
}

func TestClient_Search_EmptyQuery(t *testing.T) {
	client := NewClient(&config.SearXNGConfig{
		URL:     "http://localhost:8888",
		Timeout: 5 * time.Second,
	}, nil)

	_, err := client.Search(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query cannot be empty")
}

func TestClient_Search_ConnectionRefused(t *testing.T) {
	client := NewClient(&config.SearXNGConfig{
		URL:     "http://localhost:59999",
		Timeout: 1 * time.Second,
	}, nil)

	_, err := client.Search(context.Background(), "test", nil)
	assert.Error(t, err)
}

func TestClient_Search_FullResponseParsing(t *testing.T) {
	fullResponse := Response{
		Query:           "full test",
		NumberOfResults: 12345,
		Results: []Result{
			{
				URL:           "https://example.com/page",
				Title:         "Full Test Title",
				Content:       "This is the content snippet",
				Engines:       []string{"google", "duckduckgo", "bing"},
				Score:         142.5,
				Positions:     []int{1, 3, 5},
				PublishedDate: "2024-01-15",
				Category:      "general",
			},
		},
		Answers:     json.RawMessage(`[{"answer":"42","engine":"calculator"}]`),
		Corrections: []string{"corrected query"},
		Suggestions: []string{"suggestion 1", "suggestion 2"},
		UnresponsiveEngines: [][]string{
			{"yahoo", "timeout"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(fullResponse)
	}))
	defer server.Close()

	client := NewClient(&config.SearXNGConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}, nil)

	resp, err := client.Search(context.Background(), "full test", nil)
	require.NoError(t, err)

	assert.Equal(t, "full test", resp.Query)
	assert.Equal(t, int64(12345), resp.NumberOfResults)
	assert.Len(t, resp.Results, 1)
	assert.Equal(t, []int{1, 3, 5}, resp.Results[0].Positions)
	assert.Equal(t, "2024-01-15", resp.Results[0].PublishedDate)
	legacy := resp.LegacyAnswers()
	assert.Len(t, legacy, 1)
	assert.Equal(t, "42", legacy[0].Answer)
	assert.Equal(t, "calculator", legacy[0].Engine)
	assert.Equal(t, []string{"corrected query"}, resp.Corrections)
	assert.Equal(t, []string{"suggestion 1", "suggestion 2"}, resp.Suggestions)
	assert.Len(t, resp.UnresponsiveEngines, 1)
}
