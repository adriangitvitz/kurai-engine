package ranker

import (
	"strings"
)

type QueryCategory string

const (
	CategoryGeneral  QueryCategory = "general"
	CategoryTech     QueryCategory = "tech"
	CategoryAcademic QueryCategory = "academic"
	CategoryNews     QueryCategory = "news"
	CategorySports   QueryCategory = "sports"
)

type IntentModifiers struct {
	IsQuestion   bool
	IsTutorial   bool
	IsRecent     bool
	IsComparison bool
}

type QueryIntent struct {
	Category          QueryCategory
	IsDocumentation   bool
	IsErrorResolution bool
	IsPaperSearch     bool
	IsNews            bool
	Confidence        float64

	DomainDistribution DomainScores
	PrimaryDomain      Domain
	Modifiers          IntentModifiers
}

var programmingLanguages = map[string]bool{
	"go": true, "golang": true, "python": true, "javascript": true, "js": true,
	"typescript": true, "ts": true, "java": true, "kotlin": true, "swift": true,
	"rust": true, "c": true, "cpp": true, "c++": true, "csharp": true, "c#": true,
	"ruby": true, "php": true, "perl": true, "scala": true, "haskell": true,
	"elixir": true, "erlang": true, "clojure": true, "lua": true, "r": true,
	"matlab": true, "julia": true, "dart": true, "objective-c": true,
	"assembly": true, "asm": true, "sql": true, "bash": true, "shell": true,
	"powershell": true, "html": true, "css": true, "scss": true, "sass": true,
	"vue": true, "react": true, "angular": true, "node": true, "nodejs": true,
	"deno": true, "bun": true,
}

var techTerms = map[string]bool{
	"api": true, "sdk": true, "library": true, "framework": true, "package": true,
	"module": true, "function": true, "method": true, "class": true, "interface": true,
	"struct": true, "type": true, "error": true, "exception": true, "debug": true,
	"fix": true, "bug": true, "issue": true, "crash": true, "memory": true,
	"performance": true, "optimize": true, "async": true, "sync": true, "thread": true,
	"mutex": true, "channel": true, "goroutine": true, "coroutine": true,
	"docker": true, "kubernetes": true, "k8s": true, "container": true, "pod": true,
	"deployment": true, "service": true, "ingress": true, "helm": true,
	"aws": true, "azure": true, "gcp": true, "cloud": true, "serverless": true,
	"lambda": true, "ec2": true, "s3": true, "rds": true, "vpc": true,
	"git": true, "github": true, "gitlab": true, "bitbucket": true, "commit": true,
	"branch": true, "merge": true, "pull": true, "push": true, "clone": true,
	"http": true, "https": true, "tcp": true, "udp": true, "websocket": true,
	"rest": true, "graphql": true, "grpc": true, "json": true, "xml": true,
	"yaml": true, "toml": true, "config": true, "configuration": true,
	"database": true, "db": true, "mysql": true, "postgres": true, "postgresql": true,
	"mongodb": true, "redis": true, "elasticsearch": true, "sqlite": true,
	"tls": true, "ssl": true, "cipher": true, "encryption": true, "hash": true,
	"jwt": true, "oauth": true, "auth": true, "authentication": true, "authorization": true,
	"linux": true, "unix": true, "windows": true, "macos": true, "ubuntu": true,
	"debian": true, "centos": true, "fedora": true, "arch": true,
	"nginx": true, "apache": true, "caddy": true, "traefik": true,
	"ci": true, "cd": true, "cicd": true, "jenkins": true, "actions": true,
	"terraform": true, "ansible": true, "puppet": true, "chef": true,
	"vim": true, "emacs": true, "vscode": true, "ide": true, "editor": true,
	"terminal": true, "console": true, "cli": true, "command": true,
	"algorithm": true, "data": true, "structure": true, "array": true, "list": true,
	"map": true, "set": true, "tree": true, "graph": true, "heap": true, "stack": true,
	"queue": true, "linked": true, "binary": true, "search": true, "sort": true,
	"regex": true, "regexp": true, "pattern": true, "match": true,
}

