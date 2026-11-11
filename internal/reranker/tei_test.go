package reranker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTEIServer(t *testing.T, scoresByText map[string]float64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/rerank", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req teiRequest
		require.NoError(t, json.Unmarshal(body, &req))

		out := []teiResultItem{}
		for i, txt := range req.Texts {
			s := scoresByText[txt]
			out = append(out, teiResultItem{Index: i, Score: s})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
}

func TestTEIClient_RerankReturnsScoresInInputOrder(t *testing.T) {
	srv := newTEIServer(t, map[string]float64{
		"alpha": 4.0,
		"beta":  -1.0,
		"gamma": 2.5,
	})
	defer srv.Close()

	c := NewTEIClient(srv.URL, time.Second)
	docs := []Doc{
		{ID: "1", Text: "alpha"},
		{ID: "2", Text: "beta"},
		{ID: "3", Text: "gamma"},
	}
	got, err := c.Rerank(context.Background(), "q", docs)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "1", got[0].ID)
	assert.Equal(t, 4.0, got[0].Score)
	assert.Equal(t, "2", got[1].ID)
	assert.Equal(t, -1.0, got[1].Score)
	assert.Equal(t, "3", got[2].ID)
	assert.Equal(t, 2.5, got[2].Score)
}

func TestTEIClient_HandlesUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "model not loaded", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewTEIClient(srv.URL, time.Second)
	_, err := c.Rerank(context.Background(), "q", []Doc{{ID: "1", Text: "x"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestTEIClient_HonorsContextDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
			w.Write([]byte(`[]`))
		}
	}))
	defer srv.Close()

	c := NewTEIClient(srv.URL, 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err := c.Rerank(ctx, "q", []Doc{{ID: "1", Text: "x"}})
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "deadline") ||
			strings.Contains(err.Error(), "context") ||
			strings.Contains(err.Error(), "canceled"),
		"err should reflect context deadline; got: %v", err)
}

func TestNoop_PreservesInputOrderAndIDs(t *testing.T) {
	docs := []Doc{
		{ID: "a", Text: "alpha"},
		{ID: "b", Text: "beta"},
	}
	got, err := Noop{}.Rerank(context.Background(), "q", docs)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "a", got[0].ID)
	assert.Equal(t, 0.0, got[0].Score)
	assert.Equal(t, "b", got[1].ID)
}
