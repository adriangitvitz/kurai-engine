package ranker

import (
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

type DiversityConfig struct {
	Enabled          bool     `yaml:"enabled"`
	MaxPerDomain     int      `yaml:"max_per_domain"`
	PenaltyPerExtra  float64  `yaml:"penalty_per_extra"`
	MinScore         float64  `yaml:"min_score"`
	ExcludedDomains  []string `yaml:"excluded_domains"`
	EnableForNews    bool     `yaml:"enable_for_news"`
	NewsMaxPerSource int      `yaml:"news_max_per_source"`
}

func DefaultDiversityConfig() DiversityConfig {
	return DiversityConfig{
		Enabled:          true,
		MaxPerDomain:     3,
		PenaltyPerExtra:  0.15,
		MinScore:         0.0,
		ExcludedDomains:  []string{"github.com", "stackoverflow.com"},
		EnableForNews:    true,
		NewsMaxPerSource: 2,
	}
}

type DiversityEnforcer struct {
	config          DiversityConfig
	excludedDomains map[string]bool
}

func NewDiversityEnforcer(config DiversityConfig) *DiversityEnforcer {
	excluded := make(map[string]bool)
	for _, domain := range config.ExcludedDomains {
		excluded[domain] = true
	}

	return &DiversityEnforcer{
		config:          config,
		excludedDomains: excluded,
	}
}

func (de *DiversityEnforcer) EnforceDiversity(results []ScoredResult) []ScoredResult {
	if !de.config.Enabled || len(results) == 0 {
		return results
	}

	domainCounts := make(map[string]int)
	diversified := make([]ScoredResult, 0, len(results))

	for _, result := range results {
		domain := urlutil.ExtractDomain(result.URL)

		if de.excludedDomains[domain] {
			diversified = append(diversified, result)
			continue
		}

		count := domainCounts[domain]

		if count < de.config.MaxPerDomain {
			diversified = append(diversified, result)
			domainCounts[domain] = count + 1
		} else {
			penalty := de.calculatePenalty(count, de.config.MaxPerDomain)
			penalized := result
			penalized.FinalScore = result.FinalScore * (1.0 - penalty)
			penalized.AdjustedScore = result.AdjustedScore * (1.0 - penalty)

			if penalized.FinalScore >= de.config.MinScore {
				diversified = append(diversified, penalized)
				domainCounts[domain] = count + 1
			}
		}
	}

	return diversified
}

func (de *DiversityEnforcer) EnforceDiversityWithReorder(results []ScoredResult) []ScoredResult {
	if !de.config.Enabled || len(results) == 0 {
		return results
	}

	domainGroups := make(map[string][]ScoredResult)
	var domainOrder []string

	for _, result := range results {
		domain := urlutil.ExtractDomain(result.URL)
		if _, seen := domainGroups[domain]; !seen {
			domainOrder = append(domainOrder, domain)
		}
		domainGroups[domain] = append(domainGroups[domain], result)
	}

	diversified := make([]ScoredResult, 0, len(results))
	domainIndices := make(map[string]int)

	for len(diversified) < len(results) {
		addedThisRound := false

		for _, domain := range domainOrder {
			idx := domainIndices[domain]
			group := domainGroups[domain]

			if idx < len(group) {
				if de.excludedDomains[domain] || idx < de.config.MaxPerDomain {
					diversified = append(diversified, group[idx])
					domainIndices[domain] = idx + 1
					addedThisRound = true
				} else {
					penalty := de.calculatePenalty(idx, de.config.MaxPerDomain)
					penalized := group[idx]
					penalized.FinalScore = group[idx].FinalScore * (1.0 - penalty)
					penalized.AdjustedScore = group[idx].AdjustedScore * (1.0 - penalty)

					if penalized.FinalScore >= de.config.MinScore {
						diversified = append(diversified, penalized)
						domainIndices[domain] = idx + 1
						addedThisRound = true
					}
				}
			}
		}

		if !addedThisRound {
			break
		}
	}

	return diversified
}

func (de *DiversityEnforcer) calculatePenalty(count, maxAllowed int) float64 {
	if count < maxAllowed {
		return 0
	}

	extraCount := count - maxAllowed + 1
	penalty := float64(extraCount) * de.config.PenaltyPerExtra

	if penalty > 0.9 {
		penalty = 0.9
	}

	return penalty
}

func (de *DiversityEnforcer) GetDomainStats(results []ScoredResult) DomainStats {
	stats := DomainStats{
		DomainCounts: make(map[string]int),
		TotalResults: len(results),
	}

	for _, result := range results {
		domain := urlutil.ExtractDomain(result.URL)
		stats.DomainCounts[domain]++
	}

	stats.UniqueDomains = len(stats.DomainCounts)

	maxCount := 0
	for domain, count := range stats.DomainCounts {
		if count > maxCount {
			maxCount = count
			stats.TopDomain = domain
			stats.TopDomainCount = count
		}
	}

	if stats.TotalResults > 0 {
		stats.DiversityScore = float64(stats.UniqueDomains) / float64(stats.TotalResults)
	}

	return stats
}

type DomainStats struct {
	TotalResults   int            `json:"total_results"`
	UniqueDomains  int            `json:"unique_domains"`
	DomainCounts   map[string]int `json:"domain_counts"`
	TopDomain      string         `json:"top_domain"`
	TopDomainCount int            `json:"top_domain_count"`
	DiversityScore float64        `json:"diversity_score"`
}

func (de *DiversityEnforcer) FilterExcessive(results []ScoredResult, hardLimit int) []ScoredResult {
	if hardLimit <= 0 {
		hardLimit = de.config.MaxPerDomain * 2
	}

	domainCounts := make(map[string]int)
	filtered := make([]ScoredResult, 0, len(results))

	for _, result := range results {
		domain := urlutil.ExtractDomain(result.URL)

		if de.excludedDomains[domain] {
			filtered = append(filtered, result)
			continue
		}

		if domainCounts[domain] < hardLimit {
			filtered = append(filtered, result)
			domainCounts[domain]++
		}
	}

	return filtered
}

func (de *DiversityEnforcer) EnforceNewsDiversity(results []ScoredResult) []ScoredResult {
	if !de.config.EnableForNews || len(results) == 0 {
		return results
	}

	maxPerSource := de.config.NewsMaxPerSource
	if maxPerSource <= 0 {
		maxPerSource = 2
	}

	sourceCounts := make(map[string]int)
	diversified := make([]ScoredResult, 0, len(results))

	for _, result := range results {
		domain := urlutil.ExtractDomain(result.URL)

		if sourceCounts[domain] < maxPerSource {
			diversified = append(diversified, result)
			sourceCounts[domain]++
		}
	}

	return diversified
}

func (de *DiversityEnforcer) SetConfig(config DiversityConfig) {
	de.config = config
	de.excludedDomains = make(map[string]bool)
	for _, domain := range config.ExcludedDomains {
		de.excludedDomains[domain] = true
	}
}

func (de *DiversityEnforcer) AddExcludedDomain(domain string) {
	de.excludedDomains[domain] = true
	de.config.ExcludedDomains = append(de.config.ExcludedDomains, domain)
}

func (de *DiversityEnforcer) IsExcluded(domain string) bool {
	return de.excludedDomains[domain]
}

func (de *DiversityEnforcer) ShouldPenalize(domain string, currentCount int) bool {
	if de.excludedDomains[domain] {
		return false
	}
	return currentCount >= de.config.MaxPerDomain
}

func (de *DiversityEnforcer) GetPenalty(domain string, domainPosition int) float64 {
	if de.excludedDomains[domain] {
		return 0
	}
	return de.calculatePenalty(domainPosition, de.config.MaxPerDomain)
}
