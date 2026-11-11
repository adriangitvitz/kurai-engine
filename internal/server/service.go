package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/adriangitvitz/kurai/internal/answers"
	"github.com/adriangitvitz/kurai/internal/apierror"
	"github.com/adriangitvitz/kurai/internal/privacy"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
	"github.com/adriangitvitz/kurai/internal/tavily"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

const tavilyFallbackThreshold = 5

const answerBudget = 1000 * time.Millisecond

const answersCachePrefix = "sxa:"

func computeFacets(results []ranker.ScoredResult) map[string]int {
	facets := map[string]int{
		string(ranker.ContentArticle):    0,
		string(ranker.ContentStudy):      0,
		string(ranker.ContentVideo):      0,
		string(ranker.ContentDiscussion): 0,
		string(ranker.ContentGuide):      0,
	}
	for _, r := range results {
		ct := r.ContentType
		if ct == "" {
			ct = ranker.ContentArticle
		}
		facets[string(ct)]++
	}
	return facets
}

func scoredResultsFromAny(items []any) []ranker.ScoredResult {
	out := make([]ranker.ScoredResult, 0, len(items))
	for _, it := range items {
		if sr, ok := it.(ranker.ScoredResult); ok {
			out = append(out, sr)
		}
	}
	return out
}

func convertTavilyResults(results []tavily.Result) []searxng.Result {
	out := make([]searxng.Result, 0, len(results))
	for _, r := range results {
		out = append(out, searxng.Result{
			URL:           r.URL,
			Title:         r.Title,
			Content:       r.Content,
			Score:         r.Score,
			Engines:       []string{"tavily"},
			PublishedDate: r.PublishedDate,
		})
	}
	return out
}

type ServiceError struct {
	Status  int
	Code    string
	Message string
}

func (e *ServiceError) Error() string { return e.Message }

func newServiceError(status int, code, message string) *ServiceError {
	return &ServiceError{Status: status, Code: code, Message: message}
}

type ServiceSearchReq struct {
	UserID   string
	Query    string
	Page     int
	Category string
	Debug    bool

	Language       string
	IncludeDomains []string
	ExcludeDomains []string
	DateFrom       string
	DateTo         string
	DateStrict     bool
	SnippetLength  int
	Limit          int

	ExtractTop int

	KeepUnreachable bool

	ContentType string
}

func validContentType(s string) string {
	switch s {
	case "article", "study", "video", "discussion", "guide":
		return s
	default:
		return ""
	}
}

