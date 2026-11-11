package ranker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mmrCfg() MMRConfig {
	return MMRConfig{Lambda: 0.7, TopN: 50, SimilarityFloor: 0.5}
}

func TestMMR_LambdaOneIsIdentity(t *testing.T) {
	results := []ScoredResult{
		{URL: "https://a.com/1", Title: "Apple pie recipe", Snippet: "How to bake an apple pie."},
		{URL: "https://b.com/2", Title: "Quantum entanglement explained", Snippet: "Physics paper."},
		{URL: "https://c.com/3", Title: "Apple pie variations", Snippet: "Spiced and savory."},
	}
	cfg := mmrCfg()
	cfg.Lambda = 1.0

	got := EnforceMMR("apple pie", results, cfg)
	require.Len(t, got, 3)
	assert.Equal(t, "https://a.com/1", got[0].URL)
}

func TestMMR_PromotesDiversityAcrossNearDuplicates(t *testing.T) {
	results := []ScoredResult{
		{URL: "https://bbc.com/election", Title: "2024 Election Results: Candidate Wins by Narrow Margin"},
		{URL: "https://cnn.com/election", Title: "2024 Election Results: Candidate Wins by Narrow Margin Tonight"},
		{URL: "https://reuters.com/election", Title: "2024 Election Results - Candidate Wins by Narrow Margin"},
		{URL: "https://espn.com/scores", Title: "Lakers beat Celtics 110-105 - game recap and box score"},
		{URL: "https://nasa.gov/launch", Title: "NASA launches new Mars rover from Cape Canaveral on Saturday"},
	}
	cfg := mmrCfg()
	cfg.Lambda = 0.6

	got := EnforceMMR("2024 election results", results, cfg)
	require.Len(t, got, 5)

	dups := map[string]bool{
		"https://bbc.com/election":     true,
		"https://cnn.com/election":     true,
		"https://reuters.com/election": true,
	}
	dupHits := 0
	for _, r := range got[:3] {
		if dups[r.URL] {
			dupHits++
		}
	}
	assert.LessOrEqual(t, dupHits, 1, "top 3 should not be 3 near-duplicates; got %v", urlList(got[:3]))
}

func TestMMR_PreservesAllResults(t *testing.T) {
	results := []ScoredResult{
		{URL: "https://a.com/x", Title: "alpha"},
		{URL: "https://b.com/y", Title: "beta"},
		{URL: "https://c.com/z", Title: "gamma"},
	}
	got := EnforceMMR("anything", results, mmrCfg())
	assert.Len(t, got, len(results), "MMR must reorder, not drop")
}

func TestMMR_EmptyAndSingle(t *testing.T) {
	assert.Empty(t, EnforceMMR("q", nil, mmrCfg()))
	one := []ScoredResult{{URL: "https://a.com/x", Title: "alpha"}}
	got := EnforceMMR("q", one, mmrCfg())
	assert.Equal(t, one, got)
}

func TestMMR_ConfinedToTopN(t *testing.T) {
	results := []ScoredResult{
		{URL: "https://1.com/", Title: "near duplicate one"},
		{URL: "https://2.com/", Title: "near duplicate two"},
		{URL: "https://3.com/", Title: "near duplicate three"},
		{URL: "https://4.com/", Title: "tail four"},
		{URL: "https://5.com/", Title: "tail five"},
		{URL: "https://6.com/", Title: "tail six"},
	}
	cfg := mmrCfg()
	cfg.TopN = 3
	got := EnforceMMR("near duplicate", results, cfg)
	require.Len(t, got, 6)
	assert.Equal(t, "https://4.com/", got[3].URL)
	assert.Equal(t, "https://5.com/", got[4].URL)
	assert.Equal(t, "https://6.com/", got[5].URL)
}

func BenchmarkMMR50(b *testing.B) {
	results := make([]ScoredResult, 50)
	for i := range results {
		results[i] = ScoredResult{
			URL:     "https://example.com/" + string(rune(i)),
			Title:   "Generic article number " + string(rune(i)) + " about machine learning and ranking",
			Snippet: "Some snippet text discussing variations of search and ranking algorithms.",
		}
	}
	cfg := mmrCfg()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EnforceMMR("machine learning ranking", results, cfg)
	}
}

func urlList(rs []ScoredResult) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.URL
	}
	return out
}
