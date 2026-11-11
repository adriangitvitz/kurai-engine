package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware_GeneratesIDWhenAbsent(t *testing.T) {
	srv := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	require.NotEmpty(t, id, "X-Request-ID header must be set")
	assert.True(t, strings.HasPrefix(id, "req_"), "request id must use req_ prefix; got %q", id)
}

func TestRequestIDMiddleware_HonorsClientProvidedID(t *testing.T) {
	srv := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "client-supplied-12345")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	assert.Equal(t, "client-supplied-12345", rec.Header().Get("X-Request-ID"))
}

func TestBuildPagination(t *testing.T) {
	cases := []struct {
		name              string
		total, page, ps   int
		expectNil         bool
		expectHasMore     bool
		expectNextPagePtr bool
	}{
		{"empty", 0, 1, 10, true, false, false},
		{"single page exactly fills", 10, 1, 10, false, false, false},
		{"first of many", 25, 1, 10, false, true, true},
		{"last of many", 25, 3, 10, false, false, false},
		{"middle of many", 25, 2, 10, false, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := buildPagination(c.total, c.page, c.ps)
			if c.expectNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, c.expectHasMore, got.HasMore)
			if c.expectNextPagePtr {
				require.NotNil(t, got.NextPage)
				assert.Equal(t, c.page+1, *got.NextPage)
			} else {
				assert.Nil(t, got.NextPage)
			}
		})
	}
}
