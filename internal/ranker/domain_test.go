package ranker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainScores_PrimaryDomain(t *testing.T) {
	tests := []struct {
		name     string
		scores   DomainScores
		expected Domain
	}{
		{
			name:     "empty scores",
			scores:   DomainScores{},
			expected: DomainGeneral,
		},
		{
			name: "single domain",
			scores: DomainScores{
				DomainProgramming: 0.8,
			},
			expected: DomainProgramming,
		},
		{
			name: "multiple domains",
			scores: DomainScores{
				DomainProgramming: 0.3,
				DomainNews:        0.5,
				DomainAcademic:    0.2,
			},
			expected: DomainNews,
		},
		{
			name: "tie goes to first encountered (map iteration)",
			scores: DomainScores{
				DomainProgramming: 0.5,
				DomainNews:        0.5,
			},
			expected: DomainProgramming,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.scores.PrimaryDomain()
			if tt.name == "tie goes to first encountered (map iteration)" {
				assert.True(t, result == DomainProgramming || result == DomainNews)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDomainScores_Confidence(t *testing.T) {
	scores := DomainScores{
		DomainProgramming: 0.7,
		DomainNews:        0.3,
	}

	confidence := scores.Confidence()
	assert.Equal(t, 0.7, confidence)
}

func TestDomainScores_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		scores   DomainScores
		expected DomainScores
	}{
		{
			name:     "empty",
			scores:   DomainScores{},
			expected: DomainScores{},
		},
		{
			name: "already normalized",
			scores: DomainScores{
				DomainProgramming: 0.5,
				DomainNews:        0.5,
			},
			expected: DomainScores{
				DomainProgramming: 0.5,
				DomainNews:        0.5,
			},
		},
		{
			name: "needs normalization",
			scores: DomainScores{
				DomainProgramming: 6.0,
				DomainNews:        4.0,
			},
			expected: DomainScores{
				DomainProgramming: 0.6,
				DomainNews:        0.4,
			},
		},
		{
			name: "all zeros",
			scores: DomainScores{
				DomainProgramming: 0.0,
				DomainNews:        0.0,
			},
			expected: DomainScores{
				DomainProgramming: 0.0,
				DomainNews:        0.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.scores.Normalize()
			for domain, expectedScore := range tt.expected {
				assert.InDelta(t, expectedScore, result[domain], 0.0001)
			}
		})
	}
}

func TestClassifyDomain_Programming(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		snippet string
		url     string
	}{
		{
			name:    "REST API tutorial",
			title:   "How to implement a REST API in Go",
			snippet: "This tutorial shows you how to build a REST API using the net/http package with proper error handling and JSON responses.",
			url:     "https://example.com/tutorials/go-rest-api",
		},
		{
			name:    "GitHub repository",
			title:   "kubernetes/kubernetes: Production-Grade Container Scheduling",
			snippet: "Production-Grade Container Scheduling and Management. Kubernetes is an open-source container orchestration system.",
			url:     "https://github.com/kubernetes/kubernetes",
		},
		{
			name:    "Stack Overflow question",
			title:   "How to fix nil pointer dereference in Go",
			snippet: "I'm getting a panic: runtime error: invalid memory address or nil pointer dereference. Here's my code with the function that causes the error.",
			url:     "https://stackoverflow.com/questions/123456/nil-pointer-go",
		},
		{
			name:    "Code documentation",
			title:   "Package http - Go Documentation",
			snippet: "Package http provides HTTP client and server implementations. func Get(url string) (resp *Response, err error)",
			url:     "https://pkg.go.dev/net/http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores := ClassifyDomain(tt.title, tt.snippet, tt.url)
			primary := scores.PrimaryDomain()
			assert.Equal(t, DomainProgramming, primary,
				"Expected programming domain, got %s (scores: %v)", primary, scores)
		})
	}
}

func TestClassifyDomain_News(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		snippet string
		url     string
	}{
		{
			name:    "Breaking news",
			title:   "Breaking: New Economic Policy Announced by Government",
			snippet: "Officials reported today that the government has announced sweeping economic reforms. According to sources, the president will address the nation.",
			url:     "https://news.example.com/politics/2024/economic-policy",
		},
		{
			name:    "Election coverage",
			title:   "Election Results: Democrats Win Key Senate Seats",
			snippet: "Latest update: The election results confirm Democrats have secured crucial victories. Political analysts say this developing story could reshape Congress.",
			url:     "https://example.com/news/election-2024",
		},
		{
			name:    "Business news",
			title:   "Tech Giant Announces Major Layoffs",
			snippet: "According to sources, the company confirmed today that it will cut 10,000 jobs. A spokesperson said the restructuring aims to reduce costs.",
			url:     "https://example.com/business/tech-layoffs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores := ClassifyDomain(tt.title, tt.snippet, tt.url)
			primary := scores.PrimaryDomain()
			assert.Equal(t, DomainNews, primary,
				"Expected news domain, got %s (scores: %v)", primary, scores)
		})
	}
}