var docTerms = map[string]bool{
	"docs": true, "documentation": true, "reference": true, "guide": true,
	"tutorial": true, "howto": true, "how-to": true, "manual": true,
	"spec": true, "specification": true, "rfc": true, "standard": true,
	"example": true, "examples": true, "sample": true, "samples": true,
	"usage": true, "syntax": true, "parameters": true, "arguments": true,
	"return": true, "returns": true, "options": true, "flags": true,
}

var errorTerms = map[string]bool{
	"error": true, "exception": true, "bug": true, "issue": true, "problem": true,
	"fix": true, "solve": true, "debug": true, "troubleshoot": true, "help": true,
	"crash": true, "fail": true, "failed": true, "failing": true, "broken": true,
	"not": true, "working": true, "why": true, "how": true, "does": true,
	"undefined": true, "null": true, "nil": true, "panic": true, "segfault": true,
	"timeout": true, "connection": true, "refused": true, "denied": true,
}

var academicTerms = map[string]bool{
	"arxiv": true, "paper": true, "papers": true, "research": true, "study": true,
	"journal": true, "article": true, "publication": true, "abstract": true,
	"citation": true, "cite": true, "reference": true, "bibliography": true,
	"thesis": true, "dissertation": true, "phd": true, "peer": true, "reviewed": true,
	"experiment": true, "hypothesis": true, "methodology": true, "results": true,
	"conclusion": true, "findings": true, "analysis": true, "data": true,
	"statistical": true, "significance": true, "correlation": true, "regression": true,
	"neural": true, "network": true, "deep": true, "learning": true, "machine": true,
	"ai": true, "artificial": true, "intelligence": true, "nlp": true, "cv": true,
	"transformer": true, "transformers": true, "attention": true, "bert": true,
	"gpt": true, "llm": true, "model": true, "training": true, "inference": true,
	"dataset": true, "benchmark": true, "evaluation": true, "metrics": true,
	"alphaxiv": true, "biorxiv": true, "medrxiv": true, "ssrn": true, "pubmed": true,
}

var newsTerms = map[string]bool{
	"news": true, "breaking": true, "latest": true, "today": true, "yesterday": true,
	"update": true, "updates": true, "report": true, "reports": true, "reporter": true,
	"announcement": true, "press": true, "release": true, "statement": true,
	"headline": true, "headlines": true, "article": true, "story": true, "stories": true,
	"current": true, "events": true, "happening": true, "developing": true,
	"live": true, "coverage": true, "exclusive": true, "interview": true,
	"politics": true, "political": true, "election": true, "vote": true, "voting": true,
	"government": true, "congress": true, "senate": true, "president": true,
	"economy": true, "economic": true, "market": true, "markets": true, "stock": true,
	"stocks": true, "trading": true, "finance": true, "financial": true,
	"weather": true, "forecast": true, "storm": true, "hurricane": true,
	"crisis": true, "emergency": true, "disaster": true, "war": true, "conflict": true,
	"buys": true, "acquires": true, "acquisition": true, "merger": true, "deal": true,
	"partnership": true, "partner": true, "invests": true, "investment": true,
	"lawsuit": true, "sues": true, "sued": true, "controversy": true, "scandal": true,
	"fired": true, "resigned": true, "appointed": true, "ceo": true, "layoffs": true,
	"bankruptcy": true, "ipo": true, "valuation": true, "funding": true,
	"trump": true, "biden": true, "putin": true, "xi": true, "zelensky": true,
	"tariff": true, "tariffs": true, "sanctions": true, "treaty": true, "diplomacy": true,
	"immigration": true, "border": true, "deportation": true, "visa": true,
	"criminals": true, "crime": true, "arrest": true, "arrested": true, "indicted": true,
	"openai": true, "chatgpt": true, "tiktok": true, "meta": true, "tesla": true,
	"spacex": true, "nvidia": true, "antitrust": true, "regulation": true,
	"noticias": true, "filtrado": true, "filtrados": true, "escandalo": true,
	"corrupcion": true, "datos": true, "hackeo": true, "ataque": true,
}

