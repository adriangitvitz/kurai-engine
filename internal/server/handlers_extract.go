package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/adriangitvitz/kurai/internal/apierror"
	"github.com/adriangitvitz/kurai/internal/auth"
	"github.com/adriangitvitz/kurai/internal/extract"
)

type ExtractResponse struct {
	Result   *extract.Article `json:"result"`
	Passages []ScoredPassage  `json:"passages,omitempty"`
}

func (s *Server) handleExtract(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var timeout time.Duration
	if raw := q.Get("timeout_ms"); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil && ms > 0 {
			timeout = time.Duration(ms) * time.Millisecond
		}
	}

	resp, err := s.ServiceExtract(r.Context(), ServiceExtractReq{
		UserID:        auth.UserIDFromContext(r.Context()),
		URL:           q.Get("url"),
		Format:        q.Get("format"),
		IncludeImages: q.Get("include_images") == "true",
		IncludeLinks:  q.Get("include_links") == "true",
		Timeout:       timeout,
		Focus:         q.Get("focus"),
		PassagesOnly:  q.Get("passages_only") == "true",
		TopPassages:   parsePositiveInt(q.Get("top_passages"), 0),
	})
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, resp)
}

type ServiceExtractReq struct {
	UserID        string
	URL           string
	Format        string
	IncludeImages bool
	IncludeLinks  bool
	Timeout       time.Duration

	Focus        string
	PassagesOnly bool
	TopPassages  int
}

func (s *Server) ServiceExtract(ctx context.Context, req ServiceExtractReq) (*ExtractResponse, error) {
	if s.extractor == nil {
		return nil, newServiceError(http.StatusNotFound, apierror.CodeNotFound, "/extract is disabled in config")
	}
	if req.URL == "" {
		return nil, newServiceError(http.StatusBadRequest, apierror.CodeInvalidURL, "url parameter is required")
	}

	host := extractHostFromURL(req.URL)
	if host != "" && s.blocklist.IsBlocked(host) {
		return nil, newServiceError(http.StatusUnavailableForLegalReasons, apierror.CodeBlockedDomain, "host is in blocklist")
	}

	cacheKey := buildExtractCacheKey(req)

	if cached, _, ok := s.cache.GetPage(req.UserID, cacheKey, 1, 1); ok && len(cached) > 0 {
		if a, ok := cached[0].(*extract.Article); ok {
			a.CacheHit = true
			return s.shapeExtractResponse(a, req), nil
		}
	}

	fetchCtx := ctx
	if req.Timeout > 0 {
		capped := req.Timeout
		if capped > 30*time.Second {
			capped = 30 * time.Second
		}
		var cancel context.CancelFunc
		fetchCtx, cancel = context.WithTimeout(ctx, capped)
		defer cancel()
	}

	article, err := s.extractor.Fetch(fetchCtx, req.URL, extract.FetchOpts{
		Format:        extract.Format(req.Format),
		IncludeImages: req.IncludeImages,
		IncludeLinks:  req.IncludeLinks,
	})
	if err != nil {
		return nil, mapExtractError(err)
	}

	s.cache.Set(req.UserID, cacheKey, []any{article})

	return s.shapeExtractResponse(article, req), nil
}

func (s *Server) shapeExtractResponse(article *extract.Article, req ServiceExtractReq) *ExtractResponse {
	resp := &ExtractResponse{Result: article}
	if req.Focus == "" {
		return resp
	}

	source := article.ContentMarkdown
	if source == "" {
		source = article.ContentText
	}
	resp.Passages = scorePassages(source, req.Focus, req.TopPassages)

	if req.PassagesOnly {
		slim := *article
		slim.ContentMarkdown = ""
		slim.ContentText = ""
		resp.Result = &slim
	}
	return resp
}

func mapExtractError(err error) error {
	switch {
	case errors.Is(err, extract.ErrInvalidURL):
		return newServiceError(http.StatusBadRequest, apierror.CodeInvalidURL, err.Error())
	case errors.Is(err, extract.ErrPrivateAddress):
		return newServiceError(http.StatusBadRequest, apierror.CodeInvalidURL, err.Error())
	case errors.Is(err, extract.ErrUnsupportedType):
		return newServiceError(http.StatusUnsupportedMediaType, apierror.CodeUnsupportedContentType, err.Error())
	case errors.Is(err, extract.ErrResponseTooLarge):
		return newServiceError(http.StatusRequestEntityTooLarge, apierror.CodeResponseTooLarge, err.Error())
	case errors.Is(err, extract.ErrExtractionFailed):
		return newServiceError(http.StatusUnprocessableEntity, apierror.CodeExtractionFailed, err.Error())
	case errors.Is(err, extract.ErrFetchFailed):
		return newServiceError(http.StatusBadGateway, apierror.CodeUpstreamUnavailable, err.Error())
	default:
		return newServiceError(http.StatusInternalServerError, apierror.CodeInternalError, err.Error())
	}
}

func buildExtractCacheKey(req ServiceExtractReq) string {
	format := req.Format
	if format == "" {
		format = "markdown"
	}
	flags := "0"
	if req.IncludeImages {
		flags = "i"
	}
	if req.IncludeLinks {
		flags += "l"
	}
	return "extract:" + format + ":" + flags + ":" + req.URL
}

func extractHostFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}
