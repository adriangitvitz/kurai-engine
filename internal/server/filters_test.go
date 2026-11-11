package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/adriangitvitz/kurai/internal/ranker"
)

func mkScored(url string) ranker.ScoredResult {
	return ranker.ScoredResult{URL: url, Title: url, Snippet: "Some snippet text that is moderately long and contains punctuation."}
}

func TestApplyPostRankFilters_IncludeDomains(t *testing.T) {
	in := []ranker.ScoredResult{
		mkScored("https://arxiv.org/abs/1234"),
		mkScored("https://reddit.com/r/x"),
		mkScored("https://github.com/y/z"),
	}
	got, _ := applyPostRankFilters(in, ServiceSearchReq{IncludeDomains: []string{"arxiv.org", "github.com"}})
	assert.Len(t, got, 2)
}

func TestApplyPostRankFilters_ExcludeDomains(t *testing.T) {
	in := []ranker.ScoredResult{
		mkScored("https://arxiv.org/abs/1234"),
		mkScored("https://reddit.com/r/x"),
	}
	got, _ := applyPostRankFilters(in, ServiceSearchReq{ExcludeDomains: []string{"reddit.com"}})
	assert.Len(t, got, 1)
	assert.Contains(t, got[0].URL, "arxiv.org")
}

func TestApplyPostRankFilters_DateFromTo(t *testing.T) {
	in := []ranker.ScoredResult{
		{URL: "a", PublishedDate: "2024-01-15"},
		{URL: "b", PublishedDate: "2025-06-01"},
		{URL: "c", PublishedDate: "2026-04-29"},
		{URL: "d"},
	}
	got, _ := applyPostRankFilters(in, ServiceSearchReq{
		DateFrom: "2025-01-01",
		DateTo:   "2025-12-31",
	})
	urls := map[string]bool{}
	for _, r := range got {
		urls[r.URL] = true
	}
	assert.True(t, urls["b"], "in-range result should pass")
	assert.True(t, urls["d"], "undated result should pass (no evidence to drop)")
	assert.False(t, urls["a"], "before date_from must be dropped")
	assert.False(t, urls["c"], "after date_to must be dropped")
}

func TestApplyPostRankFilters_SnippetLengthAddsEllipsis(t *testing.T) {
	in := []ranker.ScoredResult{mkScored("https://a.com/x")}
	got, _ := applyPostRankFilters(in, ServiceSearchReq{SnippetLength: 20})
	assert.Equal(t, 21, len([]rune(got[0].Snippet)), "20 runes + ellipsis = 21")
	assert.True(t, len(got[0].Snippet) > 0)
}

func TestApplyPostRankFilters_LimitDoesNotShrinkTheFilteredSet(t *testing.T) {
	in := []ranker.ScoredResult{mkScored("https://a.com/1"), mkScored("https://b.com/2"), mkScored("https://c.com/3")}
	got, _ := applyPostRankFilters(in, ServiceSearchReq{Limit: 2})
	assert.Len(t, got, 3)
}

func TestApplyPostRankFilters_DomainNormalization(t *testing.T) {
	in := []ranker.ScoredResult{mkScored("https://www.Example.COM/x")}
	got, _ := applyPostRankFilters(in, ServiceSearchReq{IncludeDomains: []string{"example.com"}})
	assert.Len(t, got, 1, "domain comparison must be case-insensitive + www-stripped on both sides")
}

func TestSplitCSV(t *testing.T) {
	assert.Nil(t, splitCSV(""))
	assert.Equal(t, []string{"a", "b"}, splitCSV("a,b"))
	assert.Equal(t, []string{"a", "b"}, splitCSV(" a , b "))
	assert.Equal(t, []string{"a"}, splitCSV("a,,"))
}

func TestApplyPostRankFilters_DateStrictDropsUndated(t *testing.T) {
	in := []ranker.ScoredResult{
		{URL: "in-range", PublishedDate: "2026-03-01"},
		{URL: "undated"},
		{URL: "unparseable", PublishedDate: "last spring"},
		{URL: "too-old", PublishedDate: "2025-12-31"},
	}
	got, stats := applyPostRankFilters(in, ServiceSearchReq{DateFrom: "2026-01-01", DateStrict: true})
	assert.Len(t, got, 1)
	assert.Equal(t, "in-range", got[0].URL)
	assert.Equal(t, 2, stats.UndatedFiltered)
}

