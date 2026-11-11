package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/adriangitvitz/kurai/internal/apierror"
	"github.com/adriangitvitz/kurai/internal/auth"
	"github.com/adriangitvitz/kurai/internal/server"
)

const stdioMCPUser = "mcp-stdio"

func callerID(ctx context.Context) string {
	if id := auth.UserIDFromContext(ctx); id != "" {
		return id
	}
	return stdioMCPUser
}

func readOnly(title string) *mcpsdk.ToolAnnotations {
	openWorld := true
	return &mcpsdk.ToolAnnotations{
		Title:          title,
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  &openWorld,
	}
}

const defaultMaxContentChars = 32000

func capContent(body string, max int) string {
	if max <= 0 || len(body) <= max {
		return body
	}
	r := []rune(body)
	if len(r) <= max {
		return body
	}
	return string(r[:max]) + fmt.Sprintf(
		"\n\n[truncated: %d of %d characters. Re-read this URL with "+
			"extract_content(focus=\"<your question>\", passages_only=true) for the "+
			"relevant section, or raise max_chars.]", max, len(r))
}

func resolveMaxChars(v *int) int {
	if v == nil {
		return defaultMaxContentChars
	}
	return *v
}

type webSearchInput struct {
	Query          string   `json:"query" jsonschema:"Search query (required, non-empty)."`
	Limit          int      `json:"limit,omitempty" jsonschema:"Max results to return (1–20, default page_size)."`
	Category       string   `json:"category,omitempty" jsonschema:"Force a query category: tech, academic, news, sports, or general. Auto-detected when empty."`
	Page           int      `json:"page,omitempty" jsonschema:"1-based page number (default 1)."`
	Debug          bool     `json:"debug,omitempty" jsonschema:"Return the intent-detection breakdown alongside results. Implies response_format=full, which costs roughly 2x the tokens."`
	ContentType    string   `json:"content_type,omitempty" jsonschema:"Keep only one content type: article, study, video, discussion, or guide. meta.facets shows the count per type; unknown values are ignored."`
	IncludeDomains []string `json:"include_domains,omitempty" jsonschema:"Keep only results whose host matches one of these domains, e.g. [\"docs.python.org\"]. Also sent upstream as site: so the engines search inside those sites."`
	ExcludeDomains []string `json:"exclude_domains,omitempty" jsonschema:"Drop results whose host matches one of these domains, e.g. [\"pinterest.com\"]."`
	DateFrom       string   `json:"date_from,omitempty" jsonschema:"Keep only results published on/after this date (YYYY-MM-DD). A date within the past year also narrows the upstream fetch to day, week, month or year (meta.time_range). Undated results pass through unless date_strict is set."`
	DateTo         string   `json:"date_to,omitempty" jsonschema:"Keep only results published on/before this date (YYYY-MM-DD). Undated results pass through unless date_strict is set."`
	DateStrict     bool     `json:"date_strict,omitempty" jsonschema:"With date_from/date_to, also drop results that carry no date. Most web results are undated, so this often returns nothing; meta.undated_filtered reports how many were dropped."`
	Language       string   `json:"language,omitempty" jsonschema:"BCP-47 language for the upstream fetch, e.g. en or es (applied when the query is first fetched)."`
	ResponseFormat string   `json:"response_format,omitempty" jsonschema:"Result verbosity: compact (default — url/title/snippet/score/ranks only) or full (adds badges, signal_ranks, domain_scores for debugging)."`
	ExtractTop     int      `json:"extract_top,omitempty" jsonschema:"Fused search+extract: also extract the top N results' pages server-side (1–5) and attach their markdown as 'content'. Saves a separate extract_content round trip when you know you'll read the top hits."`
	SnippetLength  int      `json:"snippet_length,omitempty" jsonschema:"Truncate each result's snippet to this many characters (50–500). 0 = no truncation."`
	MaxChars       *int     `json:"max_chars,omitempty" jsonschema:"Total character budget for extracted page bodies across this whole response (default 32000, 0 = no limit), split evenly across the pages extract_top attached. Truncated bodies carry a marker saying how much was dropped."`
}

type newsSearchInput struct {
	Query     string `json:"query" jsonschema:"Search query (required, non-empty)."`
	Limit     int    `json:"limit,omitempty" jsonschema:"Max results (1–50, default 20)."`
	TimeRange string `json:"time_range,omitempty" jsonschema:"Time-range filter: day, week, month, or year."`
	Category  string `json:"category,omitempty" jsonschema:"Optional news category filter."`
	Cluster   *bool  `json:"cluster,omitempty" jsonschema:"Group near-duplicate stories into clusters (default true)."`
	Page      int    `json:"page,omitempty" jsonschema:"1-based page number (default 1)."`
}

