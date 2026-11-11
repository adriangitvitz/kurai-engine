package ranker

import (
	"math"
	"strings"
	"unicode"
)

type BM25Config struct {
	K1          float64 `yaml:"k1"`
	B           float64 `yaml:"b"`
	Delta       float64 `yaml:"delta"`
	TitleWeight float64 `yaml:"title_weight"`
}

func DefaultBM25Config() BM25Config {
	return BM25Config{
		K1:          1.2,
		B:           0.75,
		Delta:       0.5,
		TitleWeight: 2.0,
	}
}

type BM25Scorer struct {
	cfg    BM25Config
	idf    map[string]float64
	avgdl  float64
	maxObs float64
}

func NewBM25Scorer(corpus [][]string, cfg BM25Config) *BM25Scorer {
	s := &BM25Scorer{cfg: cfg}
	if len(corpus) == 0 {
		s.idf = map[string]float64{}
		return s
	}

	var totalLen int
	for _, d := range corpus {
		totalLen += len(d)
	}
	s.avgdl = float64(totalLen) / float64(len(corpus))
	if s.avgdl == 0 {
		s.avgdl = 1
	}

	df := make(map[string]int)
	for _, d := range corpus {
		seen := make(map[string]struct{}, len(d))
		for _, tok := range d {
			if _, ok := seen[tok]; ok {
				continue
			}
			seen[tok] = struct{}{}
			df[tok]++
		}
	}
	N := float64(len(corpus))
	s.idf = make(map[string]float64, len(df))
	for tok, dft := range df {
		s.idf[tok] = math.Log(1 + (N-float64(dft)+0.5)/(float64(dft)+0.5))
	}

	for _, w := range s.idf {
		if w > s.maxObs {
			s.maxObs = w
		}
	}
	return s
}

func (s *BM25Scorer) Score(queryTerms []string, doc []string) float64 {
	if s == nil || len(doc) == 0 || len(queryTerms) == 0 {
		return 0
	}
	tf := make(map[string]int, len(doc))
	for _, t := range doc {
		tf[t]++
	}

	dl := float64(len(doc))
	lengthNorm := 1 - s.cfg.B + s.cfg.B*(dl/s.avgdl)

	var score float64
	for _, q := range queryTerms {
		idf, ok := s.idf[q]
		if !ok {
			continue
		}
		f := float64(tf[q])
		if f == 0 {
			continue
		}
		score += idf * ((f*(s.cfg.K1+1))/(f+s.cfg.K1*lengthNorm) + s.cfg.Delta)
	}
	return score
}

func (s *BM25Scorer) MaxObserved() float64 {
	if s == nil {
		return 0
	}
	return s.maxObs
}

func bm25Tokens(text string, titleWeight float64, title string) []string {
	bodyTokens := simpleTokens(text)
	if titleWeight <= 1 || title == "" {
		return bodyTokens
	}
	titleSet := make(map[string]struct{})
	for _, t := range simpleTokens(title) {
		titleSet[t] = struct{}{}
	}
	multiplier := int(titleWeight) - 1
	out := make([]string, 0, len(bodyTokens)+len(titleSet)*multiplier)
	out = append(out, bodyTokens...)
	for t := range titleSet {
		for i := 0; i < multiplier; i++ {
			out = append(out, t)
		}
	}
	return out
}

func simpleTokens(s string) []string {
	if s == "" {
		return nil
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	parts := strings.Fields(b.String())
	out := parts[:0]
	for _, p := range parts {
		if len(p) >= 2 {
			out = append(out, p)
		}
	}
	return out
}
