package server

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/searxng"
)

func TestDeduplicateResults_CollapsesURLVariantsAndMergesEngines(t *testing.T) {
	in := []searxng.Result{
		{
			URL:     "https://example.com/article?utm_source=twitter&fbclid=abc",
			Title:   "An Article",
			Engines: []string{"google"},
			Score:   0.9,
		},
		{
			URL:     "http://example.com/article/",
			Title:   "An Article",
			Engines: []string{"bing"},
			Score:   0.8,
		},
		{
			URL:     "https://amp.example.com/article",
			Title:   "An Article",
			Engines: []string{"brave"},
			Score:   0.5,
		},
		{
			URL:     "https://other.com/different",
			Title:   "Different Article",
			Engines: []string{"google"},
			Score:   0.7,
		},
	}

	got := deduplicateResults(in)

	require.Len(t, got, 2, "three URL variants of the same article must collapse")

	var merged searxng.Result
	for _, r := range got {
		if r.Title == "An Article" {
			merged = r
		}
	}
	require.NotEmpty(t, merged.URL)

	engines := append([]string(nil), merged.Engines...)
	sort.Strings(engines)
	assert.Equal(t, []string{"bing", "brave", "google"}, engines, "engine sets must merge")

	assert.InDelta(t, 0.9, merged.Score, 1e-9, "higher-scored URL must be retained")
}

func TestDeduplicateResults_PreservesUniqueURLs(t *testing.T) {
	in := []searxng.Result{
		{URL: "https://a.com/x", Engines: []string{"google"}},
		{URL: "https://b.com/y", Engines: []string{"bing"}},
		{URL: "https://c.com/z", Engines: []string{"brave"}},
	}
	got := deduplicateResults(in)
	assert.Len(t, got, 3)
}

func TestDeduplicateResults_EmptyInput(t *testing.T) {
	got := deduplicateResults(nil)
	assert.Empty(t, got)
}
