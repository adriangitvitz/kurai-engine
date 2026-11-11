package extract

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	trafilatura "github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"

	"github.com/adriangitvitz/kurai/internal/privacy"
)

type Format string

const (
	FormatMarkdown Format = "markdown"
	FormatText     Format = "text"
	FormatBoth     Format = "both"
)

var (
	ErrInvalidURL       = errors.New("extract: invalid url")
	ErrPrivateAddress   = errors.New("extract: private/internal address not allowed")
	ErrUnsupportedType  = errors.New("extract: unsupported content type")
	ErrResponseTooLarge = errors.New("extract: response too large")
	ErrFetchFailed      = errors.New("extract: upstream fetch failed")
	ErrExtractionFailed = errors.New("extract: failed to isolate main content")
)

type Config struct {
	Timeout              time.Duration
	MaxBytes             int64
	MaxRedirects         int
	UserAgent            string
	AllowPrivateNetworks bool
}

type FetchOpts struct {
	Format        Format
	IncludeImages bool
	IncludeLinks  bool
}

type Article struct {
	URL             string     `json:"url"`
	FinalURL        string     `json:"final_url"`
	Title           string     `json:"title,omitempty"`
	Byline          string     `json:"byline,omitempty"`
	PublishedDate   *time.Time `json:"published_date,omitempty"`
	SiteName        string     `json:"site_name,omitempty"`
	Language        string     `json:"language,omitempty"`
	ContentMarkdown string     `json:"content_markdown,omitempty"`
	ContentText     string     `json:"content_text,omitempty"`
	Excerpt         string     `json:"excerpt,omitempty"`
	WordCount       int        `json:"word_count"`
	Images          []string   `json:"images,omitempty"`
	Links           []Link     `json:"links,omitempty"`
	Favicon         string     `json:"favicon,omitempty"`
	FetchedAt       time.Time  `json:"fetched_at"`
	CacheHit        bool       `json:"cache_hit"`
}

type Link struct {
	Href string `json:"href"`
	Text string `json:"text,omitempty"`
}

type Fetcher struct {
	cfg    Config
	client *http.Client
}

func NewFetcher(cfg Config) *Fetcher {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 5 << 20
	}
	if cfg.MaxRedirects <= 0 {
		cfg.MaxRedirects = 5
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "kurai-extract/1.0"
	}

	client := &http.Client{
		Timeout: cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= cfg.MaxRedirects {
				return fmt.Errorf("redirect cap (%d) exceeded", cfg.MaxRedirects)
			}
			return nil
		},
	}

	return &Fetcher{cfg: cfg, client: client}
}

func (f *Fetcher) Fetch(ctx context.Context, rawURL string, opts FetchOpts) (*Article, error) {
	if opts.Format == "" {
		opts.Format = FormatMarkdown
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidURL, rawURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%w: scheme %q not supported", ErrInvalidURL, parsed.Scheme)
	}
	if !f.cfg.AllowPrivateNetworks && IsInternalURL(rawURL) {
		return nil, fmt.Errorf("%w: %s", ErrPrivateAddress, parsed.Host)
	}

	body, finalURL, ct, err := f.do(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	if isPDFContentType(ct) {
		text, perr := pdfToText(ctx, body, f.cfg.Timeout)
		if perr == nil {
			return buildPDFArticle(rawURL, finalURL, text, opts), nil
		}
		if !errors.Is(perr, ErrPDFToolMissing) {
			return nil, perr
		}
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedType, ct)
	}

	if !isHTMLContentType(ct) {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedType, ct)
	}

	doc, err := html.ParseWithOptions(bytes.NewReader(body), html.ParseOptionEnableScripting(false))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExtractionFailed, err)
	}
	promoteNoscriptContent(doc)
	answers := collectAnswers(doc)

	result, err := trafilatura.ExtractDocument(doc, trafilatura.Options{
		OriginalURL:     parsed,
		IncludeImages:   opts.IncludeImages,
		IncludeLinks:    opts.IncludeLinks,
		Deduplicate:     true,
		ExcludeComments: true,
		EnableFallback:  true,
	})
	if err != nil || result == nil || result.ContentNode == nil {
		return nil, fmt.Errorf("%w: %v", ErrExtractionFailed, err)
	}

	appendMissingAnswers(result, answers)

	article := &Article{
		URL:         rawURL,
		FinalURL:    finalURL,
		Title:       result.Metadata.Title,
		Byline:      result.Metadata.Author,
		SiteName:    result.Metadata.Sitename,
		Language:    result.Metadata.Language,
		ContentText: result.ContentText,
		WordCount:   countWords(result.ContentText),
		Excerpt:     buildExcerpt(result.ContentText, 240),
		FetchedAt:   time.Now().UTC(),
	}

	if !result.Metadata.Date.IsZero() {
		d := result.Metadata.Date.UTC()
		article.PublishedDate = &d
	}

	if opts.Format == FormatMarkdown || opts.Format == FormatBoth {
		article.ContentMarkdown = nodeToMarkdown(result.ContentNode)
	}
	if opts.Format == FormatMarkdown {
		article.ContentText = ""
	}

	return article, nil
}

type answer struct {
	author string
	date   string
	body   *html.Node
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func findItemprop(n *html.Node, prop string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		if attr(c, "itemprop") == prop {
			return c
		}
		if found := findItemprop(c, prop); found != nil {
			return found
		}
	}
	return nil
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

func cloneTree(n *html.Node) *html.Node {
	clone := &html.Node{Type: n.Type, DataAtom: n.DataAtom, Data: n.Data, Attr: append([]html.Attribute(nil), n.Attr...)}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		clone.AppendChild(cloneTree(c))
	}
	return clone
}