func TestClassifyDomain_Academic(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		snippet string
		url     string
	}{
		{
			name:    "Research paper",
			title:   "A Study of Machine Learning in Healthcare: Methodology and Results",
			snippet: "Abstract: This peer-reviewed publication presents findings from our research. The study included 500 participants (n=500). Results show statistical significance (p<0.05).",
			url:     "https://arxiv.org/abs/2024.12345",
		},
		{
			name:    "Journal article",
			title:   "Correlation Between Exercise and Mental Health: A Meta-Analysis",
			snippet: "Our methodology involved analyzing 50 studies. Smith et al. found significant correlation (r=0.65). DOI: 10.1234/example",
			url:     "https://www.nature.com/articles/s12345-024-00001",
		},
		{
			name:    "University research",
			title:   "PhD Thesis: Neural Network Applications in Linguistics",
			snippet: "This dissertation presents a comprehensive analysis of deep learning models. The hypothesis was tested using randomized controlled experiments.",
			url:     "https://www.stanford.edu/thesis/neural-linguistics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores := ClassifyDomain(tt.title, tt.snippet, tt.url)
			primary := scores.PrimaryDomain()
			assert.Equal(t, DomainAcademic, primary,
				"Expected academic domain, got %s (scores: %v)", primary, scores)
		})
	}
}

func TestClassifyDomain_Psychology(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		snippet string
		url     string
	}{
		{
			name:    "CBT therapy",
			title:   "Cognitive Behavioral Therapy for Anxiety Disorders",
			snippet: "CBT is an evidence-based psychotherapy treatment for anxiety and depression. This clinical approach focuses on cognitive patterns and behavioral responses.",
			url:     "https://psychology.org/treatments/cbt-anxiety",
		},
		{
			name:    "Mental health guide",
			title:   "Understanding PTSD: Symptoms, Diagnosis, and Treatment",
			snippet: "Post-traumatic stress disorder (PTSD) is a mental health condition. The DSM-5 diagnostic criteria include trauma exposure and emotional dysregulation.",
			url:     "https://example.com/mental-health/ptsd-guide",
		},
		{
			name:    "Therapy techniques",
			title:   "DBT Skills for Emotional Regulation",
			snippet: "Dialectical behavior therapy (DBT) helps patients with mindfulness and emotional regulation. This psychotherapy approach addresses attachment and trauma.",
			url:     "https://example.com/therapy/dbt-skills",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores := ClassifyDomain(tt.title, tt.snippet, tt.url)
			primary := scores.PrimaryDomain()
			assert.Equal(t, DomainPsychology, primary,
				"Expected psychology domain, got %s (scores: %v)", primary, scores)
		})
	}
}

func TestClassifyDomain_Documentation(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		snippet string
		url     string
	}{
		{
			name:    "API reference",
			title:   "API Reference - User Authentication",
			snippet: "Parameters: username (string), password (string). Returns: authentication token. See also: OAuth documentation. Example usage included.",
			url:     "https://example.com/docs/api/authentication",
		},
		{
			name:    "Getting started guide",
			title:   "Getting Started - Installation and Configuration",
			snippet: "Prerequisites: Node.js v18+. Installation: npm install example-lib. Configuration options and flags are documented below.",
			url:     "https://example.com/docs/getting-started",
		},
		{
			name:    "Tutorial",
			title:   "Tutorial: Building Your First Application",
			snippet: "This quickstart guide covers the basics. Important: Follow these steps in order. Tip: Use the example code provided. Warning: Deprecated in v2.0.",
			url:     "https://example.com/guide/first-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores := ClassifyDomain(tt.title, tt.snippet, tt.url)
			primary := scores.PrimaryDomain()
			assert.Equal(t, DomainDocumentation, primary,
				"Expected documentation domain, got %s (scores: %v)", primary, scores)
		})
	}
}

func TestClassifyDomain_Commercial(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		snippet string
		url     string
	}{
		{
			name:    "Product page",
			title:   "Buy iPhone 15 Pro - Best Price $999",
			snippet: "Order now and get free shipping! Add to cart for exclusive discount. Premium features with enterprise support. Subscribe for updates.",
			url:     "https://shop.example.com/product/iphone-15-pro",
		},
		{
			name:    "Pricing page",
			title:   "Pricing Plans - Choose Your Plan",
			snippet: "Free trial available. Premium plan: $29/month. Enterprise pricing on request. 20% off annual subscription. Buy now, pay later.",
			url:     "https://example.com/pricing",
		},
		{
			name:    "E-commerce listing",
			title:   "Best Wireless Headphones - Compare and Buy",
			snippet: "Customer reviews: 4.5/5 rating. Price: $149. Delivery in 2-3 days. Add to cart. Compare features and benefits. Shop now!",
			url:     "https://store.example.com/checkout/headphones",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores := ClassifyDomain(tt.title, tt.snippet, tt.url)
			primary := scores.PrimaryDomain()
			assert.Equal(t, DomainCommercial, primary,
				"Expected commercial domain, got %s (scores: %v)", primary, scores)
		})
	}
}

