package reranker

import "context"

type Noop struct{}

func (Noop) Rerank(_ context.Context, _ string, docs []Doc) ([]Score, error) {
	out := make([]Score, len(docs))
	for i, d := range docs {
		out[i] = Score{ID: d.ID, Score: 0}
	}
	return out, nil
}
