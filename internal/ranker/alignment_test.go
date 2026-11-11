package ranker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlignmentScore(t *testing.T) {
	tests := []struct {
		name          string
		queryDomains  DomainScores
		resultDomains DomainScores
		expectedMin   float64
		expectedMax   float64
	}{
		{
			name: "Perfect alignment - identical distributions",
			queryDomains: DomainScores{
				DomainProgramming: 1.0,
			},
			resultDomains: DomainScores{
				DomainProgramming: 1.0,
			},
			expectedMin: 0.99,
			expectedMax: 1.01,
		},
		{
			name: "No alignment - completely different domains",
			queryDomains: DomainScores{
				DomainProgramming: 1.0,
			},
			resultDomains: DomainScores{
				DomainNews: 1.0,
			},
			expectedMin: 0.0,
			expectedMax: 0.01,
		},
		{
			name: "Partial alignment - overlapping distributions",
			queryDomains: DomainScores{
				DomainProgramming: 0.6,
				DomainGeneral:     0.4,
			},
			resultDomains: DomainScores{
				DomainProgramming: 0.4,
				DomainGeneral:     0.3,
				DomainNews:        0.3,
			},
			expectedMin: 0.35,
			expectedMax: 0.37,
		},
		{
			name:          "Empty query domains",
			queryDomains:  DomainScores{},
			resultDomains: DomainScores{DomainProgramming: 1.0},
			expectedMin:   0.0,
			expectedMax:   0.01,
		},
		{
			name:          "Empty result domains",
			queryDomains:  DomainScores{DomainProgramming: 1.0},
			resultDomains: DomainScores{},
			expectedMin:   0.0,
			expectedMax:   0.01,
		},
		{
			name: "Multi-domain academic alignment",
			queryDomains: DomainScores{
				DomainAcademic:   0.7,
				DomainPsychology: 0.3,
			},
			resultDomains: DomainScores{
				DomainAcademic:   0.5,
				DomainPsychology: 0.5,
			},
			expectedMin: 0.49,
			expectedMax: 0.51,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := AlignmentScore(tt.queryDomains, tt.resultDomains)
			assert.GreaterOrEqual(t, score, tt.expectedMin, "Score should be >= expected min")
			assert.LessOrEqual(t, score, tt.expectedMax, "Score should be <= expected max")
		})
	}
}

func TestAlignmentBoost(t *testing.T) {
	config := DefaultAlignmentConfig()

	tests := []struct {
		name        string
		alignment   float64
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "Perfect alignment - max boost",
			alignment:   1.0,
			expectedMin: 1.0 + config.MaxBoost - 0.01,
			expectedMax: 1.0 + config.MaxBoost + 0.01,
		},
		{
			name:        "High alignment (0.75) - partial boost",
			alignment:   0.75,
			expectedMin: 1.2,
			expectedMax: 1.3,
		},
		{
			name:        "Threshold alignment (0.5) - no boost yet",
			alignment:   0.5,
			expectedMin: 0.99,
			expectedMax: 1.01,
		},
		{
			name:        "Neutral zone (0.4) - no change",
			alignment:   0.4,
			expectedMin: 0.99,
			expectedMax: 1.01,
		},
		{
			name:        "Low alignment (0.15) - partial penalty",
			alignment:   0.15,
			expectedMin: 0.85,
			expectedMax: 0.95,
		},
		{
			name:        "Zero alignment - max penalty",
			alignment:   0.0,
			expectedMin: 1.0 - config.MismatchPenalty - 0.01,
			expectedMax: 1.0 - config.MismatchPenalty + 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boost := AlignmentBoost(tt.alignment, config)
			assert.GreaterOrEqual(t, boost, tt.expectedMin, "Boost should be >= expected min")
			assert.LessOrEqual(t, boost, tt.expectedMax, "Boost should be <= expected max")
		})
	}
}

