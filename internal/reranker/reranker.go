package reranker

import "context"

type Doc struct {
	ID   string
	Text string
}

type Score struct {
	ID    string
	Score float64
}

type Reranker interface {
	Rerank(ctx context.Context, query string, docs []Doc) ([]Score, error)
}
