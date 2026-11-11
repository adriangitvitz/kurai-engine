package server

import (
	"regexp"
	"strconv"
	"time"

	"github.com/adriangitvitz/kurai/internal/searxng"
)

var (
	snippetAbsoluteDate = regexp.MustCompile(`^\s*([A-Z][a-z]{2,8})\.? (\d{1,2}), (\d{4})\s*(?:·|-|—|–|\|)\s`)
	snippetRelativeDate = regexp.MustCompile(`^\s*(\d{1,3}) (hour|day|week|month|year)s? ago\s*(?:·|-|—|–|\|)\s`)
)

var relativeUnit = map[string]time.Duration{
	"hour":  time.Hour,
	"day":   24 * time.Hour,
	"week":  7 * 24 * time.Hour,
	"month": 30 * 24 * time.Hour,
	"year":  365 * 24 * time.Hour,
}

func snippetDate(snippet string, now time.Time) (time.Time, bool) {
	if m := snippetAbsoluteDate.FindStringSubmatch(snippet); m != nil {
		for _, layout := range []string{"Jan 2, 2006", "January 2, 2006"} {
			if t, err := time.Parse(layout, m[1]+" "+m[2]+", "+m[3]); err == nil {
				return t, true
			}
		}
		return time.Time{}, false
	}
	if m := snippetRelativeDate.FindStringSubmatch(snippet); m != nil {
		n, _ := strconv.Atoi(m[1])
		return now.Add(-time.Duration(n) * relativeUnit[m[2]]), true
	}
	return time.Time{}, false
}

func inferPublishedDates(results []searxng.Result, now time.Time) {
	for i := range results {
		if results[i].PublishedDate != "" {
			continue
		}
		if t, ok := snippetDate(results[i].Content, now); ok {
			results[i].PublishedDate = t.Format("2006-01-02")
		}
	}
}
