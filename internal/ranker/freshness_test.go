package ranker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFreshnessScore_NewsHalfLife24h(t *testing.T) {
	now := time.Now().UTC()
	cases := []struct {
		age  time.Duration
		want float64
	}{
		{0, 1.0},
		{24 * time.Hour, 0.5},
		{48 * time.Hour, 0.25},
		{72 * time.Hour, 0.125},
	}
	for _, c := range cases {
		got := FreshnessScore(now.Add(-c.age).Format(time.RFC3339), CategoryNews)
		assert.InDelta(t, c.want, got, 0.05, "age=%v", c.age)
	}
}

func TestFreshnessScore_GeneralHalfLifeOneYear(t *testing.T) {
	now := time.Now().UTC()
	score24h := FreshnessScore(now.Add(-24*time.Hour).Format(time.RFC3339), CategoryGeneral)
	assert.Greater(t, score24h, 0.99, "general intent: 24h-old doc should still be ~1.0")
	score180d := FreshnessScore(now.Add(-180*24*time.Hour).Format(time.RFC3339), CategoryGeneral)
	assert.InDelta(t, 0.71, score180d, 0.05, "general intent: 180d-old doc should be ~0.71")
}

func TestFreshnessScore_AcademicEffectivelyNoDecay(t *testing.T) {
	now := time.Now().UTC()
	score := FreshnessScore(now.Add(-365*24*time.Hour).Format(time.RFC3339), CategoryAcademic)
	assert.Greater(t, score, 0.85, "academic intent: 1y-old paper still highly fresh")
}

func TestFreshnessScore_SentinelOnUnparseable(t *testing.T) {
	got := FreshnessScore("garbage-not-a-date", CategoryNews)
	assert.Equal(t, FreshnessSentinel, got, "unparseable dates must return sentinel for skip-in-RRF")
}

func TestFreshnessScore_SentinelOnEmpty(t *testing.T) {
	assert.Equal(t, FreshnessSentinel, FreshnessScore("", CategoryNews))
}

func TestFreshnessScore_FutureDateClamps(t *testing.T) {
	now := time.Now().UTC()
	got := FreshnessScore(now.Add(48*time.Hour).Format(time.RFC3339), CategoryNews)
	assert.InDelta(t, 1.0, got, 1e-6, "future-dated junk should not exceed 1.0")
}

func TestFreshnessScore_ParsesMultipleFormats(t *testing.T) {
	now := time.Now().UTC()
	formats := []string{
		now.Format(time.RFC3339),
		now.Format("2006-01-02"),
		now.Format("Jan 2, 2006"),
		now.Format("2006-01-02T15:04:05"),
	}
	for _, f := range formats {
		got := FreshnessScore(f, CategoryNews)
		assert.Greater(t, got, 0.5, "format=%q must parse", f)
	}
}