func (s *Server) ServiceSearch(ctx context.Context, req ServiceSearchReq) (*SearchResponse, error) {
	if req.Query == "" {
		return nil, newServiceError(400, apierror.CodeMissingQuery, "a non-empty search query is required")
	}

	page := req.Page
	if page < 1 {
		page = 1
	}

	start := time.Now()

	var (
		results            []ranker.ScoredResult
		totalFiltered      int
		irrelevantFiltered int
		fromCache          bool
		totalResults       int
		totalPages         int
		enginesUsed        []string
		facets             map[string]int
	)

	req.ContentType = validContentType(req.ContentType)

	var cardsCh chan *answers.Block
	if s.answers != nil {
		cardsCh = make(chan *answers.Block, 1)
		cardsCtx, cardsCancel := context.WithTimeout(ctx, 30*time.Second)
		defer cardsCancel()
		go func() { cardsCh <- s.answers.ResolveCards(cardsCtx, req.UserID, req.Query) }()
	}

	var upstreamExtras *answers.Block

	var fullSet []ranker.ScoredResult

	upstreamQuery := req.Query
	if sites := siteOperator(req.IncludeDomains); sites != "" {
		upstreamQuery += " " + sites
	}

	timeRange := upstreamTimeRange(req.DateFrom, start)
	fetchKey := upstreamQuery
	if timeRange != "" {
		fetchKey += "\x00time_range=" + timeRange
	}

	if all, ok := s.cache.Get(req.UserID, fetchKey); ok {
		fullSet = scoredResultsFromAny(all)
		fromCache = true
		if s.answers != nil {
			if items, ok := s.cache.Get(req.UserID, answersCachePrefix+fetchKey); ok && len(items) > 0 {
				if b, ok := items[0].(answers.Block); ok {
					upstreamExtras = &b
				}
			}
		}
	} else if s.searxClient != nil {
		var allResults []searxng.Result

		for p := 1; p <= s.prefetchPages; p++ {
			if p > 1 {
				privacy.TimingJitter(50*time.Millisecond, 150*time.Millisecond)
			}

			pageCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			opts := &searxng.SearchOptions{Page: p, Language: req.Language, TimeRange: timeRange}
			searxResp, err := s.searxClient.Search(pageCtx, upstreamQuery, opts)
			cancel()

			if err != nil {
				slog.Error("searxng search failed", "error", err, "query", req.Query, "page", p)
				break
			}
			slog.Debug("searxng page fetched", "query", req.Query, "page", p, "results", len(searxResp.Results))

			allResults = append(allResults, searxResp.Results...)

			if s.answers != nil && upstreamExtras == nil {
				upstreamExtras = answers.ExtractUpstream(searxResp, s.answers.InfoboxEnabled())
			}

			if len(searxResp.Results) == 0 {
				slog.Debug("searxng pagination exhausted", "query", req.Query, "page", p)
				break
			}
		}

		slog.Debug("searxng fetch complete", "query", req.Query, "total_results", len(allResults))

		if s.tavily != nil && len(allResults) < tavilyFallbackThreshold {
			tavilyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			if s.tavily.Available(tavilyCtx) {
				tavilyResults, err := s.tavily.Search(tavilyCtx, req.Query, tavily.SearchParams{
					IncludeDomains: req.IncludeDomains,
					ExcludeDomains: req.ExcludeDomains,
					StartDate:      req.DateFrom,
					EndDate:        req.DateTo,
				})
				if err != nil {
					slog.Warn("tavily fallback failed", "error", err, "query", req.Query)
				} else {
					slog.Debug("tavily fallback used", "query", req.Query, "results", len(tavilyResults))
					allResults = append(allResults, convertTavilyResults(tavilyResults)...)
				}
			}
			cancel()
		}

		allResults = deduplicateResults(allResults)
		inferPublishedDates(allResults, start)

		preRankCount := len(allResults)
		var rankStats ranker.RankStats
		allRanked := s.ranker.RankWithOptions(ctx, req.Query, allResults, ranker.RankOptions{
			IncludeBreakdown: req.Debug,
			Category:         req.Category,
			Stats:            &rankStats,
		})
		totalFiltered = preRankCount - len(allRanked) - rankStats.IrrelevantFiltered
		if totalFiltered < 0 {
			totalFiltered = 0
		}
		irrelevantFiltered = rankStats.IrrelevantFiltered
		allRanked = collapseSameSiteTitles(allRanked)
		s.engineStats.record(&rankStats)
		if len(rankStats.PerEngine) > 0 {
			slog.Debug("engine filtering", "query", req.Query, "per_engine", rankStats.PerEngine)
		}

		cacheData := make([]any, len(allRanked))
		for i, r := range allRanked {
			cacheData[i] = r
		}
		s.cache.Set(req.UserID, fetchKey, cacheData)

		if s.answers != nil && upstreamExtras != nil {
			s.cache.Set(req.UserID, answersCachePrefix+fetchKey, []any{*upstreamExtras})
		}

		fullSet = allRanked
	}

	domainDateFiltered, filterStats := applyPostRankFilters(fullSet, req)
	facets = computeFacets(domainDateFiltered)
	enginesUsed = collectScoredEngines(domainDateFiltered)

	pageSize := s.pageSize
	if req.Limit > 0 && req.Limit < pageSize {
		pageSize = req.Limit
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	filtered := filterByContentType(domainDateFiltered, req.ContentType)
	totalResults = len(filtered)
	totalPages = (totalResults + pageSize - 1) / pageSize
	results = pageSlice(filtered, page, pageSize)

	if s.prober != nil {
		drop := s.probeDrop && !req.KeepUnreachable
		probed := computeFacets(results)
		results = s.prober.probeAndFilter(ctx, results, drop)
		for contentType, kept := range computeFacets(results) {
			dropped := probed[contentType] - kept
			facets[contentType] -= dropped
			totalResults -= dropped
		}
		totalPages = (totalResults + pageSize - 1) / pageSize
	}

	if req.ExtractTop > 0 {
		s.attachExtractedContent(ctx, results, req.ExtractTop)
	}

	var answerBlock *answers.Block
	if s.answers != nil {
		var cards *answers.Block
		select {
		case cards = <-cardsCh:
		case <-time.After(answerBudget):
			slog.Warn("answers: resolution exceeded budget, degrading to plain results",
				"code", apierror.CodeFetchTimeout, "request_id", RequestIDFromContext(ctx))
		}
		answerBlock = answers.Finalize(answers.Merge(cards, upstreamExtras))
	}

	resolvedPageSize := s.pageSize
	if req.Limit > 0 && req.Limit < resolvedPageSize {
		resolvedPageSize = req.Limit
	}
	resp := &SearchResponse{
		Query:   req.Query,
		Answers: answerBlock,
		Results: results,
		Meta: SearchMeta{
			Total:              totalResults,
			Page:               page,
			PageSize:           resolvedPageSize,
			TotalPages:         totalPages,
			BlocklistFiltered:  totalFiltered,
			IrrelevantFiltered: irrelevantFiltered,
			ProcessingTimeMs:   time.Since(start).Milliseconds(),
			FromCache:          fromCache,
			Engines:            enginesUsed,
			Facets:             facets,
			OffTopicFiltered:   filterStats.OffTopicFiltered,
			UndatedFiltered:    filterStats.UndatedFiltered,
			PhraseFiltered:     filterStats.PhraseFiltered,
			PhraseRelaxed:      filterStats.PhraseRelaxed,
			TimeRange:          timeRange,
		},
		Envelope: Envelope{
			RequestID:  RequestIDFromContext(ctx),
			Pagination: buildPagination(totalResults, page, resolvedPageSize),
		},
	}
	if fromCache {
		now := time.Now()
		resp.Envelope.CachedAt = &now
	}

	if req.Debug {
		resp.Intent = computeIntent(req.Query, req.Category)
	}

	return resp, nil
}

type ServiceNewsReq struct {
	UserID        string
	Query         string
	Page          int
	PageSize      int
	TimeRange     string
	Category      string
	EnableCluster bool
}

func (s *Server) ServiceNews(ctx context.Context, req ServiceNewsReq) (*NewsSearchResponse, error) {
	if req.Query == "" {
		return nil, newServiceError(400, apierror.CodeMissingQuery, "a non-empty search query is required")
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	timeRange := ranker.ParseTimeRange(req.TimeRange)
	start := time.Now()

	var (
		results       []ranker.NewsResult
		clusters      []ranker.StoryCluster
		totalResults  int
		totalPages    int
		trendingCount int
		fromCache     bool
		enginesUsed   []string
	)

	cacheKey := "news:" + req.Query + ":" + timeRange
	if cachedResults, total, ok := s.cache.GetPage(req.UserID, cacheKey, page, pageSize); ok {
		results = make([]ranker.NewsResult, 0, len(cachedResults))
		for _, r := range cachedResults {
			if nr, ok := r.(ranker.NewsResult); ok {
				results = append(results, nr)
			}
		}
		fromCache = true
		totalResults = total
		totalPages = (total + pageSize - 1) / pageSize
	} else if s.searxClient != nil {
		var allResults []searxng.Result

		for p := 1; p <= 3; p++ {
			if p > 1 {
				privacy.TimingJitter(50*time.Millisecond, 150*time.Millisecond)
			}
			pageCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			searxResp, err := s.searxClient.Search(pageCtx, req.Query, &searxng.SearchOptions{
				Page:       p,
				Categories: []string{"news"},
				TimeRange:  timeRange,
			})
			cancel()
			if err != nil {
				slog.Error("searxng news search failed", "error", err, "query", req.Query, "page", p)
				break
			}
			slog.Debug("searxng news page fetched", "query", req.Query, "page", p, "results", len(searxResp.Results))
			allResults = append(allResults, searxResp.Results...)
		}

		newsResults := convertToNewsResults(allResults)
		newsResults = ranker.FilterByTimeRange(newsResults, timeRange)

		queryTerms := urlutil.Tokenize(req.Query)
		newsRanker := ranker.NewNewsRanker(ranker.DefaultNewsRankingConfig(), s.blocklist)
		newsResults = newsRanker.RankNews(queryTerms, newsResults)

		if req.EnableCluster {
			clusterer := ranker.NewNewsClusterer(ranker.DefaultClusterConfig())
			clusters = clusterer.ClusterStories(newsResults)
			for _, c := range clusters {
				if c.IsTrending {
					trendingCount++
				}
			}
		}

		enginesUsed = collectNewsEngines(newsResults)

		cacheData := make([]any, len(newsResults))
		for i, r := range newsResults {
			cacheData[i] = r
		}
		s.cache.Set(req.UserID, cacheKey, cacheData)

		totalResults = len(newsResults)
		totalPages = (totalResults + pageSize - 1) / pageSize
		results = pageSliceNews(newsResults, page, pageSize)
	}

	resp := &NewsSearchResponse{
		Query:   req.Query,
		Results: results,
		Meta: NewsSearchMeta{
			Total:            totalResults,
			Page:             page,
			PageSize:         pageSize,
			TotalPages:       totalPages,
			TimeRange:        timeRange,
			Category:         req.Category,
			TrendingCount:    trendingCount,
			ProcessingTimeMs: time.Since(start).Milliseconds(),
			FromCache:        fromCache,
			EnginesUsed:      enginesUsed,
		},
	}

	if req.EnableCluster && page == 1 && len(clusters) > 0 {
		resp.Clusters = clusters
	}

	return resp, nil
}

func (s *Server) ServiceTrending(ctx context.Context, userID, category string) (*TrendingResponse, error) {
	start := time.Now()
	cacheKey := "trending:" + category
	var clusters []ranker.StoryCluster
	fromCache := false

	if cachedData, _, ok := s.cache.GetPage(userID, cacheKey, 1, 100); ok {
		for _, c := range cachedData {
			if cluster, ok := c.(ranker.StoryCluster); ok {
				clusters = append(clusters, cluster)
			}
		}
		fromCache = true
	} else if s.searxClient != nil {
		fetchCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		query := "news"
		if category != "" && category != "all" {
			query = category + " news"
		}

		searxResp, err := s.searxClient.Search(fetchCtx, query, &searxng.SearchOptions{
			Categories: []string{"news"},
		})
		if err != nil {
			slog.Error("searxng trending search failed", "error", err)
			return nil, newServiceError(502, apierror.CodeUpstreamUnavailable, "failed to fetch trending news")
		}

		newsResults := convertToNewsResults(searxResp.Results)
		newsRanker := ranker.NewNewsRanker(ranker.DefaultNewsRankingConfig(), s.blocklist)
		newsResults = newsRanker.RankNews([]string{}, newsResults)

		clusterer := ranker.NewNewsClusterer(ranker.DefaultClusterConfig())
		allClusters := clusterer.ClusterStories(newsResults)
		for _, c := range allClusters {
			if c.IsTrending {
				clusters = append(clusters, c)
			}
		}

		cacheData := make([]any, len(clusters))
		for i, c := range clusters {
			cacheData[i] = c
		}
		s.cache.Set(userID, cacheKey, cacheData)
	}

	return &TrendingResponse{
		Clusters: clusters,
		Meta: TrendingMeta{
			Total:            len(clusters),
			Category:         category,
			ProcessingTimeMs: time.Since(start).Milliseconds(),
			FromCache:        fromCache,
		},
	}, nil
}

type ServiceImagesReq struct {
	UserID     string
	Query      string
	Page       int
	PageSize   int
	SafeSearch int
	Size       string
}

func (s *Server) ServiceImages(ctx context.Context, req ServiceImagesReq) (*ImageSearchResponse, error) {
	if req.Query == "" {
		return nil, newServiceError(400, apierror.CodeMissingQuery, "a non-empty search query is required")
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 30
	}
	if pageSize > 100 {
		pageSize = 100
	}
	safeSearch := ranker.ValidateSafeSearch(req.SafeSearch)
	size := ranker.ParseImageSize(req.Size)

	start := time.Now()

	var (
		results      []ranker.ImageResult
		totalResults int
		totalPages   int
		fromCache    bool
		enginesUsed  []string
	)

	cacheKey := "images:" + req.Query
	if cachedResults, total, ok := s.cache.GetPage(req.UserID, cacheKey, page, pageSize); ok {
		results = make([]ranker.ImageResult, 0, len(cachedResults))
		for _, r := range cachedResults {
			if ir, ok := r.(ranker.ImageResult); ok {
				results = append(results, ir)
			}
		}
		fromCache = true
		totalResults = total
		totalPages = (total + pageSize - 1) / pageSize
	} else if s.searxClient != nil {
		var allResults []searxng.Result
		for p := 1; p <= 2; p++ {
			if p > 1 {
				privacy.TimingJitter(50*time.Millisecond, 150*time.Millisecond)
			}
			pageCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			searxResp, err := s.searxClient.Search(pageCtx, req.Query, &searxng.SearchOptions{
				Page:       p,
				Categories: []string{"images"},
				SafeSearch: safeSearch,
			})
			cancel()
			if err != nil {
				slog.Error("searxng image search failed", "error", err, "query", req.Query, "page", p)
				break
			}
			slog.Debug("searxng image page fetched", "query", req.Query, "page", p, "results", len(searxResp.Results))
			allResults = append(allResults, searxResp.Results...)
		}

		imageResults := convertToImageResults(allResults)
		queryTerms := urlutil.Tokenize(req.Query)
		imageRanker := ranker.NewImageRanker(ranker.DefaultImageRankingConfig(), s.blocklist)
		imageResults = imageRanker.RankImages(queryTerms, imageResults)
		if size != "" {
			imageResults = imageRanker.FilterBySize(imageResults, size)
		}

		enginesUsed = collectImageEngines(imageResults)

		cacheData := make([]any, len(imageResults))
		for i, r := range imageResults {
			cacheData[i] = r
		}
		s.cache.Set(req.UserID, cacheKey, cacheData)

		totalResults = len(imageResults)
		totalPages = (totalResults + pageSize - 1) / pageSize
		results = pageSliceImages(imageResults, page, pageSize)
	}

	return &ImageSearchResponse{
		Query:   req.Query,
		Results: results,
		Meta: ImageSearchMeta{
			Total:            totalResults,
			Page:             page,
			PageSize:         pageSize,
			TotalPages:       totalPages,
			SafeSearch:       safeSearch,
			ProcessingTimeMs: time.Since(start).Milliseconds(),
			FromCache:        fromCache,
			EnginesUsed:      enginesUsed,
		},
	}, nil
}

func pageSlice[T any](all []T, page, size int) []T {
	startIdx := (page - 1) * size
	if startIdx >= len(all) {
		return []T{}
	}
	endIdx := startIdx + size
	if endIdx > len(all) {
		endIdx = len(all)
	}
	return all[startIdx:endIdx]
}

func pageSliceNews(all []ranker.NewsResult, page, size int) []ranker.NewsResult {
	return pageSlice(all, page, size)
}

func pageSliceImages(all []ranker.ImageResult, page, size int) []ranker.ImageResult {
	return pageSlice(all, page, size)
}

func collectNewsEngines(results []ranker.NewsResult) []string {
	set := make(map[string]struct{})
	for _, n := range results {
		for _, e := range n.Engines {
			set[e] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for e := range set {
		out = append(out, e)
	}
	return out
}

func collectScoredEngines(results []ranker.ScoredResult) []string {
	set := make(map[string]struct{})
	for _, r := range results {
		for _, e := range r.Engines {
			set[e] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for e := range set {
		out = append(out, e)
	}
	return out
}

func collectImageEngines(results []ranker.ImageResult) []string {
	set := make(map[string]struct{})
	for _, img := range results {
		for _, e := range img.Engines {
			set[e] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for e := range set {
		out = append(out, e)
	}
	return out
}

func computeIntent(query, category string) *IntentOutput {
	queryTerms := urlutil.Tokenize(query)

	var intent ranker.QueryIntent
	if category != "" && category != "general" {
		parsedCat := ranker.ParseCategory(category)
		intent = ranker.QueryIntent{
			Category:           parsedCat,
			Confidence:         1.0,
			DomainDistribution: ranker.QueryCategoryToDomainScores(parsedCat, 1.0),
		}
		intent.PrimaryDomain = intent.DomainDistribution.PrimaryDomain()
	} else {
		intent = ranker.DetectIntent(queryTerms)
	}

	distribution := make(map[string]float64, len(intent.DomainDistribution))
	for domain, score := range intent.DomainDistribution {
		distribution[string(domain)] = score
	}

	return &IntentOutput{
		PrimaryDomain: string(intent.PrimaryDomain),
		Category:      string(intent.Category),
		Confidence:    intent.Confidence,
		Distribution:  distribution,
		Modifiers: &IntentModifiers{
			IsQuestion:   intent.Modifiers.IsQuestion,
			IsTutorial:   intent.Modifiers.IsTutorial,
			IsRecent:     intent.Modifiers.IsRecent,
			IsComparison: intent.Modifiers.IsComparison,
		},
	}
}
