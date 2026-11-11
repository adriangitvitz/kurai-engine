package ranker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/searxng"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

func TestCoverage_RealNoiseObservedLive(t *testing.T) {
	cases := []struct {
		name  string
		query string
		title string
		url   string
		keep  bool
	}{
		{"splash sports for kubernetes", "kubernetes operator pattern",
			"Splash Sports", "https://splashsports.com/", false},
		{"celebrity news for kubernetes", "kubernetes operator pattern",
			"Sydney Sweeney returns", "https://www.msn.com/en-us/entertainment/tv/sydney-sweeney-returns", false},
		{"MSFT quote for kubernetes", "kubernetes operator pattern",
			"MSFT Stock Quote", "https://finance.yahoo.com/quote/MSFT/", false},
		{"uber poll for postgres", "postgres index bloat",
			"Uber vs. Lyft poll for Phoenix area residents", "https://www.city-data.com/forum/phoenix-area/3426204-uber-vs-lyft-poll", false},
		{"cruise thread for postgres", "postgres index bloat",
			"Very early disembarkation, hours until hotel check-in", "https://www.city-data.com/forum/cruises/3030393-very-early", false},

		{"k8s docs", "kubernetes operator pattern",
			"Operator pattern", "https://kubernetes.io/docs/concepts/extend-kubernetes/operator/", true},
		{"cncf", "kubernetes operator pattern",
			"Cloud Native Live: enabling self-service with operators", "https://www.cncf.io/online-programs/cloud-native-live-enabling", true},

		{"rust the video game", "rust ownership borrow checker",
			"Rust (video game) - Wikipedia", "https://en.wikipedia.org/wiki/Rust_(video_game)", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cov := CoverageForTest(tc.query, tc.title, "", tc.url)
			got := cov > 0
			assert.Equal(t, tc.keep, got,
				"coverage=%.2f for %q vs %q", cov, tc.query, tc.title)
		})
	}
}

func CoverageForTest(query, title, snippet, rawURL string) float64 {
	return CoreTermCoverage(urlutil.Tokenize(query), title, snippet, rawURL)
}

func TestCoverage_MatchSources(t *testing.T) {
	t.Run("term found only in the URL path still counts", func(t *testing.T) {
		cov := CoverageForTest("pod terminating", "Troubleshooting guide", "",
			"https://example.com/kubernetes/pod-terminating-finalizer")
		assert.Equal(t, 1.0, cov)
	})
	t.Run("term found only in the host still counts", func(t *testing.T) {
		cov := CoverageForTest("postgres tuning", "Index maintenance", "",
			"https://postgres.example.com/tuning")
		assert.Equal(t, 1.0, cov)
	})
	t.Run("substring match covers inflections", func(t *testing.T) {
		assert.Positive(t, CoverageForTest("operator", "Writing Operators", "", "https://x.com/"))
		assert.Positive(t, CoverageForTest("index", "Indexing strategies", "", "https://x.com/"))
	})
	t.Run("scheme never matches", func(t *testing.T) {
		assert.Zero(t, CoverageForTest("https", "Unrelated title", "", "https://example.com/page"))
	})
}

func TestCoverage_DegenerateQueries(t *testing.T) {
	t.Run("all-stopword query never filters", func(t *testing.T) {
		assert.Positive(t, CoverageForTest("how to", "How to do anything", "", "https://x.com/"))
	})
	t.Run("empty query never filters", func(t *testing.T) {
		assert.Equal(t, 1.0, CoverageForTest("", "Anything", "", "https://x.com/"))
	})
}

func TestRelevanceFloor_Config(t *testing.T) {
	q := urlutil.Tokenize("kubernetes operator pattern")
	off := RelevanceFloorConfig{}
	on := DefaultRelevanceFloorConfig()

	assert.True(t, off.meetsFloor(q, "Splash Sports", "", "https://splashsports.com/"),
		"a disabled floor admits everything")
	assert.False(t, on.meetsFloor(q, "Splash Sports", "", "https://splashsports.com/"))

	t.Run("MinCoverage demands a fraction of the terms", func(t *testing.T) {
		strict := RelevanceFloorConfig{Enabled: true, MinCoverage: 0.9}
		assert.True(t, on.meetsFloor(q, "Kubernetes basics", "", "https://x.com/"))
		assert.False(t, strict.meetsFloor(q, "Kubernetes basics", "", "https://x.com/"))
		assert.True(t, strict.meetsFloor(q, "Kubernetes operator pattern guide", "", "https://x.com/"))
	})
}