var newsEntities = map[string]bool{
	"apple": true, "google": true, "microsoft": true, "amazon": true, "facebook": true,
	"oracle": true, "nvidia": true, "intel": true, "amd": true, "twitter": true,
	"mexico": true, "china": true, "russia": true, "ukraine": true, "israel": true,
	"iran": true, "korea": true, "taiwan": true, "gaza": true, "palestine": true,
}

var sportsTerms = map[string]bool{
	"score": true, "scores": true, "game": true, "games": true, "match": true,
	"vs": true, "versus": true, "play": true, "plays": true, "playoff": true,
	"playoffs": true, "championship": true, "final": true, "finals": true,
	"season": true, "league": true, "team": true, "teams": true, "player": true,
	"players": true, "roster": true, "draft": true, "trade": true, "trades": true,
	"nfl": true, "nba": true, "mlb": true, "nhl": true, "mls": true, "fifa": true,
	"soccer": true, "football": true, "basketball": true, "baseball": true,
	"hockey": true, "tennis": true, "golf": true, "boxing": true, "ufc": true,
	"mma": true, "wrestling": true, "olympics": true, "olympic": true,
	"win": true, "won": true, "lose": true, "lost": true, "tie": true, "draw": true,
	"goal": true, "goals": true, "touchdown": true, "homerun": true, "strikeout": true,
	"assist": true, "rebound": true, "point": true, "points": true,
	"standings": true, "stats": true, "statistics": true, "record": true,
	"injury": true, "injured": true, "coach": true, "manager": true,
	"espn": true, "sports": true, "athletic": true,
}

func DetectIntent(queryTerms []string) QueryIntent {
	intent := QueryIntent{
		Category:           CategoryGeneral,
		Confidence:         0.5,
		DomainDistribution: make(DomainScores),
		PrimaryDomain:      DomainGeneral,
	}

	if len(queryTerms) == 0 {
		intent.DomainDistribution[DomainGeneral] = 1.0
		return intent
	}

	var techScore, academicScore, newsScore, sportsScore int
	var hasProgLang, hasDoc, hasError bool
	var hasNewsEntity, hasNewsAction bool

	for _, term := range queryTerms {
		termLower := strings.ToLower(term)

		if programmingLanguages[termLower] {
			hasProgLang = true
			techScore += 2
		}

		if techTerms[termLower] {
			techScore++
		}

		if docTerms[termLower] {
			hasDoc = true
			techScore++
		}

		if errorTerms[termLower] {
			hasError = true
			techScore++
		}

		if academicTerms[termLower] {
			academicScore++
		}

		if newsTerms[termLower] {
			newsScore++
			if isNewsActionWord(termLower) {
				hasNewsAction = true
			}
		}

		if newsEntities[termLower] {
			hasNewsEntity = true
			newsScore++
		}

		if sportsTerms[termLower] {
			sportsScore++
		}
	}

	if hasNewsEntity && hasNewsAction {
		newsScore += 3
	}
	if hasNewsEntity && newsScore >= 2 {
		newsScore += 2
	}

	switch detectLeadQuestion(queryTerms) {
	case questionWhatIs, questionHowDoes, questionWhyIs:
		academicScore += 3
	case questionWhenDid:
		newsScore += 2
	}

	maxScore := max(techScore, academicScore, newsScore, sportsScore)

	if maxScore == 0 {
		intent.DomainDistribution[DomainGeneral] = 1.0
		return intent
	}

	nonTechAtMax := 0
	if academicScore == maxScore {
		nonTechAtMax++
	}
	if newsScore == maxScore {
		nonTechAtMax++
	}
	if sportsScore == maxScore {
		nonTechAtMax++
	}

	switch {
	case techScore == maxScore:
		intent.Category = CategoryTech
		intent.Confidence = min(0.9, 0.5+float64(techScore)*0.1)
	case nonTechAtMax > 1:
		intent.Category = CategoryGeneral
		intent.Confidence = 0.5
	case academicScore == maxScore:
		intent.Category = CategoryAcademic
		intent.Confidence = min(0.9, 0.5+float64(academicScore)*0.1)
	case newsScore == maxScore:
		intent.Category = CategoryNews
		intent.Confidence = min(0.9, 0.5+float64(newsScore)*0.1)
	case sportsScore == maxScore:
		intent.Category = CategorySports
		intent.Confidence = min(0.9, 0.5+float64(sportsScore)*0.1)
	}

	intent.IsDocumentation = hasDoc || (hasProgLang && techScore >= 2)
	intent.IsErrorResolution = hasError
	intent.IsPaperSearch = academicScore >= 2
	intent.IsNews = newsScore >= 2

	intent.Modifiers = detectModifiers(queryTerms)

	intent.DomainDistribution = computeDomainDistribution(techScore, academicScore, newsScore, sportsScore, intent)
	intent.PrimaryDomain = intent.DomainDistribution.PrimaryDomain()

	return intent
}

