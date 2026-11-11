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

	"github.com/adriangitvitz/kurai/internal/extract"
)

func TestCapContent(t *testing.T) {
	t.Run("short body passes through untouched", func(t *testing.T) {
		assert.Equal(t, "hello", capContent("hello", 100))
	})
	t.Run("zero or negative max means no limit", func(t *testing.T) {
		long := strings.Repeat("x", 1000)
		assert.Equal(t, long, capContent(long, 0))
		assert.Equal(t, long, capContent(long, -1))
	})
	t.Run("long body is cut and marked", func(t *testing.T) {
		got := capContent(strings.Repeat("x", 500), 100)
		assert.True(t, strings.HasPrefix(got, strings.Repeat("x", 100)))
		assert.Contains(t, got, "[truncated: 100 of 500 characters")
		assert.Contains(t, got, "passages_only", "marker must point at the cheaper path")
	})
	t.Run("counts runes, not bytes", func(t *testing.T) {
		got := capContent(strings.Repeat("é", 200), 100)
		body, _, _ := strings.Cut(got, "\n\n[truncated")
		assert.Equal(t, 100, len([]rune(body)))
		assert.True(t, strings.HasPrefix(body, "é"))
	})
}

func TestResolveMaxChars(t *testing.T) {
	assert.Equal(t, defaultMaxContentChars, resolveMaxChars(nil), "absent means default")
	zero := 0
	assert.Equal(t, 0, resolveMaxChars(&zero), "explicit 0 means unlimited")
	n := 123
	assert.Equal(t, 123, resolveMaxChars(&n))
}

func newExtractClient(t *testing.T) (*mcpsdk.ClientSession, context.Context) {
	t.Helper()
	ks := newTestKuraiServer(t)
	mcpsrv := NewToolServer(ks, "test", nil)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	_, err := mcpsrv.Connect(ctx, t1, nil)
	require.NoError(t, err)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	require.NoError(t, err)
	t.Cleanup(func() { cs.Close() })
	return cs, ctx
}

func TestExtractContent_RespectsMaxChars(t *testing.T) {
	body := strings.Repeat("All work and no play. ", 4000)
	html := "<html><head><title>Long</title></head><body><article><h1>Long</h1><p>" +
		body + "</p></article></body></html>"
	tsrv := newHTMLServer(t, html)
	defer tsrv.Close()

	cs, ctx := newExtractClient(t)

	readArticle := func(t *testing.T, args map[string]any) extract.Article {
		t.Helper()
		res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "extract_content", Arguments: args})
		require.NoError(t, err)
		require.False(t, res.IsError, "got: %s", textContents(res.Content))
		var a extract.Article
		require.NoError(t, json.Unmarshal([]byte(textContents(res.Content)), &a))
		return a
	}

	t.Run("explicit max_chars is honored", func(t *testing.T) {
		a := readArticle(t, map[string]any{"url": tsrv.URL, "max_chars": 500})
		assert.Contains(t, a.ContentMarkdown, "[truncated:")
		head, _, _ := strings.Cut(a.ContentMarkdown, "\n\n[truncated")
		assert.Len(t, []rune(head), 500)
	})

	t.Run("default cap applies when max_chars is absent", func(t *testing.T) {
		a := readArticle(t, map[string]any{"url": tsrv.URL})
		assert.Contains(t, a.ContentMarkdown, "[truncated:",
			"an unbounded body is what blows an agent's context")
		head, _, _ := strings.Cut(a.ContentMarkdown, "\n\n[truncated")
		assert.Len(t, []rune(head), defaultMaxContentChars)
	})

	t.Run("max_chars=0 opts out", func(t *testing.T) {
		a := readArticle(t, map[string]any{"url": tsrv.URL, "max_chars": 0})
		assert.NotContains(t, a.ContentMarkdown, "[truncated:")
		assert.Greater(t, len([]rune(a.ContentMarkdown)), defaultMaxContentChars)
	})
}

func TestCapContent_BudgetIsShared(t *testing.T) {
	const budget = 1000
	bodies := []string{
		strings.Repeat("a", 5000),
		strings.Repeat("b", 5000),
		strings.Repeat("c", 5000),
		strings.Repeat("d", 5000),
		strings.Repeat("e", 5000),
	}
	per := budget / len(bodies)

	total := 0
	for _, b := range bodies {
		got := capContent(b, per)
		head, _, _ := strings.Cut(got, "\n\n[truncated")
		total += len([]rune(head))
	}
	assert.Equal(t, budget, total, "five pages must share one budget, not each get it")
}
