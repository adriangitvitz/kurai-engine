package ranker

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mkResult(url string) ScoredResult {
	return ScoredResult{URL: url}
}

func mkRanked(urls ...string) []ScoredResult {
	out := make([]ScoredResult, 0, len(urls))
	for _, u := range urls {
		out = append(out, mkResult(u))
	}
	return out
}

func TestNDCGAtK_PerfectRankingScoresOne(t *testing.T) {
	j := Judgments{
		"https://a.com": 2,
		"https://b.com": 1,
		"https://c.com": 0,
	}
	got := NDCGAtK(mkRanked("https://a.com", "https://b.com", "https://c.com"), j, 3)
	assert.InDelta(t, 1.0, got, 1e-9)
}

func TestNDCGAtK_WorstReasonableRankingIsLow(t *testing.T) {
	j := Judgments{
		"https://a.com": 2,
		"https://b.com": 1,
		"https://c.com": 0,
	}
	got := NDCGAtK(mkRanked("https://c.com", "https://b.com", "https://a.com"), j, 3)
	assert.Less(t, got, 0.7)
	assert.Greater(t, got, 0.0)
}

func TestNDCGAtK_EmptyJudgmentsIsZero(t *testing.T) {
	got := NDCGAtK(mkRanked("https://a.com"), Judgments{}, 10)
	assert.Equal(t, 0.0, got)
}

func TestNDCGAtK_HonorsKCutoff(t *testing.T) {
	j := Judgments{
		"https://a.com": 2,
		"https://b.com": 2,
	}
	got := NDCGAtK(mkRanked("https://a.com", "https://b.com"), j, 1)
	assert.InDelta(t, 1.0, got, 1e-9)
}

func TestNDCGAtK_NaNOnZeroIDCG(t *testing.T) {
	j := Judgments{"https://a.com": 0, "https://b.com": 0}
	got := NDCGAtK(mkRanked("https://a.com"), j, 5)
	assert.False(t, math.IsNaN(got))
	assert.Equal(t, 0.0, got)
}

func TestMRR_FirstRelevantAtPositionTwo(t *testing.T) {
	j := Judgments{"https://a.com": 0, "https://b.com": 1}
	got := MRR(mkRanked("https://a.com", "https://b.com"), j)
	assert.InDelta(t, 0.5, got, 1e-9)
}

func TestMRR_NoRelevantIsZero(t *testing.T) {
	got := MRR(mkRanked("https://a.com"), Judgments{"https://a.com": 0})
	assert.Equal(t, 0.0, got)
}

func TestRecallAtK_AllRelevantInTopK(t *testing.T) {
	j := Judgments{
		"https://a.com": 1,
		"https://b.com": 2,
		"https://c.com": 0,
	}
	got := RecallAtK(mkRanked("https://a.com", "https://b.com"), j, 2)
	assert.InDelta(t, 1.0, got, 1e-9)
}

func TestRecallAtK_HalfMissing(t *testing.T) {
	j := Judgments{
		"https://a.com": 1,
		"https://b.com": 2,
	}
	got := RecallAtK(mkRanked("https://a.com"), j, 1)
	assert.InDelta(t, 0.5, got, 1e-9)
}

func TestJudgments_ByDomainExpandsToURLLookups(t *testing.T) {
	results := mkRanked("https://a.com/x", "https://b.com/y", "https://www.a.com/z")
	j := JudgmentsForDomain(results, map[string]int{
		"a.com": 2,
		"b.com": 1,
	})
	assert.Equal(t, 2, j["https://a.com/x"])
	assert.Equal(t, 1, j["https://b.com/y"])
	assert.Equal(t, 2, j["https://www.a.com/z"], "www. prefix must match a.com")
}