type leadQuestionForm int

const (
	questionNone leadQuestionForm = iota
	questionWhatIs
	questionHowDoes
	questionWhyIs
	questionWhenDid
)

func detectLeadQuestion(queryTerms []string) leadQuestionForm {
	if len(queryTerms) == 0 {
		return questionNone
	}
	first := strings.ToLower(queryTerms[0])
	var second string
	if len(queryTerms) > 1 {
		second = strings.ToLower(queryTerms[1])
	}

	switch first {
	case "what":
		if second == "is" || second == "are" || second == "was" || second == "were" || second == "" {
			return questionWhatIs
		}
		return questionWhatIs
	case "how":
		if second == "does" || second == "do" || second == "is" || second == "are" {
			return questionHowDoes
		}
	case "why":
		if second == "is" || second == "are" || second == "does" || second == "do" || second == "did" || second == "" {
			return questionWhyIs
		}
		return questionWhyIs
	case "when":
		if second == "did" || second == "was" || second == "is" || second == "are" {
			return questionWhenDid
		}
	}
	return questionNone
}

func detectModifiers(queryTerms []string) IntentModifiers {
	modifiers := IntentModifiers{}

	questionWords := map[string]bool{
		"what": true, "how": true, "why": true, "when": true, "where": true,
		"which": true, "who": true, "can": true, "does": true, "is": true,
	}

	tutorialWords := map[string]bool{
		"tutorial": true, "guide": true, "howto": true, "how-to": true,
		"learn": true, "beginner": true, "introduction": true, "intro": true,
		"getting": true, "started": true, "basics": true, "step": true,
	}

	recentWords := map[string]bool{
		"latest": true, "new": true, "recent": true, "2024": true, "2025": true,
		"2026": true, "today": true, "current": true, "now": true,
	}

	comparisonWords := map[string]bool{
		"vs": true, "versus": true, "compare": true, "comparison": true,
		"difference": true, "between": true, "better": true, "best": true,
		"alternative": true, "alternatives": true, "or": true,
	}

	for _, term := range queryTerms {
		termLower := strings.ToLower(term)

		if questionWords[termLower] {
			modifiers.IsQuestion = true
		}
		if tutorialWords[termLower] {
			modifiers.IsTutorial = true
		}
		if recentWords[termLower] {
			modifiers.IsRecent = true
		}
		if comparisonWords[termLower] {
			modifiers.IsComparison = true
		}
	}

	return modifiers
}

