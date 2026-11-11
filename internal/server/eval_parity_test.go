package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
)

func loadFixtureResults(t *testing.T, id string) []searxng.Result {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "ranker", "testdata", "fixtures", id+".json"))
	require.NoError(t, err)
	var f struct {
		Query   string           `json:"query"`
		Results []searxng.Result `json:"results"`
	}
	require.NoError(t, json.Unmarshal(data, &f))
	return f.Results
}

var fixtureCases = []struct {
	id, query, intent string
}{
	{"rust_ownership", "rust ownership tutorial", "tech"},
	{"kubernetes_pod_terminating", "kubernetes pod stuck terminating", "tech"},
	{"react_useeffect_cleanup", "react useEffect cleanup function", "tech"},
	{"cbt_overview", "what is cognitive behavioral therapy", "academic"},
	{"sqlite_vs_postgres", "sqlite vs postgres performance benchmark", "tech"},
}

func parityRanker(t *testing.T) *ranker.Ranker {
	t.Helper()
	cfg := &config.RankingConfig{
		RRF: config.RRFConfig{
			Enabled: true, K: 60, DefaultRank: 1000,
			SignalWeights: map[string]float64{
				"searxng_position": 1.0, "text_relevance": 1.8, "domain_alignment": 0.8,
				"tranco_authority": 0.5, "engine_agreement": 0.6, "freshness": 0.4,
			},
		},
	}
	bl := blocklist.NewManager(&config.BlocklistConfig{}, nil)
	return ranker.New(cfg, bl)
}

func TestEvalLive_SameFixtureSameOrder(t *testing.T) {
	for _, c := range fixtureCases {
		t.Run(c.id, func(t *testing.T) {
			results := loadFixtureResults(t, c.id)

			rA := parityRanker(t)
			evalOut := rA.RankWithOptions(context.Background(), c.query, results,
				ranker.RankOptions{Category: c.intent})

			deduped := deduplicateResults(results)
			rB := parityRanker(t)
			rankedB := rB.RankWithOptions(context.Background(), c.query, deduped,
				ranker.RankOptions{Category: c.intent})
			serviceOut, _ := applyPostRankFilters(rankedB, ServiceSearchReq{})

			limit := 10
			if len(evalOut) < limit {
				limit = len(evalOut)
			}
			if len(serviceOut) < limit {
				limit = len(serviceOut)
			}
			require.Greater(t, limit, 0, "no results returned")

			for i := 0; i < limit; i++ {
				assert.Equal(t, evalOut[i].URL, serviceOut[i].URL,
					"divergence at position %d:\n  eval=%s\n  live=%s",
					i+1, evalOut[i].URL, serviceOut[i].URL)
			}

			t.Logf("%s: dedup %d → %d (-%d), top-10 identical",
				c.id, len(results), len(deduped), len(results)-len(deduped))
		})
	}
}
