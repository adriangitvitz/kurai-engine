package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/extract"
	"github.com/adriangitvitz/kurai/internal/ranker"
)

const fusedArticleHTML = `<!doctype html><html lang="en"><head><title>Doc %d</title></head>
<body><main><article><h1>Doc %d</h1>
<p>This is the main content of document %d. It has enough words to satisfy
trafilatura's minimum content heuristics, talking about Go concurrency,
channels, goroutines, and the scheduler in moderate detail across sentences.</p>
<p>Second paragraph adds more body so extraction reliably succeeds in tests
without network access or flaky external dependencies.</p>
</article></main></body></html>`

func fusedTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	pages := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/broken" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var id int
		fmt.Sscanf(r.URL.Path, "/doc/%d", &id)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, fusedArticleHTML, id, id, id)
	})
	origin := httptest.NewServer(pages)
	t.Cleanup(origin.Close)

	srv := setupTestServer()
	srv.extractor = extract.NewFetcher(extract.Config{
		Timeout:              3 * time.Second,
		MaxBytes:             1 << 20,
		MaxRedirects:         3,
		AllowPrivateNetworks: true,
	})
	return srv, origin
}

func scoredResultsFor(origin string, n int) []ranker.ScoredResult {
	out := make([]ranker.ScoredResult, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, ranker.ScoredResult{
			URL:   fmt.Sprintf("%s/doc/%d", origin, i),
			Title: fmt.Sprintf("Doc %d", i),
		})
	}
	return out
}

func TestAttachExtractedContent_TopNOnly(t *testing.T) {
	srv, origin := fusedTestServer(t)
	results := scoredResultsFor(origin.URL, 5)

	srv.attachExtractedContent(context.Background(), results, 3)

	for i := range 3 {
		assert.NotEmpty(t, results[i].ExtractedContent, "result %d (top-3) must carry content", i+1)
		assert.Contains(t, results[i].ExtractedContent, fmt.Sprintf("document %d", i+1))
	}
	for i := 3; i < 5; i++ {
		assert.Empty(t, results[i].ExtractedContent, "result %d (beyond top-3) must NOT be extracted", i+1)
	}
}

func TestAttachExtractedContent_FailureIsNonFatal(t *testing.T) {
	srv, origin := fusedTestServer(t)
	results := []ranker.ScoredResult{
		{URL: origin.URL + "/doc/1", Title: "ok"},
		{URL: origin.URL + "/broken", Title: "broken"},
		{URL: origin.URL + "/doc/3", Title: "ok too"},
	}

	srv.attachExtractedContent(context.Background(), results, 3)

	assert.NotEmpty(t, results[0].ExtractedContent)
	assert.Empty(t, results[1].ExtractedContent, "failed extraction must leave content empty, not error out")
	assert.NotEmpty(t, results[2].ExtractedContent, "failure of one result must not block the others")
}

func TestAttachExtractedContent_CapAndNilSafety(t *testing.T) {
	srv, origin := fusedTestServer(t)

	short := scoredResultsFor(origin.URL, 2)
	srv.attachExtractedContent(context.Background(), short, 10)
	assert.NotEmpty(t, short[0].ExtractedContent)
	assert.NotEmpty(t, short[1].ExtractedContent)

	srv.extractor = nil
	plain := scoredResultsFor(origin.URL, 2)
	srv.attachExtractedContent(context.Background(), plain, 2)
	assert.Empty(t, plain[0].ExtractedContent)
}

func TestServiceSearch_ExtractTopValidation(t *testing.T) {
	srv := setupTestServer()
	resp, err := srv.ServiceSearch(context.Background(), ServiceSearchReq{
		UserID: "u", Query: "anything", ExtractTop: 99,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestAttachExtractedContent_BudgetBoundsSlowPages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/fast", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, fusedArticleHTML, 1, 1, 1)
	})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, fusedArticleHTML, 2, 2, 2)
		}
	})
	origin := httptest.NewServer(mux)
	t.Cleanup(origin.Close)

	srv := setupTestServer()
	srv.extractor = extract.NewFetcher(extract.Config{
		Timeout:              10 * time.Second,
		MaxBytes:             1 << 20,
		AllowPrivateNetworks: true,
	})
	srv.fusedExtractBudget = 700 * time.Millisecond

	results := []ranker.ScoredResult{
		{URL: origin.URL + "/fast", Title: "fast"},
		{URL: origin.URL + "/slow", Title: "slow"},
	}

	start := time.Now()
	srv.attachExtractedContent(context.Background(), results, 2)
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 3*time.Second,
		"wave must return near the budget, not wait out the slow page (took %v)", elapsed)
	assert.NotEmpty(t, results[0].ExtractedContent, "fast page inside budget must be attached")
	assert.Empty(t, results[1].ExtractedContent, "slow page past budget ships without content")
}

func TestAttachExtractedContent_ZeroBudgetMeansNoCap(t *testing.T) {
	srv, origin := fusedTestServer(t)
	srv.fusedExtractBudget = 0
	results := scoredResultsFor(origin.URL, 1)
	srv.attachExtractedContent(context.Background(), results, 1)
	assert.NotEmpty(t, results[0].ExtractedContent)
}
