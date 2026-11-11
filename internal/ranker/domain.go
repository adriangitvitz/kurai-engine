package ranker

import (
	"regexp"
	"strings"
)

type Domain string

const (
	DomainProgramming   Domain = "programming"
	DomainNews          Domain = "news"
	DomainAcademic      Domain = "academic"
	DomainPsychology    Domain = "psychology"
	DomainDocumentation Domain = "documentation"
	DomainCommercial    Domain = "commercial"
	DomainGeneral       Domain = "general"
)

type DomainScores map[Domain]float64

func (ds DomainScores) PrimaryDomain() Domain {
	if len(ds) == 0 {
		return DomainGeneral
	}

	maxDomain := DomainGeneral
	maxScore := 0.0
	for domain, score := range ds {
		if score > maxScore {
			maxScore = score
			maxDomain = domain
		}
	}
	return maxDomain
}

func (ds DomainScores) Confidence() float64 {
	if len(ds) == 0 {
		return 0.0
	}
	return ds[ds.PrimaryDomain()]
}

func (ds DomainScores) Normalize() DomainScores {
	if len(ds) == 0 {
		return ds
	}

	total := 0.0
	for _, s := range ds {
		total += s
	}

	if total == 0 {
		return ds
	}

	normalized := make(DomainScores, len(ds))
	for d, s := range ds {
		normalized[d] = s / total
	}
	return normalized
}

type DomainVocabulary struct {
	HighSignal   []string
	MediumSignal []string
	Patterns     []*regexp.Regexp
}

