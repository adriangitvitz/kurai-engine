package ranker

import (
	"context"
	"sort"

	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/reranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

type BlocklistChecker interface {
	IsBlocked(domain string) bool
	GetTrancoRank(domain string) int
}

type Ranker struct {
	config    *config.RankingConfig
	blocklist BlocklistChecker

	entityRecognizer  *EntityRecognizer
	synonymExpander   *SynonymExpander
	dynamicWeights    *DynamicWeights
	diversityEnforcer *DiversityEnforcer
	mmrConfig         MMRConfig
	collapseConfig    CollapseConfig

	relevanceFloor RelevanceFloorConfig

	reranker    reranker.Reranker
	rerankTopN  int
	rerankAlpha float64
}

func New(cfg *config.RankingConfig, bl BlocklistChecker) *Ranker {
	return &Ranker{
		config:         cfg,
		blocklist:      bl,
		mmrConfig:      DefaultMMRConfig(),
		collapseConfig: DefaultCollapseConfig(),
		relevanceFloor: relevanceFloorFromConfig(cfg),
	}
}

func (r *Ranker) SetRelevanceFloor(c RelevanceFloorConfig) {
	r.relevanceFloor = c
}

func (r *Ranker) SetMMRConfig(c MMRConfig) {
	r.mmrConfig = c
}

func (r *Ranker) SetCollapseConfig(c CollapseConfig) {
	r.collapseConfig = c
}

func (r *Ranker) SetReranker(rr reranker.Reranker, topN int, alpha float64) {
	r.reranker = rr
	r.rerankTopN = topN
	r.rerankAlpha = alpha
}

func NewWithEnhancements(cfg *config.RankingConfig, bl BlocklistChecker, opts EnhancementOptions) *Ranker {
	r := &Ranker{
		config:    cfg,
		blocklist: bl,
	}

	if opts.EnableEntityRecognition {
		r.entityRecognizer = NewEntityRecognizer()
		r.entityRecognizer.LoadDefaults()
		if opts.EntitiesPath != "" {
			_ = r.entityRecognizer.LoadFromFile(opts.EntitiesPath)
		}
	}

	if opts.EnableSynonymExpansion {
		r.synonymExpander = NewSynonymExpander()
		r.synonymExpander.LoadDefaults()
		if opts.SynonymsPath != "" {
			_ = r.synonymExpander.LoadSynonymsFromFile(opts.SynonymsPath)
		}
		if opts.AcronymsPath != "" {
			_ = r.synonymExpander.LoadAcronymsFromFile(opts.AcronymsPath)
		}
	}

	if opts.EnableDynamicWeights {
		r.dynamicWeights = NewDynamicWeights()
	}

	if opts.EnableDiversity {
		r.diversityEnforcer = NewDiversityEnforcer(opts.DiversityConfig)
	}

	return r
}

type EnhancementOptions struct {
	EnableEntityRecognition bool
	EnableSynonymExpansion  bool
	EnableDynamicWeights    bool
	EnableDiversity         bool

	EntitiesPath    string
	SynonymsPath    string
	AcronymsPath    string
	DiversityConfig DiversityConfig
}

func DefaultEnhancementOptions() EnhancementOptions {
	return EnhancementOptions{
		EnableEntityRecognition: true,
		EnableSynonymExpansion:  true,
		EnableDynamicWeights:    true,
		EnableDiversity:         true,
		DiversityConfig:         DefaultDiversityConfig(),
	}
}

func (r *Ranker) SetEntityRecognizer(er *EntityRecognizer) {
	r.entityRecognizer = er
}

func (r *Ranker) SetSynonymExpander(se *SynonymExpander) {
	r.synonymExpander = se
}

func (r *Ranker) SetDiversityEnforcer(de *DiversityEnforcer) {
	r.diversityEnforcer = de
}

func (r *Ranker) SetDynamicWeights(dw *DynamicWeights) {
	r.dynamicWeights = dw
}

type EngineTally struct {
	Contributed int `json:"contributed"`
	Blocked     int `json:"blocked"`
	Irrelevant  int `json:"irrelevant"`
	Kept        int `json:"kept"`
}

type RankStats struct {
	BlockedFiltered    int
	IrrelevantFiltered int

	PerEngine map[string]EngineTally
}

func (rs *RankStats) tally(engines []string, outcome func(*EngineTally)) {
	if rs == nil {
		return
	}
	if rs.PerEngine == nil {
		rs.PerEngine = make(map[string]EngineTally, len(engines))
	}
	for _, e := range engines {
		if e == "" {
			continue
		}
		t := rs.PerEngine[e]
		t.Contributed++
		outcome(&t)
		rs.PerEngine[e] = t
	}
}

type RankOptions struct {
	Stats            *RankStats
	IncludeBreakdown bool
	Category         string
	EnableDiversity  bool
	UseQueryAnalysis bool
	ExpandSynonyms   bool
}

func (r *Ranker) Rank(ctx context.Context, query string, results []searxng.Result, includeBreakdown bool) []ScoredResult {
	return r.RankWithOptions(ctx, query, results, RankOptions{
		IncludeBreakdown: includeBreakdown,
	})
}

func (r *Ranker) RankWithOptions(ctx context.Context, query string, results []searxng.Result, opts RankOptions) []ScoredResult {
	if len(results) == 0 {
		return []ScoredResult{}
	}

	if r.config.RRF.Enabled {
		return r.rankWithRRF(ctx, query, results, opts)
	}

	return r.rankAdditive(ctx, query, results, opts)
}

func (r *Ranker) rankAdditive(ctx context.Context, query string, results []searxng.Result, opts RankOptions) []ScoredResult {
	queryTerms := urlutil.Tokenize(query)

	var intent QueryIntent
	if opts.Category != "" && opts.Category != "general" {
		intent = QueryIntent{
			Category:   ParseCategory(opts.Category),
			Confidence: 1.0,
		}
	} else {
		intent = DetectIntent(queryTerms)
	}

	scored := make([]ScoredResult, 0, len(results))

	for _, result := range results {
		domain := urlutil.ExtractDomain(result.URL)

		if !r.relevanceFloor.meetsFloor(queryTerms, result.Title, result.Content, result.URL) {
			if opts.Stats != nil {
				opts.Stats.IrrelevantFiltered++
			}
			continue
		}

		breakdown := r.computeBreakdown(queryTerms, result, domain, intent)
		adjustment := breakdown.Total()
		adjustedScore := result.Score + float64(adjustment)

		if adjustedScore < 0 {
			continue
		}

		badges := r.computeBadges(domain, breakdown)

		sr := ScoredResult{
			URL:           result.URL,
			Title:         result.Title,
			Snippet:       result.Content,
			Engines:       result.Engines,
			OriginalScore: result.Score,
			AdjustedScore: adjustedScore,
			Badges:        badges,
		}

		if opts.IncludeBreakdown {
			sr.ScoreBreakdown = &breakdown
		}

		scored = append(scored, sr)
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].AdjustedScore > scored[j].AdjustedScore
	})

	if opts.EnableDiversity && r.diversityEnforcer != nil {
		scored = r.diversityEnforcer.EnforceDiversity(scored)
	}

	return scored
}

