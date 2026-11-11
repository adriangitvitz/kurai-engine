package server

import (
	"net/http"
	"time"

	"github.com/adriangitvitz/kurai/internal/auth"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

type NewsSearchResponse struct {
	Query    string                `json:"query"`
	Results  []ranker.NewsResult   `json:"results"`
	Clusters []ranker.StoryCluster `json:"clusters,omitempty"`
	Meta     NewsSearchMeta        `json:"meta"`
}

type NewsSearchMeta struct {
	Total            int      `json:"total"`
	Page             int      `json:"page"`
	PageSize         int      `json:"page_size"`
	TotalPages       int      `json:"total_pages"`
	TimeRange        string   `json:"time_range"`
	Category         string   `json:"category,omitempty"`
	TrendingCount    int      `json:"trending_count"`
	ProcessingTimeMs int64    `json:"processing_time_ms"`
	FromCache        bool     `json:"from_cache"`
	EnginesUsed      []string `json:"engines_used,omitempty"`
}

type TrendingResponse struct {
	Clusters []ranker.StoryCluster `json:"clusters"`
	Meta     TrendingMeta          `json:"meta"`
}

type TrendingMeta struct {
	Total            int    `json:"total"`
	Category         string `json:"category,omitempty"`
	ProcessingTimeMs int64  `json:"processing_time_ms"`
	FromCache        bool   `json:"from_cache"`
}

func (s *Server) handleNews(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	resp, err := s.ServiceNews(r.Context(), ServiceNewsReq{
		UserID:        auth.UserIDFromContext(r.Context()),
		Query:         q.Get("q"),
		Page:          parsePositiveInt(q.Get("page"), 1),
		PageSize:      parsePositiveInt(q.Get("page_size"), 0),
		TimeRange:     q.Get("time_range"),
		Category:      q.Get("category"),
		EnableCluster: q.Get("cluster") != "false",
	})
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleNewsTrending(w http.ResponseWriter, r *http.Request) {
	resp, err := s.ServiceTrending(
		r.Context(),
		auth.UserIDFromContext(r.Context()),
		r.URL.Query().Get("category"),
	)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func convertToNewsResults(results []searxng.Result) []ranker.NewsResult {
	newsResults := make([]ranker.NewsResult, 0, len(results))

	for _, r := range results {
		nr := ranker.NewsResult{
			URL:          r.URL,
			Title:        r.Title,
			Snippet:      r.Content,
			Source:       extractSourceName(r),
			SourceDomain: urlutil.ExtractDomain(r.URL),
			Author:       r.Author,
			Thumbnail:    r.Thumbnail,
			Category:     r.Category,
			Engines:      r.Engines,
		}

		if r.PublishedDate != "" {
			if t, err := parsePublishedDate(r.PublishedDate); err == nil {
				nr.PublishedDate = &t
			}
		}

		newsResults = append(newsResults, nr)
	}

	return newsResults
}

func extractSourceName(r searxng.Result) string {
	if len(r.Engines) > 0 {
		for _, eng := range r.Engines {
			if isNewsEngine(eng) {
				return formatEngineName(eng)
			}
		}
	}
	return formatDomainAsSource(urlutil.ExtractDomain(r.URL))
}

func isNewsEngine(engine string) bool {
	newsEngines := map[string]bool{
		"google news": true, "bing news": true, "yahoo news": true,
		"duckduckgo news": true, "brave news": true,
	}
	return newsEngines[engine]
}

func formatEngineName(engine string) string {
	switch engine {
	case "google news":
		return "Google News"
	case "bing news":
		return "Bing News"
	case "yahoo news":
		return "Yahoo News"
	default:
		return engine
	}
}

func formatDomainAsSource(domain string) string {
	domain = removePrefix(domain, "www.")
	domain = removePrefix(domain, "news.")
	parts := splitByDot(domain)
	if len(parts) >= 2 {
		return capitalizeFirst(parts[len(parts)-2])
	}
	return capitalizeFirst(domain)
}

func removePrefix(s, prefix string) string {
	if len(s) > len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func splitByDot(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}

func parsePublishedDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC1123,
		time.RFC1123Z,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"Jan 2, 2006",
		"January 2, 2006",
		"02 Jan 2006",
		"2 Jan 2006",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errInvalidDateFormat
}

type dateFormatError struct{}

func (e dateFormatError) Error() string {
	return "invalid date format"
}

var errInvalidDateFormat = dateFormatError{}
