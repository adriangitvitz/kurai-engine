package ranker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/searxng"
)

type goldenQuery struct {
	ID                string         `yaml:"id"`
	Query             string         `yaml:"query"`
	Intent            string         `yaml:"intent"`
	JudgmentsByDomain map[string]int `yaml:"judgments_by_domain"`
}

type goldenSuite struct {
	Queries []goldenQuery `yaml:"queries"`
}

func loadGoldenSuite(t *testing.T) *goldenSuite {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "golden_queries.yaml"))
	require.NoError(t, err)
	var s goldenSuite
	require.NoError(t, yaml.Unmarshal(data, &s))
	require.NotEmpty(t, s.Queries)
	return &s
}

func loadFixture(t *testing.T, q goldenQuery) []searxng.Result {
	t.Helper()
	path := filepath.Join("testdata", "fixtures", q.ID+".json")
	if data, err := os.ReadFile(path); err == nil {
		var f struct {
			Results []searxng.Result `json:"results"`
		}
		require.NoError(t, json.Unmarshal(data, &f), "fixture %s", path)
		return f.Results
	}
	return synthesizeFixture(q)
}

func synthesizeFixture(q goldenQuery) []searxng.Result {
	rng := rand.New(rand.NewPCG(uint64(hashID(q.ID)), 0xfeedbeef))

	noiseDomains := []string{
		"medium.com", "reddit.com", "quora.com", "stackoverflow.com",
		"github.com", "dev.to", "youtube.com", "pinterest.com",
		"slideshare.net", "scribd.com",
	}

	results := make([]searxng.Result, 0, len(q.JudgmentsByDomain)+len(noiseDomains))

	for i, d := range noiseDomains {
		if _, judged := q.JudgmentsByDomain[d]; judged {
			continue
		}
		results = append(results, searxng.Result{
			URL:     fmt.Sprintf("https://%s/post/%d", d, i),
			Title:   fmt.Sprintf("%s on %s - short reaction", titleCase(q.Query), d),
			Content: fmt.Sprintf("Quick take on %s. Read more at %s.", q.Query, d),
			Engines: pickEngines(rng, 1),
			Score:   1.0 / float64(i+1),
		})
	}

	for domain, grade := range orderedJudgments(q.JudgmentsByDomain) {
		title, snippet := factSnippet(q.Query, domain, grade)
		engines := 1
		if grade >= 2 {
			engines = 2
		}
		results = append(results, searxng.Result{
			URL:     fmt.Sprintf("https://%s/article-%d", domain, grade),
			Title:   title,
			Content: snippet,
			Engines: pickEngines(rng, engines),
			Score:   0.5 / float64(len(results)+1),
		})
	}

	return results
}

func orderedJudgments(j map[string]int) func(yield func(string, int) bool) {
	keys := make([]string, 0, len(j))
	for k := range j {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return func(yield func(string, int) bool) {
		for _, k := range keys {
			if !yield(k, j[k]) {
				return
			}
		}
	}
}

func factSnippet(query, domain string, grade int) (title, snippet string) {
	switch grade {
	case 2:
		title = fmt.Sprintf("%s - official guide on %s", titleCase(query), domain)
		snippet = fmt.Sprintf("%s. Comprehensive coverage with examples, semantics, and edge cases. Maintained by %s.",
			titleCase(query), domain)
	case 1:
		title = fmt.Sprintf("Notes on %s", titleCase(query))
		snippet = fmt.Sprintf("Working notes about %s. Useful background but not authoritative.", query)
	default:
		title = fmt.Sprintf("%s - opinion piece", titleCase(query))
		snippet = fmt.Sprintf("Personal blog about %s with anecdotes and ads.", query)
	}
	return
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Fields(s)
	for i, p := range parts {
		if len(p) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func pickEngines(rng *rand.Rand, n int) []string {
	pool := []string{"google", "bing", "brave", "duckduckgo", "qwant"}
	if n > len(pool) {
		n = len(pool)
	}
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	return pool[:n]
}

func hashID(id string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(id); i++ {
		h ^= uint32(id[i])
		h *= 16777619
	}
	return h
}

func goldenRanker(t *testing.T) *Ranker {
	t.Helper()
	cfg := &config.RankingConfig{
		Weights:   config.WeightConfig{TitleMatch: 20, URLMatch: 10, EngineAgreement: 5, TrancoTop10k: 15, TrancoTop100k: 10, TrancoTop1M: 5, Freshness: 5},
		Penalties: config.PenaltyConfig{Blocklist: -50, ContentFarm: -30, ExcessiveParams: -10, ParamThreshold: 5},
	}
	bl := blocklist.NewManager(&config.BlocklistConfig{}, nil)
	return New(cfg, bl)
}

func TestGolden_NDCGFloor(t *testing.T) {
	suite := loadGoldenSuite(t)
	r := goldenRanker(t)

	type result struct {
		ID   string
		NDCG float64
		MRR  float64
		R10  float64
		Top3 []string
	}
	var rows []result
	var sumNDCG, sumMRR, sumR10 float64

	for _, q := range suite.Queries {
		fixture := loadFixture(t, q)
		ranked := r.RankWithOptions(context.Background(), q.Query, fixture, RankOptions{
			Category: q.Intent,
		})
		j := JudgmentsForDomain(ranked, q.JudgmentsByDomain)
		nd := NDCGAtK(ranked, j, 10)
		mr := MRR(ranked, j)
		r10 := RecallAtK(ranked, j, 10)

		top3 := []string{}
		for i := 0; i < 3 && i < len(ranked); i++ {
			top3 = append(top3, domainOf(ranked[i].URL))
		}
		rows = append(rows, result{ID: q.ID, NDCG: nd, MRR: mr, R10: r10, Top3: top3})
		sumNDCG += nd
		sumMRR += mr
		sumR10 += r10
	}

	t.Logf("golden eval (n=%d):", len(suite.Queries))
	t.Logf("  %-32s %-7s %-7s %-7s  %s", "id", "nDCG@10", "MRR", "R@10", "top-3 domains")
	for _, row := range rows {
		t.Logf("  %-32s %.3f   %.3f   %.3f   %s", row.ID, row.NDCG, row.MRR, row.R10, strings.Join(row.Top3, ", "))
	}
	mean := sumNDCG / float64(len(suite.Queries))
	t.Logf("  mean nDCG@10 = %.3f, mean MRR = %.3f, mean R@10 = %.3f",
		mean, sumMRR/float64(len(suite.Queries)), sumR10/float64(len(suite.Queries)))

	const floorMeanNDCG = 0.55
	require.GreaterOrEqual(t, mean, floorMeanNDCG,
		"mean nDCG@10 (%.3f) below floor (%.3f) - ranking regression",
		mean, floorMeanNDCG)
}

func domainOf(u string) string {
	for _, prefix := range []string{"https://", "http://"} {
		u = strings.TrimPrefix(u, prefix)
	}
	if i := strings.Index(u, "/"); i >= 0 {
		u = u[:i]
	}
	return strings.TrimPrefix(u, "www.")
}
