package ranker

type SignalWeightProfile struct {
	SearXNGPosition float64 `json:"searxng_position"`
	TextRelevance   float64 `json:"text_relevance"`
	DomainAlignment float64 `json:"domain_alignment"`
	TrancoAuthority float64 `json:"tranco_authority"`
	EngineAgreement float64 `json:"engine_agreement"`
	Freshness       float64 `json:"freshness"`

	TitleMatchMult float64 `json:"title_match_mult"`
	URLMatchMult   float64 `json:"url_match_mult"`
	SnippetMult    float64 `json:"snippet_mult"`
	AuthorityMult  float64 `json:"authority_mult"`
	FreshnessMult  float64 `json:"freshness_mult"`
}

func DefaultWeightProfile() SignalWeightProfile {
	return SignalWeightProfile{
		SearXNGPosition: 1.0,
		TextRelevance:   1.2,
		DomainAlignment: 0.8,
		TrancoAuthority: 0.5,
		EngineAgreement: 0.6,
		Freshness:       0.3,

		TitleMatchMult: 1.0,
		URLMatchMult:   1.0,
		SnippetMult:    1.0,
		AuthorityMult:  1.0,
		FreshnessMult:  1.0,
	}
}

var weightProfiles = map[QueryType]map[QueryCategory]SignalWeightProfile{
	QueryTypeNavigational: {
		CategoryGeneral: {
			SearXNGPosition: 0.8,
			TextRelevance:   0.6,
			DomainAlignment: 0.5,
			TrancoAuthority: 1.5,
			EngineAgreement: 0.8,
			Freshness:       0.1,
			TitleMatchMult:  1.5,
			URLMatchMult:    2.0,
			SnippetMult:     0.5,
			AuthorityMult:   1.5,
			FreshnessMult:   0.3,
		},
		CategoryTech: {
			SearXNGPosition: 0.8,
			TextRelevance:   0.7,
			DomainAlignment: 0.6,
			TrancoAuthority: 1.3,
			EngineAgreement: 0.7,
			Freshness:       0.2,
			TitleMatchMult:  1.5,
			URLMatchMult:    2.0,
			SnippetMult:     0.6,
			AuthorityMult:   1.4,
			FreshnessMult:   0.4,
		},
	},
	QueryTypeInformational: {
		CategoryGeneral: {
			SearXNGPosition: 1.0,
			TextRelevance:   1.2,
			DomainAlignment: 0.8,
			TrancoAuthority: 0.5,
			EngineAgreement: 0.6,
			Freshness:       0.3,
			TitleMatchMult:  1.0,
			URLMatchMult:    1.0,
			SnippetMult:     1.2,
			AuthorityMult:   1.0,
			FreshnessMult:   1.0,
		},
		CategoryTech: {
			SearXNGPosition: 1.0,
			TextRelevance:   1.4,
			DomainAlignment: 1.0,
			TrancoAuthority: 0.4,
			EngineAgreement: 0.7,
			Freshness:       0.4,
			TitleMatchMult:  1.2,
			URLMatchMult:    1.1,
			SnippetMult:     1.3,
			AuthorityMult:   1.2,
			FreshnessMult:   1.2,
		},
		CategoryAcademic: {
			SearXNGPosition: 1.0,
			TextRelevance:   1.3,
			DomainAlignment: 1.2,
			TrancoAuthority: 0.3,
			EngineAgreement: 0.5,
			Freshness:       0.2,
			TitleMatchMult:  1.3,
			URLMatchMult:    0.8,
			SnippetMult:     1.4,
			AuthorityMult:   1.4,
			FreshnessMult:   0.5,
		},
		CategoryNews: {
			SearXNGPosition: 0.8,
			TextRelevance:   1.1,
			DomainAlignment: 1.0,
			TrancoAuthority: 0.6,
			EngineAgreement: 0.5,
			Freshness:       2.0,
			TitleMatchMult:  1.1,
			URLMatchMult:    0.7,
			SnippetMult:     1.0,
			AuthorityMult:   1.3,
			FreshnessMult:   2.0,
		},
		CategorySports: {
			SearXNGPosition: 0.8,
			TextRelevance:   1.0,
			DomainAlignment: 0.9,
			TrancoAuthority: 0.5,
			EngineAgreement: 0.5,
			Freshness:       1.8,
			TitleMatchMult:  1.2,
			URLMatchMult:    0.6,
			SnippetMult:     1.0,
			AuthorityMult:   1.2,
			FreshnessMult:   1.8,
		},
	},
	QueryTypeTransactional: {
		CategoryGeneral: {
			SearXNGPosition: 1.0,
			TextRelevance:   1.0,
			DomainAlignment: 0.7,
			TrancoAuthority: 1.2,
			EngineAgreement: 0.8,
			Freshness:       0.5,
			TitleMatchMult:  1.1,
			URLMatchMult:    1.3,
			SnippetMult:     0.9,
			AuthorityMult:   1.3,
			FreshnessMult:   0.8,
		},
		CategoryTech: {
			SearXNGPosition: 1.0,
			TextRelevance:   1.1,
			DomainAlignment: 0.8,
			TrancoAuthority: 1.0,
			EngineAgreement: 0.7,
			Freshness:       0.6,
			TitleMatchMult:  1.2,
			URLMatchMult:    1.4,
			SnippetMult:     1.0,
			AuthorityMult:   1.4,
			FreshnessMult:   1.0,
		},
	},
}

