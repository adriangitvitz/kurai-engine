package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
	"github.com/adriangitvitz/kurai/internal/server"
)

func newAnswersKuraiServer(t *testing.T, body string) *server.Server {
	t.Helper()
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(fake.Close)

	cfg := &config.Config{
		Server:     config.ServerConfig{Port: 0, ReadTimeout: time.Second, WriteTimeout: time.Second, PageSize: 10, PrefetchPages: 1, CacheTTL: time.Minute},
		SearXNG:    config.SearXNGConfig{URL: fake.URL, Timeout: 5 * time.Second},
		Ranking:    config.RankingConfig{},
		Blocklists: config.BlocklistConfig{},
		Answers: config.AnswersConfig{
			Enabled: true,
			Infobox: config.InfoboxAnswerConfig{Enabled: true},
		},
	}
	bl := blocklist.NewManager(&cfg.Blocklists, nil)
	rk := ranker.New(&cfg.Ranking, bl)
	return server.New(cfg, searxng.NewClient(&cfg.SearXNG, nil), rk, bl)
}

func callWebSearch(t *testing.T, ks *server.Server, query string) map[string]any {
	t.Helper()
	srv := NewToolServer(ks, "test", nil)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := srv.Connect(ctx, t1, nil)
	require.NoError(t, err)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	require.NoError(t, err)
	defer cs.Close()

	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "web_search",
		Arguments: map[string]any{"query": query},
	})
	require.NoError(t, err)
	require.False(t, res.IsError, "web_search failed: %v", textContents(res.Content))

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContents(res.Content)), &m))
	return m
}

const mcpResultsFixture = `[
	{"url":"https://alpha.example/a","title":"Alpha","content":"snippet","engines":["bing"],"score":10}
]`

func TestWebSearch_CarriesBoundedAnswerBlock(t *testing.T) {
	body := `{"query":"q","results":` + mcpResultsFixture + `,"infoboxes":[{
		"infobox":"Albert Einstein","engine":"wikipedia",
		"content":"German-born theoretical physicist.",
		"urls":[{"title":"Wikipedia","url":"https://en.wikipedia.org/wiki/Albert_Einstein"}]
	}]}`
	m := callWebSearch(t, newAnswersKuraiServer(t, body), "albert einstein")

	require.Contains(t, m, "answers", "MCP result carries the bounded block when an answer exists")
	ib := m["answers"].(map[string]any)["infobox"].(map[string]any)
	assert.Equal(t, "Albert Einstein", ib["title"])
	assert.Equal(t, "wikipedia", ib["source"])

	data, err := json.Marshal(m["answers"])
	require.NoError(t, err)
	assert.LessOrEqual(t, len(data), 4096)
}

func TestWebSearch_NoAnswerMeansNoScaffolding(t *testing.T) {
	body := `{"query":"q","results":` + mcpResultsFixture + `}`
	m := callWebSearch(t, newAnswersKuraiServer(t, body), "plain query")

	assert.NotContains(t, m, "answers",
		"tool-result JSON must contain zero instant-answer keys when no answer exists")
}