func (r *Ranker) computeBreakdown(queryTerms []string, result searxng.Result, domain string, intent QueryIntent) ScoreBreakdown {
	isBlocked := r.blocklist.IsBlocked(domain)
	trancoRank := r.blocklist.GetTrancoRank(domain)

	snippetScore := AnalyzeSnippet(queryTerms, result.Content, &r.config.Snippet)

	freshnessBonus := FreshnessBonus(result.PublishedDate, &r.config.Weights)
	if intent.Category == CategoryNews && freshnessBonus > 0 {
		freshnessBonus = int(float64(freshnessBonus) * 1.5)
	}

	return ScoreBreakdown{
		TitleMatchBonus: TitleMatchBonus(queryTerms, result.Title, &r.config.Weights),
		URLMatchBonus:   URLMatchBonus(queryTerms, result.URL, &r.config.Weights),

		TrancoBonus:          TrancoBonus(trancoRank, &r.config.Weights),
		EngineAgreementBonus: EngineAgreementBonus(result.Engines, &r.config.Weights),
		FreshnessBonus:       freshnessBonus,

		BlocklistPenalty:   BlocklistPenalty(isBlocked, &r.config.Penalties),
		ContentFarmPenalty: ContentFarmPenalty(domain, &r.config.Penalties),
		ParamPenalty:       ExcessiveParamsPenalty(result.URL, &r.config.Penalties),

		SnippetBonus:      snippetScore.Total,
		URLStructureBonus: URLStructureBonus(result.URL, &r.config.URLStructure),

		DetectedCategory: string(intent.Category),
	}
}

func (r *Ranker) computeBadges(domain string, breakdown ScoreBreakdown) Badges {
	trancoRank := r.blocklist.GetTrancoRank(domain)
	isBlocked := r.blocklist.IsBlocked(domain)

	return Badges{
		HasAds:        isBlocked,
		TrackerHeavy:  false,
		KnownSpam:     isBlocked,
		Reputable:     trancoRank > 0 && trancoRank <= 100000,
		ContentFarm:   IsContentFarm(domain),
		Authoritative: trancoRank > 0 && trancoRank <= 10000,
	}
}

