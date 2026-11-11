package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
)

func TestInferPublishedDates(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	results := []searxng.Result{
		{Content: "Apr 17, 2026 · Northern Trust is working with Digital Asset"},
		{Content: "September 3, 2026 - A longer month name"},
		{Content: "3 days ago · Recent coverage"},
		{Content: "Learn more. Founded Apr 17, 1889 · in Chicago"},
		{Content: "Feb 31, 2026 · an impossible date"},
		{Content: "Apr 17, 2026 · already dated upstream", PublishedDate: "2026-01-01T00:00:00Z"},
		{Content: "No date here"},
	}
	inferPublishedDates(results, now)

	got := make([]string, len(results))
	for i, r := range results {
		got[i] = r.PublishedDate
	}
	assert.Equal(t, []string{"2026-04-17", "2026-09-03", "2026-11-09", "", "", "2026-01-01T00:00:00Z", ""}, got)
}

func TestApplyPostRankFilters_InferredDatesAreFilteredExactly(t *testing.T) {
	in := []searxng.Result{
		{URL: "https://a.example/april", Content: "Apr 17, 2026 · Northern Trust moves into custody"},
		{URL: "https://b.example/september", Content: "Sep 2, 2026 · Follow-up coverage"},
	}
	inferPublishedDates(in, time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC))
	scored := make([]ranker.ScoredResult, len(in))
	for i, r := range in {
		scored[i] = ranker.ScoredResult{URL: r.URL, Snippet: r.Content, PublishedDate: r.PublishedDate}
	}

	got, _ := applyPostRankFilters(scored, ServiceSearchReq{DateFrom: "2026-08-01", DateTo: "2026-11-12"})
	assert.Len(t, got, 1)
	assert.Equal(t, "https://b.example/september", got[0].URL)
}
