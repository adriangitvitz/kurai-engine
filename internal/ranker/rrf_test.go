package ranker

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRRFConfig(t *testing.T) {
	cfg := DefaultRRFConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 60.0, cfg.K)
	assert.Equal(t, 1000, cfg.DefaultRank)
	assert.Len(t, cfg.SignalWeights, 6)
	assert.Equal(t, 1.0, cfg.SignalWeights["searxng_position"])
	assert.Equal(t, 1.8, cfg.SignalWeights["text_relevance"])
	assert.Equal(t, 0.8, cfg.SignalWeights["domain_alignment"])
	assert.Equal(t, 0.4, cfg.SignalWeights["freshness"])
}

func TestRRFScore_SingleSignal(t *testing.T) {
	config := RRFConfig{
		K:           60.0,
		DefaultRank: 1000,
		SignalWeights: map[string]float64{
			"test_signal": 1.0,
		},
	}

	ranks := map[string]int{"test_signal": 1}
	score := RRFScore(ranks, config)

	expected := 1.0 / 61.0
	assert.InDelta(t, expected, score, 0.0001)
}

func TestRRFScore_MultipleSignals(t *testing.T) {
	config := RRFConfig{
		K:           60.0,
		DefaultRank: 1000,
		SignalWeights: map[string]float64{
			"signal_a": 1.0,
			"signal_b": 1.0,
		},
	}

	ranks := map[string]int{
		"signal_a": 1,
		"signal_b": 1,
	}
	score := RRFScore(ranks, config)

	expected := 2.0 / 61.0
	assert.InDelta(t, expected, score, 0.0001)
}

func TestRRFScore_WeightedSignals(t *testing.T) {
	config := RRFConfig{
		K:           60.0,
		DefaultRank: 1000,
		SignalWeights: map[string]float64{
			"high_weight": 2.0,
			"low_weight":  0.5,
		},
	}

	ranks := map[string]int{
		"high_weight": 1,
		"low_weight":  1,
	}
	score := RRFScore(ranks, config)

	expected := (2.0 + 0.5) / 61.0
	assert.InDelta(t, expected, score, 0.0001)
}

func TestRRFScore_MissingSignal(t *testing.T) {
	config := RRFConfig{
		K:           60.0,
		DefaultRank: 1000,
		SignalWeights: map[string]float64{
			"present": 1.0,
			"missing": 1.0,
		},
	}

	ranks := map[string]int{
		"present": 1,
	}
	score := RRFScore(ranks, config)

	expected := (1.0 / 61.0) + (1.0 / 1060.0)
	assert.InDelta(t, expected, score, 0.0001)
}

func TestRRFScore_HighRank(t *testing.T) {
	config := RRFConfig{
		K:           60.0,
		DefaultRank: 1000,
		SignalWeights: map[string]float64{
			"signal": 1.0,
		},
	}

	scoreRank1 := RRFScore(map[string]int{"signal": 1}, config)
	scoreRank100 := RRFScore(map[string]int{"signal": 100}, config)

	assert.Greater(t, scoreRank1, scoreRank100)
	assert.InDelta(t, 1.0/61.0, scoreRank1, 0.0001)
	assert.InDelta(t, 1.0/160.0, scoreRank100, 0.0001)
}

func TestRRFScore_DifferentK(t *testing.T) {
	configLowK := RRFConfig{
		K:             10.0,
		DefaultRank:   1000,
		SignalWeights: map[string]float64{"signal": 1.0},
	}
	configHighK := RRFConfig{
		K:             100.0,
		DefaultRank:   1000,
		SignalWeights: map[string]float64{"signal": 1.0},
	}

	ranks := map[string]int{"signal": 1}

	scoreLowK := RRFScore(ranks, configLowK)
	scoreHighK := RRFScore(ranks, configHighK)

	assert.Greater(t, scoreLowK, scoreHighK)
}

func TestRRFContributions(t *testing.T) {
	config := RRFConfig{
		K:           60.0,
		DefaultRank: 1000,
		SignalWeights: map[string]float64{
			"signal_a": 1.0,
			"signal_b": 0.5,
		},
	}

	ranks := map[string]int{
		"signal_a": 1,
		"signal_b": 10,
	}

	contributions := RRFContributions(ranks, config)

	assert.InDelta(t, 1.0/61.0, contributions["signal_a"], 0.0001)
	assert.InDelta(t, 0.5/70.0, contributions["signal_b"], 0.0001)
}

