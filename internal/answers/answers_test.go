package answers

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/searxng"
)

func TestBoundWeather_TruncatesForecastToEightPeriods(t *testing.T) {
	w := &Weather{Place: "Oaxaca City, Mexico", Source: "open-meteo", AsOf: "2026-07-17T14:00:00Z"}
	for i := 0; i < 229; i++ {
		w.Forecast = append(w.Forecast, ForecastPeriod{Date: fmt.Sprintf("2026-07-%02d", i%28+1)})
	}

	bounded := boundWeather(w)
	require.NotNil(t, bounded)
	assert.Len(t, bounded.Forecast, 8)
	assert.Equal(t, "2026-07-01", bounded.Forecast[0].Date)
}

func TestBoundInfobox_EnforcesPerClassCeilings(t *testing.T) {
	long := strings.Repeat("lorem ipsum dolor sit amet ", 40)
	ib := &Infobox{Title: "Albert Einstein", Summary: long, Source: "wikipedia"}
	for i := 0; i < 12; i++ {
		ib.Attributes = append(ib.Attributes, Attribute{Label: fmt.Sprintf("L%d", i), Value: "v"})
	}
	for i := 0; i < 5; i++ {
		ib.Links = append(ib.Links, Link{Title: fmt.Sprintf("link %d", i), URL: "https://example.org"})
	}

	bounded := boundInfobox(ib)
	require.NotNil(t, bounded)

	assert.LessOrEqual(t, utf8.RuneCountInString(bounded.Summary), 500)
	assert.True(t, strings.HasSuffix(bounded.Summary, "…"), "truncation must end in an ellipsis")
	body := strings.TrimSuffix(bounded.Summary, "…")
	assert.False(t, strings.HasSuffix(body, " "))
	assert.True(t, strings.HasSuffix(body, "amet") || strings.HasSuffix(body, "lorem") ||
		strings.HasSuffix(body, "ipsum") || strings.HasSuffix(body, "dolor") ||
		strings.HasSuffix(body, "sit"), "cut must land on a whole word, got %q", body)

	assert.Len(t, bounded.Attributes, 8)
	assert.Equal(t, "L0", bounded.Attributes[0].Label, "attribute order preserved")
	assert.Len(t, bounded.Links, 3)
}

func TestBoundInfobox_ShortSummaryUntouched(t *testing.T) {
	ib := &Infobox{Title: "X", Summary: "short and sweet", Source: "wikipedia"}
	assert.Equal(t, "short and sweet", boundInfobox(ib).Summary)
}

func TestFinalize_TotalCeilingDropOrder(t *testing.T) {
	b := &Block{
		Infobox: &Infobox{Title: "Big Entity", Summary: strings.Repeat("word ", 90), Source: "wikidata"},
	}
	for i := 0; i < 8; i++ {
		b.Infobox.Attributes = append(b.Infobox.Attributes,
			Attribute{Label: fmt.Sprintf("L%d", i), Value: strings.Repeat("v", 700)})
	}

	out := Finalize(b)
	require.NotNil(t, out)
	require.NotNil(t, out.Infobox)

	data, err := json.Marshal(out)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(data), 4096)
	for i, a := range out.Infobox.Attributes {
		assert.Equal(t, fmt.Sprintf("L%d", i), a.Label)
	}
	assert.Less(t, len(out.Infobox.Attributes), 8)
}

func TestFinalize_DropSequenceThroughCorrections(t *testing.T) {
	b := &Block{
		Infobox: &Infobox{
			Title:   "E",
			Summary: strings.Repeat("word ", 99),
			Source:  "wikipedia",
			Links: []Link{
				{Title: "a", URL: "https://a.example"},
				{Title: "b", URL: "https://b.example"},
				{Title: "c", URL: "https://c.example"},
			},
			Attributes: []Attribute{{Label: "L0", Value: strings.Repeat("v", 3000)}},
		},
		Corrections: []string{
			strings.Repeat("x", 1500),
			strings.Repeat("y", 1500),
			strings.Repeat("z", 1500),
		},
	}

	out := Finalize(b)
	require.NotNil(t, out)
	assert.Nil(t, out.Infobox, "infobox dropped at step 4 when trimming cannot fit it")
	assert.Len(t, out.Corrections, 1, "corrections beyond the first dropped at step 5")

	data, err := json.Marshal(out)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(data), 4096)
}

func TestFinalize_UnderBudgetUntouched(t *testing.T) {
	b := &Block{Corrections: []string{"did you mean"}}
	out := Finalize(b)
	require.NotNil(t, out)
	assert.Equal(t, []string{"did you mean"}, out.Corrections)
}

