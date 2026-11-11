package ranker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adriangitvitz/kurai/internal/config"
)

func defaultSnippetConfig() *config.SnippetConfig {
	return &config.SnippetConfig{
		PartialMatchMax:    20,
		TermFrequencyMax:   15,
		ProximityBonus:     5,
		CodeDetectionBonus: 10,
		LowQualityPenalty:  -10,
		HighQualityBonus:   8,
	}
}

func TestAnalyzeSnippet_PartialMatch(t *testing.T) {
	cfg := defaultSnippetConfig()

	tests := []struct {
		name               string
		queryTerms         []string
		snippet            string
		expectedMinPartial int
	}{
		{
			name:               "all terms match",
			queryTerms:         []string{"go", "tls", "cipher"},
			snippet:            "The Go programming language supports TLS cipher suites",
			expectedMinPartial: 18,
		},
		{
			name:               "half terms match",
			queryTerms:         []string{"go", "tls", "python", "ruby"},
			snippet:            "The Go programming language supports TLS",
			expectedMinPartial: 8,
		},
		{
			name:               "no terms match",
			queryTerms:         []string{"python", "django"},
			snippet:            "The Go programming language supports TLS",
			expectedMinPartial: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := AnalyzeSnippet(tt.queryTerms, tt.snippet, cfg)
			assert.GreaterOrEqual(t, score.PartialMatch, tt.expectedMinPartial)
		})
	}
}

func TestAnalyzeSnippet_TermFrequency(t *testing.T) {
	cfg := defaultSnippetConfig()

	tests := []struct {
		name                 string
		queryTerms           []string
		snippet              string
		expectedMinFrequency int
	}{
		{
			name:                 "multiple occurrences",
			queryTerms:           []string{"tls"},
			snippet:              "TLS cipher suites. TLS configuration. TLS certificates. TLS handshake.",
			expectedMinFrequency: 5,
		},
		{
			name:                 "single occurrence",
			queryTerms:           []string{"tls"},
			snippet:              "TLS cipher suites documentation",
			expectedMinFrequency: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := AnalyzeSnippet(tt.queryTerms, tt.snippet, cfg)
			assert.GreaterOrEqual(t, score.TermFrequency, tt.expectedMinFrequency)
		})
	}
}

func TestAnalyzeSnippet_Proximity(t *testing.T) {
	cfg := defaultSnippetConfig()

	tests := []struct {
		name              string
		queryTerms        []string
		snippet           string
		expectedProximity int
	}{
		{
			name:              "terms adjacent",
			queryTerms:        []string{"go", "tls"},
			snippet:           "The Go TLS package provides secure communication",
			expectedProximity: 5,
		},
		{
			name:              "terms far apart",
			queryTerms:        []string{"go", "tls"},
			snippet:           "Go is a programming language that was created at Google and it has many features including support for network programming and TLS",
			expectedProximity: 0,
		},
		{
			name:              "single term",
			queryTerms:        []string{"go"},
			snippet:           "Go programming language",
			expectedProximity: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := AnalyzeSnippet(tt.queryTerms, tt.snippet, cfg)
			assert.Equal(t, tt.expectedProximity, score.Proximity)
		})
	}
}

func TestAnalyzeSnippet_CodeDetection(t *testing.T) {
	cfg := defaultSnippetConfig()

	tests := []struct {
		name            string
		snippet         string
		expectedMinCode int
	}{
		{
			name:            "code with backticks and function calls",
			snippet:         "Use `tls.Config{}` to configure TLS. Call conn.Handshake() to initiate.",
			expectedMinCode: 5,
		},
		{
			name:            "code with keywords",
			snippet:         "func main() { if err != nil { return } }",
			expectedMinCode: 5,
		},
		{
			name:            "no code patterns",
			snippet:         "TLS provides secure communication over networks.",
			expectedMinCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := AnalyzeSnippet([]string{"tls"}, tt.snippet, cfg)
			assert.GreaterOrEqual(t, score.CodeDetection, tt.expectedMinCode)
		})
	}
}

func TestAnalyzeSnippet_QualityModifier(t *testing.T) {
	cfg := defaultSnippetConfig()

	tests := []struct {
		name        string
		snippet     string
		expectedMin int
		expectedMax int
	}{
		{
			name:        "high quality technical",
			snippet:     "Documentation reference for the API. See the specification for parameters and return values.",
			expectedMin: 1,
			expectedMax: 10,
		},
		{
			name:        "low quality clickbait",
			snippet:     "You won't believe these amazing tricks! Click here to learn more. 10 tips that will shock you!",
			expectedMin: -15,
			expectedMax: -1,
		},
		{
			name:        "neutral content",
			snippet:     "This is a regular article about programming concepts.",
			expectedMin: -5,
			expectedMax: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := AnalyzeSnippet([]string{"programming"}, tt.snippet, cfg)
			assert.GreaterOrEqual(t, score.QualityModifier, tt.expectedMin)
			assert.LessOrEqual(t, score.QualityModifier, tt.expectedMax)
		})
	}
}

func TestAnalyzeSnippet_EmptyInputs(t *testing.T) {
	cfg := defaultSnippetConfig()

	t.Run("empty snippet", func(t *testing.T) {
		score := AnalyzeSnippet([]string{"test"}, "", cfg)
		assert.Equal(t, 0, score.Total)
	})

	t.Run("empty query terms", func(t *testing.T) {
		score := AnalyzeSnippet([]string{}, "some snippet", cfg)
		assert.Equal(t, 0, score.PartialMatch)
		assert.Equal(t, 0, score.TermFrequency)
	})
}

func TestAnalyzeSnippet_RealWorldExamples(t *testing.T) {
	cfg := defaultSnippetConfig()

	tests := []struct {
		name        string
		queryTerms  []string
		snippet     string
		minExpected int
		maxExpected int
	}{
		{
			name:        "pkg.go.dev TLS snippet",
			queryTerms:  []string{"go", "tls", "cipher", "suites"},
			snippet:     "Package tls partially implements TLS 1.2, as specified in RFC 5246, and TLS 1.3, as specified in RFC 8446. func CipherSuites() []*CipherSuite",
			minExpected: 20,
			maxExpected: 60,
		},
		{
			name:        "StackOverflow answer",
			queryTerms:  []string{"go", "error", "nil"},
			snippet:     "if err != nil { return err } This is the standard Go error handling pattern. func main() { if err := doSomething(); err != nil { log.Fatal(err) } }",
			minExpected: 15,
			maxExpected: 50,
		},
		{
			name:        "Wikipedia article",
			queryTerms:  []string{"gustave", "le", "bon", "crowd"},
			snippet:     "Gustave Le Bon was a French polymath whose areas of interest included anthropology, psychology, sociology, and physics. He is best known for his book The Crowd: A Study of the Popular Mind.",
			minExpected: 15,
			maxExpected: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := AnalyzeSnippet(tt.queryTerms, tt.snippet, cfg)
			assert.GreaterOrEqual(t, score.Total, tt.minExpected, "Total score too low")
			assert.LessOrEqual(t, score.Total, tt.maxExpected, "Total score too high")
		})
	}
}
