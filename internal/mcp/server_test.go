package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/extract"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/server"
)

func newTestKuraiServer(t *testing.T) *server.Server {
	t.Helper()
	cfg := &config.Config{
		Server:     config.ServerConfig{Port: 0, ReadTimeout: time.Second, WriteTimeout: time.Second, PageSize: 10, PrefetchPages: 1, CacheTTL: time.Minute},
		Ranking:    config.RankingConfig{Weights: config.WeightConfig{}, Penalties: config.PenaltyConfig{}},
		Blocklists: config.BlocklistConfig{},
		Extract:    config.ExtractConfig{Enabled: true, Timeout: 5 * time.Second, MaxBytes: 1 << 20, MaxRedirects: 5, AllowPrivateNetworks: true},
	}
	bl := blocklist.NewManager(&cfg.Blocklists, nil)
	r := ranker.New(&cfg.Ranking, bl)
	return server.New(cfg, nil, r, bl)
}

func TestNewToolServer_RegistersExpectedTools(t *testing.T) {
	ks := newTestKuraiServer(t)
	srv := NewToolServer(ks, "test", nil)
	require.NotNil(t, srv)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := srv.Connect(ctx, t1, nil)
	require.NoError(t, err)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	require.NoError(t, err)
	defer cs.Close()

	listed, err := cs.ListTools(ctx, &mcpsdk.ListToolsParams{})
	require.NoError(t, err)
	gotNames := map[string]bool{}
	for _, tool := range listed.Tools {
		gotNames[tool.Name] = true
	}
	for _, want := range []string{"web_search", "news_search", "image_search", "extract_content"} {
		assert.True(t, gotNames[want], "expected tool %q to be registered, got: %v", want, gotNames)
	}
}

func TestWebSearchTool_RejectsEmptyQuery(t *testing.T) {
	ks := newTestKuraiServer(t)
	srv := NewToolServer(ks, "test", nil)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := srv.Connect(ctx, t1, nil)
	require.NoError(t, err)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	require.NoError(t, err)
	defer cs.Close()

	got, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "web_search",
		Arguments: map[string]any{"query": ""},
	})
	require.NoError(t, err)
	assert.True(t, got.IsError, "expected IsError on empty query, got %#v", got)
	require.NotEmpty(t, got.Content)
}

func TestExtractContentTool_ReturnsArticle(t *testing.T) {
	html := "<html><head><title>Fixture</title></head><body><article><h1>Fixture</h1><p>Hello world from a unit test fixture.</p></article></body></html>"
	srv := newTestKuraiServer(t)
	mcpsrv := NewToolServer(srv, "test", nil)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := mcpsrv.Connect(ctx, t1, nil)
	require.NoError(t, err)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	require.NoError(t, err)
	defer cs.Close()

	tsrv := newHTMLServer(t, html)
	defer tsrv.Close()

	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "extract_content",
		Arguments: map[string]any{"url": tsrv.URL, "format": "markdown"},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError, "extract_content should succeed; got: %v", textContents(res.Content))

	body := textContents(res.Content)
	require.NotEmpty(t, body)
	var article extract.Article
	require.NoError(t, json.Unmarshal([]byte(body), &article), "body=%s", body)
	assert.Equal(t, "Fixture", article.Title)
	assert.Contains(t, strings.ToLower(article.ContentMarkdown), "hello world")
}

func textContents(items []mcpsdk.Content) string {
	var b strings.Builder
	for _, c := range items {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}
