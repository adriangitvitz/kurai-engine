package ranker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMMR_HighestFinalScoreSurvivesWhenUnique(t *testing.T) {
	results := []ScoredResult{
		{URL: "https://react.dev/reference/react/useEffect", Title: "useEffect – React",
			Snippet: "API reference.", FinalScore: 0.0748},
		{URL: "https://blog.example.com/long",
			Title:      "Understanding the React useEffect Cleanup Function in Detail with Examples",
			Snippet:    "Long-form blog post about useEffect cleanup function semantics.",
			FinalScore: 0.0680},
		{URL: "https://example.com/recipe", Title: "Apple pie recipe",
			Snippet: "Cooking instructions.", FinalScore: 0.0400},
	}
	cfg := DefaultMMRConfig()
	got := EnforceMMR("react useEffect cleanup function", results, cfg)
	require.Len(t, got, 3)
	assert.Equal(t, "https://react.dev/reference/react/useEffect", got[0].URL,
		"highest-FinalScore doc must remain at #1 when no near-duplicates exist")
}

func TestMMR_BlendStillPromotesDiversity(t *testing.T) {
	results := []ScoredResult{
		{URL: "https://bbc.com/election", Title: "2024 Election Results: Candidate Wins by Narrow Margin",
			FinalScore: 0.10},
		{URL: "https://cnn.com/election", Title: "2024 Election Results: Candidate Wins by Narrow Margin Tonight",
			FinalScore: 0.09},
		{URL: "https://reuters.com/election", Title: "2024 Election Results - Candidate Wins by Narrow Margin",
			FinalScore: 0.08},
		{URL: "https://espn.com/scores", Title: "Lakers beat Celtics 110-105 - game recap and box score",
			FinalScore: 0.07},
		{URL: "https://nasa.gov/launch", Title: "NASA launches new Mars rover from Cape Canaveral on Saturday",
			FinalScore: 0.06},
	}
	cfg := DefaultMMRConfig()
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
	assert.LessOrEqual(t, dupHits, 1,
		"top 3 must not contain >1 of the dup cluster; got %v", urlList(got[:3]))
}

func TestMMR_GracefulOnZeroFinalScores(t *testing.T) {
	results := []ScoredResult{
		{URL: "https://a.com/", Title: "alpha", FinalScore: 0},
		{URL: "https://b.com/", Title: "beta", FinalScore: 0},
		{URL: "https://c.com/", Title: "gamma", FinalScore: 0},
	}
	got := EnforceMMR("anything", results, DefaultMMRConfig())
	require.Len(t, got, 3)
	assert.Equal(t, "https://a.com/", got[0].URL)
}