func TestFinalize_EmptyBlockIsNil(t *testing.T) {
	assert.Nil(t, Finalize(&Block{}))
	assert.Nil(t, Finalize(nil))
}

func TestNormalizeContent_ArrayFlattensToReadableText(t *testing.T) {
	got := normalizeContent([]any{"German-born theoretical physicist.", "Developed relativity."})
	assert.Equal(t, "German-born theoretical physicist. Developed relativity.", got)
	assert.NotContains(t, got, "[")
	assert.NotContains(t, got, "{")
}

func TestNormalizeContent_StringPassesThrough(t *testing.T) {
	assert.Equal(t, "plain summary", normalizeContent("plain summary"))
}

func TestNormalizeContent_UnknownShapeYieldsEmpty(t *testing.T) {
	assert.Equal(t, "", normalizeContent(map[string]any{"weird": true}))
	assert.Equal(t, "", normalizeContent(nil))
	assert.Equal(t, "", normalizeContent(42.0))
}

func TestSelectInfobox_DeterministicRichestSource(t *testing.T) {
	boxes := []searxng.Infobox{
		{Infobox: "Entity", Engine: "wikipedia", Attributes: make([]searxng.InfoboxAttribute, 5)},
		{Infobox: "Entity", Engine: "wikidata", Attributes: make([]searxng.InfoboxAttribute, 2)},
	}

	first := selectInfobox(boxes)
	require.NotNil(t, first)
	assert.Equal(t, "wikidata", first.Engine, "wikidata preferred over wikipedia regardless of attribute count")

	second := selectInfobox(boxes)
	assert.Equal(t, first, second)
}

func TestSelectInfobox_AttributeCountTiebreakThenFirstSeen(t *testing.T) {
	sameEngine := []searxng.Infobox{
		{ID: "a", Engine: "wikipedia", Attributes: make([]searxng.InfoboxAttribute, 3)},
		{ID: "b", Engine: "wikipedia", Attributes: make([]searxng.InfoboxAttribute, 5)},
	}
	assert.Equal(t, "b", selectInfobox(sameEngine).ID, "more attributes wins within one engine")

	equal := []searxng.Infobox{
		{ID: "first", Engine: "wikipedia", Attributes: make([]searxng.InfoboxAttribute, 2)},
		{ID: "second", Engine: "wikipedia", Attributes: make([]searxng.InfoboxAttribute, 2)},
	}
	assert.Equal(t, "first", selectInfobox(equal).ID, "first-seen wins on full tie")
}

func TestSelectInfobox_NonAllowlistedEnginesDropped(t *testing.T) {
	boxes := []searxng.Infobox{
		{Infobox: "DDG thing", Engine: "duckduckgo"},
		{Infobox: "No engine at all"},
	}
	assert.Nil(t, selectInfobox(boxes), "only wikipedia/wikidata infoboxes are allowlisted")
}

func TestExtractUpstream_DropsLegacyAnswersCapsCorrections(t *testing.T) {
	resp := &searxng.Response{
		Answers:     json.RawMessage(`[{"answer":"Your IP is: 172.20.0.1","engine":"plugin: self_info"}]`),
		Corrections: []string{"one", "two", "three", "four", "five"},
	}

	b := ExtractUpstream(resp, true)
	require.NotNil(t, b)
	assert.Nil(t, b.Currency)
	assert.Nil(t, b.Weather)
	assert.Nil(t, b.Infobox)
	assert.Equal(t, []string{"one", "two", "three"}, b.Corrections, "corrections capped at 3")
}

func TestExtractUpstream_InfoboxDisabledClassGate(t *testing.T) {
	resp := &searxng.Response{
		Infoboxes: []searxng.Infobox{{Infobox: "Entity", Engine: "wikipedia", Content: "summary"}},
	}
	assert.Nil(t, ExtractUpstream(resp, false), "disabled class contributes nothing → nil block")

	b := ExtractUpstream(resp, true)
	require.NotNil(t, b)
	require.NotNil(t, b.Infobox)
	assert.Equal(t, "Entity", b.Infobox.Title)
	assert.Equal(t, "wikipedia", b.Infobox.Source, "machine-readable source identifier carried")
}

func TestExtractUpstream_NothingSurvivesIsNil(t *testing.T) {
	assert.Nil(t, ExtractUpstream(&searxng.Response{}, true))
	assert.Nil(t, ExtractUpstream(nil, true))
}
