package ranker

import (
	"bufio"
	"math"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

var (
	contentFarmDomains = map[string]struct{}{}
	contentFarmMu      sync.RWMutex
)

var defaultContentFarms = []string{
	"about.com",
	"buzzfeed.com",
	"businessinsider.com",
	"ehow.com",
	"examiner.com",
	"forbes.com",
	"howstuffworks.com",
	"lifehack.org",
	"thebalance.com",
	"thoughtco.com",
	"wikihow.com",
}

func LoadContentFarms(path string) error {
	contentFarmMu.Lock()
	defer contentFarmMu.Unlock()

	contentFarmDomains = make(map[string]struct{})

	if path != "" {
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				contentFarmDomains[strings.ToLower(line)] = struct{}{}
			}
			if err := scanner.Err(); err != nil {
				return err
			}
			if len(contentFarmDomains) > 0 {
				return nil
			}
		}
	}

	for _, domain := range defaultContentFarms {
		contentFarmDomains[domain] = struct{}{}
	}
	return nil
}

func TitleMatchBonus(queryTerms []string, title string, weights *config.WeightConfig) int {
	if len(queryTerms) == 0 {
		return 0
	}

	titleLower := strings.ToLower(title)
	matchCount := 0

	for _, term := range queryTerms {
		if strings.Contains(titleLower, strings.ToLower(term)) {
			matchCount++
		}
	}

	if matchCount == 0 {
		return 0
	}

	matchRatio := float64(matchCount) / float64(len(queryTerms))

	switch {
	case matchRatio >= 1.0:
		return weights.TitleMatch
	case matchRatio >= 0.75:
		return (weights.TitleMatch * 3) / 4
	case matchRatio >= 0.5:
		return weights.TitleMatch / 2
	default:
		return weights.TitleMatch / 4
	}
}

func URLMatchBonus(queryTerms []string, rawURL string, weights *config.WeightConfig) int {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}

	pathTerms := urlutil.TokenizePath(parsed.Path)
	if urlutil.HasTermOverlap(pathTerms, queryTerms) {
		return weights.URLMatch
	}
	return 0
}

func TrancoBonus(rank int, weights *config.WeightConfig) int {
	if rank <= 0 {
		return 0
	}

	switch {
	case rank <= 10000:
		return weights.TrancoTop10k
	case rank <= 100000:
		return weights.TrancoTop100k
	case rank <= 1000000:
		return weights.TrancoTop1M
	default:
		return 0
	}
}

func EngineAgreementBonus(engines []string, weights *config.WeightConfig) int {
	if len(engines) <= 1 {
		return 0
	}
	return (len(engines) - 1) * weights.EngineAgreement
}

func FreshnessBonus(publishedDate string, weights *config.WeightConfig) int {
	if publishedDate == "" {
		return 0
	}

	var parsed time.Time
	var err error

	formats := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"Jan 2, 2006",
		"January 2, 2006",
	}

	for _, format := range formats {
		parsed, err = time.Parse(format, publishedDate)
		if err == nil {
			break
		}
	}

	if err != nil {
		return 0
	}

	age := time.Since(parsed)
	if age < 30*24*time.Hour {
		return weights.Freshness
	}

	return 0
}

func BlocklistPenalty(isBlocked bool, penalties *config.PenaltyConfig) int {
	if isBlocked {
		return penalties.Blocklist
	}
	return 0
}

func ContentFarmPenalty(domain string, penalties *config.PenaltyConfig) int {
	if IsContentFarm(domain) {
		return penalties.ContentFarm
	}
	return 0
}

func IsContentFarm(domain string) bool {
	if domain == "" {
		return false
	}
	contentFarmMu.RLock()
	defer contentFarmMu.RUnlock()
	_, exists := contentFarmDomains[strings.ToLower(domain)]
	return exists
}

func ExcessiveParamsPenalty(rawURL string, penalties *config.PenaltyConfig) int {
	count := urlutil.CountURLParams(rawURL)
	if count > penalties.ParamThreshold {
		return penalties.ExcessiveParams
	}
	return 0
}

var documentationPatterns = []string{
	"/docs/",
	"/doc/",
	"/documentation/",
	"/api/",
	"/reference/",
	"/guide/",
	"/manual/",
	"/handbook/",
	"/tutorial/",
	"/learn/",
	"/getting-started/",
	"/quickstart/",
}

var githubCodePatterns = []string{
	"/blob/",
	"/tree/",
	"/raw/",
}

var githubIssuePatterns = []string{
	"/issues/",
	"/issue/",
	"/pull/",
	"/discussions/",
	"/wiki/",
	"/releases/",
	"/commits/",
	"/compare/",
}

var aggregatorPatterns = []string{
	"/search",
	"/tag/",
	"/tags/",
	"/category/",
	"/categories/",
	"/archive/",
	"/archives/",
	"/page/",
	"/topic/",
	"/topics/",
	"/collection/",
	"/collections/",
	"/explore/",
	"/trending/",
	"/popular/",
	"/latest/",
	"/feed/",
	"/rss/",
	"/sitemap",
	"/index.php",
}

