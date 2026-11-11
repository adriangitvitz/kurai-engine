package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
)

func searxFixture(results, answers, infoboxes, corrections, unresponsive string) string {
	if results == "" {
		results = fixtureResults
	}
	body := `{"query":"q","number_of_results":3,"results":` + results
	if answers != "" {
		body += `,"answers":` + answers
	}
	if infoboxes != "" {
		body += `,"infoboxes":` + infoboxes
	}
	if corrections != "" {
		body += `,"corrections":` + corrections
	}
	if unresponsive != "" {
		body += `,"unresponsive_engines":` + unresponsive
	}
	return body + "}"
}

const fixtureResults = `[
	{"url":"https://alpha.example/a","title":"Alpha result","content":"first snippet","engines":["bing"],"score":10},
	{"url":"https://beta.example/b","title":"Beta result","content":"second snippet","engines":["yahoo"],"score":5},
	{"url":"https://gamma.example/c","title":"Gamma result","content":"third snippet","engines":["brave"],"score":1}
]`

const fixtureInfoboxE1 = `[{
	"infobox":"Albert Einstein",
	"id":"https://en.wikipedia.org/wiki/Albert_Einstein",
	"content":"German-born theoretical physicist who developed the theory of relativity.",
	"engine":"wikipedia",
	"img_src":"https://upload.wikimedia.org/einstein.jpg",
	"urls":[{"title":"Wikipedia","url":"https://en.wikipedia.org/wiki/Albert_Einstein"}],
	"attributes":[{"label":"Born","value":"14 March 1879"}]
}]`

const fixtureSelfInfoAnswer = `[{"engine":"plugin: self_info","template":"answer/legacy.html","answer":"Your IP is: 172.20.0.1"}]`
const fixtureHashAnswer = `[{"engine":"plugin: hash_plugin","template":"answer/legacy.html","answer":"md5 hash digest: 0d5b2bb6..."}]`

func newAnswersServer(t *testing.T, body string) *Server {
	t.Helper()
	return newAnswersServerCfg(t, body, func(cfg *config.Config) {})
}

func newAnswersServerCfg(t *testing.T, body string, mutate func(*config.Config)) *Server {
	t.Helper()
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(fake.Close)

	cfg := answersTestConfig(fake.URL)
	mutate(cfg)

	bl := blocklist.NewManager(&cfg.Blocklists, nil)
	rk := ranker.New(&cfg.Ranking, bl)
	rk.SetRelevanceFloor(ranker.RelevanceFloorConfig{})
	client := searxng.NewClient(&cfg.SearXNG, nil)
	return New(cfg, client, rk, bl)
}

func answersTestConfig(searxURL string) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port: 0, ReadTimeout: time.Second, WriteTimeout: time.Second,
			PageSize: 10, PrefetchPages: 1, CacheTTL: time.Minute,
		},
		SearXNG:    config.SearXNGConfig{URL: searxURL, Timeout: 5 * time.Second},
		Ranking:    config.RankingConfig{},
		Blocklists: config.BlocklistConfig{},
		Answers: config.AnswersConfig{
			Enabled:  true,
			Currency: config.CurrencyAnswerConfig{Enabled: true, BaseURL: "http://unused.invalid", Timeout: time.Second},
			Weather:  config.WeatherAnswerConfig{Enabled: true, GeocodingURL: "http://unused.invalid", ForecastURL: "http://unused.invalid", Timeout: time.Second},
			Infobox:  config.InfoboxAnswerConfig{Enabled: true},
		},
	}
}

func doSearch(t *testing.T, s *Server, query string, params string) ([]byte, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/search?q="+query+params, nil)
	rec := httptest.NewRecorder()
	s.handleSearch(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var m map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &m))
	return rec.Body.Bytes(), m
}

func resultOrder(t *testing.T, m map[string]any) []string {
	t.Helper()
	raw, ok := m["results"].([]any)
	require.True(t, ok, "results missing")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.(map[string]any)["url"].(string))
	}
	return out
}

func TestAnswers_AdditiveToRankedResults(t *testing.T) {
	withExtras := newAnswersServer(t, searxFixture("", fixtureSelfInfoAnswer, fixtureInfoboxE1, `["did you mean"]`, ""))
	without := newAnswersServer(t, searxFixture("", "", "", "", ""))

	_, mWith := doSearch(t, withExtras, "einstein", "")
	_, mWithout := doSearch(t, without, "einstein", "")

	require.Contains(t, mWith, "answers")
	assert.Equal(t, resultOrder(t, mWithout), resultOrder(t, mWith),
		"ranked results identical in content and order with and without answer content")
	assert.Equal(t, mWithout["results"], mWith["results"],
		"per-result fields byte-equal across the answer/no-answer runs")
}

