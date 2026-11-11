package ranker

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

type NewsResult struct {
	URL           string     `json:"url"`
	Title         string     `json:"title"`
	Snippet       string     `json:"snippet"`
	Source        string     `json:"source"`
	SourceDomain  string     `json:"source_domain"`
	Author        string     `json:"author,omitempty"`
	PublishedDate *time.Time `json:"published_date,omitempty"`
	PublishedAgo  string     `json:"published_ago"`
	Thumbnail     string     `json:"thumbnail,omitempty"`
	Category      string     `json:"category,omitempty"`
	Engines       []string   `json:"engines"`

	Badges NewsBadges `json:"badges"`

	RecencyScore     float64 `json:"recency_score"`
	CredibilityScore float64 `json:"credibility_score"`
	RelevanceScore   float64 `json:"relevance_score"`
	FinalScore       float64 `json:"final_score"`

	ClusterID   string `json:"cluster_id,omitempty"`
	IsMainStory bool   `json:"is_main_story,omitempty"`
}

type NewsBadges struct {
	IsBreaking      bool `json:"is_breaking"`
	IsTrending      bool `json:"is_trending"`
	FromMajorSource bool `json:"from_major_source"`
	IsOpinion       bool `json:"is_opinion"`
	IsAnalysis      bool `json:"is_analysis"`
}

type NewsRankingConfig struct {
	RecencyWeight     float64 `yaml:"recency_weight"`
	CredibilityWeight float64 `yaml:"credibility_weight"`
	RelevanceWeight   float64 `yaml:"relevance_weight"`

	RecencyHalfLife   time.Duration `yaml:"recency_half_life"`
	BreakingThreshold time.Duration `yaml:"breaking_threshold"`

	MajorSourceThreshold int `yaml:"major_source_threshold"`
}

func DefaultNewsRankingConfig() NewsRankingConfig {
	return NewsRankingConfig{
		RecencyWeight:        0.4,
		CredibilityWeight:    0.35,
		RelevanceWeight:      0.25,
		RecencyHalfLife:      6 * time.Hour,
		BreakingThreshold:    4 * time.Hour,
		MajorSourceThreshold: 50000,
	}
}

type NewsRanker struct {
	config    NewsRankingConfig
	blocklist BlocklistChecker
}

func NewNewsRanker(config NewsRankingConfig, blocklist BlocklistChecker) *NewsRanker {
	return &NewsRanker{
		config:    config,
		blocklist: blocklist,
	}
}

func (nr *NewsRanker) ScoreRecency(publishedDate time.Time) float64 {
	if publishedDate.IsZero() {
		return 0.3
	}

	age := time.Since(publishedDate)

	if age < 1*time.Hour {
		return 1.0
	}

	halfLife := nr.config.RecencyHalfLife.Hours()
	ageHours := age.Hours()

	score := math.Pow(2, -ageHours/halfLife)

	if score < 0.1 {
		return 0.1
	}

	return score
}

