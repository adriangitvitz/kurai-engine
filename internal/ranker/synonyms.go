package ranker

import (
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type SynonymExpander struct {
	mu sync.RWMutex

	synonyms map[string]map[string][]string

	acronyms map[string]string

	maxExpansionsPerTerm int
}

type SynonymConfig map[string]map[string][]string

type AcronymConfig map[string]map[string]string

func NewSynonymExpander() *SynonymExpander {
	return &SynonymExpander{
		synonyms:             make(map[string]map[string][]string),
		acronyms:             make(map[string]string),
		maxExpansionsPerTerm: 3,
	}
}

func (se *SynonymExpander) LoadSynonymsFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return se.LoadSynonymsFromYAML(data)
}

func (se *SynonymExpander) LoadSynonymsFromYAML(data []byte) error {
	var config SynonymConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	se.mu.Lock()
	defer se.mu.Unlock()

	for domain, terms := range config {
		if se.synonyms[domain] == nil {
			se.synonyms[domain] = make(map[string][]string)
		}
		for term, synonyms := range terms {
			se.synonyms[domain][strings.ToLower(term)] = synonyms
		}
	}

	return nil
}

func (se *SynonymExpander) LoadAcronymsFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return se.LoadAcronymsFromYAML(data)
}

func (se *SynonymExpander) LoadAcronymsFromYAML(data []byte) error {
	var config AcronymConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	se.mu.Lock()
	defer se.mu.Unlock()

	for _, acronyms := range config {
		for acronym, expansion := range acronyms {
			se.acronyms[strings.ToLower(acronym)] = expansion
		}
	}

	return nil
}

