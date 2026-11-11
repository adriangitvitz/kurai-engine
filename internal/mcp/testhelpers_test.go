package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newHTMLServer(t *testing.T, html string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}))
	return srv
}