func computeDomainDistribution(techScore, academicScore, newsScore, sportsScore int, intent QueryIntent) DomainScores {
	scores := make(DomainScores)

	if techScore > 0 {
		if intent.IsErrorResolution {
			scores[DomainProgramming] = float64(techScore) * 0.8
			scores[DomainDocumentation] = float64(techScore) * 0.2
		} else if intent.IsDocumentation {
			scores[DomainDocumentation] = float64(techScore) * 0.6
			scores[DomainProgramming] = float64(techScore) * 0.4
		} else {
			scores[DomainProgramming] = float64(techScore) * 0.5
			scores[DomainDocumentation] = float64(techScore) * 0.5
		}
	}

	if academicScore > 0 {
		scores[DomainAcademic] = float64(academicScore)
		if academicScore >= 2 {
			scores[DomainPsychology] = float64(academicScore) * 0.2
		}
	}

	if newsScore > 0 {
		scores[DomainNews] = float64(newsScore)
	}

	if sportsScore > 0 {
		scores[DomainNews] += float64(sportsScore) * 0.8
	}

	totalSpecific := 0.0
	for _, s := range scores {
		totalSpecific += s
	}

	if totalSpecific == 0 {
		scores[DomainGeneral] = 1.0
	} else {
		scores[DomainGeneral] = totalSpecific * 0.1
	}

	return scores.Normalize()
}

func GetIntentBoostMultiplier(intent QueryIntent, authCategory AuthorityCategory) float64 {
	multiplier := 1.0

	switch intent.Category {
	case CategoryTech:
		switch authCategory {
		case CategoryOfficialDocs:
			if intent.IsDocumentation {
				multiplier = 1.5
			} else {
				multiplier = 1.2
			}
		case CategoryCodeRepository:
			multiplier = 1.3
		case CategoryDeveloperQA:
			if intent.IsErrorResolution {
				multiplier = 1.5
			} else {
				multiplier = 1.2
			}
		}

	case CategoryAcademic:
		switch authCategory {
		case CategoryPaperArchive:
			multiplier = 1.5
		case CategoryAcademicPublisher:
			multiplier = 1.3
		case CategoryWikipedia:
			multiplier = 1.2
		}

	case CategoryNews:
		switch authCategory {
		case CategoryMajorNews:
			multiplier = 1.5
		case CategoryQualityNews:
			multiplier = 1.3
		}

	case CategorySports:
		switch authCategory {
		case CategoryMajorNews, CategoryQualityNews:
			multiplier = 1.3
		}
	}

	if intent.Confidence < 0.7 {
		multiplier = 1.0 + (multiplier-1.0)*intent.Confidence
	}

	return multiplier
}

func ParseCategory(s string) QueryCategory {
	switch strings.ToLower(s) {
	case "tech", "technical", "programming", "code", "software", "engineering":
		return CategoryTech

	case "academic", "research", "science", "paper", "papers",
		"psychology", "sociology", "anthropology",
		"biology", "chemistry", "physics", "medicine", "medical",
		"history", "philosophy", "literature", "humanities",
		"law", "legal", "economics":
		return CategoryAcademic

	case "news", "current", "politics", "business":
		return CategoryNews

	case "sports", "sport":
		return CategorySports

	default:
		return CategoryGeneral
	}
}

func isNewsActionWord(term string) bool {
	actionWords := map[string]bool{
		"buys": true, "acquires": true, "acquisition": true, "merger": true,
		"deal": true, "partnership": true, "invests": true, "investment": true,
		"lawsuit": true, "sues": true, "sued": true, "controversy": true,
		"scandal": true, "fired": true, "resigned": true, "appointed": true,
		"layoffs": true, "bankruptcy": true, "downfall": true, "collapse": true,
		"announces": true, "launches": true, "releases": true, "unveils": true,
		"arrested": true, "indicted": true, "convicted": true, "sentenced": true,
		"dies": true, "killed": true, "attacked": true, "bombing": true,
		"wins": true, "loses": true, "defeats": true, "elected": true,
		"filtrado": true, "filtrados": true, "hackeo": true, "ataque": true,
	}
	return actionWords[term]
}