func (se *SynonymExpander) LoadDefaults() {
	se.mu.Lock()
	defer se.mu.Unlock()

	se.synonyms["programming"] = map[string][]string{
		"error":     {"exception", "bug", "issue", "fault"},
		"exception": {"error", "throw", "catch"},
		"debug":     {"troubleshoot", "diagnose", "fix"},
		"crash":     {"segfault", "panic", "abort"},
		"fix":       {"resolve", "patch", "repair"},
		"function":  {"method", "procedure", "subroutine"},
		"variable":  {"var", "field", "property"},
		"class":     {"object", "type", "struct"},
		"interface": {"contract", "protocol", "trait"},
		"module":    {"package", "library", "crate"},
		"create":    {"make", "initialize", "new"},
		"delete":    {"remove", "drop", "destroy"},
		"update":    {"modify", "change", "alter"},
		"read":      {"get", "fetch", "retrieve"},
		"api":       {"endpoint", "interface", "service"},
		"database":  {"db", "datastore", "storage"},
		"server":    {"backend", "service", "daemon"},
		"client":    {"frontend", "consumer", "app"},
		"slow":      {"performance", "latency", "bottleneck"},
		"fast":      {"quick", "optimized", "efficient"},
		"memory":    {"ram", "heap", "allocation"},
		"test":      {"unit test", "spec", "assertion"},
		"mock":      {"stub", "fake", "spy"},
	}

	se.synonyms["web"] = map[string][]string{
		"request":        {"req", "call", "fetch"},
		"response":       {"res", "reply", "return"},
		"authentication": {"auth", "login", "signin"},
		"authorization":  {"authz", "permission", "access"},
		"password":       {"credential", "secret"},
		"encrypt":        {"cipher", "encode", "secure"},
		"component":      {"widget", "element"},
		"render":         {"display", "draw", "paint"},
		"style":          {"css", "stylesheet", "theme"},
	}

	se.synonyms["devops"] = map[string][]string{
		"deploy":    {"ship", "release", "rollout"},
		"container": {"docker", "pod", "image"},
		"cluster":   {"swarm", "fleet", "group"},
		"build":     {"compile", "assemble", "package"},
		"pipeline":  {"workflow", "ci", "automation"},
		"log":       {"logging", "trace", "audit"},
		"metric":    {"measurement", "stat", "gauge"},
		"alert":     {"notification", "alarm", "warning"},
	}

	se.synonyms["data"] = map[string][]string{
		"query":  {"search", "find", "lookup", "select"},
		"filter": {"where", "condition", "predicate"},
		"sort":   {"order", "rank", "arrange"},
		"group":  {"aggregate", "collect", "bucket"},
	}

	se.acronyms = map[string]string{
		"api":  "application programming interface",
		"sdk":  "software development kit",
		"cli":  "command line interface",
		"gui":  "graphical user interface",
		"ui":   "user interface",
		"ux":   "user experience",
		"ide":  "integrated development environment",
		"orm":  "object relational mapping",
		"rest": "representational state transfer",
		"rpc":  "remote procedure call",
		"grpc": "google remote procedure call",

		"http":  "hypertext transfer protocol",
		"https": "hypertext transfer protocol secure",
		"ssl":   "secure sockets layer",
		"tls":   "transport layer security",
		"tcp":   "transmission control protocol",
		"udp":   "user datagram protocol",
		"dns":   "domain name system",
		"cdn":   "content delivery network",
		"url":   "uniform resource locator",
		"json":  "javascript object notation",
		"xml":   "extensible markup language",
		"yaml":  "yaml ain't markup language",
		"jwt":   "json web token",
		"oauth": "open authorization",
		"cors":  "cross origin resource sharing",
		"csp":   "content security policy",
		"xss":   "cross site scripting",
		"csrf":  "cross site request forgery",

		"ci":   "continuous integration",
		"cd":   "continuous deployment",
		"cicd": "continuous integration continuous deployment",
		"vm":   "virtual machine",
		"vpc":  "virtual private cloud",
		"iam":  "identity access management",
		"sso":  "single sign on",
		"mfa":  "multi factor authentication",
		"rbac": "role based access control",
		"doh":  "dns over https",

		"sql":   "structured query language",
		"nosql": "not only sql",
		"acid":  "atomicity consistency isolation durability",
		"etl":   "extract transform load",
		"bi":    "business intelligence",

		"ai":   "artificial intelligence",
		"ml":   "machine learning",
		"dl":   "deep learning",
		"nlp":  "natural language processing",
		"llm":  "large language model",
		"gpt":  "generative pretrained transformer",
		"bert": "bidirectional encoder representations from transformers",
		"cnn":  "convolutional neural network",
		"rnn":  "recurrent neural network",
		"gan":  "generative adversarial network",
		"rag":  "retrieval augmented generation",

		"os":  "operating system",
		"cpu": "central processing unit",
		"gpu": "graphics processing unit",
		"ram": "random access memory",
		"ssd": "solid state drive",
		"gc":  "garbage collection",
		"jit": "just in time",

		"tdd": "test driven development",
		"bdd": "behavior driven development",
		"qa":  "quality assurance",
		"e2e": "end to end",

		"pr":   "pull request",
		"mr":   "merge request",
		"cr":   "code review",
		"mvp":  "minimum viable product",
		"poc":  "proof of concept",
		"lgtm": "looks good to me",
		"wip":  "work in progress",
	}
}

func (se *SynonymExpander) SetMaxExpansions(max int) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.maxExpansionsPerTerm = max
}

