package bookmarks

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleQ(url, title, query string) Bookmark {
	b := sample(url, title)
	b.Query = query
	return b
}

func TestProvenance_QueryStoredVerbatim(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	const typed = "Clima en Guadalajara, Jalisco"
	saved, err := s.Save(ctx, "anonymous", sampleQ("https://a.example/1", "Alpha", typed))
	require.NoError(t, err)
	assert.Equal(t, typed, saved.Query)

	list, _ := s.List(ctx, "anonymous")
	require.Len(t, list, 1)
	assert.Equal(t, typed, list[0].Query)
}

func TestProvenance_BookmarkWithoutQueryIsValid(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	saved, err := s.Save(ctx, "anonymous", sample("https://legacy.example/1", "Imported"))
	require.NoError(t, err)
	assert.Empty(t, saved.Query)

	list, err := s.List(ctx, "anonymous")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Empty(t, list[0].Query, "no query is a legal state, never a fabricated one")
}

func TestProvenance_ReSaveFromNewSearchKeepsFirstProvenance(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	base := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	atClock(s, base)
	_, err := s.Save(ctx, "anonymous", sampleQ("https://a.example/1", "Alpha", "clima en guadalajara"))
	require.NoError(t, err)

	atClock(s, base.Add(time.Hour))
	again, err := s.Save(ctx, "anonymous", sampleQ("https://a.example/1", "Alpha", "weather jalisco"))
	require.NoError(t, err)

	assert.Equal(t, "clima en guadalajara", again.Query, "provenance must not be rewritten")
	assert.Equal(t, base, again.SavedAt.UTC(), "original saved_at is preserved too")

	list, _ := s.List(ctx, "anonymous")
	require.Len(t, list, 1, "still one bookmark for the URL")
	assert.Equal(t, "clima en guadalajara", list[0].Query)
}

func TestProvenance_ReSaveCanBackfillMissingProvenance(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Save(ctx, "anonymous", sample("https://a.example/1", "Alpha"))
	require.NoError(t, err)
	again, err := s.Save(ctx, "anonymous", sampleQ("https://a.example/1", "Alpha", "clima en guadalajara"))
	require.NoError(t, err)
	assert.Equal(t, "clima en guadalajara", again.Query,
		"backfilling absent provenance is not overwriting it")
}

func TestProvenance_ListGroupedByQuery(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	base := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	atClock(s, base)
	_, _ = s.Save(ctx, "anonymous", sampleQ("https://a.example/1", "GDL one", "clima en guadalajara"))
	atClock(s, base.Add(1*time.Minute))
	_, _ = s.Save(ctx, "anonymous", sampleQ("https://a.example/2", "GDL two", "clima en guadalajara"))
	atClock(s, base.Add(2*time.Minute))
	_, _ = s.Save(ctx, "anonymous", sampleQ("https://k.example/3", "Keyboard", "mechanical keyboards"))

	groups, err := s.ListGrouped(ctx, "anonymous")
	require.NoError(t, err)
	require.Len(t, groups, 2)

	assert.Equal(t, "mechanical keyboards", groups[0].Query)
	require.Len(t, groups[0].Bookmarks, 1)

	assert.Equal(t, "clima en guadalajara", groups[1].Query)
	require.Len(t, groups[1].Bookmarks, 2)
	assert.Equal(t, "https://a.example/2", groups[1].Bookmarks[0].URL, "newest-first within a group")
	assert.Equal(t, "https://a.example/1", groups[1].Bookmarks[1].URL)
}

func TestProvenance_ProvenancelessBookmarksFormTheirOwnGroup(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	base := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	atClock(s, base)
	_, _ = s.Save(ctx, "anonymous", sample("https://legacy.example/1", "Imported one"))
	atClock(s, base.Add(1*time.Minute))
	_, _ = s.Save(ctx, "anonymous", sample("https://legacy.example/2", "Imported two"))
	atClock(s, base.Add(2*time.Minute))
	_, _ = s.Save(ctx, "anonymous", sampleQ("https://a.example/3", "Real", "clima en guadalajara"))

	groups, err := s.ListGrouped(ctx, "anonymous")
	require.NoError(t, err)
	require.Len(t, groups, 2)

	assert.Equal(t, "clima en guadalajara", groups[0].Query)

	assert.Empty(t, groups[1].Query, "distinguishable from a real search group")
	assert.Len(t, groups[1].Bookmarks, 2, "collected into a single group")
}

func TestProvenance_GroupedEmptyBoard(t *testing.T) {
	s := newTestStore(t)
	groups, err := s.ListGrouped(context.Background(), "nobody")
	require.NoError(t, err)
	assert.Empty(t, groups)
	assert.NotNil(t, groups)
}
