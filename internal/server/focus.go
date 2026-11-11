package server

import (
	"sort"

	"github.com/adriangitvitz/kurai/internal/extract"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

type ScoredPassage struct {
	Heading string  `json:"heading,omitempty"`
	Text    string  `json:"text"`
	Score   float64 `json:"score"`
}

const defaultTopPassages = 5

func scorePassages(md, focus string, topN int) []ScoredPassage {
	if md == "" || focus == "" {
		return nil
	}
	if topN <= 0 {
		topN = defaultTopPassages
	}

	passages := extract.SplitPassages(md)
	if len(passages) == 0 {
		return nil
	}

	corpus := make([][]string, len(passages))
	for i, p := range passages {
		tokens := urlutil.Tokenize(p.Text)
		heading := urlutil.Tokenize(p.Heading)
		tokens = append(tokens, heading...)
		tokens = append(tokens, heading...)
		corpus[i] = tokens
	}

	scorer := ranker.NewBM25Scorer(corpus, ranker.DefaultBM25Config())
	queryTerms := urlutil.Tokenize(focus)

	scored := make([]ScoredPassage, len(passages))
	for i, p := range passages {
		scored[i] = ScoredPassage{
			Heading: p.Heading,
			Text:    p.Text,
			Score:   scorer.Score(queryTerms, corpus[i]),
		}
	}

	sort.SliceStable(scored, func(a, b int) bool {
		return scored[a].Score > scored[b].Score
	})

	if len(scored) > topN {
		scored = scored[:topN]
	}
	return scored
}
