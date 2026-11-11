package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
)

type pagedSearx struct {
	mu        sync.Mutex
	requested []string
	pages     map[string]string
}

func (p *pagedSearx) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageno := r.URL.Query().Get("pageno")
		if pageno == "" {
			pageno = "1"
		}
		p.mu.Lock()
		p.requested = append(p.requested, pageno)
		p.mu.Unlock()

		body, ok := p.pages[pageno]
		if !ok {
			body = `{"query":"q","results":[]}`
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

func (p *pagedSearx) pagesRequested() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.requested))
	copy(out, p.requested)
	return out
}

func resultsPage(prefix string, n int) string {
	type res struct {
		URL     string   `json:"url"`
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Engines []string `json:"engines"`
	}
	var rs []res
	for i := range n {
		rs = append(rs, res{
			URL:     fmt.Sprintf("https://example.com/%s/%d", prefix, i),
			Title:   fmt.Sprintf("kubernetes operator pattern %s %d", prefix, i),
			Content: "kubernetes operator pattern explained",
			Engines: []string{"bing"},
		})
	}
	b, _ := json.Marshal(rs)
	return `{"query":"q","results":` + string(b) + `}`
}

func newPrefetchServer(t *testing.T, p *pagedSearx, prefetchPages int) *Server {
	t.Helper()
	fake := httptest.NewServer(p.handler())
	t.Cleanup(fake.Close)

	cfg := &config.Config{
		SearXNG: config.SearXNGConfig{URL: fake.URL, Timeout: 5_000_000_000},
		Server: config.ServerConfig{
			Port: 0, ReadTimeout: 5_000_000_000, WriteTimeout: 5_000_000_000,
			PageSize: 20, PrefetchPages: prefetchPages, CacheTTL: 60_000_000_000,
		},
		Ranking:    config.RankingConfig{},
		Blocklists: config.BlocklistConfig{},
	}
	bl := blocklist.NewManager(&cfg.Blocklists, nil)
	rk := ranker.New(&cfg.Ranking, bl)
	client := searxng.NewClient(&cfg.SearXNG, nil)
	return New(cfg, client, rk, bl)
}

func TestPrefetch_StopsOnEmptyPage(t *testing.T) {
	p := &pagedSearx{pages: map[string]string{
		"1": resultsPage("p1", 5),
	}}
	s := newPrefetchServer(t, p, 3)

	resp, err := s.ServiceSearch(context.Background(), ServiceSearchReq{
		UserID: "u", Query: "kubernetes operator pattern", Page: 1,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Results, 5, "page 1 results must survive")

	assert.Equal(t, []string{"1", "2"}, p.pagesRequested(),
		"must stop after the first empty page, not walk to prefetch_pages")
}

func TestPrefetch_ContinuesWhilePagesYield(t *testing.T) {
	p := &pagedSearx{pages: map[string]string{
		"1": resultsPage("p1", 4),
		"2": resultsPage("p2", 4),
		"3": resultsPage("p3", 4),
	}}
	s := newPrefetchServer(t, p, 3)

	resp, err := s.ServiceSearch(context.Background(), ServiceSearchReq{
		UserID: "u", Query: "kubernetes operator pattern", Page: 1,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"1", "2", "3"}, p.pagesRequested(),
		"a paginating backend must still be walked to the ceiling")
	assert.Equal(t, 12, resp.Meta.Total, "every page's results must be kept")
}

func TestPrefetch_EmptyFirstPageStopsImmediately(t *testing.T) {
	p := &pagedSearx{pages: map[string]string{}}
	s := newPrefetchServer(t, p, 3)

	_, err := s.ServiceSearch(context.Background(), ServiceSearchReq{
		UserID: "u", Query: "kubernetes operator pattern", Page: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"1"}, p.pagesRequested())
}
