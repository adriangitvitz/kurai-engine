package ranker

type CollapseConfig struct {
	Enabled             bool    `yaml:"enabled"`
	SimilarityThreshold float64 `yaml:"similarity_threshold"`
}

func DefaultCollapseConfig() CollapseConfig {
	return CollapseConfig{Enabled: true, SimilarityThreshold: 0.6}
}

func CollapseByTitleSimilarity(results []ScoredResult, cfg CollapseConfig) []ScoredResult {
	if !cfg.Enabled || len(results) < 2 {
		return results
	}
	threshold := cfg.SimilarityThreshold
	if threshold <= 0 {
		threshold = 0.6
	}

	nc := NewNewsClusterer(ClusterConfig{SimilarityThreshold: threshold})
	tokens := make([]map[string]bool, len(results))
	for i, r := range results {
		tokens[i] = nc.extractTokens(r.Title)
	}

	type cluster struct {
		members []int
		repr    map[string]bool
	}
	clusters := make([]*cluster, 0, len(results))
	assigned := make([]int, len(results))
	for i := range assigned {
		assigned[i] = -1
	}

	for i := range results {
		matched := -1
		for ci, c := range clusters {
			if nc.jaccardSimilarity(tokens[i], c.repr) >= threshold {
				matched = ci
				break
			}
		}
		if matched >= 0 {
			clusters[matched].members = append(clusters[matched].members, i)
			for tok := range tokens[i] {
				clusters[matched].repr[tok] = true
			}
			assigned[i] = matched
		} else {
			repr := make(map[string]bool, len(tokens[i]))
			for tok := range tokens[i] {
				repr[tok] = true
			}
			clusters = append(clusters, &cluster{members: []int{i}, repr: repr})
			assigned[i] = len(clusters) - 1
		}
	}

	keep := make([]bool, len(results))
	for _, c := range clusters {
		bestIdx := c.members[0]
		bestScore := results[bestIdx].FinalScore
		for _, m := range c.members[1:] {
			if results[m].FinalScore > bestScore {
				bestIdx, bestScore = m, results[m].FinalScore
			}
		}
		keep[bestIdx] = true
	}

	out := make([]ScoredResult, 0, len(results))
	for i, r := range results {
		if keep[i] {
			out = append(out, r)
		}
	}
	return out
}
