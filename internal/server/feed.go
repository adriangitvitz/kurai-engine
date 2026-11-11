package server

import (
	"encoding/xml"
	"net/http"
	"time"

	"github.com/adriangitvitz/kurai/internal/apierror"
	"github.com/adriangitvitz/kurai/internal/auth"
	"github.com/adriangitvitz/kurai/internal/ranker"
)

type FeedFormat string

const (
	FeedFormatRSS  FeedFormat = "rss"
	FeedFormatAtom FeedFormat = "atom"
)

type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language,omitempty"`
	PubDate       string    `xml:"pubDate,omitempty"`
	LastBuildDate string    `xml:"lastBuildDate,omitempty"`
	Generator     string    `xml:"generator,omitempty"`
	Items         []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate,omitempty"`
	GUID        string `xml:"guid,omitempty"`
	Author      string `xml:"author,omitempty"`
	Source      string `xml:"source,omitempty"`
}

type AtomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Link    []AtomLink  `xml:"link"`
	Updated string      `xml:"updated"`
	Author  *AtomAuthor `xml:"author,omitempty"`
	Entries []AtomEntry `xml:"entry"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type AtomAuthor struct {
	Name string `xml:"name"`
}

type AtomEntry struct {
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Link    []AtomLink  `xml:"link"`
	Updated string      `xml:"updated"`
	Summary string      `xml:"summary,omitempty"`
	Content string      `xml:"content,omitempty"`
	Author  *AtomAuthor `xml:"author,omitempty"`
}

type FeedGenerator struct {
	baseURL   string
	feedTitle string
}

func NewFeedGenerator(baseURL, feedTitle string) *FeedGenerator {
	return &FeedGenerator{
		baseURL:   baseURL,
		feedTitle: feedTitle,
	}
}

func (fg *FeedGenerator) GenerateRSS(query string, news []ranker.NewsResult) *RSSFeed {
	now := time.Now().UTC()

	channel := RSSChannel{
		Title:         fg.feedTitle + " - " + query,
		Link:          fg.baseURL + "/news?q=" + query,
		Description:   "News results for: " + query,
		Language:      "en",
		PubDate:       now.Format(time.RFC1123Z),
		LastBuildDate: now.Format(time.RFC1123Z),
		Generator:     "Kurai Search",
		Items:         make([]RSSItem, 0, len(news)),
	}

	for _, n := range news {
		item := RSSItem{
			Title:       n.Title,
			Link:        n.URL,
			Description: n.Snippet,
			GUID:        n.URL,
		}

		if n.PublishedDate != nil && !n.PublishedDate.IsZero() {
			item.PubDate = n.PublishedDate.Format(time.RFC1123Z)
		}

		if n.Author != "" {
			item.Author = n.Author
		}

		if n.Source != "" {
			item.Source = n.Source
		}

		channel.Items = append(channel.Items, item)
	}

	return &RSSFeed{
		Version: "2.0",
		Channel: channel,
	}
}

func (fg *FeedGenerator) GenerateAtom(query string, news []ranker.NewsResult) *AtomFeed {
	now := time.Now().UTC()
	feedID := fg.baseURL + "/news/feed?q=" + query

	feed := &AtomFeed{
		Title:   fg.feedTitle + " - " + query,
		ID:      feedID,
		Updated: now.Format(time.RFC3339),
		Link: []AtomLink{
			{Href: fg.baseURL + "/news?q=" + query, Rel: "alternate", Type: "text/html"},
			{Href: feedID + "&format=atom", Rel: "self", Type: "application/atom+xml"},
		},
		Entries: make([]AtomEntry, 0, len(news)),
	}

	for _, n := range news {
		entry := AtomEntry{
			Title:   n.Title,
			ID:      n.URL,
			Summary: n.Snippet,
			Link: []AtomLink{
				{Href: n.URL, Rel: "alternate"},
			},
		}

		if n.PublishedDate != nil && !n.PublishedDate.IsZero() {
			entry.Updated = n.PublishedDate.Format(time.RFC3339)
		} else {
			entry.Updated = now.Format(time.RFC3339)
		}

		if n.Author != "" {
			entry.Author = &AtomAuthor{Name: n.Author}
		}

		feed.Entries = append(feed.Entries, entry)
	}

	return feed
}

func (s *Server) handleNewsFeed(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		s.writeError(w, http.StatusBadRequest, apierror.CodeMissingQuery, "query parameter 'q' is required")
		return
	}

	format := FeedFormat(r.URL.Query().Get("format"))
	if format == "" {
		format = FeedFormatRSS
	}

	if format != FeedFormatRSS && format != FeedFormatAtom {
		s.writeError(w, http.StatusBadRequest, apierror.CodeInvalidRequest, "format must be 'rss' or 'atom'")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	cacheKey := "news:" + query + ":week"
	var newsResults []ranker.NewsResult

	if cachedData, _, ok := s.cache.GetPage(userID, cacheKey, 1, 50); ok {
		for _, r := range cachedData {
			if nr, ok := r.(ranker.NewsResult); ok {
				newsResults = append(newsResults, nr)
			}
		}
	} else {
		http.Redirect(w, r, "/news?q="+query, http.StatusTemporaryRedirect)
		return
	}

	if len(newsResults) > 50 {
		newsResults = newsResults[:50]
	}

	fg := NewFeedGenerator(getBaseURL(r), "Kurai News")

	switch format {
	case FeedFormatRSS:
		feed := fg.GenerateRSS(query, newsResults)
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		xml.NewEncoder(w).Encode(feed)

	case FeedFormatAtom:
		feed := fg.GenerateAtom(query, newsResults)
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		xml.NewEncoder(w).Encode(feed)
	}
}

func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := r.Host
	if fwdHost := r.Header.Get("X-Forwarded-Host"); fwdHost != "" {
		host = fwdHost
	}

	return scheme + "://" + host
}

func ParseFeedFormat(s string) FeedFormat {
	switch s {
	case "rss", "rss2", "rss20":
		return FeedFormatRSS
	case "atom", "atom1", "atom10":
		return FeedFormatAtom
	default:
		return FeedFormatRSS
	}
}
