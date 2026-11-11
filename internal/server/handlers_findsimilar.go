package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/adriangitvitz/kurai/internal/apierror"
	"github.com/adriangitvitz/kurai/internal/auth"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

type ServiceFindSimilarReq struct {
	UserID string
	URL    string
	Limit  int
}

type FindSimilarResponse struct {
	SourceURL    string          `json:"source_url"`
	SourceTitle  string          `json:"source_title,omitempty"`
	DerivedQuery string          `json:"derived_query"`
	Search       *SearchResponse `json:"search"`
}

func (s *Server) ServiceFindSimilar(ctx context.Context, req ServiceFindSimilarReq) (*FindSimilarResponse, error) {
	if req.URL == "" {
		return nil, newServiceError(http.StatusBadRequest, apierror.CodeInvalidURL, "url is required")
	}

	extracted, err := s.ServiceExtract(ctx, ServiceExtractReq{
		UserID: req.UserID,
		URL:    req.URL,
		Format: "text",
	})
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(extracted.Result.Title)
	if title == "" {
		return nil, newServiceError(http.StatusUnprocessableEntity, apierror.CodeExtractionFailed,
			"could not derive a query: source article has no title")
	}

	query := deriveQueryFromTitle(title)
	if query == "" {
		return nil, newServiceError(http.StatusUnprocessableEntity, apierror.CodeExtractionFailed,
			"could not derive a query: title is all stop-words")
	}

	host := urlutil.ExtractDomain(extracted.Result.FinalURL)
	if host == "" {
		host = urlutil.ExtractDomain(req.URL)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	searchResp, err := s.ServiceSearch(ctx, ServiceSearchReq{
		UserID:         req.UserID,
		Query:          query,
		Page:           1,
		Limit:          limit,
		ExcludeDomains: []string{host},
	})
	if err != nil {
		return nil, err
	}

	return &FindSimilarResponse{
		SourceURL:    req.URL,
		SourceTitle:  title,
		DerivedQuery: query,
		Search:       searchResp,
	}, nil
}

func (s *Server) handleFindSimilar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	resp, err := s.ServiceFindSimilar(r.Context(), ServiceFindSimilarReq{
		UserID: auth.UserIDFromContext(r.Context()),
		URL:    q.Get("url"),
		Limit:  parsePositiveInt(q.Get("limit"), 0),
	})
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func deriveQueryFromTitle(title string) string {
	tokens := strings.Fields(strings.ToLower(strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == ' ', r == '\t':
			return r
		default:
			return ' '
		}
	}, title)))

	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if len(t) < 2 || titleStopWords[t] {
			continue
		}
		out = append(out, t)
	}
	if len(out) > 6 {
		out = out[:6]
	}
	return strings.Join(out, " ")
}

var titleStopWords = map[string]bool{
	"the": true, "a": true, "an": true, "of": true, "for": true,
	"to": true, "in": true, "on": true, "at": true, "by": true,
	"with": true, "from": true, "and": true, "or": true, "but": true,
}