func TestApplyPostRankFilters_DateStrictWithoutRangeKeepsUndated(t *testing.T) {
	in := []ranker.ScoredResult{{URL: "undated"}}
	got, stats := applyPostRankFilters(in, ServiceSearchReq{DateStrict: true})
	assert.Len(t, got, 1)
	assert.Zero(t, stats.UndatedFiltered)
}

func TestApplyPostRankFilters_QuotedPhraseIsRequired(t *testing.T) {
	in := []ranker.ScoredResult{
		{URL: "https://a.example/1", Title: "Northern  Trust adopts HSMs", Snippet: "custody"},
		{URL: "https://b.example/2", Title: "Thales Luna HSM", Snippet: "generic vendor page"},
		{URL: "https://c.example/northern-trust", Title: "Custody news", Snippet: "the bank NORTHERN TRUST said"},
	}
	got, stats := applyPostRankFilters(in, ServiceSearchReq{Query: `"Northern Trust" HSM`})
	assert.Len(t, got, 2)
	assert.Equal(t, 1, stats.PhraseFiltered)
	assert.False(t, stats.PhraseRelaxed)
	assert.Len(t, in, 3, "the cached set must not be mutated")
}

func TestApplyPostRankFilters_PhraseRelaxedWhenNothingMatches(t *testing.T) {
	in := []ranker.ScoredResult{
		{URL: "https://b.example/2", Title: "Thales Luna HSM"},
		{URL: "https://d.example/4", Title: "Hardware security module"},
	}
	got, stats := applyPostRankFilters(in, ServiceSearchReq{Query: `"Northern Trust" HSM OR Thales OR "hardware security module"`})
	assert.Len(t, got, 2)
	assert.True(t, stats.PhraseRelaxed)
	assert.Zero(t, stats.PhraseFiltered)
}

func TestApplyPostRankFilters_ApostrophesAreNotQuotes(t *testing.T) {
	in := []ranker.ScoredResult{{URL: "https://a.example/1", Title: "Unrelated"}}
	got, stats := applyPostRankFilters(in, ServiceSearchReq{Query: "why don't banks publish the company's HSM vendor"})
	assert.Len(t, got, 1)
	assert.False(t, stats.PhraseRelaxed)
}

func TestApplyPostRankFilters_PhraseMatchesBeforeSnippetTruncation(t *testing.T) {
	in := []ranker.ScoredResult{{URL: "https://a.example/1", Snippet: "A long lead-in sentence that pushes the phrase far away: Northern Trust"}}
	got, stats := applyPostRankFilters(in, ServiceSearchReq{Query: `"northern trust"`, SnippetLength: 10})
	assert.Len(t, got, 1)
	assert.False(t, stats.PhraseRelaxed)
	assert.Equal(t, 11, len([]rune(got[0].Snippet)))
}

func TestRequiredPhraseGroups(t *testing.T) {
	cases := map[string][][]string{
		`plain words only`:                 nil,
		`"Northern Trust" HSM`:             {{"northern trust"}},
		`"a b" OR "c d" extra`:             {{"a b", "c d"}},
		`"a b" x OR y`:                     {{"a b"}},
		`x OR "a b" OR y "c d"`:            {{"x", "a b", "y"}, {"c d"}},
		`OR "a b"`:                         {{"a b"}},
		`"a b" or "c d"`:                   {{"a b"}, {"c d"}},
		`don't touch the company's phrase`: nil,
	}
	for query, want := range cases {
		assert.Equal(t, want, requiredPhraseGroups(query), query)
	}
}

func TestSiteOperator(t *testing.T) {
	assert.Equal(t, "", siteOperator(nil))
	assert.Equal(t, "site:arxiv.org", siteOperator([]string{"WWW.ArXiv.org"}))
	assert.Equal(t, "site:a.example OR site:b.example", siteOperator([]string{"b.example", "a.example", "a.example"}))
	assert.Equal(t, "site:ok.example", siteOperator([]string{"ok.example", "bad.example OR intitle:x", "-x.example"}))
}

func TestApplyPostRankFilters_DomainsMatchSubdomains(t *testing.T) {
	in := []ranker.ScoredResult{
		mkScored("https://locations.northerntrust.com/a"),
		mkScored("https://www.northerntrust.com/b"),
		mkScored("https://notnortherntrust.com/c"),
		mkScored("https://northerntrust.com.evil.example/d"),
	}
	kept, _ := applyPostRankFilters(in, ServiceSearchReq{IncludeDomains: []string{"northerntrust.com"}})
	assert.Len(t, kept, 2)

	rest, _ := applyPostRankFilters(in, ServiceSearchReq{ExcludeDomains: []string{"northerntrust.com"}})
	assert.Len(t, rest, 2)
	assert.Contains(t, rest[0].URL, "notnortherntrust.com")
}

