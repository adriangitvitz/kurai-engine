package bookmarks

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := OpenSQLite(filepath.Join(t.TempDir(), "bookmarks.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func atClock(s *SQLiteStore, ts time.Time) { s.now = func() time.Time { return ts } }

func sample(url, title string) Bookmark {
	return Bookmark{URL: url, Title: title, Snippet: "snippet", Domain: "example.com",
		ContentType: "article", Meta: "2 engines", Score: 0.42}
}

func TestSQLite_SaveThenListReturnsFields(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	saved, err := s.Save(ctx, "anonymous", sample("https://a.example/1", "Alpha"))
	require.NoError(t, err)
	assert.Equal(t, "https://a.example/1", saved.URL)
	assert.False(t, saved.SavedAt.IsZero(), "server assigns saved_at")

	list, err := s.List(ctx, "anonymous")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Alpha", list[0].Title)
	assert.Equal(t, "snippet", list[0].Snippet)
	assert.Equal(t, "example.com", list[0].Domain)
	assert.Equal(t, "article", list[0].ContentType)
	assert.Equal(t, "2 engines", list[0].Meta)
	assert.InDelta(t, 0.42, list[0].Score, 0.0001)
}

func TestSQLite_SaveSameURLTwiceUpserts(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Save(ctx, "anonymous", sample("https://a.example/1", "Original title"))
	require.NoError(t, err)
	_, err = s.Save(ctx, "anonymous", sample("https://a.example/1", "Updated title"))
	require.NoError(t, err, "re-saving must not error")

	list, err := s.List(ctx, "anonymous")
	require.NoError(t, err)
	require.Len(t, list, 1, "the composite key makes duplicates structurally impossible")
	assert.Equal(t, "Updated title", list[0].Title, "card fields refresh on re-save")
}

func TestSQLite_SaveRequiresURL(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Save(context.Background(), "anonymous", Bookmark{Title: "no url"})
	assert.ErrorIs(t, err, ErrNoURL)

	list, err := s.List(context.Background(), "anonymous")
	require.NoError(t, err)
	assert.Empty(t, list, "a rejected save stores nothing")
}

func TestSQLite_DeleteRemovesOnlyTheTarget(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	_, _ = s.Save(ctx, "anonymous", sample("https://a.example/1", "Alpha"))
	_, _ = s.Save(ctx, "anonymous", sample("https://b.example/2", "Beta"))

	require.NoError(t, s.Delete(ctx, "anonymous", "https://a.example/1"))

	list, err := s.List(ctx, "anonymous")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "https://b.example/2", list[0].URL)
}

func TestSQLite_DeleteAbsentIsNotAnError(t *testing.T) {
	s := newTestStore(t)
	assert.NoError(t, s.Delete(context.Background(), "anonymous", "https://never.saved/x"))
}

func TestSQLite_DeletionPersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bookmarks.db")
	ctx := context.Background()

	s1, err := OpenSQLite(path)
	require.NoError(t, err)
	_, _ = s1.Save(ctx, "anonymous", sample("https://a.example/1", "Alpha"))
	require.NoError(t, s1.Delete(ctx, "anonymous", "https://a.example/1"))
	require.NoError(t, s1.Close())

	s2, err := OpenSQLite(path)
	require.NoError(t, err)
	defer s2.Close()

	list, err := s2.List(ctx, "anonymous")
	require.NoError(t, err)
	assert.Empty(t, list, "a deleted bookmark must not reappear after a restart")
}

func TestSQLite_ListIsNewestFirst(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	base := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	for i, url := range []string{"https://a.example/1", "https://b.example/2", "https://c.example/3"} {
		atClock(s, base.Add(time.Duration(i)*time.Minute))
		_, err := s.Save(ctx, "anonymous", sample(url, "T"))
		require.NoError(t, err)
	}

	list, err := s.List(ctx, "anonymous")
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, "https://c.example/3", list[0].URL, "most recently saved first")
	assert.Equal(t, "https://a.example/1", list[2].URL)
}

func TestSQLite_EmptyBoardIsEmptyNotError(t *testing.T) {
	s := newTestStore(t)
	list, err := s.List(context.Background(), "nobody")
	require.NoError(t, err)
	assert.Empty(t, list)
	assert.NotNil(t, list, "serializes as [] not null")
}

func TestSQLite_BoardsAreUserScoped(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	_, _ = s.Save(ctx, "alice", sample("https://a.example/1", "Alice's"))
	_, _ = s.Save(ctx, "bob", sample("https://b.example/2", "Bob's"))

	alice, err := s.List(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, alice, 1)
	assert.Equal(t, "Alice's", alice[0].Title, "alice sees only her own")

	require.NoError(t, s.Delete(ctx, "bob", "https://a.example/1"))
	alice, err = s.List(ctx, "alice")
	require.NoError(t, err)
	assert.Len(t, alice, 1, "another user's delete cannot affect this board")
}

func TestSQLite_SameURLDifferentUsersAreDistinctBookmarks(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	_, err := s.Save(ctx, "alice", sample("https://shared.example/x", "Alice's copy"))
	require.NoError(t, err)
	_, err = s.Save(ctx, "bob", sample("https://shared.example/x", "Bob's copy"))
	require.NoError(t, err)

	alice, _ := s.List(ctx, "alice")
	bob, _ := s.List(ctx, "bob")
	require.Len(t, alice, 1)
	require.Len(t, bob, 1)
	assert.Equal(t, "Alice's copy", alice[0].Title)
	assert.Equal(t, "Bob's copy", bob[0].Title)
}

func TestSQLite_BookmarksSurviveStoreRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bookmarks.db")
	ctx := context.Background()

	s1, err := OpenSQLite(path)
	require.NoError(t, err)
	_, err = s1.Save(ctx, "anonymous", sample("https://a.example/1", "Survivor"))
	require.NoError(t, err)
	require.NoError(t, s1.Close())

	s2, err := OpenSQLite(path)
	require.NoError(t, err)
	defer s2.Close()

	list, err := s2.List(ctx, "anonymous")
	require.NoError(t, err)
	require.Len(t, list, 1, "the bookmark survives a process restart")
	assert.Equal(t, "Survivor", list[0].Title, "with the fields it was saved with")
}

func TestSQLite_FreshPathYieldsEmptyBoardNotError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "bookmarks.db")
	s, err := OpenSQLite(path)
	require.NoError(t, err)
	defer s.Close()

	list, err := s.List(context.Background(), "anonymous")
	require.NoError(t, err)
	assert.Empty(t, list)
}
