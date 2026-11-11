package ranker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollapseByTitleSimilarity_KeepsHighestScoredPerCluster(t *testing.T) {
	results := []ScoredResult{
		{URL: "https://bbc.com/election", Title: "2024 Election Results: Candidate Wins by Narrow Margin", FinalScore: 0.7},
		{URL: "https://cnn.com/election", Title: "2024 Election Results: Candidate Wins by Narrow Margin Tonight", FinalScore: 0.9},
		{URL: "https://reuters.com/election", Title: "2024 Election Results - Candidate Wins by Narrow Margin", FinalScore: 0.5},
		{URL: "https://espn.com/scores", Title: "Lakers beat Celtics 110-105", FinalScore: 0.6},
		{URL: "https://nasa.gov/launch", Title: "NASA launches new Mars rover", FinalScore: 0.4},
	}
	got := CollapseByTitleSimilarity(results, DefaultCollapseConfig())

	require.Len(t, got, 3, "the three near-duplicate election headlines must collapse to 1")
	urls := make(map[string]bool)
	for _, r := range got {
		urls[r.URL] = true
	}
	assert.True(t, urls["https://cnn.com/election"], "highest-scored election variant should survive")
	assert.True(t, urls["https://espn.com/scores"])
	assert.True(t, urls["https://nasa.gov/launch"])
}

func TestCollapseByTitleSimilarity_DisabledByConfig(t *testing.T) {
	results := []ScoredResult{
		{URL: "https://a.com/x", Title: "Same title", FinalScore: 0.5},
		{URL: "https://b.com/y", Title: "Same title", FinalScore: 0.6},
	}
	got := CollapseByTitleSimilarity(results, CollapseConfig{Enabled: false})
	assert.Len(t, got, 2)
}

func TestCollapseByTitleSimilarity_PreservesUniqueTitles(t *testing.T) {
	results := []ScoredResult{
		{URL: "https://a.com/x", Title: "Apple pie recipe", FinalScore: 0.5},
		{URL: "https://b.com/y", Title: "Linux kernel security audit", FinalScore: 0.5},
		{URL: "https://c.com/z", Title: "Climate change report", FinalScore: 0.5},
	}
	got := CollapseByTitleSimilarity(results, DefaultCollapseConfig())
	assert.Len(t, got, 3)
}

func TestCollapseByTitleSimilarity_PreservesOriginalOrder(t *testing.T) {
	results := []ScoredResult{
		{URL: "https://a.com/", Title: "alpha topic", FinalScore: 0.9},
		{URL: "https://b.com/", Title: "beta topic", FinalScore: 0.8},
		{URL: "https://c.com/", Title: "gamma topic", FinalScore: 0.7},
	}
	got := CollapseByTitleSimilarity(results, DefaultCollapseConfig())
	require.Len(t, got, 3)
	assert.Equal(t, "https://a.com/", got[0].URL)
	assert.Equal(t, "https://b.com/", got[1].URL)
	assert.Equal(t, "https://c.com/", got[2].URL)
}
