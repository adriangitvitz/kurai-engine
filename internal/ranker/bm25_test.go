package ranker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type docPair struct{ title, snippet string }

func newScorerFromDocs(docs []docPair, cfg BM25Config) *BM25Scorer {
	corpus := make([][]string, len(docs))
	for i, d := range docs {
		corpus[i] = bm25Tokens(d.title+" "+d.snippet, cfg.TitleWeight, d.title)
	}
	return NewBM25Scorer(corpus, cfg)
}

func TestBM25_PicksOnTopicOverKeywordStuffed(t *testing.T) {
	cfg := DefaultBM25Config()
	docs := []docPair{
		{"python python python python python python error error error error",
			"python python python python python error error error error"},
		{"How to debug a Python error in production",
			"This guide walks through diagnosing a Python error: how to read tracebacks, find the root cause, and fix the underlying bug."},
		{"Cooking recipes for beginners",
			"Cake and pasta recipes."},
	}
	s := newScorerFromDocs(docs, cfg)
	queryTerms := []string{"python", "error"}

	scoreStuffed := s.Score(queryTerms, bm25Tokens(docs[0].title+" "+docs[0].snippet, cfg.TitleWeight, docs[0].title))
	scoreOnTopic := s.Score(queryTerms, bm25Tokens(docs[1].title+" "+docs[1].snippet, cfg.TitleWeight, docs[1].title))
	scoreOffTopic := s.Score(queryTerms, bm25Tokens(docs[2].title+" "+docs[2].snippet, cfg.TitleWeight, docs[2].title))

	t.Logf("stuffed=%.3f on-topic=%.3f off-topic=%.3f", scoreStuffed, scoreOnTopic, scoreOffTopic)
	assert.Greater(t, scoreOnTopic, scoreOffTopic, "on-topic must beat unrelated")
	assert.Less(t, scoreStuffed, scoreOnTopic*3,
		"stuffed should not score >3x the on-topic doc; got stuffed=%.3f on-topic=%.3f",
		scoreStuffed, scoreOnTopic)
}

func TestBM25_RareTermBoostsScore(t *testing.T) {
	cfg := DefaultBM25Config()
	docs := []docPair{
		{"the the", "the"},
		{"the the", "the"},
		{"the the", "the"},
		{"the rust ownership", "rust language"},
		{"the the", "the"},
	}
	s := newScorerFromDocs(docs, cfg)

	rustScore := s.Score([]string{"rust"}, bm25Tokens(docs[3].title+" "+docs[3].snippet, cfg.TitleWeight, docs[3].title))
	theScore := s.Score([]string{"the"}, bm25Tokens(docs[3].title+" "+docs[3].snippet, cfg.TitleWeight, docs[3].title))
	t.Logf("rare=%.3f common=%.3f", rustScore, theScore)
	assert.Greater(t, rustScore, theScore*2, "rare term should dominate common term")
}

func TestBM25_TitleTokensCountMore(t *testing.T) {
	cfg := DefaultBM25Config()
	cfg.TitleWeight = 3.0

	corpus := [][]string{
		bm25Tokens("python error tutorial xxx yyy zzz", cfg.TitleWeight, "python error tutorial"),
		bm25Tokens("tutorial xxx yyy zzz with the term python error mentioned once in body",
			cfg.TitleWeight, "tutorial xxx yyy zzz"),
	}
	s := NewBM25Scorer(corpus, cfg)

	scoreA := s.Score([]string{"python", "error"}, corpus[0])
	scoreB := s.Score([]string{"python", "error"}, corpus[1])
	require.Greater(t, scoreA, scoreB,
		"doc with query terms in title (multiplied by TitleWeight) should outrank body-only mention")
}

func TestBM25Tokens_EmitsLowercasedTokensWithTitleMultiplier(t *testing.T) {
	got := bm25Tokens("Python Error Tutorial: How To Fix It", 3.0, "Python Error Tutorial:")
	count := func(token string) int {
		c := 0
		for _, t := range got {
			if t == token {
				c++
			}
		}
		return c
	}
	assert.Equal(t, 3, count("python"), "title token should be multiplied by TitleWeight")
	assert.Equal(t, 3, count("error"))
	assert.Equal(t, 3, count("tutorial"))
	assert.Equal(t, 1, count("how"))
	assert.Equal(t, 1, count("fix"))
}

func TestAnalyzeTextRelevanceBM25_NormalizedToZeroOne(t *testing.T) {
	cfg := DefaultBM25Config()
	corpus := [][]string{
		bm25Tokens("python error help", cfg.TitleWeight, "python error help"),
		bm25Tokens("python error other", cfg.TitleWeight, "python error other"),
	}
	s := NewBM25Scorer(corpus, cfg)
	score := AnalyzeTextRelevanceBM25([]string{"python"}, s, "python error help", "")
	assert.GreaterOrEqual(t, score.Combined, 0.0)
	assert.LessOrEqual(t, score.Combined, 1.0, "combined must be 0..1 for RRF ranking comparability")
}
