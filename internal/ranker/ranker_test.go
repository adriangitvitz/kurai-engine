package ranker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/searxng"
)

type MockBlocklistManager struct {
	blocked map[string]bool
	ranks   map[string]int
}

func NewMockBlocklistManager() *MockBlocklistManager {
	return &MockBlocklistManager{
		blocked: make(map[string]bool),
		ranks:   make(map[string]int),
	}
}

func (m *MockBlocklistManager) IsBlocked(domain string) bool {
	return m.blocked[domain]
}

func (m *MockBlocklistManager) GetTrancoRank(domain string) int {
	return m.ranks[domain]
}

func (m *MockBlocklistManager) SetBlocked(domain string) {
	m.blocked[domain] = true
}

func (m *MockBlocklistManager) SetTrancoRank(domain string, rank int) {
	m.ranks[domain] = rank
}

func newScoringRanker(cfg *config.RankingConfig, bl BlocklistChecker) *Ranker {
	r := New(cfg, bl)
	r.SetRelevanceFloor(RelevanceFloorConfig{})
	return r
}

func defaultConfig() *config.RankingConfig {
	return &config.RankingConfig{
		Weights: config.WeightConfig{
			TitleMatch:      20,
			URLMatch:        10,
			EngineAgreement: 5,
			TrancoTop10k:    15,
			TrancoTop100k:   10,
			TrancoTop1M:     5,
			Freshness:       5,
		},
		Penalties: config.PenaltyConfig{
			Blocklist:       -50,
			ContentFarm:     -30,
			ExcessiveParams: -10,
			ParamThreshold:  5,
		},
	}
}

func TestRanker_Rank_BasicSort(t *testing.T) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	ranker := newScoringRanker(cfg, bl)

	results := []searxng.Result{
		{
			URL:     "https://lowscore.com",
			Title:   "Low Score Result",
			Content: "Some content",
			Engines: []string{"google"},
			Score:   50,
		},
		{
			URL:     "https://highscore.com",
			Title:   "High Score Result",
			Content: "Some content",
			Engines: []string{"google"},
			Score:   100,
		},
	}

	scored := ranker.Rank(context.Background(), "test", results, false)

	require.Len(t, scored, 2)
	assert.Equal(t, "https://highscore.com", scored[0].URL)
	assert.Equal(t, "https://lowscore.com", scored[1].URL)
}

func TestRanker_Rank_TitleMatchBoost(t *testing.T) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	ranker := newScoringRanker(cfg, bl)

	results := []searxng.Result{
		{
			URL:     "https://nomatch.com",
			Title:   "Unrelated Title",
			Content: "Some content",
			Engines: []string{"google"},
			Score:   100,
		},
		{
			URL:     "https://match.com",
			Title:   "Rust Programming Guide",
			Content: "Some content",
			Engines: []string{"google"},
			Score:   90,
		},
	}

	scored := ranker.Rank(context.Background(), "rust programming", results, false)

	require.Len(t, scored, 2)
	assert.Equal(t, "https://match.com", scored[0].URL)
	assert.Equal(t, float64(90+20), scored[0].AdjustedScore)
}

func TestRanker_Rank_EngineAgreementBoost(t *testing.T) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	ranker := newScoringRanker(cfg, bl)

	results := []searxng.Result{
		{
			URL:     "https://single.com",
			Title:   "Single Engine",
			Engines: []string{"google"},
			Score:   100,
		},
		{
			URL:     "https://multiple.com",
			Title:   "Multiple Engines",
			Engines: []string{"google", "duckduckgo", "bing"},
			Score:   92,
		},
	}

	scored := ranker.Rank(context.Background(), "test", results, false)

	require.Len(t, scored, 2)
	assert.Equal(t, "https://multiple.com", scored[0].URL)
	assert.Equal(t, float64(92+10), scored[0].AdjustedScore)
}

func TestRanker_Rank_TrancoBoost(t *testing.T) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	bl.SetTrancoRank("reputable.com", 5000)
	ranker := newScoringRanker(cfg, bl)

	results := []searxng.Result{
		{
			URL:     "https://unknown.com",
			Title:   "Unknown Site",
			Engines: []string{"google"},
			Score:   100,
		},
		{
			URL:     "https://reputable.com",
			Title:   "Reputable Site",
			Engines: []string{"google"},
			Score:   90,
		},
	}

	scored := ranker.Rank(context.Background(), "test", results, false)

	require.Len(t, scored, 2)
	assert.Equal(t, "https://reputable.com", scored[0].URL)
	assert.Equal(t, float64(90+15), scored[0].AdjustedScore)
}

func TestRanker_Rank_BlocklistPenalty(t *testing.T) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	bl.SetBlocked("spam.com")
	ranker := newScoringRanker(cfg, bl)

	results := []searxng.Result{
		{
			URL:     "https://spam.com",
			Title:   "Spam Site",
			Engines: []string{"google"},
			Score:   100,
		},
		{
			URL:     "https://good.com",
			Title:   "Good Site",
			Engines: []string{"google"},
			Score:   60,
		},
	}

	scored := ranker.Rank(context.Background(), "test", results, false)

	require.Len(t, scored, 2)
	assert.Equal(t, "https://good.com", scored[0].URL)
	assert.Equal(t, "https://spam.com", scored[1].URL)
	assert.Equal(t, float64(100-50), scored[1].AdjustedScore)
}