func TestRelevanceFloor_DropsAndCounts(t *testing.T) {
	r := New(defaultConfig(), NewMockBlocklistManager())

	results := []searxng.Result{
		{URL: "https://kubernetes.io/docs/concepts/extend-kubernetes/operator/",
			Title: "Operator pattern", Content: "The operator pattern...", Engines: []string{"mwmbl"}},
		{URL: "https://splashsports.com/", Title: "Splash Sports",
			Content: "Play pools", Engines: []string{"bing"}},
		{URL: "https://finance.yahoo.com/quote/MSFT/", Title: "MSFT Stock Quote",
			Content: "Microsoft Corporation", Engines: []string{"bing"}},
	}

	var stats RankStats
	scored := r.RankWithOptions(context.Background(), "kubernetes operator pattern",
		results, RankOptions{Stats: &stats})

	require.Len(t, scored, 1, "only the on-topic result should survive")
	assert.Contains(t, scored[0].URL, "kubernetes.io")
	assert.Equal(t, 2, stats.IrrelevantFiltered)
	assert.Equal(t, 0, stats.BlockedFiltered,
		"floor drops must not be attributed to the blocklist")
}

func TestRelevanceFloor_DisabledKeepsEverything(t *testing.T) {
	r := New(defaultConfig(), NewMockBlocklistManager())
	r.SetRelevanceFloor(RelevanceFloorConfig{})

	results := []searxng.Result{
		{URL: "https://splashsports.com/", Title: "Splash Sports", Engines: []string{"bing"}},
		{URL: "https://finance.yahoo.com/quote/MSFT/", Title: "MSFT", Engines: []string{"bing"}},
	}
	var stats RankStats
	scored := r.RankWithOptions(context.Background(), "kubernetes operator pattern",
		results, RankOptions{Stats: &stats})
	assert.Len(t, scored, 2)
	assert.Zero(t, stats.IrrelevantFiltered)
}

func rrfConfig() *config.RankingConfig {
	cfg := defaultConfig()
	cfg.RRF = config.RRFConfig{
		Enabled: true, K: 60, DefaultRank: 1000,
		SignalWeights: map[string]float64{"searxng_position": 1.0, "text_relevance": 1.0},
	}
	return cfg
}

func TestPerEngine_BlockedIsNotIrrelevant(t *testing.T) {
	bl := NewMockBlocklistManager()
	bl.SetBlocked("adfarm.example")
	r := New(rrfConfig(), bl)

	results := []searxng.Result{
		{URL: "https://adfarm.example/kubernetes-operator-pattern",
			Title: "Kubernetes operator pattern explained", Engines: []string{"adengine"}},
		{URL: "https://finance.example/quote/MSFT",
			Title: "MSFT Stock Quote", Engines: []string{"noisyengine"}},
		{URL: "https://kubernetes.io/docs/concepts/extend-kubernetes/operator/",
			Title: "Operator pattern", Engines: []string{"goodengine"}},
	}

	var stats RankStats
	scored := r.RankWithOptions(context.Background(), "kubernetes operator pattern",
		results, RankOptions{Stats: &stats})
	require.Len(t, scored, 1)

	ad := stats.PerEngine["adengine"]
	assert.Equal(t, 1, ad.Blocked, "ad domain must count as policy, not noise")
	assert.Zero(t, ad.Irrelevant, "an on-topic result is never irrelevant, blocked or not")

	noisy := stats.PerEngine["noisyengine"]
	assert.Equal(t, 1, noisy.Irrelevant)
	assert.Zero(t, noisy.Blocked)

	good := stats.PerEngine["goodengine"]
	assert.Equal(t, 1, good.Kept)
	assert.Zero(t, good.Blocked)
	assert.Zero(t, good.Irrelevant)
}

func TestPerEngine_MultiEngineResultsCreditEveryEngine(t *testing.T) {
	r := New(rrfConfig(), NewMockBlocklistManager())
	results := []searxng.Result{
		{URL: "https://finance.example/msft", Title: "MSFT Stock Quote",
			Engines: []string{"bing", "yahoo"}},
	}
	var stats RankStats
	r.RankWithOptions(context.Background(), "kubernetes operator pattern",
		results, RankOptions{Stats: &stats})

	for _, e := range []string{"bing", "yahoo"} {
		assert.Equal(t, 1, stats.PerEngine[e].Contributed, e)
		assert.Equal(t, 1, stats.PerEngine[e].Irrelevant, e)
	}
}

func TestPerEngine_NilStatsIsSafe(t *testing.T) {
	r := New(defaultConfig(), NewMockBlocklistManager())
	assert.NotPanics(t, func() {
		r.RankWithOptions(context.Background(), "anything",
			[]searxng.Result{{URL: "https://x.example/", Title: "y", Engines: []string{"e"}}},
			RankOptions{})
	})
}
