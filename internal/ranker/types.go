package ranker

type ScoredResult struct {
	URL            string          `json:"url"`
	Title          string          `json:"title"`
	Snippet        string          `json:"snippet"`
	Engines        []string        `json:"engines"`
	OriginalScore  float64         `json:"original_score"`
	AdjustedScore  float64         `json:"adjusted_score"`
	PublishedDate  string          `json:"published_date,omitempty"`
	Badges         Badges          `json:"badges"`
	ScoreBreakdown *ScoreBreakdown `json:"score_breakdown,omitempty"`

	RRFScore        float64        `json:"rrf_score,omitempty"`
	FinalScore      float64        `json:"final_score"`
	IntentAlignment float64        `json:"intent_alignment,omitempty"`
	DomainScores    DomainScores   `json:"domain_scores,omitempty"`
	PrimaryDomain   Domain         `json:"primary_domain,omitempty"`
	ContentType     ContentType    `json:"content_type,omitempty"`
	SignalRanks     map[string]int `json:"signal_ranks,omitempty"`

	RRFRank   int `json:"rrf_rank,omitempty"`
	FinalRank int `json:"final_rank,omitempty"`

	ExtractedContent string `json:"extracted_content,omitempty"`

	Availability string `json:"availability,omitempty"`
	HTTPStatus   int    `json:"http_status,omitempty"`
}

type Badges struct {
	HasAds        bool `json:"has_ads"`
	TrackerHeavy  bool `json:"tracker_heavy"`
	KnownSpam     bool `json:"known_spam"`
	Reputable     bool `json:"reputable"`
	ContentFarm   bool `json:"content_farm"`
	Authoritative bool `json:"authoritative"`
}

type ScoreBreakdown struct {
	TitleMatchBonus int `json:"title_match_bonus"`
	URLMatchBonus   int `json:"url_match_bonus"`

	TrancoBonus          int `json:"tranco_bonus"`
	EngineAgreementBonus int `json:"engine_agreement_bonus"`
	FreshnessBonus       int `json:"freshness_bonus"`

	BlocklistPenalty   int `json:"blocklist_penalty"`
	ContentFarmPenalty int `json:"content_farm_penalty"`
	ParamPenalty       int `json:"param_penalty"`

	SnippetBonus      int `json:"snippet_bonus"`
	URLStructureBonus int `json:"url_structure_bonus"`

	DetectedCategory string `json:"detected_category,omitempty"`

	RRFBase             float64            `json:"rrf_base,omitempty"`
	DomainBoost         float64            `json:"domain_boost,omitempty"`
	SignalContributions map[string]float64 `json:"signal_contributions,omitempty"`
	DetectedIntent      string             `json:"detected_intent,omitempty"`
	IntentConfidence    float64            `json:"intent_confidence,omitempty"`
	DomainMatch         string             `json:"domain_match,omitempty"`

	TextRelevance *TextRelevanceBreakdown `json:"text_relevance,omitempty"`
}

type TextRelevanceBreakdown struct {
	TitleMatch       float64 `json:"title_match"`
	SnippetMatch     float64 `json:"snippet_match"`
	TermFrequency    float64 `json:"term_frequency"`
	Proximity        float64 `json:"proximity"`
	CoreTermCoverage float64 `json:"core_term_coverage"`
	Combined         float64 `json:"combined"`
}

func (b *ScoreBreakdown) Total() int {
	return b.TitleMatchBonus +
		b.URLMatchBonus +
		b.TrancoBonus +
		b.EngineAgreementBonus +
		b.FreshnessBonus +
		b.BlocklistPenalty +
		b.ContentFarmPenalty +
		b.ParamPenalty +
		b.SnippetBonus +
		b.URLStructureBonus
}
