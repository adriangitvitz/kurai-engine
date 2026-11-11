package ranker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntent_TiedNewsAcademic_LeavesGeneralWithBalancedDistribution(t *testing.T) {
	intent := DetectIntent(tokens("election results"))

	assert.Equal(t, CategoryGeneral, intent.Category,
		"news/academic tie must NOT pick a winner - Category should fall to general")
	require.NotEmpty(t, intent.DomainDistribution,
		"DomainDistribution must still reflect the tied categories")
	assert.Greater(t, intent.DomainDistribution[DomainNews], 0.0,
		"news domain should still appear in the distribution")
	assert.Greater(t, intent.DomainDistribution[DomainAcademic], 0.0,
		"academic domain should still appear in the distribution")
}

func TestIntent_StrongCategorySignalStillWins(t *testing.T) {
	intent := DetectIntent(tokens("what is cognitive behavioral therapy"))
	assert.Equal(t, CategoryAcademic, intent.Category,
		"a clear academic-only signal must still resolve to academic")
}

func TestIntent_TechSignalAlwaysWinsOverTied(t *testing.T) {
	intent := DetectIntent(tokens("python error"))
	assert.Equal(t, CategoryTech, intent.Category)
}

func TestIntent_AcademicSearchOutrightStillAcademic(t *testing.T) {
	intent := DetectIntent(tokens("neural network paper arxiv"))
	assert.Equal(t, CategoryAcademic, intent.Category)
}

func TestIntent_TiedDistribution_AlignsBothNewsAndAcademic(t *testing.T) {
	intent := DetectIntent(tokens("election results"))

	newsResultDomain := DomainScores{DomainNews: 1.0}
	academicResultDomain := DomainScores{DomainAcademic: 1.0}

	newsAlign := AlignmentScore(intent.DomainDistribution, newsResultDomain)
	acadAlign := AlignmentScore(intent.DomainDistribution, academicResultDomain)

	t.Logf("news-result alignment=%.4f academic-result alignment=%.4f", newsAlign, acadAlign)
	assert.Greater(t, newsAlign, 0.0,
		"news result must get a positive alignment under balanced distribution")
	assert.InDelta(t, newsAlign, acadAlign, 0.05,
		"news and academic result alignments should be ~equal under tied distribution")
}
