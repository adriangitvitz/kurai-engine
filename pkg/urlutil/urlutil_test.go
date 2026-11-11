package urlutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple words",
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with punctuation",
			input:    "hello, world! how are you?",
			expected: []string{"hello", "world", "how", "are", "you"},
		},
		{
			name:     "mixed case",
			input:    "Hello World UPPERCASE",
			expected: []string{"hello", "world", "uppercase"},
		},
		{
			name:     "extra whitespace",
			input:    "  hello   world  ",
			expected: []string{"hello", "world"},
		},
		{
			name:     "short words filtered",
			input:    "a to be or not to be",
			expected: []string{"to", "be", "or", "not", "to", "be"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only punctuation",
			input:    "!@#$%",
			expected: []string{},
		},
		{
			name:     "numbers",
			input:    "rust 2024 tutorial",
			expected: []string{"rust", "2024", "tutorial"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Tokenize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTokenizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple path",
			input:    "/blog/post",
			expected: []string{"blog", "post"},
		},
		{
			name:     "with hyphens",
			input:    "/rust-programming-guide",
			expected: []string{"rust", "programming", "guide"},
		},
		{
			name:     "with underscores",
			input:    "/api/user_profile",
			expected: []string{"api", "user", "profile"},
		},
		{
			name:     "with dots",
			input:    "/assets/style.css",
			expected: []string{"assets", "style", "css"},
		},
		{
			name:     "mixed separators",
			input:    "/blog/2024-01/rust_tutorial.html",
			expected: []string{"blog", "2024", "01", "rust", "tutorial", "html"},
		},
		{
			name:     "empty path",
			input:    "",
			expected: []string{},
		},
		{
			name:     "root only",
			input:    "/",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TokenizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple URL",
			input:    "https://example.com/path",
			expected: "example.com",
		},
		{
			name:     "with www",
			input:    "https://www.example.com/path",
			expected: "example.com",
		},
		{
			name:     "with port",
			input:    "https://example.com:8080/path",
			expected: "example.com",
		},
		{
			name:     "subdomain",
			input:    "https://blog.example.com/post",
			expected: "blog.example.com",
		},
		{
			name:     "http",
			input:    "http://example.org",
			expected: "example.org",
		},
		{
			name:     "no scheme",
			input:    "example.com/path",
			expected: "example.com",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "invalid URL",
			input:    "not a url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractDomain(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountURLParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "no params",
			input:    "https://example.com/path",
			expected: 0,
		},
		{
			name:     "one param",
			input:    "https://example.com?q=search",
			expected: 1,
		},
		{
			name:     "multiple params",
			input:    "https://example.com?q=search&page=1&sort=date",
			expected: 3,
		},
		{
			name:     "tracking params",
			input:    "https://example.com?utm_source=google&utm_medium=cpc&utm_campaign=test&fbclid=123",
			expected: 4,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "empty query string",
			input:    "https://example.com?",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountURLParams(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsAllTerms(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		terms    []string
		expected bool
	}{
		{
			name:     "all terms present",
			text:     "rust programming guide for beginners",
			terms:    []string{"rust", "guide"},
			expected: true,
		},
		{
			name:     "some terms missing",
			text:     "rust programming guide",
			terms:    []string{"rust", "python"},
			expected: false,
		},
		{
			name:     "empty terms",
			text:     "some text",
			terms:    []string{},
			expected: true,
		},
		{
			name:     "empty text",
			text:     "",
			terms:    []string{"term"},
			expected: false,
		},
		{
			name:     "case sensitive",
			text:     "rust programming",
			terms:    []string{"rust"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsAllTerms(tt.text, tt.terms)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasTermOverlap(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "overlap exists",
			a:        []string{"rust", "go", "python"},
			b:        []string{"java", "rust"},
			expected: true,
		},
		{
			name:     "no overlap",
			a:        []string{"rust", "go"},
			b:        []string{"java", "python"},
			expected: false,
		},
		{
			name:     "empty first",
			a:        []string{},
			b:        []string{"rust"},
			expected: false,
		},
		{
			name:     "empty second",
			a:        []string{"rust"},
			b:        []string{},
			expected: false,
		},
		{
			name:     "both empty",
			a:        []string{},
			b:        []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasTermOverlap(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkTokenize(b *testing.B) {
	text := "This is a sample search query with multiple words"
	for i := 0; i < b.N; i++ {
		Tokenize(text)
	}
}

func BenchmarkExtractDomain(b *testing.B) {
	url := "https://www.example.com/path/to/page?query=value"
	for i := 0; i < b.N; i++ {
		ExtractDomain(url)
	}
}