func collectAnswers(doc *html.Node) []answer {
	var answers []answer
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if prop := attr(n, "itemprop"); prop == "suggestedAnswer" || prop == "acceptedAnswer" {
				if body := findItemprop(n, "text"); body != nil {
					a := answer{body: cloneTree(body)}
					if author := findItemprop(n, "author"); author != nil {
						if name := findItemprop(author, "name"); name != nil {
							a.author = nodeText(name)
						}
					}
					if published := findItemprop(n, "datePublished"); published != nil {
						a.date = attr(published, "datetime")
						if len(a.date) >= 10 {
							a.date = a.date[:10]
						}
					}
					answers = append(answers, a)
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return answers
}

const answerProbeChars = 60

func appendMissingAnswers(result *trafilatura.ExtractResult, answers []answer) {
	extracted := strings.Join(strings.Fields(result.ContentText), " ")
	for _, a := range answers {
		text := nodeText(a.body)
		if text == "" {
			continue
		}
		probe := text
		if len(probe) > answerProbeChars {
			probe = strings.ToValidUTF8(probe[:answerProbeChars], "")
		}
		if strings.Contains(extracted, probe) {
			continue
		}

		byline := strings.TrimSpace(strings.Join([]string{a.author, a.date}, " · "))
		byline = strings.Trim(byline, " ·")
		result.ContentNode.AppendChild(&html.Node{Type: html.ElementNode, Data: "hr"})
		if byline != "" {
			strong := &html.Node{Type: html.ElementNode, Data: "strong"}
			strong.AppendChild(&html.Node{Type: html.TextNode, Data: byline})
			p := &html.Node{Type: html.ElementNode, Data: "p"}
			p.AppendChild(strong)
			result.ContentNode.AppendChild(p)
			result.ContentText += "\n\n" + byline
		}
		a.body.Data = "div"
		result.ContentNode.AppendChild(a.body)
		result.ContentText += "\n" + text
	}
}

const noscriptContentMinChars = 500

func promoteNoscriptContent(doc *html.Node) {
	var substantial []*html.Node
	var walk func(n *html.Node, inBody bool)
	walk = func(n *html.Node, inBody bool) {
		if n.Type == html.ElementNode && n.Data == "body" {
			inBody = true
		}
		if inBody && n.Type == html.ElementNode && n.Data == "noscript" {
			if textLength(n) >= noscriptContentMinChars {
				substantial = append(substantial, n)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inBody)
		}
	}
	walk(doc, false)

	for _, n := range substantial {
		for c := n.FirstChild; c != nil; c = n.FirstChild {
			n.RemoveChild(c)
			n.Parent.InsertBefore(c, n)
		}
		n.Parent.RemoveChild(n)
	}
}

func textLength(n *html.Node) int {
	if n.Type == html.TextNode {
		return len(strings.TrimSpace(n.Data))
	}
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
		return 0
	}
	total := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		total += textLength(c)
	}
	return total
}

func nodeToMarkdown(node *html.Node) (md string) {
	if node == nil {
		return ""
	}

	defer func() {
		if r := recover(); r != nil {
			slog.Warn("extract: html-to-markdown panicked, returning empty markdown", "panic", r)
			md = ""
		}
	}()

	var buf bytes.Buffer
	if err := html.Render(&buf, node); err != nil {
		slog.Warn("extract: failed to render content node to html", "error", err)
		return ""
	}

	out, err := htmltomarkdown.ConvertString(buf.String())
	if err != nil {
		slog.Warn("extract: html-to-markdown conversion failed", "error", err)
		return ""
	}
	return strings.TrimSpace(out)
}

func buildPDFArticle(rawURL, finalURL, text string, opts FetchOpts) *Article {
	a := &Article{
		URL:         rawURL,
		FinalURL:    finalURL,
		Title:       pdfTitleFromURL(finalURL),
		ContentText: text,
		WordCount:   countWords(text),
		Excerpt:     buildExcerpt(text, 240),
		FetchedAt:   time.Now().UTC(),
	}
	if opts.Format == FormatMarkdown || opts.Format == FormatBoth || opts.Format == "" {
		a.ContentMarkdown = text
	}
	if opts.Format == FormatMarkdown {
		a.ContentText = ""
	}
	return a
}

func pdfTitleFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	seg := path.Base(u.Path)
	if seg == "." || seg == "/" {
		return ""
	}
	return strings.TrimSuffix(seg, ".pdf")
}

func (f *Fetcher) do(ctx context.Context, rawURL string) ([]byte, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	privacy.AnonymizeRequest(req)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.1")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, "", "", fmt.Errorf("%w: status %d", ErrFetchFailed, resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, f.cfg.MaxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: read body: %v", ErrFetchFailed, err)
	}
	if int64(len(body)) > f.cfg.MaxBytes {
		return nil, "", "", fmt.Errorf("%w: %d > %d", ErrResponseTooLarge, len(body), f.cfg.MaxBytes)
	}

	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	ct := resp.Header.Get("Content-Type")
	return body, finalURL, ct, nil
}

func isHTMLContentType(ct string) bool {
	if ct == "" {
		return true
	}
	ct = strings.ToLower(ct)
	return strings.HasPrefix(ct, "text/html") || strings.HasPrefix(ct, "application/xhtml")
}

func countWords(s string) int {
	if s == "" {
		return 0
	}
	count := 0
	inWord := false
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r':
			inWord = false
		default:
			if !inWord {
				count++
				inWord = true
			}
		}
	}
	return count
}

func buildExcerpt(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	count := 0
	for i := range s {
		if count >= maxRunes {
			return strings.TrimSpace(s[:i]) + "…"
		}
		count++
	}
	return s
}