var domainVocabularies = map[Domain]DomainVocabulary{
	DomainProgramming: {
		HighSignal: []string{
			"function", "variable", "api", "debug", "compile", "repository",
			"commit", "async", "callback", "deploy", "syntax", "runtime",
			"exception", "interface", "implementation", "algorithm", "struct",
			"package", "import", "module", "dependency", "framework", "library",
			"endpoint", "request", "response", "json", "yaml", "config",
			"docker", "kubernetes", "git", "npm", "pip", "cargo", "go mod",
			"stackoverflow", "github", "gitlab", "bitbucket",
		},
		MediumSignal: []string{
			"code", "error", "output", "input", "data", "server", "client",
			"build", "test", "run", "install", "version", "update", "file",
			"path", "string", "integer", "boolean", "array", "object", "map",
			"class", "method", "parameter", "return", "type", "generic",
		},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(func|def|class|const|let|var)\s+\w+`),
			regexp.MustCompile(`(?i)\berr\s*!=?\s*nil\b`),
			regexp.MustCompile(`(?i)\.(go|py|js|ts|rs|java|cpp|c|rb|php)$`),
			regexp.MustCompile(`(?i)\bimport\s+["']`),
			regexp.MustCompile(`(?i)\bpackage\s+\w+`),
			regexp.MustCompile(`(?i)<\w+\s*/?>`),
		},
	},
	DomainNews: {
		HighSignal: []string{
			"reported", "according to", "officials", "statement", "breaking",
			"sources say", "announced", "investigation", "exclusive", "update",
			"developing", "confirmed", "alleged", "spokesperson", "press release",
			"election", "president", "congress", "parliament", "minister",
		},
		MediumSignal: []string{
			"today", "yesterday", "week", "government", "says", "told",
			"reuters", "associated press", "washington", "politics",
			"report", "news", "latest", "current", "events", "crisis",
		},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)\w*\s+\d{1,2},?\s+\d{4}`),
			regexp.MustCompile(`(?i)\d{1,2}\s+(hours?|days?|minutes?)\s+ago`),
			regexp.MustCompile(`(?i)\b(breaking|exclusive|update|developing):`),
			regexp.MustCompile(`(?i)\b(2024|2025)\b`),
		},
	},
	DomainAcademic: {
		HighSignal: []string{
			"study", "research", "findings", "hypothesis", "methodology",
			"results", "conclusion", "abstract", "citation", "peer-reviewed",
			"journal", "publication", "et al", "doi", "arxiv", "pubmed",
			"statistical", "significant", "correlation", "regression",
			"experiment", "participants", "randomized", "controlled",
		},
		MediumSignal: []string{
			"university", "professor", "paper", "review", "analysis",
			"data", "sample", "theory", "model", "evidence",
			"scholar", "academic", "thesis", "dissertation", "conference",
		},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bet\s+al\.?`),
			regexp.MustCompile(`(?i)doi:\s*10\.\d+`),
			regexp.MustCompile(`(?i)p\s*[<>=]\s*0?\.\d+`),
			regexp.MustCompile(`(?i)\(?\d{4}\)?[,;]\s*pp?\.\s*\d+`),
			regexp.MustCompile(`(?i)n\s*=\s*\d+`),
		},
	},
	DomainPsychology: {
		HighSignal: []string{
			"cognitive", "behavioral", "therapy", "disorder", "attachment",
			"trauma", "neuroplasticity", "anxiety", "depression", "mental health",
			"psychotherapy", "clinical", "diagnostic", "symptom", "treatment",
			"dsm", "icd", "psychologist", "psychiatrist", "counseling",
			"cbt", "dbt", "emdr", "mindfulness", "emotional regulation",
		},
		MediumSignal: []string{
			"mind", "emotion", "pattern", "response", "behavior", "feeling",
			"thought", "belief", "perception", "memory", "learning", "motivation",
			"psychology", "psychological", "mental", "emotional", "wellbeing",
		},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(cbt|dbt|emdr|act)\b`),
			regexp.MustCompile(`(?i)\bdsm-[iv5]+\b`),
			regexp.MustCompile(`(?i)\b(anxiety|depression|ptsd|adhd|ocd|bipolar)\b`),
			regexp.MustCompile(`(?i)mental\s+(health|illness|disorder)`),
		},
	},
	DomainDocumentation: {
		HighSignal: []string{
			"documentation", "reference", "guide", "tutorial", "getting started",
			"installation", "configuration", "usage", "parameters", "returns",
			"example", "syntax", "options", "arguments", "flags", "quickstart",
		},
		MediumSignal: []string{
			"see also", "related", "note", "warning", "tip", "important",
			"deprecated", "version", "changelog", "api reference", "manual",
			"overview", "introduction", "prerequisites", "requirements",
		},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)/docs?/`),
			regexp.MustCompile(`(?i)/api/`),
			regexp.MustCompile(`(?i)/reference/`),
			regexp.MustCompile(`(?i)/guide/`),
			regexp.MustCompile(`(?i)/tutorial/`),
			regexp.MustCompile(`(?i)\.md$`),
		},
	},
	DomainCommercial: {
		HighSignal: []string{
			"buy", "price", "discount", "sale", "offer", "cart", "checkout",
			"shipping", "delivery", "order", "purchase", "subscribe", "plan",
			"pricing", "free trial", "premium", "enterprise", "coupon",
		},
		MediumSignal: []string{
			"product", "service", "customer", "support", "contact", "about us",
			"reviews", "rating", "compare", "features", "benefits", "deal",
			"shop", "store", "marketplace", "vendor", "seller",
		},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\$\d+(\.\d{2})?`),
			regexp.MustCompile(`(?i)\d+%\s*off`),
			regexp.MustCompile(`(?i)add\s+to\s+cart`),
			regexp.MustCompile(`(?i)buy\s+now`),
			regexp.MustCompile(`(?i)free\s+shipping`),
		},
	},
}

func ClassifyDomain(title, snippet, url string) DomainScores {
	text := strings.ToLower(title + " " + snippet)
	urlLower := strings.ToLower(url)

	scores := make(DomainScores)

	for domain, vocab := range domainVocabularies {
		score := 0.0

		for _, term := range vocab.HighSignal {
			if containsTerm(text, term) {
				score += 2.0
			}
		}

		for _, term := range vocab.MediumSignal {
			if containsTerm(text, term) {
				score += 1.0
			}
		}

		for _, pattern := range vocab.Patterns {
			if pattern.MatchString(text) || pattern.MatchString(urlLower) {
				score += 3.0
			}
		}

		score += urlDomainBonus(urlLower, domain)

		if score > 0 {
			scores[domain] = score
		}
	}

	if len(scores) == 0 {
		scores[DomainGeneral] = 1.0
		return scores
	}

	return scores.Normalize()
}

func urlDomainBonus(url string, domain Domain) float64 {
	bonus := 0.0

	switch domain {
	case DomainDocumentation:
		if strings.Contains(url, "/docs") ||
			strings.Contains(url, "/api") ||
			strings.Contains(url, "/reference") ||
			strings.Contains(url, "/guide") {
			bonus += 3.0
		}
	case DomainAcademic:
		if strings.Contains(url, ".edu") ||
			strings.Contains(url, ".ac.") ||
			strings.Contains(url, "arxiv.org") ||
			strings.Contains(url, "scholar.google") ||
			strings.Contains(url, "pubmed") ||
			strings.Contains(url, "jstor.org") {
			bonus += 3.0
		}
	case DomainNews:
		if strings.Contains(url, "/news/") ||
			strings.Contains(url, "/politics/") ||
			strings.Contains(url, "/world/") ||
			strings.Contains(url, "/business/") {
			bonus += 2.0
		}
	case DomainProgramming:
		if strings.Contains(url, "github.com") ||
			strings.Contains(url, "gitlab.com") ||
			strings.Contains(url, "stackoverflow.com") ||
			strings.Contains(url, "pkg.go.dev") {
			bonus += 2.0
		}
	case DomainCommercial:
		if strings.Contains(url, "/shop") ||
			strings.Contains(url, "/product") ||
			strings.Contains(url, "/cart") ||
			strings.Contains(url, "/checkout") {
			bonus += 2.0
		}
	}

	return bonus
}

func containsTerm(text, term string) bool {
	return strings.Contains(text, strings.ToLower(term))
}

func DomainToCategory(d Domain) QueryCategory {
	switch d {
	case DomainProgramming, DomainDocumentation:
		return CategoryTech
	case DomainAcademic, DomainPsychology:
		return CategoryAcademic
	case DomainNews:
		return CategoryNews
	case DomainCommercial, DomainGeneral:
		return CategoryGeneral
	default:
		return CategoryGeneral
	}
}

func CategoryToDomain(c QueryCategory) Domain {
	switch c {
	case CategoryTech:
		return DomainProgramming
	case CategoryAcademic:
		return DomainAcademic
	case CategoryNews:
		return DomainNews
	case CategorySports:
		return DomainNews
	default:
		return DomainGeneral
	}
}