func TestRankBySignal_Basic(t *testing.T) {
	results := []ScoredResult{
		{URL: "http://a.com", OriginalScore: 10},
		{URL: "http://b.com", OriginalScore: 30},
		{URL: "http://c.com", OriginalScore: 20},
	}

	ranks := RankBySignal(results, func(r ScoredResult) float64 {
		return r.OriginalScore
	})

	assert.Equal(t, 1, ranks["http://b.com"])
	assert.Equal(t, 2, ranks["http://c.com"])
	assert.Equal(t, 3, ranks["http://a.com"])
}

func TestRankBySignal_Empty(t *testing.T) {
	ranks := RankBySignal([]ScoredResult{}, func(r ScoredResult) float64 {
		return r.OriginalScore
	})

	assert.Empty(t, ranks)
}

func TestRankBySignal_SingleResult(t *testing.T) {
	results := []ScoredResult{
		{URL: "http://only.com", OriginalScore: 50},
	}

	ranks := RankBySignal(results, func(r ScoredResult) float64 {
		return r.OriginalScore
	})

	assert.Equal(t, 1, ranks["http://only.com"])
}

func TestRankBySignal_TiesShareTheBestRank(t *testing.T) {
	results := []ScoredResult{
		{URL: "http://a.com", OriginalScore: 30},
		{URL: "http://b.com", OriginalScore: 30},
		{URL: "http://c.com", OriginalScore: 20},
		{URL: "http://d.com", OriginalScore: 20},
		{URL: "http://e.com", OriginalScore: 10},
	}

	ranks := RankBySignal(results, func(r ScoredResult) float64 {
		return r.OriginalScore
	})

	assert.Equal(t, 1, ranks["http://a.com"])
	assert.Equal(t, 1, ranks["http://b.com"])
	assert.Equal(t, 3, ranks["http://c.com"])
	assert.Equal(t, 3, ranks["http://d.com"])
	assert.Equal(t, 5, ranks["http://e.com"])
}

func TestCombineRanks(t *testing.T) {
	signalRanks := map[string]map[string]int{
		"signal_a": {
			"http://x.com": 1,
			"http://y.com": 2,
		},
		"signal_b": {
			"http://x.com": 2,
			"http://y.com": 1,
		},
	}

	combined := CombineRanks(signalRanks)

	require.Contains(t, combined, "http://x.com")
	require.Contains(t, combined, "http://y.com")

	assert.Equal(t, 1, combined["http://x.com"]["signal_a"])
	assert.Equal(t, 2, combined["http://x.com"]["signal_b"])
	assert.Equal(t, 2, combined["http://y.com"]["signal_a"])
	assert.Equal(t, 1, combined["http://y.com"]["signal_b"])
}

func TestNormalizeRRFScores(t *testing.T) {
	scores := map[string]float64{
		"http://a.com": 0.1,
		"http://b.com": 0.3,
		"http://c.com": 0.5,
	}

	normalized := NormalizeRRFScores(scores)

	assert.InDelta(t, 0.0, normalized["http://a.com"], 0.0001)
	assert.InDelta(t, 0.5, normalized["http://b.com"], 0.0001)
	assert.InDelta(t, 1.0, normalized["http://c.com"], 0.0001)
}

func TestNormalizeRRFScores_AllEqual(t *testing.T) {
	scores := map[string]float64{
		"http://a.com": 0.5,
		"http://b.com": 0.5,
		"http://c.com": 0.5,
	}

	normalized := NormalizeRRFScores(scores)

	assert.Equal(t, 1.0, normalized["http://a.com"])
	assert.Equal(t, 1.0, normalized["http://b.com"])
	assert.Equal(t, 1.0, normalized["http://c.com"])
}

func TestNormalizeRRFScores_Empty(t *testing.T) {
	scores := map[string]float64{}
	normalized := NormalizeRRFScores(scores)
	assert.Empty(t, normalized)
}

func TestRRFScore_RealWorldScenario(t *testing.T) {
	config := DefaultRRFConfig()

	ranksA := map[string]int{
		"searxng_position": 1,
		"text_relevance":   2,
		"domain_alignment": 1,
		"tranco_authority": 5,
		"engine_agreement": 1,
	}

	ranksB := map[string]int{
		"searxng_position": 50,
		"text_relevance":   30,
		"domain_alignment": 40,
		"tranco_authority": 100,
		"engine_agreement": 10,
	}

	scoreA := RRFScore(ranksA, config)
	scoreB := RRFScore(ranksB, config)

	assert.Greater(t, scoreA, scoreB)

	assert.Greater(t, scoreA, 0.05)
	assert.Less(t, scoreA, 0.15)
}

