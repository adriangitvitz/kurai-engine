package ranker

import (
	"regexp"
	"strings"
	"unicode"
)

type QueryType string

const (
	QueryTypeNavigational  QueryType = "navigational"
	QueryTypeInformational QueryType = "informational"
	QueryTypeTransactional QueryType = "transactional"
)

type QueryAnalysis struct {
	OriginalQuery   string   `json:"original_query"`
	NormalizedQuery string   `json:"normalized_query"`
	Terms           []string `json:"terms"`

	QueryType      QueryType `json:"query_type"`
	QueryTypeScore float64   `json:"query_type_score"`

	Intent QueryIntent `json:"intent"`

	Entities []RecognizedEntity `json:"entities,omitempty"`

	Phrases []DetectedPhrase `json:"phrases,omitempty"`

	MultiIntent []IntentComponent `json:"multi_intent,omitempty"`

	ExpandedTerms []string `json:"expanded_terms,omitempty"`

	IsQuestion    bool `json:"is_question"`
	IsComparison  bool `json:"is_comparison"`
	HasErrorCode  bool `json:"has_error_code"`
	HasVersion    bool `json:"has_version"`
	HasQuotedText bool `json:"has_quoted_text"`
	WordCount     int  `json:"word_count"`
}

type RecognizedEntity struct {
	Text      string `json:"text"`
	Type      string `json:"type"`
	Canonical string `json:"canonical"`
	Position  int    `json:"position"`
}

type DetectedPhrase struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Context string `json:"context"`
}

type IntentComponent struct {
	Terms      []string      `json:"terms"`
	Category   QueryCategory `json:"category"`
	Confidence float64       `json:"confidence"`
}

var (
	errorCodePattern  = regexp.MustCompile(`(?i)\b(error|err|exception|panic|fatal|failed|failure)[\s:_-]*([A-Z0-9_]{2,}|\d+)\b`)
	versionPattern    = regexp.MustCompile(`\b[vV]?\d+\.\d+(?:\.\d+)?(?:-[a-zA-Z0-9]+)?\b`)
	comparisonPattern = regexp.MustCompile(`(?i)\b(vs?\.?|versus|compared?\s*to|or|better|best|difference|between)\b`)
	quotedTextPattern = regexp.MustCompile(`"([^"]+)"`)
	questionPattern   = regexp.MustCompile(`(?i)^(what|how|why|when|where|which|who|can|does|is|are|should|could|would)\b`)
)

var navigationalIndicators = map[string]bool{
	"login":     true,
	"signin":    true,
	"sign-in":   true,
	"signup":    true,
	"sign-up":   true,
	"dashboard": true,
	"account":   true,
	"official":  true,
	"homepage":  true,
	"home":      true,
	"site":      true,
	"website":   true,
	".com":      true,
	".org":      true,
	".io":       true,
	".dev":      true,
}

var transactionalIndicators = map[string]bool{
	"buy":       true,
	"purchase":  true,
	"download":  true,
	"install":   true,
	"get":       true,
	"free":      true,
	"trial":     true,
	"price":     true,
	"pricing":   true,
	"cost":      true,
	"subscribe": true,
	"order":     true,
	"book":      true,
	"reserve":   true,
}

func AnalyzeQuery(query string, entityRecognizer *EntityRecognizer, synonymExpander *SynonymExpander) *QueryAnalysis {
	analysis := &QueryAnalysis{
		OriginalQuery:   query,
		NormalizedQuery: normalizeQuery(query),
		Terms:           tokenizeQuery(query),
	}

	analysis.WordCount = len(analysis.Terms)

	analysis.IsQuestion = questionPattern.MatchString(query)
	analysis.IsComparison = comparisonPattern.MatchString(query)
	analysis.HasErrorCode = errorCodePattern.MatchString(query)
	analysis.HasVersion = versionPattern.MatchString(query)
	analysis.HasQuotedText = quotedTextPattern.MatchString(query)

	analysis.QueryType, analysis.QueryTypeScore = classifyQueryType(query, analysis.Terms)

	analysis.Intent = DetectIntent(analysis.Terms)

	analysis.Phrases = detectPhrases(query)

	if entityRecognizer != nil {
		analysis.Entities = entityRecognizer.Recognize(analysis.Terms)
	}

	if synonymExpander != nil {
		analysis.ExpandedTerms = synonymExpander.Expand(analysis.Terms, 3)
	}

	analysis.MultiIntent = detectMultiIntent(analysis.Terms, query)

	return analysis
}

func normalizeQuery(query string) string {
	normalized := strings.ToLower(query)
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

func tokenizeQuery(query string) []string {
	var terms []string

	quotes := quotedTextPattern.FindAllStringSubmatch(query, -1)
	remaining := quotedTextPattern.ReplaceAllString(query, " ")

	for _, q := range quotes {
		if len(q) > 1 {
			terms = append(terms, strings.ToLower(q[1]))
		}
	}

	words := strings.FieldsFunc(remaining, func(r rune) bool {
		return unicode.IsSpace(r) || (unicode.IsPunct(r) && r != '-' && r != '_' && r != '.' && r != '#')
	})

	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".-_"))
		if word != "" && len(word) > 1 {
			terms = append(terms, word)
		}
	}

	return terms
}

