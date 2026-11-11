package ranker

type AlignmentConfig struct {
	MaxBoost        float64 `yaml:"max_boost"`
	MismatchPenalty float64 `yaml:"mismatch_penalty"`
	MinConfidence   float64 `yaml:"min_confidence"`
}

func DefaultAlignmentConfig() AlignmentConfig {
	return AlignmentConfig{
		MaxBoost:        0.5,
		MismatchPenalty: 0.2,
		MinConfidence:   0.3,
	}
}

func AlignmentScore(queryDomains, resultDomains DomainScores) float64 {
	if len(queryDomains) == 0 || len(resultDomains) == 0 {
		return 0.0
	}

	dotProduct := 0.0
	for domain, queryScore := range queryDomains {
		if resultScore, exists := resultDomains[domain]; exists {
			dotProduct += queryScore * resultScore
		}
	}

	return dotProduct
}

const NeutralAlignment = 0.4

func IntentAlignmentScore(intent QueryIntent, resultDomains DomainScores) float64 {
	if intent.PrimaryDomain == DomainGeneral {
		return NeutralAlignment
	}
	return AlignmentScore(intent.DomainDistribution, resultDomains)
}

func AlignmentBoost(alignment float64, config AlignmentConfig) float64 {
	const highAlignment = 0.5
	const lowAlignment = 0.3

	if alignment >= highAlignment {
		boostFactor := (alignment - highAlignment) / (1.0 - highAlignment)
		return 1.0 + (config.MaxBoost * boostFactor)
	}

	if alignment < lowAlignment {
		penaltyFactor := 1.0 - (alignment / lowAlignment)
		return 1.0 - (config.MismatchPenalty * penaltyFactor)
	}

	return 1.0
}

func AlignmentBoostWithConfidence(alignment, confidence float64, config AlignmentConfig) float64 {
	if confidence < config.MinConfidence {
		return 1.0
	}

	baseBoost := AlignmentBoost(alignment, config)

	deviation := baseBoost - 1.0
	scaledDeviation := deviation * ((confidence - config.MinConfidence) / (1.0 - config.MinConfidence))

	return 1.0 + scaledDeviation
}

func QueryCategoryToDomainScores(category QueryCategory, confidence float64) DomainScores {
	scores := make(DomainScores)

	switch category {
	case CategoryTech:
		scores[DomainProgramming] = 0.5 * confidence
		scores[DomainDocumentation] = 0.3 * confidence
		scores[DomainGeneral] = 0.2 * (1 - confidence)

	case CategoryAcademic:
		scores[DomainAcademic] = 0.6 * confidence
		scores[DomainPsychology] = 0.2 * confidence
		scores[DomainGeneral] = 0.2 * (1 - confidence)

	case CategoryNews:
		scores[DomainNews] = 0.8 * confidence
		scores[DomainGeneral] = 0.2 * (1 - confidence)

	case CategorySports:
		scores[DomainNews] = 0.6 * confidence
		scores[DomainGeneral] = 0.4 * (1 - confidence)

	default:
		scores[DomainGeneral] = 0.4
		scores[DomainProgramming] = 0.15
		scores[DomainNews] = 0.15
		scores[DomainAcademic] = 0.15
		scores[DomainCommercial] = 0.15
	}

	return scores.Normalize()
}

func ComputeIntentAlignment(intent QueryIntent, resultDomains DomainScores) float64 {
	queryDomains := QueryCategoryToDomainScores(intent.Category, intent.Confidence)
	return AlignmentScore(queryDomains, resultDomains)
}
