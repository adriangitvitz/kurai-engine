package ranker

import (
	"strings"
)

func AnalyzeTextRelevanceBM25(queryTerms []string, scorer *BM25Scorer, title, snippet string) TextRelevanceScore {
	if scorer == nil || len(queryTerms) == 0 {
		return TextRelevanceScore{}
	}
	doc := bm25Tokens(title+" "+snippet, scorer.cfg.TitleWeight, title)
	raw := scorer.Score(queryTerms, doc)

	coverage := termCoverage(queryTerms, doc)

	denom := raw + scorer.MaxObserved()
	if denom == 0 {
		return TextRelevanceScore{Combined: 0, CoreTermCoverage: coverage}
	}
	combined := (raw / denom) * coverage
	if combined > 1 {
		combined = 1
	}
	if combined < 0 {
		combined = 0
	}

	return TextRelevanceScore{
		Combined:         combined,
		CoreTermCoverage: coverage,
	}
}

func termCoverage(queryTerms, doc []string) float64 {
	if len(queryTerms) == 0 {
		return 0
	}
	docSet := make(map[string]struct{}, len(doc))
	for _, t := range doc {
		docSet[t] = struct{}{}
	}
	hit := 0
	seen := make(map[string]struct{}, len(queryTerms))
	for _, q := range queryTerms {
		q = strings.ToLower(q)
		if _, dup := seen[q]; dup {
			continue
		}
		seen[q] = struct{}{}
		if _, ok := docSet[q]; ok {
			hit++
		}
	}
	return float64(hit) / float64(len(seen))
}