func (nr *NewsRanker) ScoreSourceCredibility(domain string) float64 {
	if nr.blocklist != nil && nr.blocklist.IsBlocked(domain) {
		return 0.0
	}

	tierScores := map[string]float64{
		"reuters.com":        1.0,
		"apnews.com":         1.0,
		"bbc.com":            0.95,
		"bbc.co.uk":          0.95,
		"nytimes.com":        0.95,
		"washingtonpost.com": 0.90,
		"theguardian.com":    0.90,
		"wsj.com":            0.90,
		"economist.com":      0.90,
		"npr.org":            0.90,
		"pbs.org":            0.90,

		"cnn.com":            0.80,
		"nbcnews.com":        0.80,
		"abcnews.go.com":     0.80,
		"cbsnews.com":        0.80,
		"usatoday.com":       0.75,
		"latimes.com":        0.80,
		"chicagotribune.com": 0.75,
		"bostonglobe.com":    0.80,
		"sfchronicle.com":    0.75,

		"techcrunch.com":       0.85,
		"arstechnica.com":      0.85,
		"theverge.com":         0.80,
		"wired.com":            0.80,
		"zdnet.com":            0.75,
		"cnet.com":             0.75,
		"engadget.com":         0.75,
		"gizmodo.com":          0.70,
		"9to5mac.com":          0.75,
		"9to5google.com":       0.75,
		"macrumors.com":        0.75,
		"hackernews.com":       0.70,
		"news.ycombinator.com": 0.70,

		"bloomberg.com":       0.90,
		"ft.com":              0.90,
		"cnbc.com":            0.80,
		"marketwatch.com":     0.75,
		"forbes.com":          0.70,
		"businessinsider.com": 0.65,

		"medium.com":       0.50,
		"substack.com":     0.55,
		"huffpost.com":     0.55,
		"buzzfeednews.com": 0.55,
	}

	if score, ok := tierScores[domain]; ok {
		return score
	}

	if nr.blocklist != nil {
		trancoRank := nr.blocklist.GetTrancoRank(domain)
		if trancoRank > 0 {
			switch {
			case trancoRank <= 1000:
				return 0.85
			case trancoRank <= 10000:
				return 0.75
			case trancoRank <= 50000:
				return 0.65
			case trancoRank <= 100000:
				return 0.55
			default:
				return 0.45
			}
		}
	}

	return 0.4
}

func (nr *NewsRanker) DetectBreakingNews(news NewsResult) bool {
	if news.PublishedDate != nil && !news.PublishedDate.IsZero() {
		if time.Since(*news.PublishedDate) < nr.config.BreakingThreshold {
			return true
		}
	}

	titleLower := strings.ToLower(news.Title)
	breakingIndicators := []string{
		"breaking:",
		"breaking news",
		"just in:",
		"developing:",
		"urgent:",
		"flash:",
		"live updates",
		"just now",
		"happening now",
	}

	for _, indicator := range breakingIndicators {
		if strings.Contains(titleLower, indicator) {
			return true
		}
	}

	return false
}

func (nr *NewsRanker) DetectOpinion(news NewsResult) bool {
	titleLower := strings.ToLower(news.Title)
	urlLower := strings.ToLower(news.URL)

	opinionIndicators := []string{
		"opinion:",
		"editorial:",
		"op-ed:",
		"commentary:",
		"perspective:",
		"/opinion/",
		"/opinions/",
		"/editorial/",
		"/commentary/",
	}

	combined := titleLower + " " + urlLower

	for _, indicator := range opinionIndicators {
		if strings.Contains(combined, indicator) {
			return true
		}
	}

	return false
}

func (nr *NewsRanker) DetectAnalysis(news NewsResult) bool {
	titleLower := strings.ToLower(news.Title)
	urlLower := strings.ToLower(news.URL)

	analysisIndicators := []string{
		"analysis:",
		"explainer:",
		"in-depth:",
		"deep dive:",
		"what this means",
		"/analysis/",
		"/explainer/",
	}

	combined := titleLower + " " + urlLower

	for _, indicator := range analysisIndicators {
		if strings.Contains(combined, indicator) {
			return true
		}
	}

	return false
}

func (nr *NewsRanker) IsMajorSource(domain string) bool {
	if nr.blocklist != nil {
		trancoRank := nr.blocklist.GetTrancoRank(domain)
		if trancoRank > 0 && trancoRank <= nr.config.MajorSourceThreshold {
			return true
		}
	}

	majorSources := map[string]bool{
		"reuters.com": true, "apnews.com": true, "bbc.com": true,
		"bbc.co.uk": true, "nytimes.com": true, "washingtonpost.com": true,
		"theguardian.com": true, "wsj.com": true, "economist.com": true,
		"cnn.com": true, "bloomberg.com": true, "ft.com": true,
	}

	return majorSources[domain]
}

