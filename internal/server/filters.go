package server

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

func filterByContentType(results []ranker.ScoredResult, ct string) []ranker.ScoredResult {
	if ct == "" {
		return results
	}
	out := make([]ranker.ScoredResult, 0, len(results))
	for _, r := range results {
		rt := string(r.ContentType)
		if rt == "" {
			rt = string(ranker.ContentArticle)
		}
		if rt == ct {
			out = append(out, r)
		}
	}
	return out
}

type filterStats struct {
	OffTopicFiltered int
	UndatedFiltered  int
	PhraseFiltered   int
	PhraseRelaxed    bool
}

func applyPostRankFilters(results []ranker.ScoredResult, req ServiceSearchReq) ([]ranker.ScoredResult, filterStats) {
	var stats filterStats
	if len(results) == 0 {
		return results, stats
	}

	include := domainSet(req.IncludeDomains)
	exclude := domainSet(req.ExcludeDomains)
	dateFrom := parseFilterDate(req.DateFrom, false)
	dateTo := parseFilterDate(req.DateTo, true)
	dateRange := dateFrom != nil || dateTo != nil

	out := make([]ranker.ScoredResult, 0, len(results))
	for _, r := range results {
		domain := urlutil.ExtractDomain(r.URL)

		if len(include) > 0 && !hostInDomainSet(include, domain) {
			continue
		}
		if hostInDomainSet(exclude, domain) {
			continue
		}

		if dateRange {
			t, dated := parsePublishedAny(r.PublishedDate)
			if !dated && req.DateStrict {
				stats.UndatedFiltered++
				continue
			}
			if dated && dateFrom != nil && t.Before(*dateFrom) {
				continue
			}
			if dated && dateTo != nil && t.After(*dateTo) {
				continue
			}
		}

		out = append(out, r)
	}

	if terms := siteScopedTerms(req.Query, include); len(terms) > 0 && len(out) > 0 {
		onTopic := make([]ranker.ScoredResult, 0, len(out))
		for _, r := range out {
			if ranker.CoreTermCoverage(terms, r.Title, r.Snippet, r.URL) > 0 {
				onTopic = append(onTopic, r)
			}
		}
		if len(onTopic) > 0 {
			stats.OffTopicFiltered = len(out) - len(onTopic)
			out = onTopic
		}
	}

	if groups := requiredPhraseGroups(req.Query); len(groups) > 0 && len(out) > 0 {
		matched := make([]ranker.ScoredResult, 0, len(out))
		for _, r := range out {
			if matchesPhraseGroups(r, groups) {
				matched = append(matched, r)
			}
		}
		if len(matched) == 0 {
			stats.PhraseRelaxed = true
		} else {
			stats.PhraseFiltered = len(out) - len(matched)
			out = matched
		}
	}

	if req.SnippetLength > 0 {
		for i := range out {
			out[i].Snippet = truncateRunes(out[i].Snippet, req.SnippetLength)
		}
	}

	return out, stats
}

var queryTokenPattern = regexp.MustCompile(`"([^"]+)"|(\S+)`)

func requiredPhraseGroups(query string) [][]string {
	var groups [][]string
	var current []string
	hasPhrase, joinNext := false, false
	flush := func() {
		if hasPhrase {
			groups = append(groups, current)
		}
		current, hasPhrase = nil, false
	}
	for _, m := range queryTokenPattern.FindAllStringSubmatch(query, -1) {
		phrase, word := strings.Join(strings.Fields(m[1]), " "), m[2]
		if word == "OR" {
			joinNext = len(current) > 0
			continue
		}
		if !joinNext {
			flush()
		}
		joinNext = false
		if phrase != "" {
			current = append(current, strings.ToLower(phrase))
			hasPhrase = true
		} else if word != "" {
			current = append(current, strings.ToLower(word))
		}
	}
	flush()
	return groups
}

func matchesPhraseGroups(r ranker.ScoredResult, groups [][]string) bool {
	haystack := strings.ToLower(strings.Join(strings.Fields(r.Title+" "+r.Snippet+" "+r.URL), " "))
	for _, alternatives := range groups {
		found := false
		for _, term := range alternatives {
			if strings.Contains(haystack, term) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func siteScopedTerms(query string, include map[string]struct{}) []string {
	if len(include) == 0 {
		return nil
	}
	var terms []string
	for _, term := range urlutil.Tokenize(query) {
		named := false
		for domain := range include {
			if strings.Contains(strings.ReplaceAll(domain, ".", ""), term) {
				named = true
				break
			}
		}
		if !named {
			terms = append(terms, term)
		}
	}
	return terms
}

func hostInDomainSet(set map[string]struct{}, host string) bool {
	for host != "" {
		if _, ok := set[host]; ok {
			return true
		}
		_, parent, found := strings.Cut(host, ".")
		if !found {
			return false
		}
		host = parent
	}
	return false
}

func upstreamTimeRange(dateFrom string, now time.Time) string {
	from := parseFilterDate(dateFrom, false)
	if from == nil {
		return ""
	}
	age := now.Sub(*from)
	switch {
	case age < 0:
		return ""
	case age <= 24*time.Hour:
		return "day"
	case age <= 7*24*time.Hour:
		return "week"
	case age <= 31*24*time.Hour:
		return "month"
	case age <= 365*24*time.Hour:
		return "year"
	default:
		return ""
	}
}

var siteDomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$`)

func siteOperator(domains []string) string {
	set := domainSet(domains)
	sites := make([]string, 0, len(set))
	for d := range set {
		if siteDomainPattern.MatchString(d) {
			sites = append(sites, "site:"+d)
		}
	}
	sort.Strings(sites)
	return strings.Join(sites, " OR ")
}

func domainSet(domains []string) map[string]struct{} {
	if len(domains) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		d = strings.TrimPrefix(d, "www.")
		if d != "" {
			out[d] = struct{}{}
		}
	}
	return out
}

func parseFilterDate(s string, endOfDay bool) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Nanosecond)
	}
	return &t
}

func parsePublishedAny(s string) (time.Time, bool) {
	formats := []string{
		time.RFC3339,
		time.RFC1123,
		time.RFC1123Z,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"Jan 2, 2006",
		"January 2, 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return s
	}
	count := 0
	for i := range s {
		if count >= n {
			return strings.TrimRight(s[:i], " ") + "…"
		}
		count++
	}
	return s
}
