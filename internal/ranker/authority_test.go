package ranker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adriangitvitz/kurai/internal/config"
)

func defaultAuthorityConfig() *config.AuthorityConfig {
	return &config.AuthorityConfig{
		OfficialDocs:      25,
		CodeRepository:    15,
		DeveloperQA:       12,
		QualityBlog:       8,
		PaperArchive:      25,
		AcademicPublisher: 20,
		Wikipedia:         15,
		MajorNews:         15,
		QualityNews:       12,
		QueryDomainMatch:  30,
	}
}

func TestGetAuthorityCategory(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected AuthorityCategory
	}{
		{name: "pkg.go.dev", domain: "pkg.go.dev", expected: CategoryNone},
		{name: "docs.python.org", domain: "docs.python.org", expected: CategoryNone},
		{name: "MDN", domain: "developer.mozilla.org", expected: CategoryNone},
		{name: "kubernetes.io", domain: "kubernetes.io", expected: CategoryNone},
		{name: "github.com", domain: "github.com", expected: CategoryNone},
		{name: "gitlab.com", domain: "gitlab.com", expected: CategoryNone},
		{name: "stackoverflow", domain: "stackoverflow.com", expected: CategoryNone},
		{name: "superuser", domain: "superuser.com", expected: CategoryNone},
		{name: "arxiv", domain: "arxiv.org", expected: CategoryNone},
		{name: "scholar.google", domain: "scholar.google.com", expected: CategoryNone},
		{name: "nature", domain: "nature.com", expected: CategoryNone},
		{name: "IEEE", domain: "ieeexplore.ieee.org", expected: CategoryNone},
		{name: "wikipedia", domain: "wikipedia.org", expected: CategoryNone},
		{name: "en.wikipedia", domain: "en.wikipedia.org", expected: CategoryNone},
		{name: "reuters", domain: "reuters.com", expected: CategoryNone},
		{name: "nytimes", domain: "nytimes.com", expected: CategoryNone},
		{name: "unknown", domain: "example.com", expected: CategoryNone},
		{name: "random blog", domain: "myblog.com", expected: CategoryNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAuthorityCategory(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAuthorityCategory_Subdomains(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected AuthorityCategory
	}{
		{name: "subdomain of github", domain: "gist.github.com", expected: CategoryNone},
		{name: "subdomain of wikipedia", domain: "de.wikipedia.org", expected: CategoryNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAuthorityCategory(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthorityBonus(t *testing.T) {
	cfg := defaultAuthorityConfig()

	tests := []struct {
		name     string
		domain   string
		expected int
	}{
		{name: "official docs", domain: "pkg.go.dev", expected: 0},
		{name: "github", domain: "github.com", expected: 0},
		{name: "stackoverflow", domain: "stackoverflow.com", expected: 0},
		{name: "arxiv", domain: "arxiv.org", expected: 0},
		{name: "nature", domain: "nature.com", expected: 0},
		{name: "wikipedia", domain: "wikipedia.org", expected: 0},
		{name: "reuters", domain: "reuters.com", expected: 0},
		{name: "nytimes", domain: "nytimes.com", expected: 0},
		{name: "unknown", domain: "example.com", expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AuthorityBonus(tt.domain, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryDomainMatchBonus(t *testing.T) {
	cfg := defaultAuthorityConfig()

	tests := []struct {
		name       string
		queryTerms []string
		domain     string
		expected   int
	}{
		{
			name:       "arxiv in query matches arxiv.org",
			queryTerms: []string{"arxiv", "machine", "learning"},
			domain:     "arxiv.org",
			expected:   0,
		},
		{
			name:       "github in query matches github.com",
			queryTerms: []string{"github", "golang", "tls"},
			domain:     "github.com",
			expected:   0,
		},
		{
			name:       "wikipedia in query matches wikipedia.org",
			queryTerms: []string{"wikipedia", "gustave", "le", "bon"},
			domain:     "en.wikipedia.org",
			expected:   0,
		},
		{
			name:       "wiki shorthand matches wikipedia",
			queryTerms: []string{"wiki", "crowd", "psychology"},
			domain:     "wikipedia.org",
			expected:   0,
		},
		{
			name:       "stackoverflow in query",
			queryTerms: []string{"stackoverflow", "go", "error"},
			domain:     "stackoverflow.com",
			expected:   0,
		},
		{
			name:       "no match - different domain",
			queryTerms: []string{"arxiv", "machine", "learning"},
			domain:     "github.com",
			expected:   0,
		},
		{
			name:       "no domain keyword in query",
			queryTerms: []string{"go", "tls", "cipher", "suites"},
			domain:     "pkg.go.dev",
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QueryDomainMatchBonus(tt.queryTerms, tt.domain, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthorityBonusWithIntent(t *testing.T) {
	cfg := defaultAuthorityConfig()

	tests := []struct {
		name   string
		domain string
		intent QueryCategory
	}{
		{name: "tech query github", domain: "github.com", intent: CategoryTech},
		{name: "academic query arxiv", domain: "arxiv.org", intent: CategoryAcademic},
		{name: "news query reuters", domain: "reuters.com", intent: CategoryNews},
		{name: "general query wikipedia", domain: "wikipedia.org", intent: CategoryGeneral},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AuthorityBonusWithIntent(tt.domain, tt.intent, cfg)
			assert.Equal(t, 0, result)
		})
	}
}