func TestUpstreamTimeRange(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	cases := map[string]string{
		"":           "",
		"not-a-date": "",
		"2026-11-11": "day",
		"2026-11-06": "week",
		"2026-10-20": "month",
		"2026-01-01": "year",
		"2025-11-01": "",
		"2027-01-01": "",
	}
	for from, want := range cases {
		assert.Equal(t, want, upstreamTimeRange(from, now), from)
	}
}

func TestApplyPostRankFilters_SiteScopeRequiresATermBeyondTheSiteName(t *testing.T) {
	in := []ranker.ScoredResult{
		{URL: "https://www.northerntrust.com/pr/2026/custody", Title: "Northern Trust builds custody on the Canton Network"},
		{URL: "https://www.northerntrust.com/united-states/login", Title: "login | Northern Trust"},
		{URL: "https://www.northerntrust.com/about-us/careers", Title: "Northern Trust | Careers"},
	}
	got, stats := applyPostRankFilters(in, ServiceSearchReq{Query: `"Northern Trust" Canton`, IncludeDomains: []string{"northerntrust.com"}})
	assert.Len(t, got, 1)
	assert.Contains(t, got[0].URL, "/pr/2026/custody")
	assert.Equal(t, 2, stats.OffTopicFiltered)
}

func TestApplyPostRankFilters_SiteScopeRelaxesWhenNothingIsOnTopic(t *testing.T) {
	in := []ranker.ScoredResult{
		{URL: "https://www.northerntrust.com/united-states/login", Title: "login | Northern Trust"},
		{URL: "https://www.northerntrust.com/about-us/careers", Title: "Northern Trust | Careers"},
	}
	got, stats := applyPostRankFilters(in, ServiceSearchReq{Query: "Northern Trust Canton", IncludeDomains: []string{"northerntrust.com"}})
	assert.Len(t, got, 2)
	assert.Zero(t, stats.OffTopicFiltered)
}

func TestApplyPostRankFilters_SiteScopeIgnoresQueriesThatOnlyNameTheSite(t *testing.T) {
	in := []ranker.ScoredResult{{URL: "https://www.northerntrust.com/about-us/careers", Title: "Careers"}}
	got, stats := applyPostRankFilters(in, ServiceSearchReq{Query: "Northern Trust", IncludeDomains: []string{"northerntrust.com"}})
	assert.Len(t, got, 1)
	assert.Zero(t, stats.OffTopicFiltered)
}

func TestApplyPostRankFilters_NoSiteScopeNoTermRequirement(t *testing.T) {
	in := []ranker.ScoredResult{{URL: "https://www.northerntrust.com/united-states/login", Title: "login | Northern Trust"}}
	got, _ := applyPostRankFilters(in, ServiceSearchReq{Query: "Northern Trust Canton"})
	assert.Len(t, got, 1)
}

func TestCollapseSameSiteTitles(t *testing.T) {
	in := []ranker.ScoredResult{
		{URL: "https://www.northerntrust.com/united-states/pr/2026/custody", Title: "Northern Trust Builds Tokenized Asset Custody", FinalRank: 1},
		{URL: "https://www.businesswire.com/news/custody", Title: "Northern Trust Builds Tokenized Asset Custody", FinalRank: 2},
		{URL: "https://locations.northerntrust.com/middle-east/pr/2026/custody", Title: "Northern  Trust builds tokenized asset custody", FinalRank: 3},
		{URL: "https://docs.example.com/a", Title: "API Reference", FinalRank: 4},
		{URL: "https://docs.example.com/b", Title: "API Reference", FinalRank: 5},
	}
	got := collapseSameSiteTitles(in)

	urls := make([]string, len(got))
	for i, r := range got {
		urls[i] = r.URL
		assert.Equal(t, i+1, r.FinalRank)
	}
	assert.Equal(t, []string{
		"https://www.northerntrust.com/united-states/pr/2026/custody",
		"https://www.businesswire.com/news/custody",
		"https://docs.example.com/a",
		"https://docs.example.com/b",
	}, urls)
	assert.Len(t, in, 5)
}