func TestAnswers_AbsentMeansNoKeysInBothFormats(t *testing.T) {
	s := newAnswersServer(t, searxFixture("", "", "", "", ""))

	rawFull, mFull := doSearch(t, s, "plain", "")
	assert.NotContains(t, mFull, "answers")
	assert.NotContains(t, string(rawFull), `"answers"`)

	rawCompact, mCompact := doSearch(t, s, "plain", "&response_format=compact")
	assert.NotContains(t, mCompact, "answers")
	assert.NotContains(t, string(rawCompact), `"answers"`)
}

func TestAnswers_CorrectionsForwardedCappedAtThree(t *testing.T) {
	s := newAnswersServer(t, searxFixture("", "", "", `["one","two","three","four","five"]`, ""))

	_, m := doSearch(t, s, "misspeled", "")
	answers := m["answers"].(map[string]any)
	got := answers["corrections"].([]any)
	assert.Equal(t, []any{"one", "two", "three"}, got)
}

func TestAnswers_SelfInfoNeverForwarded(t *testing.T) {
	s := newAnswersServer(t, searxFixture("", fixtureSelfInfoAnswer, "", "", ""))

	for _, params := range []string{"", "&response_format=compact"} {
		raw, m := doSearch(t, s, "ip", params)
		assert.NotContains(t, m, "answers", "params=%q", params)
		assert.NotContains(t, string(raw), "172.20.0.1", "server identity must leave no trace")
		assert.Len(t, m["results"], 3, "ranked results returned normally")
	}
}

func TestAnswers_GenericHashAnswerNeverForwarded(t *testing.T) {
	s := newAnswersServer(t, searxFixture("", fixtureHashAnswer, "", "", ""))

	raw, m := doSearch(t, s, "md5+test", "")
	assert.NotContains(t, m, "answers")
	assert.NotContains(t, string(raw), "hash digest")
}

func TestAnswers_SuspendedEnginesDegradeSilently(t *testing.T) {
	s := newAnswersServer(t, searxFixture("", `[]`, "", "",
		`[["wikipedia","Suspended: too many requests"],["duckduckgo","CAPTCHA"]]`))

	raw, m := doSearch(t, s, "einstein", "")
	assert.NotContains(t, m, "answers")
	assert.NotContains(t, m, "error")
	assert.NotContains(t, string(raw), `"error"`)
	assert.Len(t, m["results"], 3)
}

func TestAnswers_MalformedAnswerEntriesDegradeSilently(t *testing.T) {
	cases := map[string]string{
		"answers is not an array": `"garbage"`,
		"entry is not an object":  `[42, "loose string"]`,
		"entry field wrong type":  `[{"answer": 42, "engine": true}]`,
	}
	for name, malformed := range cases {
		t.Run(name, func(t *testing.T) {
			s := newAnswersServer(t, searxFixture("", malformed, "", "", ""))
			_, m := doSearch(t, s, "anything", "")
			assert.NotContains(t, m, "answers")
			assert.NotContains(t, m, "error")
			assert.Len(t, m["results"], 3)
		})
	}
}

func TestAnswers_TavilyFallbackNoAnswerScaffolding(t *testing.T) {
	oneResult := `[{"url":"https://alpha.example/a","title":"Alpha","content":"only hit","engines":["bing"],"score":10}]`

	fakeTavily := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/usage"):
			fmt.Fprint(w, `{"account":{"plan_usage":1,"plan_limit":1000}}`)
		default:
			fmt.Fprint(w, `{"results":[{"url":"https://tavily.example/t","title":"Tavily hit","content":"fallback snippet","score":0.9}]}`)
		}
	}))
	t.Cleanup(fakeTavily.Close)

	t.Setenv("TEST_TAVILY_KEY", "test-key")
	s := newAnswersServerCfg(t, searxFixture(oneResult, "", "", "", ""), func(cfg *config.Config) {
		cfg.Tavily = config.TavilyConfig{
			Enabled: true, URL: fakeTavily.URL, APIKeyEnv: "TEST_TAVILY_KEY",
			MaxResults: 5, SearchDepth: "basic", Timeout: 3 * time.Second,
		}
	})

	raw, m := doSearch(t, s, "rare+query", "")
	assert.NotContains(t, m, "answers")
	assert.NotContains(t, m, "error")
	assert.GreaterOrEqual(t, len(m["results"].([]any)), 2, "merged searx + tavily results")
	assert.Contains(t, string(raw), "tavily.example")
}

