package ranker

import (
	"testing"
)

func TestSynonymExpander_LoadDefaults(t *testing.T) {
	se := NewSynonymExpander()
	se.LoadDefaults()

	stats := se.Stats()
	if stats["synonym_terms"] == 0 {
		t.Error("No synonym terms loaded")
	}
	if stats["domains"] == 0 {
		t.Error("No domains loaded")
	}
	if stats["acronyms"] == 0 {
		t.Error("No acronyms loaded")
	}
}

func TestSynonymExpander_Expand(t *testing.T) {
	se := NewSynonymExpander()
	se.LoadDefaults()

	tests := []struct {
		name       string
		terms      []string
		maxExpand  int
		wantMinLen int
	}{
		{"error synonyms", []string{"error"}, 5, 1},
		{"api synonyms", []string{"api"}, 5, 1},
		{"debug synonyms", []string{"debug"}, 5, 1},
		{"unknown term", []string{"xyzabc123"}, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded := se.Expand(tt.terms, tt.maxExpand)
			if len(expanded) < tt.wantMinLen {
				t.Errorf("Expand(%v) got %d terms, want at least %d", tt.terms, len(expanded), tt.wantMinLen)
			}
		})
	}
}

func TestSynonymExpander_ExpandAcronym(t *testing.T) {
	se := NewSynonymExpander()
	se.LoadDefaults()

	tests := []struct {
		acronym  string
		wantPart string
	}{
		{"api", "application programming interface"},
		{"sdk", "software development kit"},
		{"llm", "large language model"},
		{"sql", "structured query language"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.acronym, func(t *testing.T) {
			got := se.ExpandAcronym(tt.acronym)
			if got != tt.wantPart {
				t.Errorf("ExpandAcronym(%q) = %q, want %q", tt.acronym, got, tt.wantPart)
			}
		})
	}
}

func TestSynonymExpander_GetSynonyms(t *testing.T) {
	se := NewSynonymExpander()
	se.LoadDefaults()

	tests := []struct {
		term    string
		wantMin int
	}{
		{"error", 1},
		{"function", 1},
		{"deploy", 1},
		{"unknownterm123", 0},
	}

	for _, tt := range tests {
		t.Run(tt.term, func(t *testing.T) {
			synonyms := se.GetSynonyms(tt.term)
			if len(synonyms) < tt.wantMin {
				t.Errorf("GetSynonyms(%q) got %d, want at least %d", tt.term, len(synonyms), tt.wantMin)
			}
		})
	}
}

func TestSynonymExpander_IsAcronym(t *testing.T) {
	se := NewSynonymExpander()
	se.LoadDefaults()

	tests := []struct {
		term string
		want bool
	}{
		{"api", true},
		{"sdk", true},
		{"http", true},
		{"notanacronym", false},
		{"xyz123", false},
	}

	for _, tt := range tests {
		t.Run(tt.term, func(t *testing.T) {
			if got := se.IsAcronym(tt.term); got != tt.want {
				t.Errorf("IsAcronym(%q) = %v, want %v", tt.term, got, tt.want)
			}
		})
	}
}

func TestSynonymExpander_ExpandWithDomain(t *testing.T) {
	se := NewSynonymExpander()
	se.LoadDefaults()

	terms := []string{"error", "function"}
	expanded := se.ExpandWithDomain(terms, "programming", 10)

	if len(expanded) == 0 {
		t.Error("ExpandWithDomain() returned empty result")
	}
}

func TestSynonymExpander_AddSynonym(t *testing.T) {
	se := NewSynonymExpander()

	se.AddSynonym("test", "custom", []string{"synonym1", "synonym2"})

	synonyms := se.GetSynonyms("custom")
	if len(synonyms) != 2 {
		t.Errorf("GetSynonyms() got %d, want 2", len(synonyms))
	}
}

func TestSynonymExpander_AddAcronym(t *testing.T) {
	se := NewSynonymExpander()

	se.AddAcronym("xyz", "custom expansion")

	got := se.ExpandAcronym("xyz")
	if got != "custom expansion" {
		t.Errorf("ExpandAcronym(xyz) = %q, want %q", got, "custom expansion")
	}
}

func TestSynonymExpander_MaxExpansions(t *testing.T) {
	se := NewSynonymExpander()
	se.LoadDefaults()

	se.SetMaxExpansions(1)

	expanded := se.Expand([]string{"error"}, 10)

	if len(expanded) > 5 {
		t.Errorf("Expand() returned too many results despite limit: %d", len(expanded))
	}
}

func TestSynonymExpander_LoadFromYAML(t *testing.T) {
	yamlData := []byte(`
test_domain:
  test_term:
    - synonym1
    - synonym2
`)

	se := NewSynonymExpander()
	err := se.LoadSynonymsFromYAML(yamlData)
	if err != nil {
		t.Fatalf("LoadSynonymsFromYAML() error = %v", err)
	}

	synonyms := se.GetSynonyms("test_term")
	if len(synonyms) != 2 {
		t.Errorf("GetSynonyms() got %d, want 2", len(synonyms))
	}
}

func TestSynonymExpander_NoDuplicates(t *testing.T) {
	se := NewSynonymExpander()
	se.LoadDefaults()

	terms := []string{"error", "exception"}
	expanded := se.Expand(terms, 20)

	seen := make(map[string]bool)
	for _, term := range expanded {
		if seen[term] {
			t.Errorf("Expand() returned duplicate term: %s", term)
		}
		seen[term] = true
	}
}
