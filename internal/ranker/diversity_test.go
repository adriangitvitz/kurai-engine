package ranker

import (
	"testing"
)

func TestDiversityEnforcer_EnforceDiversity(t *testing.T) {
	config := DiversityConfig{
		Enabled:         true,
		MaxPerDomain:    2,
		PenaltyPerExtra: 0.2,
		MinScore:        0.0,
		ExcludedDomains: []string{"excluded.com"},
	}

	de := NewDiversityEnforcer(config)

	results := []ScoredResult{
		{URL: "https://example.com/1", FinalScore: 1.0, AdjustedScore: 1000},
		{URL: "https://example.com/2", FinalScore: 0.9, AdjustedScore: 900},
		{URL: "https://example.com/3", FinalScore: 0.8, AdjustedScore: 800},
		{URL: "https://other.com/1", FinalScore: 0.7, AdjustedScore: 700},
		{URL: "https://excluded.com/1", FinalScore: 0.6, AdjustedScore: 600},
		{URL: "https://excluded.com/2", FinalScore: 0.5, AdjustedScore: 500},
		{URL: "https://excluded.com/3", FinalScore: 0.4, AdjustedScore: 400},
	}

	diversified := de.EnforceDiversity(results)

	if len(diversified) != 7 {
		t.Errorf("EnforceDiversity() got %d results, want 7", len(diversified))
	}

	for _, r := range diversified {
		if r.URL == "https://example.com/3" {
			if r.FinalScore >= 0.8 {
				t.Errorf("Third example.com result should be penalized, got score %f", r.FinalScore)
			}
		}
		if r.URL == "https://excluded.com/3" {
			if r.FinalScore != 0.4 {
				t.Errorf("Excluded domain result should not be penalized, got score %f", r.FinalScore)
			}
		}
	}
}

func TestDiversityEnforcer_EnforceDiversityWithReorder(t *testing.T) {
	config := DefaultDiversityConfig()
	config.MaxPerDomain = 2

	de := NewDiversityEnforcer(config)

	results := []ScoredResult{
		{URL: "https://example.com/1", FinalScore: 1.0, AdjustedScore: 1000},
		{URL: "https://example.com/2", FinalScore: 0.9, AdjustedScore: 900},
		{URL: "https://example.com/3", FinalScore: 0.8, AdjustedScore: 800},
		{URL: "https://other.com/1", FinalScore: 0.7, AdjustedScore: 700},
		{URL: "https://other.com/2", FinalScore: 0.6, AdjustedScore: 600},
	}

	diversified := de.EnforceDiversityWithReorder(results)

	if len(diversified) == 0 {
		t.Error("EnforceDiversityWithReorder() returned empty results")
	}
}

func TestDiversityEnforcer_FilterExcessive(t *testing.T) {
	config := DefaultDiversityConfig()
	de := NewDiversityEnforcer(config)

	results := []ScoredResult{
		{URL: "https://example.com/1", FinalScore: 1.0},
		{URL: "https://example.com/2", FinalScore: 0.9},
		{URL: "https://example.com/3", FinalScore: 0.8},
		{URL: "https://example.com/4", FinalScore: 0.7},
		{URL: "https://example.com/5", FinalScore: 0.6},
		{URL: "https://other.com/1", FinalScore: 0.5},
	}

	filtered := de.FilterExcessive(results, 3)

	exampleCount := 0
	for _, r := range filtered {
		if r.URL[:19] == "https://example.com" {
			exampleCount++
		}
	}

	if exampleCount > 3 {
		t.Errorf("FilterExcessive() allowed %d results from same domain, want max 3", exampleCount)
	}
}

