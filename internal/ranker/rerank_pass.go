package ranker

import (
	"context"
	"log/slog"
	"sort"

	"github.com/adriangitvitz/kurai/internal/reranker"
)

func ApplyReranker(ctx context.Context, query string, scored []ScoredResult, rr reranker.Reranker, topN int, alpha float64) []ScoredResult {
	if rr == nil || len(scored) == 0 {
		return scored
	}
	if topN <= 0 || topN > len(scored) {
		topN = len(scored)
	}
	if alpha <= 0 {
		return scored
	}
	if alpha > 1 {
		alpha = 1
	}

	head := scored[:topN]
	tail := scored[topN:]

	docs := make([]reranker.Doc, len(head))
	for i, r := range head {
		docs[i] = reranker.Doc{ID: r.URL, Text: r.Title + ". " + r.Snippet}
	}

	scores, err := rr.Rerank(ctx, query, docs)
	if err != nil {
		slog.Warn("reranker failed; returning RRF-only order", "error", err, "n", len(docs))
		return scored
	}
	if len(scores) != len(head) {
		slog.Warn("reranker returned wrong-length scores", "got", len(scores), "want", len(head))
		return scored
	}

	ceMin, ceMax := scores[0].Score, scores[0].Score
	for _, s := range scores[1:] {
		if s.Score < ceMin {
			ceMin = s.Score
		}
		if s.Score > ceMax {
			ceMax = s.Score
		}
	}
	ceRange := ceMax - ceMin
	if ceRange == 0 {
		return scored
	}

	rrfMin, rrfMax := head[0].FinalScore, head[0].FinalScore
	for _, r := range head[1:] {
		if r.FinalScore < rrfMin {
			rrfMin = r.FinalScore
		}
		if r.FinalScore > rrfMax {
			rrfMax = r.FinalScore
		}
	}
	rrfRange := rrfMax - rrfMin

	type indexed struct {
		blended float64
		i       int
	}
	blends := make([]indexed, len(head))
	for i := range head {
		ceN := (scores[i].Score - ceMin) / ceRange
		var rrfN float64
		if rrfRange == 0 {
			rrfN = 1.0
		} else {
			rrfN = (head[i].FinalScore - rrfMin) / rrfRange
		}
		blends[i] = indexed{
			blended: alpha*ceN + (1-alpha)*rrfN,
			i:       i,
		}
	}
	sort.SliceStable(blends, func(i, j int) bool {
		return blends[i].blended > blends[j].blended
	})

	out := make([]ScoredResult, 0, len(scored))
	for _, b := range blends {
		out = append(out, head[b.i])
	}
	out = append(out, tail...)
	return out
}
