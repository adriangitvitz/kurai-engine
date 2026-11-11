package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/adriangitvitz/kurai/internal/apierror"
	"github.com/adriangitvitz/kurai/internal/auth"
	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/searxng"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	resp, err := s.ServiceSearch(r.Context(), ServiceSearchReq{
		UserID:          auth.UserIDFromContext(r.Context()),
		Query:           q.Get("q"),
		Page:            parsePositiveInt(q.Get("page"), 1),
		Category:        q.Get("category"),
		Debug:           q.Get("debug") == "true",
		Language:        q.Get("language"),
		IncludeDomains:  splitCSV(q.Get("include_domains")),
		ExcludeDomains:  splitCSV(q.Get("exclude_domains")),
		DateFrom:        q.Get("date_from"),
		DateTo:          q.Get("date_to"),
		DateStrict:      q.Get("date_strict") == "true",
		SnippetLength:   parsePositiveInt(q.Get("snippet_length"), 0),
		Limit:           parsePositiveInt(q.Get("limit"), 0),
		ExtractTop:      parsePositiveInt(q.Get("extract_top"), 0),
		KeepUnreachable: q.Get("keep_unreachable") == "true",
		ContentType:     validContentType(q.Get("content_type")),
	})
	if err != nil {
		s.writeServiceError(w, err)
		return
	}

	if q.Get("response_format") == "compact" {
		s.writeJSON(w, http.StatusOK, ToCompactSearchResponse(resp))
		return
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func deduplicateResults(results []searxng.Result) []searxng.Result {
	if len(results) == 0 {
		return results
	}

	type slot struct {
		idx     int
		engines map[string]struct{}
	}
	seen := make(map[string]*slot, len(results))
	unique := make([]searxng.Result, 0, len(results))

	for _, r := range results {
		key := urlutil.CanonicalKey(r.URL)
		if existing, ok := seen[key]; ok {
			for _, e := range r.Engines {
				existing.engines[e] = struct{}{}
			}
			if r.Score > unique[existing.idx].Score {
				engineList := unique[existing.idx].Engines
				r.Engines = mergeEngines(engineList, r.Engines)
				unique[existing.idx] = r
			} else {
				unique[existing.idx].Engines = engineSetToSlice(existing.engines)
			}
			continue
		}

		s := &slot{idx: len(unique), engines: make(map[string]struct{}, len(r.Engines))}
		for _, e := range r.Engines {
			s.engines[e] = struct{}{}
		}
		seen[key] = s
		unique = append(unique, r)
	}

	return unique
}

func mergeEngines(a, b []string) []string {
	set := make(map[string]struct{}, len(a)+len(b))
	for _, e := range a {
		set[e] = struct{}{}
	}
	for _, e := range b {
		set[e] = struct{}{}
	}
	return engineSetToSlice(set)
}

func engineSetToSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for e := range set {
		out = append(out, e)
	}
	return out
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:     "ok",
		Uptime:     time.Since(s.startTime).String(),
		Blocklists: s.blocklist.Status(),
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleBlocklistReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, apierror.CodeMethodNotAllowed, "POST method required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	err := s.blocklist.Reload(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, apierror.CodeInternalError, "failed to reload blocklists: "+err.Error())
		return
	}

	resp := ReloadResponse{
		Status:  "ok",
		Updated: s.blocklist.Status(),
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string) {
	apierror.Write(w, status, code, message, "")
}

func (s *Server) writeServiceError(w http.ResponseWriter, err error) {
	var se *ServiceError
	if errors.As(err, &se) {
		s.writeError(w, se.Status, se.Code, se.Message)
		return
	}
	s.writeError(w, http.StatusInternalServerError, apierror.CodeInternalError, err.Error())
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	if v, err := strconv.Atoi(raw); err == nil && v > 0 {
		return v
	}
	return fallback
}

type BlocklistManager interface {
	IsBlocked(domain string) bool
	GetTrancoRank(domain string) int
	Status() blocklist.Status
	Reload(ctx context.Context) error
}