func TestDiversityEnforcer_GetDomainStats(t *testing.T) {
	de := NewDiversityEnforcer(DefaultDiversityConfig())

	results := []ScoredResult{
		{URL: "https://a.com/1"},
		{URL: "https://a.com/2"},
		{URL: "https://b.com/1"},
		{URL: "https://c.com/1"},
	}

	stats := de.GetDomainStats(results)

	if stats.TotalResults != 4 {
		t.Errorf("TotalResults = %d, want 4", stats.TotalResults)
	}
	if stats.UniqueDomains != 3 {
		t.Errorf("UniqueDomains = %d, want 3", stats.UniqueDomains)
	}
	if stats.TopDomain != "a.com" {
		t.Errorf("TopDomain = %s, want a.com", stats.TopDomain)
	}
	if stats.TopDomainCount != 2 {
		t.Errorf("TopDomainCount = %d, want 2", stats.TopDomainCount)
	}
}

func TestDiversityEnforcer_IsExcluded(t *testing.T) {
	config := DiversityConfig{
		Enabled:         true,
		ExcludedDomains: []string{"github.com", "stackoverflow.com"},
	}
	de := NewDiversityEnforcer(config)

	tests := []struct {
		domain string
		want   bool
	}{
		{"github.com", true},
		{"stackoverflow.com", true},
		{"example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			if got := de.IsExcluded(tt.domain); got != tt.want {
				t.Errorf("IsExcluded(%q) = %v, want %v", tt.domain, got, tt.want)
			}
		})
	}
}

func TestDiversityEnforcer_ShouldPenalize(t *testing.T) {
	config := DiversityConfig{
		Enabled:         true,
		MaxPerDomain:    3,
		ExcludedDomains: []string{"excluded.com"},
	}
	de := NewDiversityEnforcer(config)

	tests := []struct {
		domain string
		count  int
		want   bool
	}{
		{"example.com", 0, false},
		{"example.com", 2, false},
		{"example.com", 3, true},
		{"example.com", 5, true},
		{"excluded.com", 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			if got := de.ShouldPenalize(tt.domain, tt.count); got != tt.want {
				t.Errorf("ShouldPenalize(%q, %d) = %v, want %v", tt.domain, tt.count, got, tt.want)
			}
		})
	}
}

func TestDiversityEnforcer_GetPenalty(t *testing.T) {
	config := DiversityConfig{
		Enabled:         true,
		MaxPerDomain:    2,
		PenaltyPerExtra: 0.2,
		ExcludedDomains: []string{"excluded.com"},
	}
	de := NewDiversityEnforcer(config)

	tests := []struct {
		domain   string
		position int
		wantMin  float64
		wantMax  float64
	}{
		{"example.com", 0, 0, 0},
		{"example.com", 1, 0, 0},
		{"example.com", 2, 0.2, 0.3},
		{"example.com", 3, 0.4, 0.5},
		{"excluded.com", 10, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := de.GetPenalty(tt.domain, tt.position)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("GetPenalty(%q, %d) = %f, want between %f and %f",
					tt.domain, tt.position, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestDiversityEnforcer_AddExcludedDomain(t *testing.T) {
	de := NewDiversityEnforcer(DefaultDiversityConfig())

	de.AddExcludedDomain("newexcluded.com")

	if !de.IsExcluded("newexcluded.com") {
		t.Error("AddExcludedDomain() did not add domain to exclusion list")
	}
}

func TestDiversityEnforcer_Disabled(t *testing.T) {
	config := DiversityConfig{
		Enabled: false,
	}
	de := NewDiversityEnforcer(config)

	results := []ScoredResult{
		{URL: "https://example.com/1", FinalScore: 1.0},
		{URL: "https://example.com/2", FinalScore: 0.9},
		{URL: "https://example.com/3", FinalScore: 0.8},
	}

	diversified := de.EnforceDiversity(results)

	if len(diversified) != 3 {
		t.Errorf("EnforceDiversity() with disabled config got %d results, want 3", len(diversified))
	}

	for i, r := range diversified {
		if r.FinalScore != results[i].FinalScore {
			t.Errorf("Result %d score changed when diversity disabled", i)
		}
	}
}
