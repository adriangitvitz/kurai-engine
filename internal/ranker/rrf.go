package ranker

import (
	"sort"
)

type RRFConfig struct {
	Enabled       bool               `yaml:"enabled"`
	K             float64            `yaml:"k"`
	SignalWeights map[string]float64 `yaml:"signal_weights"`
	DefaultRank   int                `yaml:"default_rank"`
}

func DefaultRRFConfig() RRFConfig {
	return RRFConfig{
		Enabled:     true,
		K:           60.0,
		DefaultRank: 1000,
		SignalWeights: map[string]float64{
			"searxng_position": 1.0,
			"text_relevance":   1.8,
			"domain_alignment": 0.8,
			"tranco_authority": 0.5,
			"engine_agreement": 0.6,
			"freshness":        0.4,
		},
	}
}

func RRFScore(ranks map[string]int, config RRFConfig) float64 {
	score := 0.0
	defaultRank := config.DefaultRank
	if defaultRank == 0 {
		defaultRank = 1000
	}

	for _, signal := range sortedSignals(config.SignalWeights) {
		weight := config.SignalWeights[signal]
		rank, exists := ranks[signal]
		if !exists || rank <= 0 {
			rank = defaultRank
		}

		score += weight * (1.0 / (config.K + float64(rank)))
	}

	return score
}

func RRFContributions(ranks map[string]int, config RRFConfig) map[string]float64 {
	contributions := make(map[string]float64)
	defaultRank := config.DefaultRank
	if defaultRank == 0 {
		defaultRank = 1000
	}

	for _, signal := range sortedSignals(config.SignalWeights) {
		weight := config.SignalWeights[signal]
		rank, exists := ranks[signal]
		if !exists || rank <= 0 {
			rank = defaultRank
		}
		contributions[signal] = weight * (1.0 / (config.K + float64(rank)))
	}

	return contributions
}

func sortedSignals(weights map[string]float64) []string {
	signals := make([]string, 0, len(weights))
	for signal := range weights {
		signals = append(signals, signal)
	}
	sort.Strings(signals)
	return signals
}

func RankBySignal(results []ScoredResult, scoreFunc func(ScoredResult) float64) map[string]int {
	if len(results) == 0 {
		return make(map[string]int)
	}

	type scoredPair struct {
		url   string
		score float64
	}
	pairs := make([]scoredPair, len(results))
	for i, r := range results {
		pairs[i] = scoredPair{url: r.URL, score: scoreFunc(r)}
	}

	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].score > pairs[j].score
	})

	ranks := make(map[string]int, len(pairs))
	currentRank := 1
	for i, p := range pairs {
		if i > 0 && pairs[i].score < pairs[i-1].score {
			currentRank = i + 1
		}
		ranks[p.url] = currentRank
	}

	return ranks
}

func CombineRanks(signalRanks map[string]map[string]int) map[string]map[string]int {

	combined := make(map[string]map[string]int)

	for signal, urlRanks := range signalRanks {
		for url, rank := range urlRanks {
			if combined[url] == nil {
				combined[url] = make(map[string]int)
			}
			combined[url][signal] = rank
		}
	}

	return combined
}

func NormalizeRRFScores(scores map[string]float64) map[string]float64 {
	if len(scores) == 0 {
		return scores
	}

	minScore, maxScore := 0.0, 0.0
	first := true
	for _, s := range scores {
		if first {
			minScore, maxScore = s, s
			first = false
		} else {
			if s < minScore {
				minScore = s
			}
			if s > maxScore {
				maxScore = s
			}
		}
	}

	rangeScore := maxScore - minScore
	if rangeScore == 0 {
		normalized := make(map[string]float64, len(scores))
		for url := range scores {
			normalized[url] = 1.0
		}
		return normalized
	}

	normalized := make(map[string]float64, len(scores))
	for url, score := range scores {
		normalized[url] = (score - minScore) / rangeScore
	}

	return normalized
}
