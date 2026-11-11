package ranker

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/adriangitvitz/kurai/internal/config"
)

func TestMain(m *testing.M) {
	paths := []string{
		"../../blocklists/content-farms.txt",
		"blocklists/content-farms.txt",
		filepath.Join("..", "..", "blocklists", "content-farms.txt"),
	}

	for _, path := range paths {
		if err := LoadContentFarms(path); err == nil {
			break
		}
	}

	os.Exit(m.Run())
}

func defaultWeights() *config.WeightConfig {
	return &config.WeightConfig{
		TitleMatch:      20,
		URLMatch:        10,
		EngineAgreement: 5,
		TrancoTop10k:    15,
		TrancoTop100k:   10,
		TrancoTop1M:     5,
		Freshness:       5,
	}
}

func defaultPenalties() *config.PenaltyConfig {
	return &config.PenaltyConfig{
		Blocklist:       -50,
		ContentFarm:     -30,
		ExcessiveParams: -10,
		ParamThreshold:  5,
	}
}

func TestTitleMatchBonus(t *testing.T) {
	weights := defaultWeights()

	tests := []struct {
		name     string
		query    []string
		title    string
		expected int
	}{
		{
			name:     "all terms match",
			query:    []string{"rust", "programming"},
			title:    "Rust Programming Guide for Beginners",
			expected: 20,
		},
		{
			name:     "partial match 50%",
			query:    []string{"rust", "python"},
			title:    "Rust Programming Guide",
			expected: 10,
		},
		{
			name:     "partial match 75%",
			query:    []string{"rust", "programming", "guide", "python"},
			title:    "Rust Programming Guide for Beginners",
			expected: 15,
		},
		{
			name:     "partial match 25%",
			query:    []string{"rust", "python", "java", "go"},
			title:    "Rust Programming Guide",
			expected: 5,
		},
		{
			name:     "no match",
			query:    []string{"golang"},
			title:    "Rust Programming Guide",
			expected: 0,
		},
		{
			name:     "case insensitive",
			query:    []string{"rust"},
			title:    "RUST PROGRAMMING",
			expected: 20,
		},
		{
			name:     "empty query",
			query:    []string{},
			title:    "Some Title",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TitleMatchBonus(tt.query, tt.title, weights)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLMatchBonus(t *testing.T) {
	weights := defaultWeights()

	tests := []struct {
		name     string
		query    []string
		url      string
		expected int
	}{
		{
			name:     "path contains term",
			query:    []string{"rust"},
			url:      "https://example.com/rust-tutorial",
			expected: 10,
		},
		{
			name:     "path contains multiple terms",
			query:    []string{"rust", "tutorial"},
			url:      "https://example.com/rust-programming-tutorial",
			expected: 10,
		},
		{
			name:     "no match in path",
			query:    []string{"golang"},
			url:      "https://example.com/rust-tutorial",
			expected: 0,
		},
		{
			name:     "empty path",
			query:    []string{"test"},
			url:      "https://example.com/",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := URLMatchBonus(tt.query, tt.url, weights)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrancoBonus(t *testing.T) {
	weights := defaultWeights()

	tests := []struct {
		name     string
		rank     int
		expected int
	}{
		{
			name:     "top 10k",
			rank:     5000,
			expected: 15,
		},
		{
			name:     "rank 10000 exactly",
			rank:     10000,
			expected: 15,
		},
		{
			name:     "top 100k",
			rank:     50000,
			expected: 10,
		},
		{
			name:     "rank 100000 exactly",
			rank:     100000,
			expected: 10,
		},
		{
			name:     "top 1M",
			rank:     500000,
			expected: 5,
		},
		{
			name:     "rank 1000000 exactly",
			rank:     1000000,
			expected: 5,
		},
		{
			name:     "not ranked",
			rank:     0,
			expected: 0,
		},
		{
			name:     "beyond 1M",
			rank:     2000000,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrancoBonus(tt.rank, weights)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEngineAgreementBonus(t *testing.T) {
	weights := defaultWeights()

	tests := []struct {
		name     string
		engines  []string
		expected int
	}{
		{
			name:     "single engine",
			engines:  []string{"google"},
			expected: 0,
		},
		{
			name:     "two engines",
			engines:  []string{"google", "duckduckgo"},
			expected: 5,
		},
		{
			name:     "three engines",
			engines:  []string{"google", "duckduckgo", "bing"},
			expected: 10,
		},
		{
			name:     "five engines",
			engines:  []string{"google", "duckduckgo", "bing", "brave", "yahoo"},
			expected: 20,
		},
		{
			name:     "no engines",
			engines:  []string{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EngineAgreementBonus(tt.engines, weights)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFreshnessBonus(t *testing.T) {
	weights := defaultWeights()

	tests := []struct {
		name          string
		publishedDate string
		expected      int
	}{
		{
			name:          "recent (today)",
			publishedDate: time.Now().Format("2006-01-02"),
			expected:      5,
		},
		{
			name:          "recent (10 days ago)",
			publishedDate: time.Now().AddDate(0, 0, -10).Format("2006-01-02"),
			expected:      5,
		},
		{
			name:          "old (60 days ago)",
			publishedDate: time.Now().AddDate(0, 0, -60).Format("2006-01-02"),
			expected:      0,
		},
		{
			name:          "empty date",
			publishedDate: "",
			expected:      0,
		},
		{
			name:          "invalid date",
			publishedDate: "not-a-date",
			expected:      0,
		},
		{
			name:          "ISO format",
			publishedDate: time.Now().AddDate(0, 0, -5).Format(time.RFC3339),
			expected:      5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FreshnessBonus(tt.publishedDate, weights)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBlocklistPenalty(t *testing.T) {
	penalties := defaultPenalties()

	tests := []struct {
		name      string
		isBlocked bool
		expected  int
	}{
		{
			name:      "blocked",
			isBlocked: true,
			expected:  -50,
		},
		{
			name:      "not blocked",
			isBlocked: false,
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BlocklistPenalty(tt.isBlocked, penalties)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContentFarmPenalty(t *testing.T) {
	penalties := defaultPenalties()

	tests := []struct {
		name     string
		domain   string
		expected int
	}{
		{
			name:     "known content farm",
			domain:   "about.com",
			expected: -30,
		},
		{
			name:     "another content farm",
			domain:   "buzzfeed.com",
			expected: -30,
		},
		{
			name:     "not a content farm",
			domain:   "github.com",
			expected: 0,
		},
		{
			name:     "empty domain",
			domain:   "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContentFarmPenalty(tt.domain, penalties)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExcessiveParamsPenalty(t *testing.T) {
	penalties := defaultPenalties()

	tests := []struct {
		name     string
		url      string
		expected int
	}{
		{
			name:     "no params",
			url:      "https://example.com/path",
			expected: 0,
		},
		{
			name:     "few params",
			url:      "https://example.com?a=1&b=2",
			expected: 0,
		},
		{
			name:     "exactly threshold",
			url:      "https://example.com?a=1&b=2&c=3&d=4&e=5",
			expected: 0,
		},
		{
			name:     "excessive params",
			url:      "https://example.com?a=1&b=2&c=3&d=4&e=5&f=6",
			expected: -10,
		},
		{
			name:     "tracking params",
			url:      "https://example.com?utm_source=x&utm_medium=y&utm_campaign=z&fbclid=a&gclid=b&ref=c",
			expected: -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExcessiveParamsPenalty(tt.url, penalties)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsContentFarm(t *testing.T) {
	tests := []struct {
		domain   string
		expected bool
	}{
		{"about.com", true},
		{"buzzfeed.com", true},
		{"businessinsider.com", true},
		{"github.com", false},
		{"google.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := IsContentFarm(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func defaultURLStructureConfig() *config.URLStructureConfig {
	return &config.URLStructureConfig{
		DocumentationPath: 10,
		GitHubCode:        8,
		GitHubIssue:       6,
		AggregatorPenalty: -15,
	}
}

func TestURLStructureBonus(t *testing.T) {
	cfg := defaultURLStructureConfig()

	tests := []struct {
		name     string
		url      string
		expected int
	}{
		{
			name:     "docs path",
			url:      "https://example.com/docs/api/reference",
			expected: 10,
		},
		{
			name:     "api path",
			url:      "https://example.com/api/v1/users",
			expected: 10,
		},
		{
			name:     "reference path",
			url:      "https://example.com/reference/configuration",
			expected: 10,
		},
		{
			name:     "guide path",
			url:      "https://example.com/guide/getting-started",
			expected: 10,
		},

		{
			name:     "github blob",
			url:      "https://github.com/golang/go/blob/master/src/crypto/tls/cipher_suites.go",
			expected: 8,
		},
		{
			name:     "github tree",
			url:      "https://github.com/golang/go/tree/master/src/crypto/tls",
			expected: 8,
		},

		{
			name:     "github issue",
			url:      "https://github.com/golang/go/issues/12345",
			expected: 6,
		},
		{
			name:     "github wiki",
			url:      "https://github.com/golang/go/wiki/GoUsers",
			expected: 6,
		},
		{
			name:     "github pull",
			url:      "https://github.com/golang/go/pull/12345",
			expected: 6,
		},

		{
			name:     "search page",
			url:      "https://example.com/search?q=golang",
			expected: -15,
		},
		{
			name:     "tag page",
			url:      "https://example.com/tag/programming",
			expected: -15,
		},
		{
			name:     "category page",
			url:      "https://example.com/category/tech",
			expected: -15,
		},
		{
			name:     "archive page",
			url:      "https://example.com/archive/2024",
			expected: -15,
		},

		{
			name:     "normal blog post",
			url:      "https://example.com/blog/how-to-use-tls",
			expected: 0,
		},
		{
			name:     "normal article",
			url:      "https://example.com/article/golang-features",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := URLStructureBonus(tt.url, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLStructureBonus_InvalidURL(t *testing.T) {
	cfg := defaultURLStructureConfig()

	result := URLStructureBonus("not-a-valid-url", cfg)
	assert.Equal(t, 0, result)
}

func TestAnalyzeTextRelevance(t *testing.T) {
	tests := []struct {
		name        string
		queryTerms  []string
		title       string
		snippet     string
		minCombined float64
		maxCombined float64
		checkTitle  bool
	}{
		{
			name:        "all terms in both title and snippet",
			queryTerms:  []string{"golang", "tls", "handshake"},
			title:       "Golang TLS Handshake Guide",
			snippet:     "Learn how to implement a TLS handshake in Golang with this comprehensive guide.",
			minCombined: 0.7,
			maxCombined: 1.0,
		},
		{
			name:        "terms only in title",
			queryTerms:  []string{"golang", "tls"},
			title:       "Golang TLS Package",
			snippet:     "A comprehensive guide to secure connections.",
			minCombined: 0.3,
			maxCombined: 0.6,
			checkTitle:  true,
		},
		{
			name:        "terms only in snippet",
			queryTerms:  []string{"ConnectionState", "handshake"},
			title:       "tls package - crypto/tls",
			snippet:     "ConnectionState records information about a TLS handshake, including the negotiated protocol version.",
			minCombined: 0.2,
			maxCombined: 0.5,
		},
		{
			name:        "no terms match",
			queryTerms:  []string{"python", "django"},
			title:       "Golang TLS Guide",
			snippet:     "Learn about secure connections in Go.",
			minCombined: 0.0,
			maxCombined: 0.01,
		},
		{
			name:        "empty query terms",
			queryTerms:  []string{},
			title:       "Any Title",
			snippet:     "Any snippet content",
			minCombined: 0.0,
			maxCombined: 0.0,
		},
		{
			name:        "partial match with proximity boost",
			queryTerms:  []string{"tls", "certificate", "verify"},
			title:       "TLS Certificate Guide",
			snippet:     "How to verify your TLS certificate is working correctly.",
			minCombined: 0.5,
			maxCombined: 1.0,
		},
		{
			name:        "case insensitive matching",
			queryTerms:  []string{"TLS", "GOLANG"},
			title:       "golang tls package",
			snippet:     "The Go programming language's tls implementation.",
			minCombined: 0.4,
			maxCombined: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := AnalyzeTextRelevance(tt.queryTerms, tt.title, tt.snippet)

			assert.GreaterOrEqual(t, score.Combined, tt.minCombined,
				"Combined score should be at least %f", tt.minCombined)
			assert.LessOrEqual(t, score.Combined, tt.maxCombined,
				"Combined score should be at most %f", tt.maxCombined)

			assert.GreaterOrEqual(t, score.TitleMatch, 0.0)
			assert.LessOrEqual(t, score.TitleMatch, 1.0)
			assert.GreaterOrEqual(t, score.SnippetMatch, 0.0)
			assert.LessOrEqual(t, score.SnippetMatch, 1.0)
			assert.GreaterOrEqual(t, score.TermFrequency, 0.0)
			assert.LessOrEqual(t, score.TermFrequency, 1.0)
			assert.GreaterOrEqual(t, score.Proximity, 0.0)
			assert.LessOrEqual(t, score.Proximity, 1.0)
			assert.GreaterOrEqual(t, score.CoreTermCoverage, 0.0)
			assert.LessOrEqual(t, score.CoreTermCoverage, 1.0)

			if tt.checkTitle {
				assert.Greater(t, score.TitleMatch, score.SnippetMatch,
					"Title match should be higher when terms only in title")
			}
		})
	}
}

func TestAnalyzeTextRelevance_TitleWeighting(t *testing.T) {
	queryTerms := []string{"golang", "tls"}

	scoreWithTitle := AnalyzeTextRelevance(queryTerms, "Golang TLS Guide", "A programming guide.")
	scoreWithSnippet := AnalyzeTextRelevance(queryTerms, "A programming guide.", "Golang TLS Guide")

	assert.Greater(t, scoreWithTitle.Combined, scoreWithSnippet.Combined,
		"Score with terms in title should be higher than score with terms only in snippet")
}

func TestCalculateTermFrequencyNormalized(t *testing.T) {
	tests := []struct {
		name       string
		queryTerms []string
		text       string
		minScore   float64
		maxScore   float64
	}{
		{
			name:       "no occurrences",
			queryTerms: []string{"golang"},
			text:       "python programming",
			minScore:   0.0,
			maxScore:   0.0,
		},
		{
			name:       "single occurrence",
			queryTerms: []string{"golang"},
			text:       "golang programming",
			minScore:   0.1,
			maxScore:   0.5,
		},
		{
			name:       "multiple occurrences",
			queryTerms: []string{"golang"},
			text:       "golang is great. golang is fast. golang is popular.",
			minScore:   0.3,
			maxScore:   0.8,
		},
		{
			name:       "many occurrences (log scale should cap)",
			queryTerms: []string{"the"},
			text:       "the the the the the the the the the the the the the the the the",
			minScore:   0.5,
			maxScore:   1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateTermFrequencyNormalized(tt.queryTerms, tt.text)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestCalculateProximityNormalized(t *testing.T) {
	tests := []struct {
		name       string
		queryTerms []string
		text       string
		minScore   float64
		maxScore   float64
	}{
		{
			name:       "terms adjacent",
			queryTerms: []string{"tls", "handshake"},
			text:       "the tls handshake process",
			minScore:   0.8,
			maxScore:   1.0,
		},
		{
			name:       "terms close (3-5 words)",
			queryTerms: []string{"tls", "certificate"},
			text:       "the tls protocol uses certificate validation",
			minScore:   0.5,
			maxScore:   1.0,
		},
		{
			name:       "terms far apart",
			queryTerms: []string{"first", "last"},
			text:       "first word then many other words in between until we reach the last word",
			minScore:   0.0,
			maxScore:   0.5,
		},
		{
			name:       "single term (no proximity)",
			queryTerms: []string{"golang"},
			text:       "golang programming language",
			minScore:   0.0,
			maxScore:   0.0,
		},
		{
			name:       "no terms found",
			queryTerms: []string{"python", "django"},
			text:       "golang programming language",
			minScore:   0.0,
			maxScore:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateProximityNormalized(tt.queryTerms, tt.text)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestFilterStopWords(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "removes stop words",
			input:    []string{"capital", "of", "France"},
			expected: []string{"capital", "France"},
		},
		{
			name:     "no stop words",
			input:    []string{"python", "web", "framework"},
			expected: []string{"python", "web", "framework"},
		},
		{
			name:     "all stop words falls back to original",
			input:    []string{"to", "be", "or", "not"},
			expected: []string{"to", "be", "or", "not"},
		},
		{
			name:     "mixed case stop words",
			input:    []string{"The", "capital", "OF", "France"},
			expected: []string{"capital", "France"},
		},
		{
			name:     "single non-stop word",
			input:    []string{"apple"},
			expected: []string{"apple"},
		},
		{
			name:     "single stop word falls back",
			input:    []string{"the"},
			expected: []string{"the"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterStopWords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCoreTermCoverage(t *testing.T) {
	tests := []struct {
		name           string
		queryTerms     []string
		title          string
		snippet        string
		minCoverage    float64
		maxCoverage    float64
		expectLowScore bool
	}{
		{
			name:           "capital of France - irrelevant result missing France",
			queryTerms:     []string{"capital", "of", "france"},
			title:          "Social Capital Theory",
			snippet:        "Understanding social capital in organizational dynamics.",
			minCoverage:    0.4,
			maxCoverage:    0.6,
			expectLowScore: true,
		},
		{
			name:        "capital of France - relevant result",
			queryTerms:  []string{"capital", "of", "france"},
			title:       "Capital of France - Paris",
			snippet:     "Paris is the capital city of France.",
			minCoverage: 1.0,
			maxCoverage: 1.0,
		},
		{
			name:        "best restaurants in new york - full match",
			queryTerms:  []string{"best", "restaurants", "in", "new", "york"},
			title:       "Best Restaurants in New York",
			snippet:     "Top-rated restaurants in New York City.",
			minCoverage: 1.0,
			maxCoverage: 1.0,
		},
		{
			name:           "best restaurants in new york - partial match",
			queryTerms:     []string{"best", "restaurants", "in", "new", "york"},
			title:          "New York City Guide",
			snippet:        "A comprehensive guide to New York attractions.",
			minCoverage:    0.4,
			maxCoverage:    0.6,
			expectLowScore: true,
		},
		{
			name:           "how to train a dog - irrelevant",
			queryTerms:     []string{"how", "to", "train", "a", "dog"},
			title:          "Train Schedule Timetable",
			snippet:        "View the train schedule for your local station.",
			minCoverage:    0.4,
			maxCoverage:    0.6,
			expectLowScore: true,
		},
		{
			name:        "how to train a dog - relevant",
			queryTerms:  []string{"how", "to", "train", "a", "dog"},
			title:       "How to Train Your Dog at Home",
			snippet:     "Learn effective methods to train a dog.",
			minCoverage: 1.0,
			maxCoverage: 1.0,
		},
		{
			name:        "python web framework - full match",
			queryTerms:  []string{"python", "web", "framework"},
			title:       "Python Web Framework Comparison",
			snippet:     "Comparing popular Python web frameworks.",
			minCoverage: 1.0,
			maxCoverage: 1.0,
		},
		{
			name:           "python web framework - missing python",
			queryTerms:     []string{"python", "web", "framework"},
			title:          "Web Framework for Java",
			snippet:        "A powerful web framework built for Java.",
			minCoverage:    0.6,
			maxCoverage:    0.7,
			expectLowScore: true,
		},
		{
			name:        "single term - no stop words",
			queryTerms:  []string{"apple"},
			title:       "Apple Inc. Company Profile",
			snippet:     "Apple designs consumer electronics.",
			minCoverage: 1.0,
			maxCoverage: 1.0,
		},
		{
			name:        "all stop words - falls back to original terms",
			queryTerms:  []string{"to", "be", "or", "not"},
			title:       "To Be or Not to Be - Shakespeare",
			snippet:     "The famous soliloquy from Hamlet.",
			minCoverage: 1.0,
			maxCoverage: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := AnalyzeTextRelevance(tt.queryTerms, tt.title, tt.snippet)

			assert.GreaterOrEqual(t, score.CoreTermCoverage, tt.minCoverage,
				"CoreTermCoverage should be >= %f, got %f", tt.minCoverage, score.CoreTermCoverage)
			assert.LessOrEqual(t, score.CoreTermCoverage, tt.maxCoverage,
				"CoreTermCoverage should be <= %f, got %f", tt.maxCoverage, score.CoreTermCoverage)

			if tt.expectLowScore {
				assert.Less(t, score.Combined, 0.5,
					"Combined score should be low when core terms are missing, got %f", score.Combined)
			}
		})
	}
}

func TestCoreTermCoverage_RelevantVsIrrelevant(t *testing.T) {
	queryTerms := []string{"capital", "of", "france"}

	relevant := AnalyzeTextRelevance(queryTerms, "Capital of France - Paris", "Paris is the capital of France.")
	irrelevant := AnalyzeTextRelevance(queryTerms, "Social Capital Theory", "Understanding social capital dynamics.")

	assert.Greater(t, relevant.Combined, irrelevant.Combined*1.5,
		"Relevant result (%.3f) should score at least 1.5x higher than irrelevant (%.3f)",
		relevant.Combined, irrelevant.Combined)
	assert.Equal(t, 1.0, relevant.CoreTermCoverage)
	assert.Less(t, irrelevant.CoreTermCoverage, 1.0)
}

func TestTokenizeText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "simple words",
			text:     "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "mixed case",
			text:     "Hello World",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with punctuation",
			text:     "hello, world! how are you?",
			expected: []string{"hello", "world", "how", "are", "you"},
		},
		{
			name:     "with numbers",
			text:     "golang 1.21 release",
			expected: []string{"golang", "1", "21", "release"},
		},
		{
			name:     "empty string",
			text:     "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizeText(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}
