package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToolAnnotations_AllReadOnly(t *testing.T) {
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

	res, err := cs.ListTools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, res.Tools, 5)

	for _, tool := range res.Tools {
		require.NotNil(t, tool.Annotations, "tool %q has no annotations", tool.Name)
		assert.True(t, tool.Annotations.ReadOnlyHint, "tool %q: ReadOnlyHint", tool.Name)
		assert.True(t, tool.Annotations.IdempotentHint, "tool %q: IdempotentHint", tool.Name)
		require.NotNil(t, tool.Annotations.OpenWorldHint, "tool %q: OpenWorldHint", tool.Name)
		assert.True(t, *tool.Annotations.OpenWorldHint, "tool %q: OpenWorldHint", tool.Name)
		assert.NotEmpty(t, tool.Annotations.Title, "tool %q: Title", tool.Name)
	}
}

func TestInstructions_SentOnInitialize(t *testing.T) {
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

	got := cs.InitializeResult().Instructions
	require.NotEmpty(t, got, "server sent no instructions")
	assert.Contains(t, got, "passages_only", "instructions must cover the token-saving knobs")
	assert.Contains(t, got, "extract_top")
}

func TestHTTPHandler_KeepAliveSurvivesWriteTimeout(t *testing.T) {
	ks := newTestKuraiServer(t)
	handler := NewHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return NewToolServer(ks, "test", &Options{KeepAlive: 100 * time.Millisecond})
	}, time.Minute)

	ts := httptest.NewUnstartedServer(handler)
	ts.Config.WriteTimeout = 250 * time.Millisecond
	ts.Start()
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, &mcpsdk.StreamableClientTransport{Endpoint: ts.URL}, nil)
	require.NoError(t, err)
	defer cs.Close()

	time.Sleep(2 * time.Second)

	_, err = cs.ListTools(ctx, nil)
	require.NoError(t, err, "session died after keepalive pings crossed the write deadline")
}