func TestAlignmentBoostWithConfidence(t *testing.T) {
	config := DefaultAlignmentConfig()

	tests := []struct {
		name        string
		alignment   float64
		confidence  float64
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "High alignment, high confidence - full boost",
			alignment:   1.0,
			confidence:  1.0,
			expectedMin: 1.0 + config.MaxBoost - 0.01,
			expectedMax: 1.0 + config.MaxBoost + 0.01,
		},
		{
			name:        "High alignment, low confidence - reduced boost",
			alignment:   1.0,
			confidence:  0.5,
			expectedMin: 1.1,
			expectedMax: 1.2,
		},
		{
			name:        "High alignment, below min confidence - neutral",
			alignment:   1.0,
			confidence:  0.2,
			expectedMin: 0.99,
			expectedMax: 1.01,
		},
		{
			name:        "Low alignment, high confidence - full penalty",
			alignment:   0.0,
			confidence:  1.0,
			expectedMin: 1.0 - config.MismatchPenalty - 0.01,
			expectedMax: 1.0 - config.MismatchPenalty + 0.01,
		},
		{
			name:        "Low alignment, low confidence - reduced penalty",
			alignment:   0.0,
			confidence:  0.5,
			expectedMin: 0.9,
			expectedMax: 0.96,
		},
		{
			name:        "Neutral alignment, any confidence - no change",
			alignment:   0.4,
			confidence:  1.0,
			expectedMin: 0.99,
			expectedMax: 1.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boost := AlignmentBoostWithConfidence(tt.alignment, tt.confidence, config)
			assert.GreaterOrEqual(t, boost, tt.expectedMin, "Boost should be >= expected min")
			assert.LessOrEqual(t, boost, tt.expectedMax, "Boost should be <= expected max")
		})
	}
}

func TestQueryCategoryToDomainScores(t *testing.T) {
	tests := []struct {
		name           string
		category       QueryCategory
		confidence     float64
		expectedDomain Domain
	}{
		{
			name:           "Tech category maps to programming",
			category:       CategoryTech,
			confidence:     0.9,
			expectedDomain: DomainProgramming,
		},
		{
			name:           "Academic category maps to academic",
			category:       CategoryAcademic,
			confidence:     0.9,
			expectedDomain: DomainAcademic,
		},
		{
			name:           "News category maps to news",
			category:       CategoryNews,
			confidence:     0.9,
			expectedDomain: DomainNews,
		},
		{
			name:           "Sports category maps to news (sports coverage)",
			category:       CategorySports,
			confidence:     0.9,
			expectedDomain: DomainNews,
		},
		{
			name:           "General category has general as primary",
			category:       CategoryGeneral,
			confidence:     0.9,
			expectedDomain: DomainGeneral,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores := QueryCategoryToDomainScores(tt.category, tt.confidence)

			total := 0.0
			for _, s := range scores {
				total += s
			}
			assert.InDelta(t, 1.0, total, 0.01, "Scores should sum to ~1.0")

			assert.Equal(t, tt.expectedDomain, scores.PrimaryDomain(),
				"Primary domain should match expected")
		})
	}
}

func TestComputeIntentAlignment(t *testing.T) {
	tests := []struct {
		name          string
		intent        QueryIntent
		resultDomains DomainScores
		minAlignment  float64
		maxAlignment  float64
	}{
		{
			name: "Tech intent aligns with programming result",
			intent: QueryIntent{
				Category:   CategoryTech,
				Confidence: 0.9,
			},
			resultDomains: DomainScores{
				DomainProgramming:   0.7,
				DomainDocumentation: 0.3,
			},
			minAlignment: 0.4,
			maxAlignment: 1.0,
		},
		{
			name: "Tech intent misaligns with news result",
			intent: QueryIntent{
				Category:   CategoryTech,
				Confidence: 0.9,
			},
			resultDomains: DomainScores{
				DomainNews: 1.0,
			},
			minAlignment: 0.0,
			maxAlignment: 0.2,
		},
		{
			name: "Academic intent aligns with academic result",
			intent: QueryIntent{
				Category:   CategoryAcademic,
				Confidence: 0.9,
			},
			resultDomains: DomainScores{
				DomainAcademic:   0.6,
				DomainPsychology: 0.4,
			},
			minAlignment: 0.5,
			maxAlignment: 1.0,
		},
		{
			name: "General intent has some alignment with anything",
			intent: QueryIntent{
				Category:   CategoryGeneral,
				Confidence: 0.5,
			},
			resultDomains: DomainScores{
				DomainCommercial: 0.8,
				DomainGeneral:    0.2,
			},
			minAlignment: 0.1,
			maxAlignment: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alignment := ComputeIntentAlignment(tt.intent, tt.resultDomains)
			assert.GreaterOrEqual(t, alignment, tt.minAlignment,
				"Alignment should be >= min expected")
			assert.LessOrEqual(t, alignment, tt.maxAlignment,
				"Alignment should be <= max expected")
		})
	}
}

