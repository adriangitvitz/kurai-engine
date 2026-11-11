package ranker

import "strings"

type ContentType string

const (
	ContentArticle    ContentType = "article"
	ContentStudy      ContentType = "study"
	ContentVideo      ContentType = "video"
	ContentDiscussion ContentType = "discussion"
	ContentGuide      ContentType = "guide"
)

var (
	videoHosts = []string{
		"youtube.com", "youtu.be", "vimeo.com", "dailymotion.com",
		"twitch.tv", "rumble.com", "bilibili.com",
	}
	discussionHosts = []string{
		"reddit.com", "news.ycombinator.com", "stackoverflow.com",
		"stackexchange.com", "superuser.com", "serverfault.com",
		"askubuntu.com", "quora.com", "lobste.rs", "lemmy.",
	}
	discussionPaths = []string{
		"/forum", "/discussion", "/thread/", "/threads/", "/comments/",
	}
	academicHosts = []string{
		".edu", ".ac.", "arxiv.org", "biorxiv.org", "medrxiv.org",
		"pubmed", "ncbi.nlm.nih.gov", "doi.org", "jstor.org",
		"sciencedirect.com", "springer.com", "nature.com",
		"researchgate.net", "ssrn.com", "semanticscholar.org",
		"scholar.google", "plos.org", "acm.org", "ieee.org",
	}
	guideHosts = []string{
		"developer.mozilla.org", "readthedocs.io", "readthedocs.org",
		"w3schools.com", "freecodecamp.org",
	}
	guidePaths = []string{
		"/docs", "/doc/", "/guide", "/tutorial", "/reference",
		"/getting-started", "/how-to", "/handbook", "/manual",
		"/learn/", "/quickstart",
	}
)

func ClassifyContentType(rawURL string, scores DomainScores) ContentType {
	u := strings.ToLower(rawURL)

	switch {
	case containsAnySubstring(u, videoHosts):
		return ContentVideo
	case containsAnySubstring(u, discussionHosts), containsAnySubstring(u, discussionPaths):
		return ContentDiscussion
	case containsAnySubstring(u, academicHosts):
		return ContentStudy
	case containsAnySubstring(u, guideHosts), containsAnySubstring(u, guidePaths):
		return ContentGuide
	}

	switch scores.PrimaryDomain() {
	case DomainAcademic, DomainPsychology:
		return ContentStudy
	case DomainDocumentation:
		return ContentGuide
	}

	return ContentArticle
}

func containsAnySubstring(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
