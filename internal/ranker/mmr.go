package ranker

import (
	"math"
	"strings"
	"unicode"
)

type MMRConfig struct {
	Enabled         bool    `yaml:"enabled"`
	Lambda          float64 `yaml:"lambda"`
	TopN            int     `yaml:"top_n"`
	SimilarityFloor float64 `yaml:"similarity_floor"`
}

func DefaultMMRConfig() MMRConfig {
	return MMRConfig{
		Enabled:         true,
		Lambda:          0.7,
		TopN:            50,
		SimilarityFloor: 0.5,
	}
}

func EnforceMMR(query string, results []ScoredResult, cfg MMRConfig) []ScoredResult {
	if len(results) <= 1 || cfg.Lambda >= 1.0 {
		return results
	}

	topN := cfg.TopN
	if topN <= 0 || topN > len(results) {
		topN = len(results)
	}

	cand := results[:topN]
	tail := results[topN:]

	docs := make([]map[string]float64, len(cand))
	df := make(map[string]int)
	for i, r := range cand {
		docs[i] = char3GramTF(r.Title + " " + truncRunes(r.Snippet, 200))
		for g := range docs[i] {
			df[g]++
		}
	}

	idf := make(map[string]float64, len(df))
	N := float64(len(cand))
	for g, dfg := range df {
		idf[g] = math.Log(1 + (N-float64(dfg)+0.5)/(float64(dfg)+0.5))
	}

	vecs := make([]map[string]float64, len(cand))
	norms := make([]float64, len(cand))
	for i, tf := range docs {
		v := make(map[string]float64, len(tf))
		var sumSq float64
		for g, c := range tf {
			w := c * idf[g]
			v[g] = w
			sumSq += w * w
		}
		vecs[i] = v
		norms[i] = math.Sqrt(sumSq)
	}

	_ = query
	qSim := normalizeFinalScores(cand)

	selected := make([]int, 0, len(cand))
	remaining := make(map[int]struct{}, len(cand))
	for i := range cand {
		remaining[i] = struct{}{}
	}

	for len(selected) < len(cand) {
		bestIdx := -1
		bestScore := math.Inf(-1)
		for i := range remaining {
			redund := 0.0
			for _, s := range selected {
				sim := cosine(vecs[i], norms[i], vecs[s], norms[s])
				if sim > redund {
					redund = sim
				}
			}
			score := cfg.Lambda*qSim[i] - (1-cfg.Lambda)*redund
			if cfg.SimilarityFloor > 0 && redund >= cfg.SimilarityFloor {
				score -= 100
			}
			if score > bestScore || (score == bestScore && (bestIdx == -1 || i < bestIdx)) {
				bestScore, bestIdx = score, i
			}
		}
		selected = append(selected, bestIdx)
		delete(remaining, bestIdx)
	}

	out := make([]ScoredResult, 0, len(results))
	for _, idx := range selected {
		out = append(out, cand[idx])
	}
	out = append(out, tail...)
	return out
}

func char3GramTF(s string) map[string]float64 {
	s = normalizeForGrams(s)
	if s == "" {
		return map[string]float64{}
	}
	padded := "  " + s + "  "
	tf := make(map[string]float64)
	total := 0
	rs := []rune(padded)
	if len(rs) < 3 {
		return tf
	}
	for i := 0; i+3 <= len(rs); i++ {
		g := string(rs[i : i+3])
		tf[g]++
		total++
	}
	if total == 0 {
		return tf
	}
	for g, c := range tf {
		tf[g] = c / float64(total)
	}
	return tf
}

func normalizeFinalScores(rs []ScoredResult) []float64 {
	out := make([]float64, len(rs))
	if len(rs) == 0 {
		return out
	}
	mn, mx := rs[0].FinalScore, rs[0].FinalScore
	for _, r := range rs[1:] {
		if r.FinalScore < mn {
			mn = r.FinalScore
		}
		if r.FinalScore > mx {
			mx = r.FinalScore
		}
	}
	rng := mx - mn
	if rng == 0 {
		for i := range out {
			out[i] = 0.5
		}
		return out
	}
	for i, r := range rs {
		out[i] = (r.FinalScore - mn) / rng
	}
	return out
}

func cosine(a map[string]float64, na float64, b map[string]float64, nb float64) float64 {
	if na == 0 || nb == 0 {
		return 0
	}
	if len(a) > len(b) {
		a, b = b, a
		na, nb = nb, na
	}
	var dot float64
	for g, w := range a {
		dot += w * b[g]
	}
	return dot / (na * nb)
}

func normalizeForGrams(s string) string {
	if s == "" {
		return s
	}
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r > 0x7F:
			b.WriteRune(' ')
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	return b.String()
}

func truncRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	count := 0
	for i := range s {
		if count >= n {
			return s[:i]
		}
		count++
	}
	return s
}
