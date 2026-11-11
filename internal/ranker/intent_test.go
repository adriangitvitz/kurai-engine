package ranker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectIntent(t *testing.T) {
	tests := []struct {
		name             string
		queryTerms       []string
		expectedCategory QueryCategory
		expectedMinConf  float64
	}{
		{
			name:             "Go TLS query",
			queryTerms:       []string{"go", "tls", "cipher", "suites"},
			expectedCategory: CategoryTech,
			expectedMinConf:  0.6,
		},
		{
			name:             "Python documentation",
			queryTerms:       []string{"python", "docs", "asyncio"},
			expectedCategory: CategoryTech,
			expectedMinConf:  0.6,
		},
		{
			name:             "Error debugging",
			queryTerms:       []string{"error", "nil", "pointer", "go"},
			expectedCategory: CategoryTech,
			expectedMinConf:  0.6,
		},
		{
			name:             "Docker kubernetes",
			queryTerms:       []string{"docker", "kubernetes", "deployment"},
			expectedCategory: CategoryTech,
			expectedMinConf:  0.6,
		},

		{
			name:             "ArXiv paper search",
			queryTerms:       []string{"arxiv", "machine", "learning", "transformers"},
			expectedCategory: CategoryAcademic,
			expectedMinConf:  0.6,
		},
		{
			name:             "Research paper",
			queryTerms:       []string{"research", "paper", "neural", "network"},
			expectedCategory: CategoryAcademic,
			expectedMinConf:  0.6,
		},

		{
			name:             "News today",
			queryTerms:       []string{"news", "today", "mexico"},
			expectedCategory: CategoryNews,
			expectedMinConf:  0.6,
		},
		{
			name:             "Breaking news",
			queryTerms:       []string{"breaking", "news", "election"},
			expectedCategory: CategoryNews,
			expectedMinConf:  0.6,
		},

		{
			name:             "Sports score",
			queryTerms:       []string{"nfl", "score", "game"},
			expectedCategory: CategorySports,
			expectedMinConf:  0.6,
		},
		{
			name:             "Match results",
			queryTerms:       []string{"vs", "match", "playoffs"},
			expectedCategory: CategorySports,
			expectedMinConf:  0.6,
		},

		{
			name:             "Generic search",
			queryTerms:       []string{"best", "restaurant", "near", "me"},
			expectedCategory: CategoryGeneral,
			expectedMinConf:  0.5,
		},
		{
			name:             "Empty query",
			queryTerms:       []string{},
			expectedCategory: CategoryGeneral,
			expectedMinConf:  0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := DetectIntent(tt.queryTerms)
			assert.Equal(t, tt.expectedCategory, intent.Category)
			assert.GreaterOrEqual(t, intent.Confidence, tt.expectedMinConf)
		})
	}
}

func TestDetectIntent_SpecificFlags(t *testing.T) {
	tests := []struct {
		name              string
		queryTerms        []string
		isDocumentation   bool
		isErrorResolution bool
		isPaperSearch     bool
		isNews            bool
	}{
		{
			name:            "Documentation query",
			queryTerms:      []string{"python", "docs", "reference"},
			isDocumentation: true,
		},
		{
			name:              "Error resolution",
			queryTerms:        []string{"error", "fix", "debug", "go"},
			isErrorResolution: true,
		},
		{
			name:          "Paper search",
			queryTerms:    []string{"arxiv", "paper", "research"},
			isPaperSearch: true,
		},
		{
			name:       "News search",
			queryTerms: []string{"news", "today", "latest"},
			isNews:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := DetectIntent(tt.queryTerms)
			if tt.isDocumentation {
				assert.True(t, intent.IsDocumentation)
			}
			if tt.isErrorResolution {
				assert.True(t, intent.IsErrorResolution)
			}
			if tt.isPaperSearch {
				assert.True(t, intent.IsPaperSearch)
			}
			if tt.isNews {
				assert.True(t, intent.IsNews)
			}
		})
	}
}

func TestParseCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected QueryCategory
	}{
		{"tech", CategoryTech},
		{"technical", CategoryTech},
		{"programming", CategoryTech},
		{"code", CategoryTech},
		{"software", CategoryTech},
		{"engineering", CategoryTech},

		{"academic", CategoryAcademic},
		{"research", CategoryAcademic},
		{"science", CategoryAcademic},
		{"paper", CategoryAcademic},
		{"papers", CategoryAcademic},

		{"psychology", CategoryAcademic},
		{"sociology", CategoryAcademic},
		{"anthropology", CategoryAcademic},
		{"biology", CategoryAcademic},
		{"chemistry", CategoryAcademic},
		{"physics", CategoryAcademic},
		{"medicine", CategoryAcademic},
		{"medical", CategoryAcademic},
		{"history", CategoryAcademic},
		{"philosophy", CategoryAcademic},
		{"literature", CategoryAcademic},
		{"humanities", CategoryAcademic},
		{"law", CategoryAcademic},
		{"legal", CategoryAcademic},
		{"economics", CategoryAcademic},

		{"news", CategoryNews},
		{"current", CategoryNews},
		{"politics", CategoryNews},
		{"business", CategoryNews},

		{"sports", CategorySports},
		{"sport", CategorySports},

		{"general", CategoryGeneral},
		{"unknown", CategoryGeneral},
		{"", CategoryGeneral},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseCategory(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntentBoostMultiplier(t *testing.T) {
	tests := []struct {
		name          string
		intent        QueryIntent
		authCategory  AuthorityCategory
		minMultiplier float64
		maxMultiplier float64
	}{
		{
			name: "tech intent + official docs",
			intent: QueryIntent{
				Category:        CategoryTech,
				IsDocumentation: true,
				Confidence:      1.0,
			},
			authCategory:  CategoryOfficialDocs,
			minMultiplier: 1.4,
			maxMultiplier: 1.6,
		},
		{
			name: "tech intent + developer QA with error",
			intent: QueryIntent{
				Category:          CategoryTech,
				IsErrorResolution: true,
				Confidence:        1.0,
			},
			authCategory:  CategoryDeveloperQA,
			minMultiplier: 1.4,
			maxMultiplier: 1.6,
		},
		{
			name: "academic intent + paper archive",
			intent: QueryIntent{
				Category:      CategoryAcademic,
				IsPaperSearch: true,
				Confidence:    1.0,
			},
			authCategory:  CategoryPaperArchive,
			minMultiplier: 1.4,
			maxMultiplier: 1.6,
		},
		{
			name: "news intent + major news",
			intent: QueryIntent{
				Category:   CategoryNews,
				IsNews:     true,
				Confidence: 1.0,
			},
			authCategory:  CategoryMajorNews,
			minMultiplier: 1.4,
			maxMultiplier: 1.6,
		},
		{
			name: "general intent no boost",
			intent: QueryIntent{
				Category:   CategoryGeneral,
				Confidence: 1.0,
			},
			authCategory:  CategoryOfficialDocs,
			minMultiplier: 0.9,
			maxMultiplier: 1.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiplier := GetIntentBoostMultiplier(tt.intent, tt.authCategory)
			assert.GreaterOrEqual(t, multiplier, tt.minMultiplier)
			assert.LessOrEqual(t, multiplier, tt.maxMultiplier)
		})
	}
}

func TestDetectIntent_DomainDistribution(t *testing.T) {
	tests := []struct {
		name            string
		queryTerms      []string
		expectedPrimary Domain
		minConfidence   float64
	}{
		{
			name:            "Tech query -> Programming domain",
			queryTerms:      []string{"golang", "error", "handling"},
			expectedPrimary: DomainProgramming,
			minConfidence:   0.3,
		},
		{
			name:            "Documentation query -> Documentation domain",
			queryTerms:      []string{"python", "docs", "tutorial"},
			expectedPrimary: DomainDocumentation,
			minConfidence:   0.3,
		},
		{
			name:            "Academic query -> Academic domain",
			queryTerms:      []string{"research", "paper", "methodology"},
			expectedPrimary: DomainAcademic,
			minConfidence:   0.3,
		},
		{
			name:            "News query -> News domain",
			queryTerms:      []string{"breaking", "news", "election"},
			expectedPrimary: DomainNews,
			minConfidence:   0.3,
		},
		{
			name:            "General query -> General domain",
			queryTerms:      []string{"best", "pizza", "restaurant"},
			expectedPrimary: DomainGeneral,
			minConfidence:   0.5,
		},
		{
			name:            "Empty query -> General domain",
			queryTerms:      []string{},
			expectedPrimary: DomainGeneral,
			minConfidence:   0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := DetectIntent(tt.queryTerms)

			assert.NotNil(t, intent.DomainDistribution, "Domain distribution should be populated")
			assert.NotEmpty(t, intent.DomainDistribution, "Domain distribution should not be empty")

			assert.Equal(t, tt.expectedPrimary, intent.PrimaryDomain,
				"Primary domain should match expected")

			assert.GreaterOrEqual(t, intent.DomainDistribution[tt.expectedPrimary], tt.minConfidence,
				"Primary domain should have at least minimum confidence")

			total := 0.0
			for _, score := range intent.DomainDistribution {
				total += score
			}
			assert.InDelta(t, 1.0, total, 0.01, "Domain scores should sum to 1.0")
		})
	}
}

func TestDetectModifiers(t *testing.T) {
	tests := []struct {
		name         string
		queryTerms   []string
		isQuestion   bool
		isTutorial   bool
		isRecent     bool
		isComparison bool
	}{
		{
			name:       "Question query",
			queryTerms: []string{"how", "to", "fix", "error"},
			isQuestion: true,
		},
		{
			name:       "Tutorial query",
			queryTerms: []string{"golang", "tutorial", "beginner"},
			isTutorial: true,
		},
		{
			name:       "Recent query",
			queryTerms: []string{"latest", "python", "version"},
			isRecent:   true,
		},
		{
			name:         "Comparison query",
			queryTerms:   []string{"react", "vs", "vue"},
			isComparison: true,
		},
		{
			name:         "Multiple modifiers",
			queryTerms:   []string{"what", "is", "best", "framework", "2025"},
			isQuestion:   true,
			isRecent:     true,
			isComparison: true,
		},
		{
			name:         "No modifiers",
			queryTerms:   []string{"golang", "error", "handling"},
			isQuestion:   false,
			isTutorial:   false,
			isRecent:     false,
			isComparison: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := DetectIntent(tt.queryTerms)

			assert.Equal(t, tt.isQuestion, intent.Modifiers.IsQuestion,
				"IsQuestion should match expected")
			assert.Equal(t, tt.isTutorial, intent.Modifiers.IsTutorial,
				"IsTutorial should match expected")
			assert.Equal(t, tt.isRecent, intent.Modifiers.IsRecent,
				"IsRecent should match expected")
			assert.Equal(t, tt.isComparison, intent.Modifiers.IsComparison,
				"IsComparison should match expected")
		})
	}
}
