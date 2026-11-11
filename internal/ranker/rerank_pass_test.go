package ranker

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/reranker"
)

type stubReranker struct {
	scores []float64
	err    error
}

func (s stubReranker) Rerank(_ context.Context, _ string, docs []reranker.Doc) ([]reranker.Score, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make([]reranker.Score, len(docs))
	for i, d := range docs {
		var sc float64
		if i < len(s.scores) {
			sc = s.scores[i]
		}
		out[i] = reranker.Score{ID: d.ID, Score: sc}
	}
	return out, nil
}

func TestApplyReranker_BlendsAndReorders(t *testing.T) {
	scored := []ScoredResult{
		{URL: "a", Title: "Alpha", FinalScore: 0.9},
		{URL: "b", Title: "Beta", FinalScore: 0.8},
		{URL: "c", Title: "Gamma", FinalScore: 0.7},
	}
	rr := stubReranker{scores: []float64{0.1, 0.2, 0.95}}

	got := ApplyReranker(context.Background(), "q", scored, rr, 3, 1.0)
	require.Len(t, got, 3)
	assert.Equal(t, "c", got[0].URL, "with α=1, CE wins → Gamma should rise to top")
}

func TestApplyReranker_AlphaZeroPreservesRRFOrder(t *testing.T) {
	scored := []ScoredResult{
		{URL: "a", Title: "Alpha", FinalScore: 0.9},
		{URL: "b", Title: "Beta", FinalScore: 0.8},
	}
	rr := stubReranker{scores: []float64{0.1, 0.95}}
	got := ApplyReranker(context.Background(), "q", scored, rr, 2, 0.0)
	assert.Equal(t, "a", got[0].URL, "α=0 → CE ignored, RRF order kept")
}

func TestApplyReranker_PreservesTailBeyondTopN(t *testing.T) {
	scored := []ScoredResult{
		{URL: "a", Title: "Alpha", FinalScore: 0.9},
		{URL: "b", Title: "Beta", FinalScore: 0.8},
		{URL: "c", Title: "Gamma", FinalScore: 0.7},
		{URL: "d", Title: "Delta", FinalScore: 0.6},
	}
	rr := stubReranker{scores: []float64{0.1, 0.95}}
	got := ApplyReranker(context.Background(), "q", scored, rr, 2, 1.0)
	require.Len(t, got, 4)
	assert.Equal(t, "c", got[2].URL, "tail must be unchanged")
	assert.Equal(t, "d", got[3].URL)
}

func TestApplyReranker_ErrorFallsBackToRRF(t *testing.T) {
	scored := []ScoredResult{
		{URL: "a", Title: "Alpha", FinalScore: 0.9},
		{URL: "b", Title: "Beta", FinalScore: 0.8},
	}
	rr := stubReranker{err: errors.New("model not loaded")}
	got := ApplyReranker(context.Background(), "q", scored, rr, 2, 0.6)
	assert.Equal(t, scored, got, "error path returns RRF-only order")
}

func TestApplyReranker_NilRerankerNoop(t *testing.T) {
	scored := []ScoredResult{{URL: "a"}, {URL: "b"}}
	got := ApplyReranker(context.Background(), "q", scored, nil, 10, 0.6)
	assert.Equal(t, scored, got)
}

func TestApplyReranker_FlatCEScoresPreservesRRF(t *testing.T) {
	scored := []ScoredResult{
		{URL: "a", Title: "Alpha", FinalScore: 0.9},
		{URL: "b", Title: "Beta", FinalScore: 0.8},
	}
	rr := stubReranker{scores: []float64{0.5, 0.5}}
	got := ApplyReranker(context.Background(), "q", scored, rr, 2, 0.6)
	assert.Equal(t, "a", got[0].URL)
}