func TestRanker_Rank_FilterHeavilyPenalized(t *testing.T) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	bl.SetBlocked("verybad.com")
	ranker := newScoringRanker(cfg, bl)

	results := []searxng.Result{
		{
			URL:     "https://verybad.com",
			Title:   "Very Bad Site",
			Engines: []string{"google"},
			Score:   30,
		},
		{
			URL:     "https://good.com",
			Title:   "Good Site",
			Engines: []string{"google"},
			Score:   50,
		},
	}

	scored := ranker.Rank(context.Background(), "test", results, false)

	require.Len(t, scored, 1)
	assert.Equal(t, "https://good.com", scored[0].URL)
}

func TestRanker_Rank_ContentFarmPenalty(t *testing.T) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	ranker := newScoringRanker(cfg, bl)

	results := []searxng.Result{
		{
			URL:     "https://about.com/article",
			Title:   "Content Farm Article",
			Engines: []string{"google"},
			Score:   100,
		},
		{
			URL:     "https://github.com/repo",
			Title:   "GitHub Repo",
			Engines: []string{"google"},
			Score:   80,
		},
	}

	scored := ranker.Rank(context.Background(), "test", results, false)

	require.Len(t, scored, 2)
	assert.Equal(t, "https://github.com/repo", scored[0].URL)
	assert.Equal(t, "https://about.com/article", scored[1].URL)
	assert.Equal(t, float64(100-30), scored[1].AdjustedScore)
}

func TestRanker_Rank_ScoreBreakdown(t *testing.T) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	bl.SetTrancoRank("example.com", 5000)
	ranker := newScoringRanker(cfg, bl)

	results := []searxng.Result{
		{
			URL:     "https://example.com/rust-guide",
			Title:   "Rust Programming Guide",
			Content: "Learn Rust programming",
			Engines: []string{"google", "duckduckgo"},
			Score:   100,
		},
	}

	scored := ranker.Rank(context.Background(), "rust", results, true)

	require.Len(t, scored, 1)
	require.NotNil(t, scored[0].ScoreBreakdown)

	breakdown := scored[0].ScoreBreakdown
	assert.Equal(t, 20, breakdown.TitleMatchBonus)
	assert.Equal(t, 10, breakdown.URLMatchBonus)
	assert.Equal(t, 15, breakdown.TrancoBonus)
	assert.Equal(t, 5, breakdown.EngineAgreementBonus)
	assert.Equal(t, 0, breakdown.BlocklistPenalty)
	assert.Equal(t, 0, breakdown.ContentFarmPenalty)
}

func TestRanker_Rank_Badges(t *testing.T) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	bl.SetBlocked("ads.example.com")
	bl.SetTrancoRank("reputable.com", 50000)
	ranker := newScoringRanker(cfg, bl)

	results := []searxng.Result{
		{
			URL:     "https://ads.example.com",
			Title:   "Ad Site",
			Engines: []string{"google"},
			Score:   100,
		},
		{
			URL:     "https://reputable.com",
			Title:   "Reputable Site",
			Engines: []string{"google"},
			Score:   80,
		},
		{
			URL:     "https://about.com",
			Title:   "Content Farm",
			Engines: []string{"google"},
			Score:   90,
		},
	}

	scored := ranker.Rank(context.Background(), "test", results, false)

	for _, r := range scored {
		switch r.URL {
		case "https://ads.example.com":
			assert.True(t, r.Badges.HasAds)
			assert.True(t, r.Badges.KnownSpam)
		case "https://reputable.com":
			assert.True(t, r.Badges.Reputable)
			assert.False(t, r.Badges.HasAds)
		case "https://about.com":
			assert.True(t, r.Badges.ContentFarm)
		}
	}
}

func TestRanker_Rank_EmptyResults(t *testing.T) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	ranker := newScoringRanker(cfg, bl)

	results := []searxng.Result{}

	scored := ranker.Rank(context.Background(), "test", results, false)

	assert.Empty(t, scored)
}

func TestRanker_Rank_EmptyQuery(t *testing.T) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	ranker := newScoringRanker(cfg, bl)

	results := []searxng.Result{
		{
			URL:     "https://example.com",
			Title:   "Example",
			Engines: []string{"google"},
			Score:   100,
		},
	}

	scored := ranker.Rank(context.Background(), "", results, false)

	require.Len(t, scored, 1)
}

func BenchmarkRanker_Rank100Results(b *testing.B) {
	cfg := defaultConfig()
	bl := NewMockBlocklistManager()
	for i := 0; i < 1000; i++ {
		bl.SetTrancoRank("domain"+string(rune(i))+".com", i)
	}
	ranker := newScoringRanker(cfg, bl)

	results := make([]searxng.Result, 100)
	for i := 0; i < 100; i++ {
		results[i] = searxng.Result{
			URL:     "https://domain" + string(rune(i)) + ".com/page",
			Title:   "Result " + string(rune(i)),
			Content: "Content for result",
			Engines: []string{"google", "duckduckgo"},
			Score:   float64(100 - i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ranker.Rank(context.Background(), "test query", results, false)
	}
}