func (nr *NewsRanker) RankNews(queryTerms []string, news []NewsResult) []NewsResult {
	for i := range news {
		domain := news[i].SourceDomain
		if domain == "" {
			domain = urlutil.ExtractDomain(news[i].URL)
			news[i].SourceDomain = domain
		}

		var pd time.Time
		if news[i].PublishedDate != nil {
			pd = *news[i].PublishedDate
		}
		news[i].RecencyScore = nr.ScoreRecency(pd)
		news[i].CredibilityScore = nr.ScoreSourceCredibility(domain)
		news[i].RelevanceScore = nr.scoreRelevance(queryTerms, news[i])

		news[i].Badges.IsBreaking = nr.DetectBreakingNews(news[i])
		news[i].Badges.IsOpinion = nr.DetectOpinion(news[i])
		news[i].Badges.IsAnalysis = nr.DetectAnalysis(news[i])
		news[i].Badges.FromMajorSource = nr.IsMajorSource(domain)

		news[i].FinalScore = news[i].RecencyScore*nr.config.RecencyWeight +
			news[i].CredibilityScore*nr.config.CredibilityWeight +
			news[i].RelevanceScore*nr.config.RelevanceWeight

		if news[i].Badges.IsBreaking {
			news[i].FinalScore *= 1.2
		}

		if news[i].PublishedDate != nil && !news[i].PublishedDate.IsZero() {
			news[i].PublishedAgo = formatTimeAgo(*news[i].PublishedDate)
		}
	}

	sortNewsByScore(news)

	return news
}

func (nr *NewsRanker) scoreRelevance(queryTerms []string, news NewsResult) float64 {
	if len(queryTerms) == 0 {
		return 0.5
	}

	text := strings.ToLower(news.Title + " " + news.Snippet)

	matchCount := 0
	for _, term := range queryTerms {
		if strings.Contains(text, strings.ToLower(term)) {
			matchCount++
		}
	}

	return float64(matchCount) / float64(len(queryTerms))
}

func sortNewsByScore(news []NewsResult) {
	for i := 1; i < len(news); i++ {
		for j := i; j > 0 && news[j].FinalScore > news[j-1].FinalScore; j-- {
			news[j], news[j-1] = news[j-1], news[j]
		}
	}
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return strconv.Itoa(mins) + " minutes ago"
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return strconv.Itoa(hours) + " hours ago"
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return strconv.Itoa(days) + " days ago"
	case duration < 30*24*time.Hour:
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return strconv.Itoa(weeks) + " weeks ago"
	default:
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return strconv.Itoa(months) + " months ago"
	}
}

type NewsSearchOptions struct {
	TimeRange        string `json:"time_range"`
	Category         string `json:"category"`
	EnableClustering bool   `json:"enable_clustering"`
	OnlyMajorSources bool   `json:"only_major_sources"`
}

func DefaultNewsSearchOptions() NewsSearchOptions {
	return NewsSearchOptions{
		TimeRange:        "week",
		EnableClustering: true,
		OnlyMajorSources: false,
	}
}

func ParseTimeRange(s string) string {
	switch strings.ToLower(s) {
	case "day", "d", "24h":
		return "day"
	case "week", "w", "7d":
		return "week"
	case "month", "m", "30d":
		return "month"
	case "year", "y", "365d":
		return "year"
	default:
		return "week"
	}
}

func FilterByTimeRange(news []NewsResult, timeRange string) []NewsResult {
	var maxAge time.Duration

	switch timeRange {
	case "day":
		maxAge = 24 * time.Hour
	case "week":
		maxAge = 7 * 24 * time.Hour
	case "month":
		maxAge = 30 * 24 * time.Hour
	case "year":
		maxAge = 365 * 24 * time.Hour
	default:
		return news
	}

	var filtered []NewsResult
	now := time.Now()

	for _, n := range news {
		if n.PublishedDate == nil || n.PublishedDate.IsZero() {
			filtered = append(filtered, n)
		} else if now.Sub(*n.PublishedDate) <= maxAge {
			filtered = append(filtered, n)
		}
	}

	return filtered
}

func (nr *NewsRanker) FilterByMajorSources(news []NewsResult) []NewsResult {
	var filtered []NewsResult

	for _, n := range news {
		domain := n.SourceDomain
		if domain == "" {
			domain = urlutil.ExtractDomain(n.URL)
		}

		if nr.IsMajorSource(domain) {
			filtered = append(filtered, n)
		}
	}

	return filtered
}