type imageSearchInput struct {
	Query      string `json:"query" jsonschema:"Search query (required, non-empty)."`
	Limit      int    `json:"limit,omitempty" jsonschema:"Max results (1–40, default 30)."`
	SafeSearch *int   `json:"safesearch,omitempty" jsonschema:"SafeSearch level: 0=off, 1=moderate (default), 2=strict."`
	Size       string `json:"size,omitempty" jsonschema:"Size filter: small, medium, or large."`
	Page       int    `json:"page,omitempty" jsonschema:"1-based page number (default 1)."`
}

func registerSearchTools(srv *mcpsdk.Server, svc *server.Server) {
	mcpsdk.AddTool(srv,
		&mcpsdk.Tool{
			Name:        "web_search",
			Annotations: readOnly("Web search (ranked)"),
			Description: "Search the web via Kurai's ranked meta-search (SearXNG → BM25 + RRF + MMR). " +
				"Results are compact by default: url, title, snippet, score, final_rank, rrf_rank, engines, domain, content_type, published_date. " +
				"When rrf_rank ≠ final_rank, MMR moved the result for diversity; if you specifically need a " +
				"demoted high-score result, page or filter further. Set response_format=full for debugging " +
				"metadata (badges, signal_ranks, domain_scores).\n" +
				"\n" +
				"Narrow results with content_type, include_domains/exclude_domains, date_from/date_to, or language. " +
				"meta.facets reports the count per content_type, so a strong pattern is: search once, read meta.facets, " +
				"then refine with content_type (e.g. study or video) without re-fetching.\n" +
				"\n" +
				"Tip: set extract_top=N (1–5) to also receive the top-N pages' full markdown as 'content' in one " +
				"call, skipping the separate extract_content round trip when you know you'll read the top hits.",
		},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in webSearchInput) (*mcpsdk.CallToolResult, any, error) {
			ctx = withRequestID(ctx)
			for _, e := range []error{
				validateRange("limit", in.Limit, 1, 20),
				validateRange("extract_top", in.ExtractTop, 1, 5),
				validateRange("snippet_length", in.SnippetLength, 50, 500),
				validateRange("page", in.Page, 1, 1000),
			} {
				if e != nil {
					return errResult(ctx, e), nil, nil
				}
			}
			page := in.Page
			if page <= 0 {
				page = 1
			}
			resp, err := svc.ServiceSearch(ctx, server.ServiceSearchReq{
				UserID:         callerID(ctx),
				Query:          in.Query,
				Page:           page,
				Category:       in.Category,
				Debug:          in.Debug,
				Language:       in.Language,
				ContentType:    in.ContentType,
				IncludeDomains: in.IncludeDomains,
				ExcludeDomains: in.ExcludeDomains,
				DateFrom:       in.DateFrom,
				DateTo:         in.DateTo,
				DateStrict:     in.DateStrict,
				ExtractTop:     in.ExtractTop,
				Limit:          in.Limit,
				SnippetLength:  in.SnippetLength,
			})
			if err != nil {
				return errResult(ctx, err), nil, nil
			}
			if budget := resolveMaxChars(in.MaxChars); budget > 0 {
				withContent := 0
				for i := range resp.Results {
					if resp.Results[i].ExtractedContent != "" {
						withContent++
					}
				}
				if withContent > 0 {
					per := budget / withContent
					if per < 1 {
						per = 1
					}
					for i := range resp.Results {
						resp.Results[i].ExtractedContent = capContent(resp.Results[i].ExtractedContent, per)
					}
				}
			}
			if in.ResponseFormat != "full" && !in.Debug {
				return jsonResult(server.ToCompactSearchResponse(resp)), nil, nil
			}
			return jsonResult(resp), nil, nil
		})

	mcpsdk.AddTool(srv,
		&mcpsdk.Tool{
			Name:        "news_search",
			Annotations: readOnly("News search"),
			Description: "Search news via Kurai: news-specific ranking plus clustering that collapses " +
				"wire-syndicated copies of the same story into one entry.\n" +
				"\n" +
				"Reach for this instead of web_search whenever the question is time-bound - " +
				"\"what happened this week\", \"latest release\", \"recent outage\", anything about " +
				"current events. time_range (day|week|month|year) is the cheap way to bound " +
				"recency; web_search's date_from/date_to is the fallback for a precise window.\n" +
				"\n" +
				"Caveat: upstream news results often carry no parseable date, so published_date " +
				"may be absent. Never infer one that isn't there.",
		},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in newsSearchInput) (*mcpsdk.CallToolResult, any, error) {
			ctx = withRequestID(ctx)
			if err := validateRange("limit", in.Limit, 1, 50); err != nil {
				return errResult(ctx, err), nil, nil
			}
			page := in.Page
			if page <= 0 {
				page = 1
			}
			cluster := true
			if in.Cluster != nil {
				cluster = *in.Cluster
			}
			limit := in.Limit
			pageSize := 20
			if limit > 0 && limit < pageSize {
				pageSize = limit
			}
			resp, err := svc.ServiceNews(ctx, server.ServiceNewsReq{
				UserID:        callerID(ctx),
				Query:         in.Query,
				Page:          page,
				PageSize:      pageSize,
				TimeRange:     in.TimeRange,
				Category:      in.Category,
				EnableCluster: cluster,
			})
			if err != nil {
				return errResult(ctx, err), nil, nil
			}
			if limit > 0 && len(resp.Results) > limit {
				resp.Results = resp.Results[:limit]
			}
			return jsonResult(resp), nil, nil
		})

	mcpsdk.AddTool(srv,
		&mcpsdk.Tool{
			Name:        "image_search",
			Annotations: readOnly("Image search"),
			Description: "Search images via Kurai. Returns image URLs, dimensions, the page each image " +
				"came from, and a source authority score.\n" +
				"\n" +
				"Use it only when an image is the actual deliverable - a diagram, a screenshot, a " +
				"product photo the user asked to see. It is not a way to research a topic; " +
				"web_search is.\n" +
				"\n" +
				"Dimensions are best-effort: engines that don't report a resolution yield " +
				"width=0 height=0, so filter on size rather than assuming it is populated.",
		},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in imageSearchInput) (*mcpsdk.CallToolResult, any, error) {
			ctx = withRequestID(ctx)
			if err := validateRange("limit", in.Limit, 1, 40); err != nil {
				return errResult(ctx, err), nil, nil
			}
			page := in.Page
			if page <= 0 {
				page = 1
			}
			limit := in.Limit
			pageSize := 30
			if limit > 0 && limit < pageSize {
				pageSize = limit
			}
			ss := 1
			if in.SafeSearch != nil {
				ss = *in.SafeSearch
			}
			resp, err := svc.ServiceImages(ctx, server.ServiceImagesReq{
				UserID:     callerID(ctx),
				Query:      in.Query,
				Page:       page,
				PageSize:   pageSize,
				SafeSearch: ss,
				Size:       in.Size,
			})
			if err != nil {
				return errResult(ctx, err), nil, nil
			}
			if limit > 0 && len(resp.Results) > limit {
				resp.Results = resp.Results[:limit]
			}
			return jsonResult(resp), nil, nil
		})
}