func (se *SynonymExpander) Expand(terms []string, maxTotal int) []string {
	se.mu.RLock()
	defer se.mu.RUnlock()

	var expanded []string
	seen := make(map[string]bool)

	for _, term := range terms {
		seen[strings.ToLower(term)] = true
	}

	for _, term := range terms {
		termLower := strings.ToLower(term)

		if expansion, ok := se.acronyms[termLower]; ok {
			words := strings.Fields(expansion)
			for _, word := range words {
				wordLower := strings.ToLower(word)
				if !seen[wordLower] {
					expanded = append(expanded, word)
					seen[wordLower] = true
					if len(expanded) >= maxTotal {
						return expanded
					}
				}
			}
		}

		added := 0
		for _, domainSynonyms := range se.synonyms {
			if synonyms, ok := domainSynonyms[termLower]; ok {
				for _, syn := range synonyms {
					synLower := strings.ToLower(syn)
					if !seen[synLower] {
						expanded = append(expanded, syn)
						seen[synLower] = true
						added++
						if added >= se.maxExpansionsPerTerm || len(expanded) >= maxTotal {
							break
						}
					}
				}
			}
			if len(expanded) >= maxTotal {
				break
			}
		}
	}

	return expanded
}

func (se *SynonymExpander) ExpandWithDomain(terms []string, domain string, maxTotal int) []string {
	se.mu.RLock()
	defer se.mu.RUnlock()

	var expanded []string
	seen := make(map[string]bool)

	for _, term := range terms {
		seen[strings.ToLower(term)] = true
	}

	domainSynonyms, hasDomain := se.synonyms[domain]

	for _, term := range terms {
		termLower := strings.ToLower(term)

		if expansion, ok := se.acronyms[termLower]; ok {
			words := strings.Fields(expansion)
			for _, word := range words {
				wordLower := strings.ToLower(word)
				if !seen[wordLower] {
					expanded = append(expanded, word)
					seen[wordLower] = true
					if len(expanded) >= maxTotal {
						return expanded
					}
				}
			}
		}

		if hasDomain {
			if synonyms, ok := domainSynonyms[termLower]; ok {
				added := 0
				for _, syn := range synonyms {
					synLower := strings.ToLower(syn)
					if !seen[synLower] {
						expanded = append(expanded, syn)
						seen[synLower] = true
						added++
						if added >= se.maxExpansionsPerTerm || len(expanded) >= maxTotal {
							break
						}
					}
				}
			}
		}
	}

	return expanded
}

func (se *SynonymExpander) ExpandAcronym(acronym string) string {
	se.mu.RLock()
	defer se.mu.RUnlock()

	return se.acronyms[strings.ToLower(acronym)]
}

func (se *SynonymExpander) GetSynonyms(term string) []string {
	se.mu.RLock()
	defer se.mu.RUnlock()

	termLower := strings.ToLower(term)
	var allSynonyms []string
	seen := make(map[string]bool)

	for _, domainSynonyms := range se.synonyms {
		if synonyms, ok := domainSynonyms[termLower]; ok {
			for _, syn := range synonyms {
				synLower := strings.ToLower(syn)
				if !seen[synLower] {
					allSynonyms = append(allSynonyms, syn)
					seen[synLower] = true
				}
			}
		}
	}

	return allSynonyms
}

func (se *SynonymExpander) AddSynonym(domain, term string, synonyms []string) {
	se.mu.Lock()
	defer se.mu.Unlock()

	if se.synonyms[domain] == nil {
		se.synonyms[domain] = make(map[string][]string)
	}
	se.synonyms[domain][strings.ToLower(term)] = synonyms
}

func (se *SynonymExpander) AddAcronym(acronym, expansion string) {
	se.mu.Lock()
	defer se.mu.Unlock()

	se.acronyms[strings.ToLower(acronym)] = expansion
}

func (se *SynonymExpander) Stats() map[string]int {
	se.mu.RLock()
	defer se.mu.RUnlock()

	synonymCount := 0
	for _, domainSynonyms := range se.synonyms {
		synonymCount += len(domainSynonyms)
	}

	return map[string]int{
		"synonym_terms": synonymCount,
		"domains":       len(se.synonyms),
		"acronyms":      len(se.acronyms),
	}
}

func (se *SynonymExpander) IsAcronym(term string) bool {
	se.mu.RLock()
	defer se.mu.RUnlock()

	_, ok := se.acronyms[strings.ToLower(term)]
	return ok
}