func URLStructureBonus(rawURL string, cfg *config.URLStructureConfig) int {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}

	pathLower := strings.ToLower(parsed.Path)
	domain := strings.ToLower(parsed.Host)

	for _, pattern := range aggregatorPatterns {
		if strings.Contains(pathLower, pattern) {
			return cfg.AggregatorPenalty
		}
	}

	for _, pattern := range documentationPatterns {
		if strings.Contains(pathLower, pattern) {
			return cfg.DocumentationPath
		}
	}

	if strings.Contains(domain, "github.com") || strings.Contains(domain, "gitlab.com") {
		for _, pattern := range githubCodePatterns {
			if strings.Contains(pathLower, pattern) {
				return cfg.GitHubCode
			}
		}

		for _, pattern := range githubIssuePatterns {
			if strings.Contains(pathLower, pattern) {
				return cfg.GitHubIssue
			}
		}
	}

	return 0
}

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "of": true, "in": true,
	"to": true, "for": true, "on": true, "at": true, "by": true,
	"is": true, "it": true, "or": true, "and": true, "but": true,
	"with": true, "from": true, "as": true, "be": true, "are": true,
	"was": true, "were": true, "has": true, "have": true, "had": true,
	"do": true, "does": true, "did": true, "not": true, "no": true,
	"how": true, "what": true, "where": true, "when": true, "why": true,
	"who": true, "which": true, "that": true, "this": true, "these": true,
}

func filterStopWords(terms []string) []string {
	filtered := make([]string, 0, len(terms))
	for _, t := range terms {
		if !stopWords[strings.ToLower(t)] {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) == 0 {
		return terms
	}
	return filtered
}

type TextRelevanceScore struct {
	TitleMatch       float64
	SnippetMatch     float64
	TermFrequency    float64
	Proximity        float64
	CoreTermCoverage float64
	Combined         float64
}

func AnalyzeTextRelevance(queryTerms []string, title, snippet string) TextRelevanceScore {
	score := TextRelevanceScore{}

	if len(queryTerms) == 0 {
		return score
	}

	titleLower := strings.ToLower(title)
	titleMatches := 0
	for _, term := range queryTerms {
		if strings.Contains(titleLower, strings.ToLower(term)) {
			titleMatches++
		}
	}
	score.TitleMatch = float64(titleMatches) / float64(len(queryTerms))

	snippetLower := strings.ToLower(snippet)
	snippetMatches := 0
	for _, term := range queryTerms {
		if strings.Contains(snippetLower, strings.ToLower(term)) {
			snippetMatches++
		}
	}
	score.SnippetMatch = float64(snippetMatches) / float64(len(queryTerms))

	combinedText := titleLower + " " + snippetLower
	score.TermFrequency = calculateTermFrequencyNormalized(queryTerms, combinedText)
	score.Proximity = calculateProximityNormalized(queryTerms, combinedText)

	coreTerms := filterStopWords(queryTerms)
	coreFound := 0
	for _, term := range coreTerms {
		termLower := strings.ToLower(term)
		if strings.Contains(titleLower, termLower) || strings.Contains(snippetLower, termLower) {
			coreFound++
		}
	}
	score.CoreTermCoverage = float64(coreFound) / float64(len(coreTerms))

	baseCombined := score.TitleMatch*0.4 + score.SnippetMatch*0.3 +
		score.TermFrequency*0.15 + score.Proximity*0.15
	score.Combined = baseCombined * score.CoreTermCoverage

	return score
}

func calculateTermFrequencyNormalized(queryTerms []string, text string) float64 {
	totalOccurrences := 0

	for _, term := range queryTerms {
		termLower := strings.ToLower(term)
		count := strings.Count(text, termLower)
		totalOccurrences += count
	}

	if totalOccurrences == 0 {
		return 0
	}

	logScore := math.Log2(float64(totalOccurrences) + 1)

	return math.Min(1.0, logScore/5.0)
}

func calculateProximityNormalized(queryTerms []string, text string) float64 {
	if len(queryTerms) < 2 {
		return 0
	}

	words := tokenizeText(text)
	if len(words) == 0 {
		return 0
	}

	termPositions := make(map[string][]int)
	for i, word := range words {
		for _, term := range queryTerms {
			if strings.EqualFold(word, term) {
				termPositions[strings.ToLower(term)] = append(termPositions[strings.ToLower(term)], i)
			}
		}
	}

	foundTerms := 0
	for _, positions := range termPositions {
		if len(positions) > 0 {
			foundTerms++
		}
	}

	if foundTerms < 2 {
		return 0
	}

	minDistance := len(words)

	for term1, positions1 := range termPositions {
		for term2, positions2 := range termPositions {
			if term1 >= term2 {
				continue
			}

			for _, p1 := range positions1 {
				for _, p2 := range positions2 {
					dist := p1 - p2
					if dist < 0 {
						dist = -dist
					}
					if dist < minDistance {
						minDistance = dist
					}
				}
			}
		}
	}

	switch {
	case minDistance <= 2:
		return 1.0
	case minDistance <= 5:
		return 0.8
	case minDistance <= 10:
		return 0.5
	default:
		return 0.2
	}
}

func tokenizeText(text string) []string {
	var words []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			current.WriteRune(unicode.ToLower(r))
		} else {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}
