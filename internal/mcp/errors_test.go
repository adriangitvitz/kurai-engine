package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func callErr(t *testing.T, cs *mcpsdk.ClientSession, ctx context.Context, tool string, args map[string]any) toolError {
	t.Helper()
	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: tool, Arguments: args})
	require.NoError(t, err)
	require.True(t, res.IsError, "expected a tool error for %s(%v)", tool, args)

	body := textContents(res.Content)
	var te toolError
	require.NoError(t, json.Unmarshal([]byte(body), &te),
		"error body must be JSON so agents can branch on the code; got: %s", body)
	return te
}

func TestErrors_AreStructuredAndCarryCode(t *testing.T) {
	cs, ctx := newExtractClient(t)

	t.Run("missing query", func(t *testing.T) {
		te := callErr(t, cs, ctx, "web_search", map[string]any{"query": ""})
		assert.Equal(t, "missing_query", te.Code)
		assert.False(t, te.Retryable, "bad input never becomes valid on retry")
		assert.NotEmpty(t, te.RequestID)
		assert.NotContains(t, te.Message, "'q'",
			"must not name the HTTP query parameter in a tool whose parameter is 'query'")
	})

	t.Run("invalid url", func(t *testing.T) {
		te := callErr(t, cs, ctx, "extract_content", map[string]any{"url": "not-a-url"})
		assert.Equal(t, "invalid_url", te.Code)
		assert.False(t, te.Retryable)
	})
}

func TestErrors_OutOfRangeIsRejected(t *testing.T) {
	cs, ctx := newExtractClient(t)

	cases := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{"limit above max", "web_search", map[string]any{"query": "x", "limit": 99}, "limit"},
		{"extract_top above max", "web_search", map[string]any{"query": "x", "extract_top": 20}, "extract_top"},
		{"snippet_length below min", "web_search", map[string]any{"query": "x", "snippet_length": 5}, "snippet_length"},
		{"timeout_ms below min", "extract_content", map[string]any{"url": "http://example.com", "timeout_ms": 5}, "timeout_ms"},
		{"news limit above max", "news_search", map[string]any{"query": "x", "limit": 500}, "limit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			te := callErr(t, cs, ctx, tc.tool, tc.args)
			assert.Equal(t, "invalid_request", te.Code)
			assert.Contains(t, te.Message, tc.want)
			assert.False(t, te.Retryable)
		})
	}
}

func TestRequestID_DiffersPerCall(t *testing.T) {
	cs, ctx := newExtractClient(t)

	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		te := callErr(t, cs, ctx, "web_search", map[string]any{"query": ""})
		require.NotEmpty(t, te.RequestID)
		assert.False(t, seen[te.RequestID], "request_id %q repeated across calls", te.RequestID)
		seen[te.RequestID] = true
	}
	assert.Len(t, seen, 3)
}
