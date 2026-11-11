package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
)

func TestServiceSearch_TotalCountsOnlyTheResultsReturnedAfterTheProbe(t *testing.T) {
	alive := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer alive.Close()
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) }))
	defer dead.Close()

	searx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"results":[
			{"url":"%s/custody-keys-alive","title":"custody keys alive","content":"custody keys","engine":"brave"},
			{"url":"%s/custody-keys-gone","title":"custody keys gone","content":"custody keys","engine":"brave"}]}`, alive.URL, dead.URL)
	}))
	defer searx.Close()

	cfg := answersTestConfig(searx.URL)
	cfg.Answers = config.AnswersConfig{}
	bl := blocklist.NewManager(&cfg.Blocklists, nil)
	rk := ranker.New(&cfg.Ranking, bl)
	rk.SetRelevanceFloor(ranker.RelevanceFloorConfig{})
	srv := New(cfg, searxng.NewClient(&cfg.SearXNG, nil), rk, bl)
	srv.prober = testProber()
	srv.probeDrop = true

	resp, err := srv.ServiceSearch(context.Background(), ServiceSearchReq{Query: "custody keys"})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, 1, resp.Meta.Total)
	assert.Equal(t, 1, resp.Meta.TotalPages)
	assert.Equal(t, 1, resp.Envelope.Pagination.Total)
	shown := 0
	for _, n := range resp.Meta.Facets {
		shown += n
	}
	assert.Equal(t, 1, shown, "facet counts must describe the same set as total")

	kept, err := srv.ServiceSearch(context.Background(), ServiceSearchReq{Query: "custody keys", KeepUnreachable: true})
	require.NoError(t, err)
	assert.Len(t, kept.Results, 2)
	assert.Equal(t, 2, kept.Meta.Total)
}

func TestServiceSearch_LimitIsThePageSizeNotTheTotal(t *testing.T) {
	searx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results":[
			{"url":"https://a.example/custody-keys","title":"custody keys one","content":"custody keys","engine":"brave"},
			{"url":"https://b.example/custody-keys","title":"custody keys two","content":"custody keys","engine":"brave"},
			{"url":"https://c.example/custody-keys","title":"custody keys three","content":"custody keys","engine":"brave"}]}`)
	}))
	defer searx.Close()

	cfg := answersTestConfig(searx.URL)
	cfg.Answers = config.AnswersConfig{}
	bl := blocklist.NewManager(&cfg.Blocklists, nil)
	rk := ranker.New(&cfg.Ranking, bl)
	rk.SetRelevanceFloor(ranker.RelevanceFloorConfig{})
	srv := New(cfg, searxng.NewClient(&cfg.SearXNG, nil), rk, bl)

	first, err := srv.ServiceSearch(context.Background(), ServiceSearchReq{Query: "custody keys", Limit: 2})
	require.NoError(t, err)
	assert.Len(t, first.Results, 2)
	assert.Equal(t, 3, first.Meta.Total)
	assert.Equal(t, 2, first.Meta.PageSize)
	assert.Equal(t, 2, first.Meta.TotalPages)
	assert.True(t, first.Envelope.Pagination.HasMore)
	shown := 0
	for _, n := range first.Meta.Facets {
		shown += n
	}
	assert.Equal(t, 3, shown)

	second, err := srv.ServiceSearch(context.Background(), ServiceSearchReq{Query: "custody keys", Limit: 2, Page: 2})
	require.NoError(t, err)
	require.Len(t, second.Results, 1)
	assert.NotContains(t, []string{first.Results[0].URL, first.Results[1].URL}, second.Results[0].URL)
	assert.False(t, second.Envelope.Pagination.HasMore)
}
