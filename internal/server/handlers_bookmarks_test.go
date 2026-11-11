package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/ranker"
)

func newBookmarkServer(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{
		Server:     config.ServerConfig{Port: 0, ReadTimeout: time.Second, WriteTimeout: time.Second, PageSize: 10, CacheTTL: time.Minute},
		Ranking:    config.RankingConfig{},
		Blocklists: config.BlocklistConfig{},
		Bookmarks: config.BookmarksConfig{
			Enabled: true,
			Path:    filepath.Join(t.TempDir(), "bookmarks.db"),
		},
	}
	bl := blocklist.NewManager(&cfg.Blocklists, nil)
	rk := ranker.New(&cfg.Ranking, bl)
	s := New(cfg, nil, rk, bl)
	require.NotNil(t, s.bookmarks, "store must open for these tests")
	t.Cleanup(func() { _ = s.bookmarks.Close() })
	return s
}

func saveBookmark(t *testing.T, s *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/bookmarks", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.handleBookmarks(rec, req)
	return rec
}

func listBookmarks(t *testing.T, s *Server, query string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/bookmarks"+query, nil)
	rec := httptest.NewRecorder()
	s.handleBookmarks(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var m map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &m))
	return m
}

func TestBookmarksAPI_SaveListDeleteRoundTrip(t *testing.T) {
	s := newBookmarkServer(t)

	rec := saveBookmark(t, s, `{"url":"https://a.example/1","title":"Alpha","domain":"a.example","query":"clima en guadalajara"}`)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var saved map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &saved))
	assert.Equal(t, "https://a.example/1", saved["url"])
	assert.Equal(t, "clima en guadalajara", saved["query"])
	assert.NotEmpty(t, saved["saved_at"], "server assigns saved_at")

	list := listBookmarks(t, s, "")["bookmarks"].([]any)
	require.Len(t, list, 1)
	assert.Equal(t, "Alpha", list[0].(map[string]any)["title"])

	del := httptest.NewRequest(http.MethodDelete, "/bookmarks?url="+url.QueryEscape("https://a.example/1"), nil)
	drec := httptest.NewRecorder()
	s.handleBookmarks(drec, del)
	assert.Equal(t, http.StatusNoContent, drec.Code)

	assert.Empty(t, listBookmarks(t, s, "")["bookmarks"], "deleted bookmark is gone from the board")
}

func TestBookmarksAPI_SaveWithoutURLIsRejected(t *testing.T) {
	s := newBookmarkServer(t)

	rec := saveBookmark(t, s, `{"title":"no url here"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_request", "uses the existing stable codes")

	assert.Empty(t, listBookmarks(t, s, "")["bookmarks"], "a rejected save stores nothing")
}

func TestBookmarksAPI_MalformedBodyIsRejected(t *testing.T) {
	s := newBookmarkServer(t)
	rec := saveBookmark(t, s, `{not json`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBookmarksAPI_DeleteRequiresURL(t *testing.T) {
	s := newBookmarkServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/bookmarks", nil)
	rec := httptest.NewRecorder()
	s.handleBookmarks(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBookmarksAPI_DeleteAbsentIsIdempotent(t *testing.T) {
	s := newBookmarkServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/bookmarks?url="+url.QueryEscape("https://never.saved/x"), nil)
	rec := httptest.NewRecorder()
	s.handleBookmarks(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code, "a retry or double-click never errors")
}

func TestBookmarksAPI_UnsupportedMethod(t *testing.T) {
	s := newBookmarkServer(t)
	req := httptest.NewRequest(http.MethodPatch, "/bookmarks", nil)
	rec := httptest.NewRecorder()
	s.handleBookmarks(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestBookmarksAPI_EmptyBoardIsEmptyListNotError(t *testing.T) {
	s := newBookmarkServer(t)
	m := listBookmarks(t, s, "")
	assert.Empty(t, m["bookmarks"])
	assert.NotContains(t, m, "error")
	assert.Contains(t, strings.ReplaceAll(mustJSON(t, m), " ", ""), `"bookmarks":[]`)
}

func TestBookmarksAPI_AnonymousScopingWhenAuthDisabled(t *testing.T) {
	s := newBookmarkServer(t)
	saveBookmark(t, s, `{"url":"https://a.example/1","title":"Alpha"}`)

	list, err := s.bookmarks.List(t.Context(), "anonymous")
	require.NoError(t, err)
	require.Len(t, list, 1, "stored under the anonymous identity")

	other, err := s.bookmarks.List(t.Context(), "someone-else")
	require.NoError(t, err)
	assert.Empty(t, other)
}

func TestBookmarksAPI_GroupedByQuery(t *testing.T) {
	s := newBookmarkServer(t)

	saveBookmark(t, s, `{"url":"https://a.example/1","title":"GDL one","query":"clima en guadalajara"}`)
	saveBookmark(t, s, `{"url":"https://a.example/2","title":"GDL two","query":"clima en guadalajara"}`)
	saveBookmark(t, s, `{"url":"https://k.example/3","title":"Keyboard","query":"mechanical keyboards"}`)
	saveBookmark(t, s, `{"url":"https://legacy.example/4","title":"Imported"}`)

	groups := listBookmarks(t, s, "?group=query")["groups"].([]any)
	require.Len(t, groups, 3)

	byQuery := map[string]int{}
	provenanceless := 0
	for _, g := range groups {
		gm := g.(map[string]any)
		q, _ := gm["query"].(string)
		n := len(gm["bookmarks"].([]any))
		if q == "" {
			provenanceless = n
			continue
		}
		byQuery[q] = n
	}
	assert.Equal(t, 2, byQuery["clima en guadalajara"])
	assert.Equal(t, 1, byQuery["mechanical keyboards"])
	assert.Equal(t, 1, provenanceless, "provenance-less bookmarks form their own group")
}

func TestBookmarksAPI_ReSaveKeepsFirstProvenanceThroughAPI(t *testing.T) {
	s := newBookmarkServer(t)

	saveBookmark(t, s, `{"url":"https://a.example/1","title":"Alpha","query":"clima en guadalajara"}`)
	rec := saveBookmark(t, s, `{"url":"https://a.example/1","title":"Alpha","query":"weather jalisco"}`)
	require.Equal(t, http.StatusOK, rec.Code)

	var again map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &again))
	assert.Equal(t, "clima en guadalajara", again["query"], "first provenance survives a re-save")

	assert.Len(t, listBookmarks(t, s, "")["bookmarks"], 1, "and no duplicate is created")
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