func classifyQueryType(query string, terms []string) (QueryType, float64) {
	navScore := 0.0
	transScore := 0.0
	infoScore := 0.0

	queryLower := strings.ToLower(query)

	for indicator := range navigationalIndicators {
		if strings.Contains(queryLower, indicator) {
			navScore += 0.3
		}
	}

	for indicator := range transactionalIndicators {
		if strings.Contains(queryLower, indicator) {
			transScore += 0.3
		}
	}

	if questionPattern.MatchString(query) {
		infoScore += 0.4
	}

	if comparisonPattern.MatchString(query) {
		infoScore += 0.3
	}

	if errorCodePattern.MatchString(query) {
		infoScore += 0.3
	}

	if len(terms) <= 2 {
		navScore += 0.2
	}

	if len(terms) >= 5 {
		infoScore += 0.2
	}

	if strings.Contains(query, ".com") || strings.Contains(query, ".org") || strings.Contains(query, ".io") {
		navScore += 0.4
	}

	maxScore := max(navScore, transScore, infoScore)
	if maxScore == 0 {
		return QueryTypeInformational, 0.5
	}

	var queryType QueryType
	switch {
	case navScore == maxScore:
		queryType = QueryTypeNavigational
	case transScore == maxScore:
		queryType = QueryTypeTransactional
	default:
		queryType = QueryTypeInformational
	}

	confidence := min(maxScore, 1.0)
	return queryType, confidence
}

func detectPhrases(query string) []DetectedPhrase {
	var phrases []DetectedPhrase

	if matches := errorCodePattern.FindAllStringSubmatch(query, -1); matches != nil {
		for _, match := range matches {
			phrases = append(phrases, DetectedPhrase{
				Type:    "error_code",
				Text:    match[0],
				Context: match[1],
			})
		}
	}

	if matches := versionPattern.FindAllString(query, -1); matches != nil {
		for _, match := range matches {
			phrases = append(phrases, DetectedPhrase{
				Type: "version",
				Text: match,
			})
		}
	}

	if matches := quotedTextPattern.FindAllStringSubmatch(query, -1); matches != nil {
		for _, match := range matches {
			if len(match) > 1 {
				phrases = append(phrases, DetectedPhrase{
					Type: "quoted",
					Text: match[1],
				})
			}
		}
	}

	if comparisonPattern.MatchString(query) {
		phrases = append(phrases, DetectedPhrase{
			Type: "comparison",
			Text: query,
		})
	}

	return phrases
}

func detectMultiIntent(terms []string, query string) []IntentComponent {
	var components []IntentComponent

	queryLower := strings.ToLower(query)
	if strings.Contains(queryLower, " vs ") || strings.Contains(queryLower, " versus ") {
		parts := regexp.MustCompile(`(?i)\s+vs\.?\s+|\s+versus\s+`).Split(queryLower, 2)
		if len(parts) == 2 {
			leftTerms := strings.Fields(parts[0])
			rightTerms := strings.Fields(parts[1])

			leftIntent := DetectIntent(leftTerms)
			rightIntent := DetectIntent(rightTerms)

			components = append(components, IntentComponent{
				Terms:      leftTerms,
				Category:   leftIntent.Category,
				Confidence: leftIntent.Confidence * 0.8,
			})
			components = append(components, IntentComponent{
				Terms:      rightTerms,
				Category:   rightIntent.Category,
				Confidence: rightIntent.Confidence * 0.8,
			})
		}
	}

	if strings.Contains(queryLower, " for ") {
		parts := strings.SplitN(queryLower, " for ", 2)
		if len(parts) == 2 {
			targetTerms := strings.Fields(parts[1])
			targetIntent := DetectIntent(targetTerms)
			if targetIntent.Category != CategoryGeneral {
				components = append(components, IntentComponent{
					Terms:      targetTerms,
					Category:   targetIntent.Category,
					Confidence: targetIntent.Confidence * 0.7,
				})
			}
		}
	}

	if len(components) == 0 {
		mainIntent := DetectIntent(terms)
		components = append(components, IntentComponent{
			Terms:      terms,
			Category:   mainIntent.Category,
			Confidence: mainIntent.Confidence,
		})
	}

	return components
}

func (qa *QueryAnalysis) GetPrimaryCategory() QueryCategory {
	if len(qa.MultiIntent) == 0 {
		return qa.Intent.Category
	}

	var primaryCategory QueryCategory = CategoryGeneral
	var maxConfidence float64 = 0

	for _, component := range qa.MultiIntent {
		if component.Confidence > maxConfidence {
			maxConfidence = component.Confidence
			primaryCategory = component.Category
		}
	}

	return primaryCategory
}

func (qa *QueryAnalysis) HasEntity(entityType string) bool {
	for _, entity := range qa.Entities {
		if entity.Type == entityType {
			return true
		}
	}
	return false
}

func (qa *QueryAnalysis) GetEntitiesByType(entityType string) []RecognizedEntity {
	var entities []RecognizedEntity
	for _, entity := range qa.Entities {
		if entity.Type == entityType {
			entities = append(entities, entity)
		}
	}
	return entities
}

func (qa *QueryAnalysis) IsErrorQuery() bool {
	return qa.HasErrorCode || qa.Intent.IsErrorResolution
}

func (qa *QueryAnalysis) EffectiveTerms() []string {
	if len(qa.ExpandedTerms) > 0 {
		seen := make(map[string]bool)
		var result []string

		for _, term := range qa.Terms {
			if !seen[term] {
				seen[term] = true
				result = append(result, term)
			}
		}

		for _, term := range qa.ExpandedTerms {
			if !seen[term] {
				seen[term] = true
				result = append(result, term)
			}
		}

		return result
	}

	return qa.Terms
}
