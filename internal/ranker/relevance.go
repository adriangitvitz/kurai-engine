package ranker

import (
	"net/url"
	"strings"

	"github.com/adriangitvitz/kurai/internal/config"
)

type RelevanceFloorConfig struct {
	Enabled     bool    `yaml:"enabled"`
	MinCoverage float64 `yaml:"min_coverage"`
}

func DefaultRelevanceFloorConfig() RelevanceFloorConfig {
	return RelevanceFloorConfig{Enabled: true, MinCoverage: 0}
}

func CoreTermCoverage(queryTerms []string, title, snippet, rawURL string) float64 {
	core := filterStopWords(queryTerms)
	if len(core) == 0 {
		return 1
	}

	var b strings.Builder
	b.WriteString(strings.ToLower(title))
	b.WriteByte(' ')
	b.WriteString(strings.ToLower(snippet))
	b.WriteByte(' ')
	b.WriteString(urlHaystack(rawURL))
	haystack := b.String()

	found := 0
	for _, term := range core {
		if strings.Contains(haystack, strings.ToLower(term)) {
			found++
		}
	}
	return float64(found) / float64(len(core))
}

func urlHaystack(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	lower := strings.ToLower(rawURL)
	if u, err := url.Parse(lower); err == nil && u.Host != "" {
		lower = u.Host + " " + u.Path + " " + u.RawQuery
	} else {
		if i := strings.Index(lower, "://"); i >= 0 {
			lower = lower[i+3:]
		}
	}
	return strings.NewReplacer("/", " ", "-", " ", "_", " ", ".", " ", "?", " ", "=", " ", "&", " ").Replace(lower)
}

func (c RelevanceFloorConfig) meetsFloor(queryTerms []string, title, snippet, rawURL string) bool {
	if !c.Enabled {
		return true
	}
	coverage := CoreTermCoverage(queryTerms, title, snippet, rawURL)
	if c.MinCoverage <= 0 {
		return coverage > 0
	}
	return coverage >= c.MinCoverage
}

func relevanceFloorFromConfig(cfg *config.RankingConfig) RelevanceFloorConfig {
	if cfg == nil {
		return DefaultRelevanceFloorConfig()
	}
	out := DefaultRelevanceFloorConfig()
	if cfg.RelevanceFloor.Enabled != nil {
		out.Enabled = *cfg.RelevanceFloor.Enabled
	}
	if cfg.RelevanceFloor.MinCoverage > 0 {
		out.MinCoverage = cfg.RelevanceFloor.MinCoverage
	}
	return out
}