func (r *Ranker) rankWithRRF(ctx context.Context, query string, results []searxng.Result, opts RankOptions) []ScoredResult {
	queryTerms := urlutil.Tokenize(query)

	var intent QueryIntent
	if opts.Category != "" && opts.Category != "general" {
		parsedCat := ParseCategory(opts.Category)
		intent = QueryIntent{
			Category:           parsedCat,
			Confidence:         1.0,
			DomainDistribution: QueryCategoryToDomainScores(parsedCat, 1.0),
		}
		intent.PrimaryDomain = intent.DomainDistribution.PrimaryDomain()
	} else {
		intent = DetectIntent(queryTerms)
	}

	rrfConfig := RRFConfig{
		Enabled:       r.config.RRF.Enabled,
		K:             r.config.RRF.K,
		DefaultRank:   r.config.RRF.DefaultRank,
		SignalWeights: r.config.RRF.SignalWeights,
	}

	alignConfig := AlignmentConfig{
		MaxBoost:        r.config.Alignment.MaxBoost,
		MismatchPenalty: r.config.Alignment.MismatchPenalty,
		MinConfidence:   r.config.Alignment.MinConfidence,
	}

	scored := make([]ScoredResult, 0, len(results))
	for i, result := range results {
		domain := urlutil.ExtractDomain(result.URL)

		if r.blocklist.IsBlocked(domain) {
			if opts.Stats != nil {
				opts.Stats.BlockedFiltered++
				opts.Stats.tally(result.Engines, func(t *EngineTally) { t.Blocked++ })
			}
			continue
		}

		if !r.relevanceFloor.meetsFloor(queryTerms, result.Title, result.Content, result.URL) {
			if opts.Stats != nil {
				opts.Stats.IrrelevantFiltered++
				opts.Stats.tally(result.Engines, func(t *EngineTally) { t.Irrelevant++ })
			}
			continue
		}
		if opts.Stats != nil {
			opts.Stats.tally(result.Engines, func(t *EngineTally) { t.Kept++ })
		}

		domainScores := ClassifyDomain(result.Title, result.Content, result.URL)

		alignment := IntentAlignmentScore(intent, domainScores)

		breakdown := r.computeBreakdown(queryTerms, result, domain, intent)
		badges := r.computeBadges(domain, breakdown)

		sr := ScoredResult{
			URL:             result.URL,
			Title:           result.Title,
			Snippet:         result.Content,
			Engines:         result.Engines,
			OriginalScore:   result.Score,
			PublishedDate:   result.PublishedDate,
			DomainScores:    domainScores,
			PrimaryDomain:   domainScores.PrimaryDomain(),
			ContentType:     ClassifyContentType(result.URL, domainScores),
			IntentAlignment: alignment,
			Badges:          badges,
			SignalRanks:     make(map[string]int),
		}

		sr.SignalRanks["searxng_position"] = i + 1

		if opts.IncludeBreakdown {
			sr.ScoreBreakdown = &breakdown
			sr.ScoreBreakdown.DetectedIntent = string(intent.PrimaryDomain)
			sr.ScoreBreakdown.IntentConfidence = intent.Confidence
			sr.ScoreBreakdown.DomainMatch = string(sr.PrimaryDomain)
		}

		scored = append(scored, sr)
	}

	if len(scored) == 0 {
		return scored
	}

	r.computeSignalRanks(scored, queryTerms)
	r.computeFreshnessSignal(scored, intent.Category)

	for i := range scored {
		baseRRF := RRFScore(scored[i].SignalRanks, rrfConfig)
		scored[i].RRFScore = baseRRF

		boost := AlignmentBoostWithConfidence(scored[i].IntentAlignment, intent.Confidence, alignConfig)
		scored[i].FinalScore = baseRRF * boost

		scored[i].AdjustedScore = scored[i].FinalScore * 1000

		if opts.IncludeBreakdown && scored[i].ScoreBreakdown != nil {
			scored[i].ScoreBreakdown.RRFBase = baseRRF
			scored[i].ScoreBreakdown.DomainBoost = boost
			scored[i].ScoreBreakdown.SignalContributions = RRFContributions(scored[i].SignalRanks, rrfConfig)

			textScore := AnalyzeTextRelevance(queryTerms, scored[i].Title, scored[i].Snippet)
			scored[i].ScoreBreakdown.TextRelevance = &TextRelevanceBreakdown{
				TitleMatch:       textScore.TitleMatch,
				SnippetMatch:     textScore.SnippetMatch,
				TermFrequency:    textScore.TermFrequency,
				Proximity:        textScore.Proximity,
				CoreTermCoverage: textScore.CoreTermCoverage,
				Combined:         textScore.Combined,
			}
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].FinalScore > scored[j].FinalScore
	})

	for i := range scored {
		scored[i].RRFRank = i + 1
	}

	if r.reranker != nil && r.rerankAlpha > 0 {
		topN := r.rerankTopN
		if topN <= 0 {
			topN = 30
		}
		scored = ApplyReranker(ctx, query, scored, r.reranker, topN, r.rerankAlpha)
	}

	if r.mmrConfig.Enabled && r.mmrConfig.Lambda > 0 {
		scored = EnforceMMR(query, scored, r.mmrConfig)
	}

	if intent.Category == CategoryNews && r.collapseConfig.Enabled {
		scored = CollapseByTitleSimilarity(scored, r.collapseConfig)
	}

	if opts.EnableDiversity && r.diversityEnforcer != nil {
		scored = r.diversityEnforcer.EnforceDiversity(scored)
	}

	for i := range scored {
		scored[i].FinalRank = i + 1
	}

	return scored
}

