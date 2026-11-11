package ranker

import (
	"testing"
)

func TestAnalyzeQuery_BasicParsing(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantTerms int
	}{
		{"simple query", "hello world", 2},
		{"quoted phrase", `"hello world" test`, 2},
		{"with punctuation", "hello, world!", 2},
		{"empty query", "", 0},
		{"single term", "test", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := AnalyzeQuery(tt.query, nil, nil)
			if len(analysis.Terms) != tt.wantTerms {
				t.Errorf("AnalyzeQuery() got %d terms, want %d", len(analysis.Terms), tt.wantTerms)
			}
		})
	}
}

func TestAnalyzeQuery_QueryTypeClassification(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantType QueryType
	}{
		{"navigational - login", "github login", QueryTypeNavigational},
		{"navigational - domain", "google.com", QueryTypeNavigational},
		{"transactional - buy", "buy laptop", QueryTypeTransactional},
		{"transactional - download", "download python", QueryTypeTransactional},
		{"informational - question", "how to sort array", QueryTypeInformational},
		{"informational - long query", "what is the best way to implement caching in go", QueryTypeInformational},
		{"informational - comparison", "python vs golang", QueryTypeInformational},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := AnalyzeQuery(tt.query, nil, nil)
			if analysis.QueryType != tt.wantType {
				t.Errorf("AnalyzeQuery() query type = %v, want %v", analysis.QueryType, tt.wantType)
			}
		})
	}
}

func TestAnalyzeQuery_PhraseDetection(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantError   bool
		wantVersion bool
		wantQuoted  bool
	}{
		{"error code", "error ERR_CONNECTION_REFUSED", true, false, false},
		{"version number", "python 3.12.1", false, true, false},
		{"quoted text", `"exact phrase" search`, false, false, true},
		{"error with code", "panic SIGSEGV", true, false, false},
		{"version in middle", "upgrade to v2.0.0", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := AnalyzeQuery(tt.query, nil, nil)
			if analysis.HasErrorCode != tt.wantError {
				t.Errorf("HasErrorCode = %v, want %v", analysis.HasErrorCode, tt.wantError)
			}
			if analysis.HasVersion != tt.wantVersion {
				t.Errorf("HasVersion = %v, want %v", analysis.HasVersion, tt.wantVersion)
			}
			if analysis.HasQuotedText != tt.wantQuoted {
				t.Errorf("HasQuotedText = %v, want %v", analysis.HasQuotedText, tt.wantQuoted)
			}
		})
	}
}

func TestAnalyzeQuery_QuestionDetection(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantQuestion bool
	}{
		{"what question", "what is golang", true},
		{"how question", "how to sort array", true},
		{"why question", "why use docker", true},
		{"not a question", "golang tutorial", false},
		{"can question", "can I use react", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := AnalyzeQuery(tt.query, nil, nil)
			if analysis.IsQuestion != tt.wantQuestion {
				t.Errorf("IsQuestion = %v, want %v", analysis.IsQuestion, tt.wantQuestion)
			}
		})
	}
}

func TestAnalyzeQuery_ComparisonDetection(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		wantComparison bool
	}{
		{"vs comparison", "python vs golang", true},
		{"versus comparison", "react versus vue", true},
		{"or comparison", "redis or memcached", true},
		{"better comparison", "which is better react", true},
		{"no comparison", "golang tutorial", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := AnalyzeQuery(tt.query, nil, nil)
			if analysis.IsComparison != tt.wantComparison {
				t.Errorf("IsComparison = %v, want %v", analysis.IsComparison, tt.wantComparison)
			}
		})
	}
}

func TestAnalyzeQuery_MultiIntentDetection(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantMinCount int
	}{
		{"vs query", "python vs golang performance", 2},
		{"simple query", "golang tutorial", 1},
		{"comparison", "react vs vue", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := AnalyzeQuery(tt.query, nil, nil)
			if len(analysis.MultiIntent) < tt.wantMinCount {
				t.Errorf("MultiIntent count = %d, want at least %d", len(analysis.MultiIntent), tt.wantMinCount)
			}
		})
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"  multiple   spaces  ", "multiple spaces"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeQuery(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeQuery(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTokenizeQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"hello world", []string{"hello", "world"}},
		{`"exact phrase" other`, []string{"exact phrase", "other"}},
		{"c++ programming", []string{"c++", "programming"}},
		{"test-driven development", []string{"test-driven", "development"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := tokenizeQuery(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("tokenizeQuery(%q) = %v, want %v", tt.input, got, tt.expected)
				return
			}
			for i, term := range got {
				if term != tt.expected[i] {
					t.Errorf("tokenizeQuery(%q)[%d] = %q, want %q", tt.input, i, term, tt.expected[i])
				}
			}
		})
	}
}

func TestQueryAnalysis_GetPrimaryCategory(t *testing.T) {
	analysis := &QueryAnalysis{
		MultiIntent: []IntentComponent{
			{Category: CategoryTech, Confidence: 0.8},
			{Category: CategoryGeneral, Confidence: 0.3},
		},
	}

	got := analysis.GetPrimaryCategory()
	if got != CategoryTech {
		t.Errorf("GetPrimaryCategory() = %v, want %v", got, CategoryTech)
	}
}

func TestQueryAnalysis_EffectiveTerms(t *testing.T) {
	analysis := &QueryAnalysis{
		Terms:         []string{"api", "error"},
		ExpandedTerms: []string{"endpoint", "exception"},
	}

	got := analysis.EffectiveTerms()
	if len(got) != 4 {
		t.Errorf("EffectiveTerms() got %d terms, want 4", len(got))
	}

	seen := make(map[string]bool)
	for _, term := range got {
		if seen[term] {
			t.Errorf("EffectiveTerms() has duplicate term: %s", term)
		}
		seen[term] = true
	}
}

func TestAnalyzeQuery_ApostrophesAreNotQuotedText(t *testing.T) {
	analysis := AnalyzeQuery("why don't banks publish the company's HSM vendor", nil, nil)
	if analysis.HasQuotedText {
		t.Fatalf("apostrophes must not count as quoted text")
	}
	if !AnalyzeQuery(`"northern trust" hsm`, nil, nil).HasQuotedText {
		t.Fatalf("double quotes must count as quoted text")
	}
}