func TestAnswers_CompactCarriesAnswerWhenPresent(t *testing.T) {
	s := newAnswersServer(t, searxFixture("", "", fixtureInfoboxE1, "", ""))

	_, m := doSearch(t, s, "einstein", "&response_format=compact")
	require.Contains(t, m, "answers")
	ib := m["answers"].(map[string]any)["infobox"].(map[string]any)
	assert.Equal(t, "Albert Einstein", ib["title"])
	assert.Equal(t, "wikipedia", ib["source"])
}

func TestAnswers_CompactUnchangedWhenNoAnswer(t *testing.T) {
	body := searxFixture("", "", "", "", "")
	enabled := newAnswersServer(t, body)
	disabled := newAnswersServerCfg(t, body, func(cfg *config.Config) {
		cfg.Answers.Enabled = false
	})

	_, mEnabled := doSearch(t, enabled, "plain", "&response_format=compact")
	_, mDisabled := doSearch(t, disabled, "plain", "&response_format=compact")

	for _, m := range []map[string]any{mEnabled, mDisabled} {
		delete(m, "request_id")
		if meta, ok := m["meta"].(map[string]any); ok {
			delete(meta, "processing_time_ms")
			if engines, ok := meta["engines_used"].([]any); ok {
				sort.Slice(engines, func(i, j int) bool {
					return engines[i].(string) < engines[j].(string)
				})
			}
		}
	}
	assert.Equal(t, mDisabled, mEnabled,
		"no-answer compact response must match the answers-disabled (pre-change) contract exactly")
}

func TestAnswers_APIHasNoDisplayPreferenceInput(t *testing.T) {
	s := newAnswersServer(t, searxFixture("", "", fixtureInfoboxE1, "", ""))

	req := httptest.NewRequest(http.MethodGet, "/search?q=einstein&answers=off&preferences=hidden", nil)
	req.Header.Set("Cookie", "instant_answers=off")
	rec := httptest.NewRecorder()
	s.handleSearch(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var m map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &m))
	require.Contains(t, m, "answers")
}

func TestAnswers_E1InfoboxForwarded(t *testing.T) {
	s := newAnswersServer(t, searxFixture("", "", fixtureInfoboxE1, "", ""))

	_, m := doSearch(t, s, "albert+einstein", "")
	require.Contains(t, m, "answers")
	ib := m["answers"].(map[string]any)["infobox"].(map[string]any)
	assert.Equal(t, "Albert Einstein", ib["title"])
	assert.Contains(t, ib["summary"], "theoretical physicist")
	assert.Equal(t, "https://upload.wikimedia.org/einstein.jpg", ib["image_url"])
	links := ib["links"].([]any)
	require.Len(t, links, 1)
	assert.Equal(t, "https://en.wikipedia.org/wiki/Albert_Einstein", links[0].(map[string]any)["url"])
}

func TestAnswers_NoInfoboxNoPanelKeys(t *testing.T) {
	s := newAnswersServer(t, searxFixture("", "", "", "", `[["wikidata","Suspended: access denied"]]`))

	_, m := doSearch(t, s, "obscure+entity", "")
	assert.NotContains(t, m, "answers")
	assert.Len(t, m["results"], 3)
}

func TestAnswers_InfoboxCarriesSourceIdentifier(t *testing.T) {
	s := newAnswersServer(t, searxFixture("", "", fixtureInfoboxE1, "", ""))

	_, m := doSearch(t, s, "einstein", "")
	ib := m["answers"].(map[string]any)["infobox"].(map[string]any)
	assert.Equal(t, "wikipedia", ib["source"], "machine-readable source identifier end to end (E4 engine field)")
}

func TestAnswers_InfoboxSurvivesCacheHit(t *testing.T) {
	s := newAnswersServer(t, searxFixture("", "", fixtureInfoboxE1, "", ""))

	_, first := doSearch(t, s, "einstein", "")
	require.Contains(t, first, "answers")

	_, second := doSearch(t, s, "einstein", "")
	require.Contains(t, second, "answers", "cache-hit request must show the same infobox (no flicker)")
	assert.True(t, second["meta"].(map[string]any)["from_cache"].(bool), "second request served from cache")
	assert.Equal(t, first["answers"], second["answers"])
}