func TestClassifyDomain_General(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		snippet string
		url     string
	}{
		{
			name:    "Generic content",
			title:   "Welcome to Our Website",
			snippet: "Hello world. This is a simple page with random text that has no strong domain signals.",
			url:     "https://example.com/",
		},
		{
			name:    "Empty content",
			title:   "",
			snippet: "",
			url:     "https://example.com/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores := ClassifyDomain(tt.title, tt.snippet, tt.url)
			primary := scores.PrimaryDomain()
			assert.Equal(t, DomainGeneral, primary,
				"Expected general domain, got %s (scores: %v)", primary, scores)
		})
	}
}

func TestClassifyDomain_NormalizesScores(t *testing.T) {
	scores := ClassifyDomain(
		"How to implement REST API in Go",
		"This tutorial shows error handling and JSON responses with the http package",
		"https://github.com/example/api-tutorial",
	)

	total := 0.0
	for _, s := range scores {
		total += s
	}
	assert.InDelta(t, 1.0, total, 0.0001, "Scores should sum to 1.0")
}

func TestClassifyDomain_URLBonus(t *testing.T) {

	scoresAcademicURL := ClassifyDomain(
		"Research Study",
		"Analysis of data patterns",
		"https://arxiv.org/paper/12345",
	)

	scoresGenericURL := ClassifyDomain(
		"Research Study",
		"Analysis of data patterns",
		"https://example.com/article",
	)

	academicScoreWithBonus := scoresAcademicURL[DomainAcademic]
	academicScoreWithout := scoresGenericURL[DomainAcademic]

	require.NotZero(t, academicScoreWithBonus, "Academic URL should have academic score")
	assert.GreaterOrEqual(t, academicScoreWithBonus, academicScoreWithout,
		"Academic URL should boost academic score")
}

func TestDomainToCategory(t *testing.T) {
	tests := []struct {
		domain   Domain
		expected QueryCategory
	}{
		{DomainProgramming, CategoryTech},
		{DomainDocumentation, CategoryTech},
		{DomainAcademic, CategoryAcademic},
		{DomainPsychology, CategoryAcademic},
		{DomainNews, CategoryNews},
		{DomainCommercial, CategoryGeneral},
		{DomainGeneral, CategoryGeneral},
	}

	for _, tt := range tests {
		t.Run(string(tt.domain), func(t *testing.T) {
			result := DomainToCategory(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCategoryToDomain(t *testing.T) {
	tests := []struct {
		category QueryCategory
		expected Domain
	}{
		{CategoryTech, DomainProgramming},
		{CategoryAcademic, DomainAcademic},
		{CategoryNews, DomainNews},
		{CategorySports, DomainNews},
		{CategoryGeneral, DomainGeneral},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			result := CategoryToDomain(tt.category)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsTerm(t *testing.T) {
	text := "this is a test with cognitive behavioral therapy mentioned"

	assert.True(t, containsTerm(text, "cognitive"))
	assert.True(t, containsTerm(text, "Cognitive"))
	assert.True(t, containsTerm(text, "cognitive behavioral"))
	assert.False(t, containsTerm(text, "programming"))
}

func BenchmarkClassifyDomain(b *testing.B) {
	title := "How to implement REST API in Go with proper error handling"
	snippet := "This tutorial shows you how to build a REST API using the net/http package. We cover JSON encoding, middleware, and deployment with Docker and Kubernetes."
	url := "https://github.com/example/go-rest-api-tutorial"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ClassifyDomain(title, snippet, url)
	}
}

func TestClassifyDomain_Deterministic(t *testing.T) {
	title := "Machine learning research paper"
	snippet := "This study presents findings on neural networks with statistical analysis"
	url := "https://arxiv.org/abs/2024.12345"

	scores1 := ClassifyDomain(title, snippet, url)
	scores2 := ClassifyDomain(title, snippet, url)
	scores3 := ClassifyDomain(title, snippet, url)

	assert.Equal(t, scores1.PrimaryDomain(), scores2.PrimaryDomain())
	assert.Equal(t, scores2.PrimaryDomain(), scores3.PrimaryDomain())
}

func TestClassifyDomain_MixedContent(t *testing.T) {
	scores := ClassifyDomain(
		"Go Package Documentation - http",
		"Package http provides HTTP client implementation. func Get returns Response. See installation guide.",
		"https://pkg.go.dev/net/http",
	)

	assert.Greater(t, scores[DomainProgramming]+scores[DomainDocumentation], 0.5,
		"Should have significant programming or documentation signal")
}
