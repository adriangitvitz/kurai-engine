package server

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/ranker"
)

func sampleScoredResult() ranker.ScoredResult {
	return ranker.ScoredResult{
		URL:             "https://example.com/a",
		Title:           "Example A",
		Snippet:         "Snippet for A",
		Engines:         []string{"bing", "brave"},
		OriginalScore:   1.5,
		AdjustedScore:   61.2,
		FinalScore:      0.0612,
		RRFScore:        0.0625,
		RRFRank:         2,
		FinalRank:       1,
		PrimaryDomain:   ranker.Domain("programming"),
		IntentAlignment: 0.42,
		DomainScores:    ranker.DomainScores{"programming": 0.8},
		SignalRanks:     map[string]int{"text_relevance": 1, "searxng_position": 3},
		Badges:          ranker.Badges{Reputable: true},
		PublishedDate:   "2026-01-15",
	}
}

func TestToCompact_StripsDebugFields(t *testing.T) {
	full := &SearchResponse{
		Query:   "q",
		Results: []ranker.ScoredResult{sampleScoredResult()},
		Meta:    SearchMeta{Total: 1, Page: 1, PageSize: 10},
	}

	compact := ToCompactSearchResponse(full)
	out, err := json.Marshal(compact)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	results := m["results"].([]any)
	require.Len(t, results, 1)
	r := results[0].(map[string]any)

	for _, want := range []string{"url", "title", "snippet", "score", "final_rank", "rrf_rank", "engines", "domain", "published_date"} {
		assert.Contains(t, r, want, "compact result must keep %q", want)
	}

	for _, gone := range []string{"badges", "signal_ranks", "domain_scores", "adjusted_score", "original_score", "rrf_score", "intent_alignment", "score_breakdown"} {
		assert.NotContains(t, r, gone, "compact result must NOT carry %q", gone)
	}

	assert.Equal(t, "q", m["query"])
	assert.Contains(t, m, "meta")
}

func TestToCompact_OmitsEmptyOptionalFields(t *testing.T) {
	sr := sampleScoredResult()
	sr.PublishedDate = ""
	sr.PrimaryDomain = ""
	full := &SearchResponse{Query: "q", Results: []ranker.ScoredResult{sr}}

	out, err := json.Marshal(ToCompactSearchResponse(full))
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))
	r := m["results"].([]any)[0].(map[string]any)
	assert.NotContains(t, r, "published_date", "empty date must be omitted, not nulled")
	assert.NotContains(t, r, "domain")
}

func TestToCompact_IsMeasurablySmaller(t *testing.T) {
	results := make([]ranker.ScoredResult, 10)
	for i := range results {
		results[i] = sampleScoredResult()
	}
	full := &SearchResponse{Query: "q", Results: results, Meta: SearchMeta{Total: 10}}

	fullJSON, _ := json.Marshal(full)
	compactJSON, _ := json.Marshal(ToCompactSearchResponse(full))
	reduction := 100 - len(compactJSON)*100/len(fullJSON)
	assert.GreaterOrEqual(t, reduction, 40,
		"compact must cut ≥40%% of bytes (got %d%%: %d → %d)", reduction, len(fullJSON), len(compactJSON))
}

func TestHandleSearch_ResponseFormatParam(t *testing.T) {
	srv := setupTestServer()

	resp, err := srv.ServiceSearch(t.Context(), ServiceSearchReq{UserID: "u", Query: "anything"})
	require.NoError(t, err)
	require.NotNil(t, resp)
	compact := ToCompactSearchResponse(resp)
	assert.Equal(t, resp.Query, compact.Query)
	assert.Len(t, compact.Results, len(resp.Results))
}