type DynamicWeights struct {
	defaultProfile SignalWeightProfile
	profiles       map[QueryType]map[QueryCategory]SignalWeightProfile
}

func NewDynamicWeights() *DynamicWeights {
	return &DynamicWeights{
		defaultProfile: DefaultWeightProfile(),
		profiles:       weightProfiles,
	}
}

func (dw *DynamicWeights) GetProfile(analysis *QueryAnalysis) SignalWeightProfile {
	if analysis == nil {
		return dw.defaultProfile
	}

	categoryProfiles, hasType := dw.profiles[analysis.QueryType]
	if !hasType {
		categoryProfiles = dw.profiles[QueryTypeInformational]
	}

	category := analysis.GetPrimaryCategory()
	profile, hasCategory := categoryProfiles[category]
	if !hasCategory {
		if generalProfile, ok := categoryProfiles[CategoryGeneral]; ok {
			profile = generalProfile
		} else {
			return dw.defaultProfile
		}
	}

	profile = dw.applyModifiers(profile, analysis)

	return profile
}

func (dw *DynamicWeights) applyModifiers(profile SignalWeightProfile, analysis *QueryAnalysis) SignalWeightProfile {
	modified := profile

	if analysis.IsQuestion {
		modified.TextRelevance *= 1.1
		modified.SnippetMult *= 1.1
	}

	if analysis.IsErrorQuery() {
		modified.URLMatchMult *= 1.2
		modified.TextRelevance *= 1.1
	}

	if analysis.HasVersion {
		modified.Freshness *= 1.3
		modified.FreshnessMult *= 1.2
	}

	if analysis.HasQuotedText {
		modified.TextRelevance *= 1.3
		modified.TitleMatchMult *= 1.2
	}

	if analysis.IsComparison {
		modified.TrancoAuthority *= 0.7
		modified.AuthorityMult *= 0.8
	}

	if len(analysis.Entities) > 0 {
		modified.DomainAlignment *= 1.2
	}

	return modified
}

func (dw *DynamicWeights) GetProfileForIntent(intent QueryIntent) SignalWeightProfile {
	queryType := QueryTypeInformational

	if intent.IsDocumentation {
		queryType = QueryTypeNavigational
	}

	categoryProfiles, hasType := dw.profiles[queryType]
	if !hasType {
		return dw.defaultProfile
	}

	profile, hasCategory := categoryProfiles[intent.Category]
	if !hasCategory {
		if generalProfile, ok := categoryProfiles[CategoryGeneral]; ok {
			return generalProfile
		}
		return dw.defaultProfile
	}

	return profile
}

func (dw *DynamicWeights) SetProfile(queryType QueryType, category QueryCategory, profile SignalWeightProfile) {
	if dw.profiles[queryType] == nil {
		dw.profiles[queryType] = make(map[QueryCategory]SignalWeightProfile)
	}
	dw.profiles[queryType][category] = profile
}

func (dw *DynamicWeights) SetDefaultProfile(profile SignalWeightProfile) {
	dw.defaultProfile = profile
}

func (profile SignalWeightProfile) ToRRFWeights() map[string]float64 {
	return map[string]float64{
		"searxng_position": profile.SearXNGPosition,
		"text_relevance":   profile.TextRelevance,
		"domain_alignment": profile.DomainAlignment,
		"tranco_authority": profile.TrancoAuthority,
		"engine_agreement": profile.EngineAgreement,
		"freshness":        profile.Freshness,
	}
}

func (profile SignalWeightProfile) MergeWithConfig(configWeights map[string]float64) map[string]float64 {
	merged := make(map[string]float64)

	for k, v := range configWeights {
		merged[k] = v
	}

	profileWeights := profile.ToRRFWeights()
	for k, v := range profileWeights {
		if v != 0 {
			if existing, ok := merged[k]; ok {
				merged[k] = existing * v
			} else {
				merged[k] = v
			}
		}
	}

	return merged
}

func (profile *SignalWeightProfile) AdjustForRecency(timeSensitivity float64) {
	if timeSensitivity < 0 {
		timeSensitivity = 0
	}
	if timeSensitivity > 1 {
		timeSensitivity = 1
	}

	baseFreshness := 0.3
	maxFreshness := 2.0
	profile.Freshness = baseFreshness + (maxFreshness-baseFreshness)*timeSensitivity
	profile.FreshnessMult = 1.0 + timeSensitivity
}
