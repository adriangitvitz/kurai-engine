package ranker

import (
	"math"

	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

type Judgments map[string]int

func NDCGAtK(ranked []ScoredResult, judgments Judgments, k int) float64 {
	if k <= 0 || len(judgments) == 0 {
		return 0
	}

	dcg := dcgAt(ranked, judgments, k)
	idcg := idealDCG(judgments, k)
	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}

func dcgAt(ranked []ScoredResult, judgments Judgments, k int) float64 {
	limit := k
	if len(ranked) < limit {
		limit = len(ranked)
	}
	var dcg float64
	for i := 0; i < limit; i++ {
		grade := judgments[ranked[i].URL]
		if grade == 0 {
			continue
		}
		gain := math.Pow(2, float64(grade)) - 1
		dcg += gain / math.Log2(float64(i+2))
	}
	return dcg
}

func idealDCG(judgments Judgments, k int) float64 {
	grades := make([]int, 0, len(judgments))
	for _, g := range judgments {
		if g > 0 {
			grades = append(grades, g)
		}
	}
	for i := 1; i < len(grades); i++ {
		for j := i; j > 0 && grades[j] > grades[j-1]; j-- {
			grades[j], grades[j-1] = grades[j-1], grades[j]
		}
	}
	limit := k
	if len(grades) < limit {
		limit = len(grades)
	}
	var idcg float64
	for i := 0; i < limit; i++ {
		gain := math.Pow(2, float64(grades[i])) - 1
		idcg += gain / math.Log2(float64(i+2))
	}
	return idcg
}

func MRR(ranked []ScoredResult, judgments Judgments) float64 {
	for i, r := range ranked {
		if judgments[r.URL] > 0 {
			return 1.0 / float64(i+1)
		}
	}
	return 0
}

func RecallAtK(ranked []ScoredResult, judgments Judgments, k int) float64 {
	totalRelevant := 0
	for _, g := range judgments {
		if g > 0 {
			totalRelevant++
		}
	}
	if totalRelevant == 0 {
		return 0
	}
	limit := k
	if len(ranked) < limit {
		limit = len(ranked)
	}
	hit := 0
	for i := 0; i < limit; i++ {
		if judgments[ranked[i].URL] > 0 {
			hit++
		}
	}
	return float64(hit) / float64(totalRelevant)
}

func JudgmentsForDomain(results []ScoredResult, byDomain map[string]int) Judgments {
	if len(byDomain) == 0 {
		return Judgments{}
	}
	j := make(Judgments, len(results))
	for _, r := range results {
		domain := urlutil.ExtractDomain(r.URL)
		if grade, ok := byDomain[domain]; ok {
			j[r.URL] = grade
		}
	}
	return j
}
