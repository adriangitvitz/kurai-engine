package ranker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/searxng"
)

func freshnessRanker(t *testing.T) *Ranker {
	t.Helper()
	cfg := &config.RankingConfig{
		Weights:   config.WeightConfig{},
		Penalties: config.PenaltyConfig{},
		RRF: config.RRFConfig{
			Enabled:     true,
			K:           60.0,
			DefaultRank: 1000,
			SignalWeights: map[string]float64{
				"searxng_position": 1.0,
				"text_relevance":   1.8,
				"domain_alignment": 0.8,
				"tranco_authority": 0.5,
				"engine_agreement": 0.6,
				"freshness":        0.4,
			},
		},
	}
	bl := blocklist.NewManager(&config.BlocklistConfig{}, nil)
	r := New(cfg, bl)
	r.SetMMRConfig(MMRConfig{})
	r.SetCollapseConfig(CollapseConfig{})
	r.SetRelevanceFloor(RelevanceFloorConfig{})
	return r
}

func TestNewsIntent_DatedDocBeatsUndated(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	results := []searxng.Result{
		{
			URL:           "https://example.com/dated",
			Title:         "Trump signs new executive order today",
			Content:       "Breaking news from the white house.",
			Engines:       []string{"google"},
			PublishedDate: now,
		},
		{
			URL:     "https://example.com/undated",
			Title:   "Trump signs new executive order today",
			Content: "Breaking news from the white house.",
			Engines: []string{"google"},
		},
	}

	r := freshnessRanker(t)
	scored := r.RankWithOptions(context.Background(), "trump executive order today", results, RankOptions{
		Category: "news",
	})
	require.Len(t, scored, 2)
	assert.Equal(t, "https://example.com/dated", scored[0].URL,
		"dated news result must outrank an identical undated one")
}

func TestNonNewsIntent_UndatedHasNoFreshnessRank(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	results := []searxng.Result{
		{URL: "https://example.com/dated", Title: "BM25 guide", Content: "x", Engines: []string{"google"}, PublishedDate: now},
		{URL: "https://example.com/undated", Title: "BM25 guide", Content: "x", Engines: []string{"google"}},
	}
	r := freshnessRanker(t)
	scored := r.RankWithOptions(context.Background(), "bm25", results, RankOptions{Category: "tech"})
	require.Len(t, scored, 2)
	for _, sr := range scored {
		switch sr.URL {
		case "https://example.com/dated":
			assert.NotZero(t, sr.SignalRanks["freshness"], "dated doc should have a freshness rank set")
		case "https://example.com/undated":
			_, hasFreshness := sr.SignalRanks["freshness"]
			assert.False(t, hasFreshness, "undated doc under non-news intent must NOT have a freshness rank - it falls to RRF DefaultRank")
		}
	}
}

func TestNewsIntent_UndatedAssignedWorstRankPlusOne(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	yesterday := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	results := []searxng.Result{
		{URL: "https://x/now", Title: "news A", Engines: []string{"google"}, PublishedDate: now},
		{URL: "https://x/yesterday", Title: "news B", Engines: []string{"google"}, PublishedDate: yesterday},
		{URL: "https://x/undated", Title: "news C", Engines: []string{"google"}},
	}
	r := freshnessRanker(t)
	scored := r.RankWithOptions(context.Background(), "trump", results, RankOptions{Category: "news"})
	require.Len(t, scored, 3)
	for _, sr := range scored {
		if sr.URL == "https://x/undated" {
			assert.Equal(t, 3, sr.SignalRanks["freshness"],
				"undated under news intent: rank = number of dated docs + 1 = 3")
		}
	}
}
