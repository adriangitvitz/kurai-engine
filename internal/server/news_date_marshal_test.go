package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
)

func TestNewsResult_UndatedMarshalsAsNullOrAbsent(t *testing.T) {
	results := []searxng.Result{
		{URL: "https://example.com/news/1", Title: "Headline 1", Engines: []string{"google"}},
	}
	news := convertToNewsResults(results)
	require.Len(t, news, 1)

	out, err := json.Marshal(news[0])
	require.NoError(t, err)

	got := map[string]any{}
	require.NoError(t, json.Unmarshal(out, &got))

	pd, present := got["published_date"]
	if present {
		assert.Nil(t, pd, "published_date present for undated result must be JSON null, not zero-time string; got %#v", pd)
	}
	assert.NotContains(t, string(out), "0001-01-01",
		"zero-time string in JSON breaks agent date reasoning; should be null or omitted")
}

func TestNewsResult_DatedMarshalsCleanly(t *testing.T) {
	results := []searxng.Result{
		{URL: "https://example.com/news/2", Title: "Headline 2", PublishedDate: "2026-04-29T15:30:00Z", Engines: []string{"google"}},
	}
	news := convertToNewsResults(results)
	require.Len(t, news, 1)

	out, err := json.Marshal(news[0])
	require.NoError(t, err)
	assert.Contains(t, string(out), "2026-04-29",
		"valid date should round-trip into the JSON output")
}

func TestNewsResult_NoZeroTimeSurvivesPipeline(t *testing.T) {
	results := []searxng.Result{
		{URL: "https://a/", Title: "A", Engines: []string{"g"}, PublishedDate: "2026-01-01"},
		{URL: "https://b/", Title: "B", Engines: []string{"g"}},
		{URL: "https://c/", Title: "C", Engines: []string{"g"}, PublishedDate: "garbage"},
	}
	news := convertToNewsResults(results)
	require.Len(t, news, 3)

	resp := NewsSearchResponse{Query: "q", Results: news}
	out, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "0001-01-01",
		"no NewsResult should marshal a zero-time published_date; got: %s", string(out))
}

func TestNewsRanker_HandlesUndatedResults(t *testing.T) {
	now := time.Now().UTC()
	dated := now.Add(-1 * time.Hour)
	results := []ranker.NewsResult{
		{URL: "https://undated/", Title: "Undated"},
		{URL: "https://dated/", Title: "Dated", PublishedDate: &dated},
	}
	cfg := ranker.DefaultNewsRankingConfig()
	r := ranker.NewNewsRanker(cfg, nil)
	ranked := r.RankNews([]string{"news"}, results)
	require.Len(t, ranked, 2)

	var datedScore, undatedScore float64
	for _, n := range ranked {
		switch n.URL {
		case "https://undated/":
			undatedScore = n.RecencyScore
		case "https://dated/":
			datedScore = n.RecencyScore
		}
	}
	assert.NotZero(t, datedScore, "dated result should have a positive recency score")
	assert.Greater(t, datedScore, undatedScore,
		"dated result must outrank undated on recency (dated=%.3f undated=%.3f)",
		datedScore, undatedScore)
}
