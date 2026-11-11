package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
)

type recordingSearx struct {
	mu         sync.Mutex
	queries    []string
	timeRanges []string
}

func (r *recordingSearx) seen() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.queries...)
}

func newSiteOperatorServer(t *testing.T) (*Server, *recordingSearx) {
	t.Helper()
	rec := &recordingSearx{}
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		rec.mu.Lock()
		rec.queries = append(rec.queries, q)
		rec.timeRanges = append(rec.timeRanges, r.URL.Query().Get("time_range"))
		rec.mu.Unlock()
		host := "general.example"
		if q != "custody keys" {
			host = "northerntrust.com"
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"results":[{"url":"https://%s/custody-keys","title":"custody keys","content":"custody keys","engine":"brave"}]}`, host)
	}))
	t.Cleanup(fake.Close)

	cfg := answersTestConfig(fake.URL)
	cfg.Answers = config.AnswersConfig{}
	bl := blocklist.NewManager(&cfg.Blocklists, nil)
	rk := ranker.New(&cfg.Ranking, bl)
	rk.SetRelevanceFloor(ranker.RelevanceFloorConfig{})
	return New(cfg, searxng.NewClient(&cfg.SearXNG, nil), rk, bl), rec
}

func TestServiceSearch_IncludeDomainsReachUpstreamAsSiteOperator(t *testing.T) {
	srv, rec := newSiteOperatorServer(t)

	resp, err := srv.ServiceSearch(context.Background(), ServiceSearchReq{
		Query:          "custody keys",
		IncludeDomains: []string{"www.NorthernTrust.com"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"custody keys site:northerntrust.com"}, rec.seen())
	require.Len(t, resp.Results, 1)
	assert.Equal(t, "https://northerntrust.com/custody-keys", resp.Results[0].URL)
	assert.Equal(t, "custody keys", resp.Query)
}

func TestServiceSearch_SiteScopedSetDoesNotShareCacheWithGeneralSet(t *testing.T) {
	srv, rec := newSiteOperatorServer(t)
	ctx := context.Background()

	general, err := srv.ServiceSearch(ctx, ServiceSearchReq{Query: "custody keys"})
	require.NoError(t, err)
	require.Len(t, general.Results, 1)
	assert.Equal(t, "https://general.example/custody-keys", general.Results[0].URL)

	scoped, err := srv.ServiceSearch(ctx, ServiceSearchReq{Query: "custody keys", IncludeDomains: []string{"northerntrust.com"}})
	require.NoError(t, err)
	require.Len(t, scoped.Results, 1)
	assert.Equal(t, "https://northerntrust.com/custody-keys", scoped.Results[0].URL)
	assert.False(t, scoped.Meta.FromCache)

	again, err := srv.ServiceSearch(ctx, ServiceSearchReq{Query: "custody keys"})
	require.NoError(t, err)
	assert.True(t, again.Meta.FromCache)
	assert.Equal(t, "https://general.example/custody-keys", again.Results[0].URL)

	scopedAgain, err := srv.ServiceSearch(ctx, ServiceSearchReq{Query: "custody keys", IncludeDomains: []string{"northerntrust.com"}})
	require.NoError(t, err)
	assert.True(t, scopedAgain.Meta.FromCache)
	assert.Len(t, rec.seen(), 2)
}

func TestServiceSearch_PhraseAndDateStatsReachMeta(t *testing.T) {
	srv, _ := newSiteOperatorServer(t)

	resp, err := srv.ServiceSearch(context.Background(), ServiceSearchReq{
		Query:      "custody keys",
		DateFrom:   "2026-01-01",
		DateStrict: true,
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Results)
	assert.Equal(t, 1, resp.Meta.UndatedFiltered)
}

func TestServiceSearch_RecentDateFromNarrowsTheUpstreamFetch(t *testing.T) {
	srv, rec := newSiteOperatorServer(t)
	ctx := context.Background()
	recent := time.Now().AddDate(0, -3, 0).Format("2006-01-02")

	resp, err := srv.ServiceSearch(ctx, ServiceSearchReq{Query: "custody keys", DateFrom: recent})
	require.NoError(t, err)
	assert.Equal(t, "year", resp.Meta.TimeRange)
	require.Len(t, resp.Results, 1, "undated results stay unless date_strict")

	plain, err := srv.ServiceSearch(ctx, ServiceSearchReq{Query: "custody keys"})
	require.NoError(t, err)
	assert.False(t, plain.Meta.FromCache, "the narrowed set must not serve the unfiltered query")
	assert.Empty(t, plain.Meta.TimeRange)

	rec.mu.Lock()
	defer rec.mu.Unlock()
	assert.Equal(t, []string{"year", ""}, rec.timeRanges)
}