func (r *Ranker) computeSignalRanks(scored []ScoredResult, queryTerms []string) {
	bm25Cfg := DefaultBM25Config()
	corpus := make([][]string, len(scored))
	for i := range scored {
		corpus[i] = bm25Tokens(scored[i].Title+" "+scored[i].Snippet, bm25Cfg.TitleWeight, scored[i].Title)
	}
	bm25 := NewBM25Scorer(corpus, bm25Cfg)

	textRanks := RankBySignal(scored, func(sr ScoredResult) float64 {
		return AnalyzeTextRelevanceBM25(queryTerms, bm25, sr.Title, sr.Snippet).Combined
	})
	for i := range scored {
		if rank, ok := textRanks[scored[i].URL]; ok {
			scored[i].SignalRanks["text_relevance"] = rank
		}
	}

	alignmentRanks := RankBySignal(scored, func(sr ScoredResult) float64 {
		return sr.IntentAlignment
	})
	for i := range scored {
		if rank, ok := alignmentRanks[scored[i].URL]; ok {
			scored[i].SignalRanks["domain_alignment"] = rank
		}
	}

	trancoRanks := RankBySignal(scored, func(sr ScoredResult) float64 {
		domain := urlutil.ExtractDomain(sr.URL)
		trancoRank := r.blocklist.GetTrancoRank(domain)
		if trancoRank <= 0 {
			return 0
		}
		return 1.0 / float64(trancoRank)
	})
	for i := range scored {
		if rank, ok := trancoRanks[scored[i].URL]; ok {
			scored[i].SignalRanks["tranco_authority"] = rank
		}
	}

	engineRanks := RankBySignal(scored, func(sr ScoredResult) float64 {
		return float64(len(sr.Engines))
	})
	for i := range scored {
		if rank, ok := engineRanks[scored[i].URL]; ok {
			scored[i].SignalRanks["engine_agreement"] = rank
		}
	}
}

func (r *Ranker) computeFreshnessSignal(scored []ScoredResult, category QueryCategory) {
	type dated struct {
		url   string
		score float64
	}
	dat := make([]dated, 0, len(scored))
	var undated []string
	for _, sr := range scored {
		s := FreshnessScore(sr.PublishedDate, category)
		if s == FreshnessSentinel {
			undated = append(undated, sr.URL)
			continue
		}
		dat = append(dat, dated{url: sr.URL, score: s})
	}
	if len(dat) == 0 {
		return
	}

	for i := 1; i < len(dat); i++ {
		for j := i; j > 0; j-- {
			if dat[j].score > dat[j-1].score ||
				(dat[j].score == dat[j-1].score && dat[j].url < dat[j-1].url) {
				dat[j], dat[j-1] = dat[j-1], dat[j]
			} else {
				break
			}
		}
	}
	urlToRank := make(map[string]int, len(dat))
	rank := 1
	for i, d := range dat {
		if i > 0 && d.score != dat[i-1].score {
			rank = i + 1
		}
		urlToRank[d.url] = rank
	}
	for i := range scored {
		if rk, ok := urlToRank[scored[i].URL]; ok {
			scored[i].SignalRanks["freshness"] = rk
		}
	}

	if category == CategoryNews && len(undated) > 0 {
		worstRank := len(dat) + 1
		for _, url := range undated {
			for i := range scored {
				if scored[i].URL == url {
					scored[i].SignalRanks["freshness"] = worstRank
					break
				}
			}
		}
	}
}
