package urlutil

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	punctuationRe = regexp.MustCompile(`[^\w\s]`)
	pathSplitRe   = regexp.MustCompile(`[/\-_.]`)
)

func Tokenize(s string) []string {
	cleaned := punctuationRe.ReplaceAllString(s, " ")
	parts := strings.Fields(cleaned)

	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(p)
		if len(p) >= 2 {
			result = append(result, p)
		}
	}
	return result
}

func TokenizePath(path string) []string {
	parts := pathSplitRe.Split(path, -1)

	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if len(p) >= 2 {
			result = append(result, p)
		}
	}
	return result
}

func ExtractDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	host := parsed.Hostname()
	if host == "" {
		return ""
	}

	host = strings.TrimPrefix(host, "www.")
	return strings.ToLower(host)
}

func CountURLParams(rawURL string) int {
	if rawURL == "" {
		return 0
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}

	return len(parsed.Query())
}

func ContainsAllTerms(text string, terms []string) bool {
	for _, term := range terms {
		if !strings.Contains(text, term) {
			return false
		}
	}
	return true
}

func HasTermOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}

	set := make(map[string]struct{}, len(a))
	for _, term := range a {
		set[term] = struct{}{}
	}

	for _, term := range b {
		if _, ok := set[term]; ok {
			return true
		}
	}
	return false
}
