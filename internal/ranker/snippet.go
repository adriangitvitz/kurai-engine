package ranker

import (
	"math"
	"regexp"
	"strings"
	"unicode"

	"github.com/adriangitvitz/kurai/internal/config"
)

type SnippetScore struct {
	PartialMatch    int
	TermFrequency   int
	Proximity       int
	CodeDetection   int
	QualityModifier int
	Total           int
}

var (
	codePatterns = []*regexp.Regexp{
		regexp.MustCompile("`[^`]+`"),
		regexp.MustCompile(`\b\w+\(\)`),
		regexp.MustCompile(`\b\w+\.\w+\(`),
		regexp.MustCompile(`\b(func|def|function|class|struct|interface|type)\s+\w+`),
		regexp.MustCompile(`\b(import|require|include|using)\s+`),
		regexp.MustCompile(`\b(var|let|const|int|string|bool|float)\s+\w+`),
		regexp.MustCompile(`[{}\[\]()<>]`),
		regexp.MustCompile(`\b(if|else|for|while|switch|case|return)\b`),
		regexp.MustCompile(`\b(nil|null|None|true|false|True|False)\b`),
		regexp.MustCompile(`\b(err|error|Error|Exception)\b`),
		regexp.MustCompile(`\b\w+::\w+`),
		regexp.MustCompile(`\$\w+`),
		regexp.MustCompile(`@\w+`),
		regexp.MustCompile(`#\w+`),
	}

	lowQualityPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(you won't believe|shocking|amazing|incredible)\b`),
		regexp.MustCompile(`(?i)\b(click here|learn more|read more|see more)\b`),
		regexp.MustCompile(`(?i)\b(\d+ (ways|tips|tricks|secrets|reasons))\b`),
		regexp.MustCompile(`(?i)\b(best|top|ultimate) \d+\b`),
		regexp.MustCompile(`(?i)\b(free|cheap|discount|deal|offer|sale)\b`),
		regexp.MustCompile(`(?i)\b(subscribe|sign up|newsletter)\b`),
		regexp.MustCompile(`(?i)\b(sponsored|advertisement|promoted)\b`),
	}

	highQualityPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(documentation|specification|reference|manual)\b`),
		regexp.MustCompile(`(?i)\b(implementation|algorithm|architecture|design)\b`),
		regexp.MustCompile(`(?i)\b(parameters?|arguments?|returns?|throws?)\b`),
		regexp.MustCompile(`(?i)\b(deprecated|experimental|stable|beta)\b`),
		regexp.MustCompile(`(?i)\b(example|usage|syntax|tutorial)\b`),
		regexp.MustCompile(`(?i)\bversion \d+\.\d+`),
		regexp.MustCompile(`(?i)\b(api|sdk|library|framework|package)\b`),
	}
)

func AnalyzeSnippet(queryTerms []string, snippet string, cfg *config.SnippetConfig) SnippetScore {
	score := SnippetScore{}

	if snippet == "" || len(queryTerms) == 0 {
		return score
	}

	snippetLower := strings.ToLower(snippet)
	snippetWords := tokenizeSnippet(snippetLower)

	score.PartialMatch = calculatePartialMatch(queryTerms, snippetLower, cfg.PartialMatchMax)
	score.TermFrequency = calculateTermFrequency(queryTerms, snippetLower, cfg.TermFrequencyMax)
	score.Proximity = calculateProximity(queryTerms, snippetWords, cfg.ProximityBonus)
	score.CodeDetection = detectCode(snippet, cfg.CodeDetectionBonus)
	score.QualityModifier = calculateQualityModifier(snippet, cfg.LowQualityPenalty, cfg.HighQualityBonus)

	score.Total = score.PartialMatch + score.TermFrequency + score.Proximity +
		score.CodeDetection + score.QualityModifier

	return score
}

func tokenizeSnippet(snippet string) []string {
	var words []string
	var current strings.Builder

	for _, r := range snippet {
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

func calculatePartialMatch(queryTerms []string, snippetLower string, maxScore int) int {
	if len(queryTerms) == 0 {
		return 0
	}

	matchCount := 0
	for _, term := range queryTerms {
		termLower := strings.ToLower(term)
		if strings.Contains(snippetLower, termLower) {
			matchCount++
		}
	}

	matchRatio := float64(matchCount) / float64(len(queryTerms))
	return int(matchRatio * float64(maxScore))
}

func calculateTermFrequency(queryTerms []string, snippetLower string, maxScore int) int {
	totalOccurrences := 0

	for _, term := range queryTerms {
		termLower := strings.ToLower(term)
		count := strings.Count(snippetLower, termLower)
		totalOccurrences += count
	}

	if totalOccurrences == 0 {
		return 0
	}

	logScore := math.Log2(float64(totalOccurrences) + 1)

	normalizedScore := min(1.0, logScore/5.0)
	return int(normalizedScore * float64(maxScore))
}

func calculateProximity(queryTerms []string, snippetWords []string, bonus int) int {
	if len(queryTerms) < 2 || len(snippetWords) == 0 {
		return 0
	}

	termPositions := make(map[string][]int)
	for i, word := range snippetWords {
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

	minDistance := len(snippetWords)

	for term1, positions1 := range termPositions {
		for term2, positions2 := range termPositions {
			if term1 >= term2 {
				continue
			}

			for _, p1 := range positions1 {
				for _, p2 := range positions2 {
					dist := abs(p1 - p2)
					if dist < minDistance {
						minDistance = dist
					}
				}
			}
		}
	}

	if minDistance <= 10 {
		return bonus
	}

	return 0
}

func detectCode(snippet string, bonus int) int {
	codeSignals := 0

	for _, pattern := range codePatterns {
		if pattern.MatchString(snippet) {
			codeSignals++
		}
	}

	if codeSignals >= 3 {
		return bonus
	} else if codeSignals >= 1 {
		return bonus / 2
	}

	return 0
}

func calculateQualityModifier(snippet string, penaltyValue int, bonusValue int) int {
	lowQualitySignals := 0
	highQualitySignals := 0

	for _, pattern := range lowQualityPatterns {
		if pattern.MatchString(snippet) {
			lowQualitySignals++
		}
	}

	for _, pattern := range highQualityPatterns {
		if pattern.MatchString(snippet) {
			highQualitySignals++
		}
	}

	if lowQualitySignals >= 2 && highQualitySignals == 0 {
		return penaltyValue
	} else if highQualitySignals >= 2 && lowQualitySignals == 0 {
		return bonusValue
	} else if highQualitySignals > lowQualitySignals {
		return bonusValue / 2
	} else if lowQualitySignals > highQualitySignals {
		return penaltyValue / 2
	}

	return 0
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