type findSimilarInput struct {
	URL   string `json:"url" jsonschema:"Source article URL (required)."`
	Limit int    `json:"limit,omitempty" jsonschema:"Max similar results (1–20, default 10)."`
}

func registerFindSimilarTool(srv *mcpsdk.Server, svc *server.Server) {
	mcpsdk.AddTool(srv,
		&mcpsdk.Tool{
			Name:        "find_similar",
			Annotations: readOnly("Find similar pages"),
			Description: "Given an article URL, return ranked results on the same topic from other " +
				"sites. Kurai extracts the page's title, derives a query from it, and excludes " +
				"the source host so you get corroboration rather than the same page again.\n" +
				"\n" +
				"Use it to broaden once you have one good source - a second opinion, a competing " +
				"account, the other half of a story. If you instead want to go deeper into the " +
				"page you already have, that is extract_content with focus.",
		},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in findSimilarInput) (*mcpsdk.CallToolResult, any, error) {
			ctx = withRequestID(ctx)
			if err := validateRange("limit", in.Limit, 1, 20); err != nil {
				return errResult(ctx, err), nil, nil
			}
			resp, err := svc.ServiceFindSimilar(ctx, server.ServiceFindSimilarReq{
				UserID: callerID(ctx),
				URL:    in.URL,
				Limit:  in.Limit,
			})
			if err != nil {
				return errResult(ctx, err), nil, nil
			}
			return jsonResult(resp), nil, nil
		})
}

type extractInput struct {
	URL           string `json:"url" jsonschema:"Absolute http(s) URL of the article to extract (required)."`
	Format        string `json:"format,omitempty" jsonschema:"Output format: markdown (default), text, or both."`
	IncludeImages bool   `json:"include_images,omitempty" jsonschema:"Include image references in the extracted body."`
	IncludeLinks  bool   `json:"include_links,omitempty" jsonschema:"Include outbound link URLs in the extracted body."`
	TimeoutMs     int    `json:"timeout_ms,omitempty" jsonschema:"Per-request timeout in milliseconds (1000–30000)."`
	Focus         string `json:"focus,omitempty" jsonschema:"Query-focused passages: chunk the content by headings and rank chunks against this query, most relevant first. Use when you only need one section of a long page."`
	PassagesOnly  bool   `json:"passages_only,omitempty" jsonschema:"With focus: omit the full content fields and return only the ranked passages — maximum token saving."`
	TopPassages   int    `json:"top_passages,omitempty" jsonschema:"With focus: how many passages to return (default 5)."`
	MaxChars      *int   `json:"max_chars,omitempty" jsonschema:"Cap the returned body at this many characters (default 32000, 0 = no limit). A truncated body carries a marker saying how much was dropped; prefer focus + passages_only when you only need one section."`
}

