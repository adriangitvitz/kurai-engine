package ranker

import (
	"math"
	"time"
)

const FreshnessSentinel = -1.0

func FreshnessScore(publishedDate string, category QueryCategory) float64 {
	if publishedDate == "" {
		return FreshnessSentinel
	}
	t, ok := parseFlexibleDate(publishedDate)
	if !ok {
		return FreshnessSentinel
	}

	age := time.Since(t)
	if age < 0 {
		age = 0
	}
	hl := halfLifeFor(category)
	return math.Pow(0.5, age.Hours()/hl.Hours())
}

func halfLifeFor(category QueryCategory) time.Duration {
	const day = 24 * time.Hour
	switch category {
	case CategoryNews:
		return 24 * time.Hour
	case CategoryAcademic:
		return 5 * 365 * day
	default:
		return 365 * day
	}
}

func parseFlexibleDate(s string) (time.Time, bool) {
	formats := []string{
		time.RFC3339,
		time.RFC1123,
		time.RFC1123Z,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"Jan 2, 2006",
		"January 2, 2006",
		"02 Jan 2006",
		"2 Jan 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