func BenchmarkRRFScore(b *testing.B) {
	config := DefaultRRFConfig()
	ranks := map[string]int{
		"searxng_position": 5,
		"text_relevance":   10,
		"domain_alignment": 3,
		"tranco_authority": 50,
		"engine_agreement": 2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RRFScore(ranks, config)
	}
}

func BenchmarkRankBySignal(b *testing.B) {
	results := make([]ScoredResult, 100)
	for i := 0; i < 100; i++ {
		results[i] = ScoredResult{
			URL:           "http://example" + string(rune(i)) + ".com",
			OriginalScore: float64(100 - i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RankBySignal(results, func(r ScoredResult) float64 {
			return r.OriginalScore
		})
	}
}

func TestRRF_RobustnessToOutliers(t *testing.T) {
	config := RRFConfig{
		K:           60.0,
		DefaultRank: 1000,
		SignalWeights: map[string]float64{
			"signal_a": 1.0,
			"signal_b": 1.0,
			"signal_c": 1.0,
		},
	}

	ranksX := map[string]int{"signal_a": 5, "signal_b": 5, "signal_c": 5}

	ranksY := map[string]int{"signal_a": 1, "signal_b": 100, "signal_c": 100}

	scoreX := RRFScore(ranksX, config)
	scoreY := RRFScore(ranksY, config)

	assert.Greater(t, scoreX, scoreY,
		"RRF should favor consistent rankings over outlier performance")

	expectedX := 3.0 / 65.0
	expectedY := (1.0 / 61.0) + (2.0 / 160.0)
	assert.InDelta(t, expectedX, scoreX, 0.001)
	assert.InDelta(t, expectedY, scoreY, 0.001)
}

func TestRRF_KParameterEffect(t *testing.T) {
	tests := []struct {
		name     string
		k        float64
		rank1    int
		rank10   int
		expected float64
	}{
		{"k=10 (aggressive)", 10.0, 1, 10, 20.0 / 11.0},
		{"k=60 (standard)", 60.0, 1, 10, 70.0 / 61.0},
		{"k=100 (conservative)", 100.0, 1, 10, 110.0 / 101.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := RRFConfig{
				K:             tt.k,
				DefaultRank:   1000,
				SignalWeights: map[string]float64{"signal": 1.0},
			}

			score1 := RRFScore(map[string]int{"signal": tt.rank1}, config)
			score10 := RRFScore(map[string]int{"signal": tt.rank10}, config)

			ratio := score1 / score10
			assert.InDelta(t, tt.expected, ratio, 0.01,
				"K=%v should give ratio %.2f for rank 1 vs 10", tt.k, tt.expected)
		})
	}
}

func TestRRFScore_InvalidRanks(t *testing.T) {
	config := RRFConfig{
		K:           60.0,
		DefaultRank: 1000,
		SignalWeights: map[string]float64{
			"signal": 1.0,
		},
	}

	scoreZero := RRFScore(map[string]int{"signal": 0}, config)
	scoreDefault := RRFScore(map[string]int{}, config)

	assert.InDelta(t, scoreDefault, scoreZero, 0.0001,
		"Zero rank should use default rank")

	expected := 1.0 / (60.0 + 1000.0)
	assert.InDelta(t, expected, scoreZero, 0.0001)
}

func TestRRFScore_Deterministic(t *testing.T) {
	config := DefaultRRFConfig()
	ranks := map[string]int{
		"searxng_position": 3,
		"text_relevance":   7,
	}

	score1 := RRFScore(ranks, config)
	score2 := RRFScore(ranks, config)
	score3 := RRFScore(ranks, config)

	assert.Equal(t, score1, score2)
	assert.Equal(t, score2, score3)
}

func TestRRFScore_Precision(t *testing.T) {
	config := RRFConfig{
		K:           60.0,
		DefaultRank: 1000,
		SignalWeights: map[string]float64{
			"signal": 1.0,
		},
	}

	score := RRFScore(map[string]int{"signal": 1000000}, config)

	assert.False(t, math.IsNaN(score))
	assert.False(t, math.IsInf(score, 0))
	assert.Greater(t, score, 0.0)
}

func TestRankBySignal_UninformativeSignalDoesNotEchoInputOrder(t *testing.T) {
	results := []ScoredResult{{URL: "first"}, {URL: "second"}, {URL: "third"}}
	ranks := RankBySignal(results, func(ScoredResult) float64 { return 1 })
	for _, r := range results {
		if ranks[r.URL] != 1 {
			t.Fatalf("%s ranked %d; a signal equal for every result must rank them all 1", r.URL, ranks[r.URL])
		}
	}
}