func TestDefaultAlignmentConfig(t *testing.T) {
	config := DefaultAlignmentConfig()

	assert.Equal(t, 0.5, config.MaxBoost, "Default max boost should be 0.5")
	assert.Equal(t, 0.2, config.MismatchPenalty, "Default mismatch penalty should be 0.2")
	assert.Equal(t, 0.3, config.MinConfidence, "Default min confidence should be 0.3")
}

func TestAlignmentBoostSymmetry(t *testing.T) {
	config := DefaultAlignmentConfig()

	alignments := []float64{0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0}
	var prevBoost float64 = -1.0

	for _, alignment := range alignments {
		boost := AlignmentBoost(alignment, config)

		assert.GreaterOrEqual(t, boost, prevBoost,
			"Boost should not decrease with alignment (alignment=%.1f)", alignment)
		prevBoost = boost

		assert.Greater(t, boost, 0.5, "Boost should be > 0.5")
		assert.Less(t, boost, 2.0, "Boost should be < 2.0")
	}

	assert.Less(t, AlignmentBoost(0.0, config), 1.0, "Zero alignment should have penalty")
	assert.Equal(t, 1.0, AlignmentBoost(0.4, config), "Neutral zone should return 1.0")
	assert.Greater(t, AlignmentBoost(1.0, config), 1.0, "Perfect alignment should have boost")
}

func BenchmarkAlignmentScore(b *testing.B) {
	queryDomains := DomainScores{
		DomainProgramming:   0.5,
		DomainDocumentation: 0.3,
		DomainGeneral:       0.2,
	}
	resultDomains := DomainScores{
		DomainProgramming:   0.6,
		DomainDocumentation: 0.2,
		DomainCommercial:    0.2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AlignmentScore(queryDomains, resultDomains)
	}
}

func BenchmarkComputeIntentAlignment(b *testing.B) {
	intent := QueryIntent{
		Category:   CategoryTech,
		Confidence: 0.8,
	}
	resultDomains := DomainScores{
		DomainProgramming:   0.6,
		DomainDocumentation: 0.2,
		DomainGeneral:       0.2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ComputeIntentAlignment(intent, resultDomains)
	}
}

func TestIntentAlignmentScore_GeneralIntentIsNeutralForEveryResult(t *testing.T) {
	general := QueryIntent{PrimaryDomain: DomainGeneral, DomainDistribution: DomainScores{DomainGeneral: 1}}
	unclassified := DomainScores{DomainGeneral: 1}
	news := DomainScores{DomainNews: 0.6, DomainCommercial: 0.4}

	assert.Equal(t, IntentAlignmentScore(general, unclassified), IntentAlignmentScore(general, news))
	assert.Equal(t, 1.0, AlignmentBoost(IntentAlignmentScore(general, news), DefaultAlignmentConfig()))
}

func TestIntentAlignmentScore_SpecificIntentStillDiscriminates(t *testing.T) {
	newsIntent := QueryIntent{PrimaryDomain: DomainNews, DomainDistribution: DomainScores{DomainNews: 1}}
	assert.Greater(t,
		IntentAlignmentScore(newsIntent, DomainScores{DomainNews: 1}),
		IntentAlignmentScore(newsIntent, DomainScores{DomainGeneral: 1}),
	)
}
