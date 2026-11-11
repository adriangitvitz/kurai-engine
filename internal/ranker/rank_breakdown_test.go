package ranker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/searxng"
)

func TestRankBreakdown_RRFAndFinalRanksDifferAfterMMR(t *testing.T) {
	results := []searxng.Result{
		{URL: "https://a.com/x", Title: "election results candidate wins narrow margin", Engines: []string{"google", "bing", "brave"}, Score: 1.0},
		{URL: "https://b.com/x", Title: "election results candidate wins narrow margin tonight", Engines: []string{"google", "bing"}, Score: 0.9},
		{URL: "https://c.com/x", Title: "election results candidate wins by narrow margin", Engines: []string{"google"}, Score: 0.85},
		{URL: "https://d.com/x", Title: "lakers beat celtics in overtime", Engines: []string{"espn"}, Score: 0.5},
	}
	r := freshnessRanker(t)
	r.SetMMRConfig(MMRConfig{Enabled: true, Lambda: 0.5, TopN: 50, SimilarityFloor: 0.5})

	scored := r.RankWithOptions(context.Background(), "election results", results, RankOptions{})
	require.Len(t, scored, 4)

	for i, sr := range scored {
		assert.Equal(t, i+1, sr.FinalRank, "FinalRank must match position")
		assert.NotZero(t, sr.RRFRank, "RRFRank must be set")
	}

	moved := 0
	for _, sr := range scored {
		if sr.RRFRank != sr.FinalRank {
			moved++
		}
	}
	assert.Greater(t, moved, 0, "expected MMR to reorder at least one result, got 0 - test no longer exercises rank delta")
}

func TestRankBreakdown_RanksIdenticalWhenMMRDisabled(t *testing.T) {
	results := []searxng.Result{
		{URL: "https://a.com/x", Title: "alpha", Engines: []string{"google"}, Score: 1.0},
		{URL: "https://b.com/x", Title: "beta", Engines: []string{"google"}, Score: 0.5},
	}
	r := freshnessRanker(t)
	scored := r.RankWithOptions(context.Background(), "anything", results, RankOptions{})
	for _, sr := range scored {
		assert.Equal(t, sr.RRFRank, sr.FinalRank, "without MMR, RRF rank must equal final rank")
	}
}