func registerExtractTool(srv *mcpsdk.Server, svc *server.Server) {
	mcpsdk.AddTool(srv,
		&mcpsdk.Tool{
			Name:        "extract_content",
			Annotations: readOnly("Read a page"),
			Description: "Fetch a URL and return its cleaned main content - title, byline, published " +
				"date, and markdown with headings preserved. Navigation, ads, footers and " +
				"related-post rails are stripped. This is your page-reading tool.\n" +
				"\n" +
				"Token cost is the thing to get right. A long article runs to thousands of " +
				"tokens, and bodies are capped (default ~32k characters, see max_chars) with an " +
				"inline marker when truncated. When you have a specific question about the page, " +
				"pass focus=\"<your question>\" with passages_only=true: you get the " +
				"heading-chunks ranked against your question, typically a couple hundred tokens " +
				"instead of the whole body. Ask for the full body only when you genuinely need it.\n" +
				"\n" +
				"HTML only - a PDF or other binary returns unsupported_content_type. The " +
				"underlying extraction is cached for 24h and is focus-independent, so asking the " +
				"same URL with different focus queries is cheap.",
		},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in extractInput) (*mcpsdk.CallToolResult, any, error) {
			ctx = withRequestID(ctx)
			if err := validateRange("timeout_ms", in.TimeoutMs, 1000, 30000); err != nil {
				return errResult(ctx, err), nil, nil
			}
			req := server.ServiceExtractReq{
				UserID:        callerID(ctx),
				URL:           in.URL,
				Format:        in.Format,
				IncludeImages: in.IncludeImages,
				IncludeLinks:  in.IncludeLinks,
				Focus:         in.Focus,
				PassagesOnly:  in.PassagesOnly,
				TopPassages:   in.TopPassages,
			}
			if in.TimeoutMs > 0 {
				req.Timeout = time.Duration(in.TimeoutMs) * time.Millisecond
			}
			resp, err := svc.ServiceExtract(ctx, req)
			if err != nil {
				return errResult(ctx, err), nil, nil
			}
			if max := resolveMaxChars(in.MaxChars); max > 0 && resp.Result != nil {
				trimmed := *resp.Result
				trimmed.ContentMarkdown = capContent(trimmed.ContentMarkdown, max)
				trimmed.ContentText = capContent(trimmed.ContentText, max)
				resp.Result = &trimmed
			}
			if in.Focus != "" {
				return jsonResult(resp), nil, nil
			}
			return jsonResult(resp.Result), nil, nil
		})
}

func jsonResult(v any) *mcpsdk.CallToolResult {
	b, err := json.Marshal(v)
	if err != nil {
		return errResult(context.Background(), fmt.Errorf("marshal result: %w", err))
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(b)}},
	}
}

type toolError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
	Status    int    `json:"status,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

var retryableCodes = map[string]bool{
	apierror.CodeFetchTimeout:        true,
	apierror.CodeUpstreamUnavailable: true,
}

func errResult(ctx context.Context, err error) *mcpsdk.CallToolResult {
	te := toolError{
		Code:      apierror.CodeInternalError,
		Message:   err.Error(),
		RequestID: server.RequestIDFromContext(ctx),
	}
	var se *server.ServiceError
	if errors.As(err, &se) {
		te.Code = se.Code
		te.Message = se.Message
		te.Status = se.Status
	}
	te.Retryable = retryableCodes[te.Code]

	b, mErr := json.Marshal(te)
	if mErr != nil {
		b = []byte(te.Code + ": " + te.Message)
	}
	return &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(b)}},
	}
}

func validateRange(name string, v, min, max int) error {
	if v == 0 || (v >= min && v <= max) {
		return nil
	}
	return newInputError(fmt.Sprintf("%s must be between %d and %d, got %d", name, min, max, v))
}

func newInputError(msg string) error {
	return &server.ServiceError{Status: 400, Code: apierror.CodeInvalidRequest, Message: msg}
}

func withRequestID(ctx context.Context) context.Context {
	return server.WithRequestID(ctx, server.NewRequestID())
}
