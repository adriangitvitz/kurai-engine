package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/adriangitvitz/kurai/internal/auth"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

type ImageSearchResponse struct {
	Query   string               `json:"query"`
	Results []ranker.ImageResult `json:"results"`
	Meta    ImageSearchMeta      `json:"meta"`
}

type ImageSearchMeta struct {
	Total            int      `json:"total"`
	Page             int      `json:"page"`
	PageSize         int      `json:"page_size"`
	TotalPages       int      `json:"total_pages"`
	SafeSearch       int      `json:"safesearch"`
	ProcessingTimeMs int64    `json:"processing_time_ms"`
	FromCache        bool     `json:"from_cache"`
	EnginesUsed      []string `json:"engines_used,omitempty"`
}

func (s *Server) handleImages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	safeSearch := 1
	if ss := q.Get("safesearch"); ss != "" {
		if parsed, err := strconv.Atoi(ss); err == nil {
			safeSearch = parsed
		}
	}

	resp, err := s.ServiceImages(r.Context(), ServiceImagesReq{
		UserID:     auth.UserIDFromContext(r.Context()),
		Query:      q.Get("q"),
		Page:       parsePositiveInt(q.Get("page"), 1),
		PageSize:   parsePositiveInt(q.Get("page_size"), 0),
		SafeSearch: safeSearch,
		Size:       q.Get("size"),
	})
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func convertToImageResults(results []searxng.Result) []ranker.ImageResult {
	imageResults := make([]ranker.ImageResult, 0, len(results))

	for _, r := range results {
		imgURL := r.ImgSrc
		if imgURL == "" {
			imgURL = r.URL
		}
		if !ranker.IsSafeImageURL(imgURL) {
			continue
		}

		thumb := r.ThumbnailSrc
		if thumb == "" {
			thumb = r.Thumbnail
		}

		ir := ranker.ImageResult{
			URL:          imgURL,
			ThumbnailURL: thumb,
			Title:        r.Title,
			AltText:      r.Content,
			Source:       r.URL,
			SourceDomain: urlutil.ExtractDomain(r.URL),
			Engines:      r.Engines,
		}

		if r.ImgFormat != "" {
			ir.Format = ranker.ImageFormat(strings.ToLower(r.ImgFormat))
		} else {
			ir.Format = ranker.DetectImageFormat(imgURL)
		}

		if r.Resolution != "" {
			ir.Width, ir.Height = parseResolution(r.Resolution)
		}
		if ir.Width == 0 && ir.Height == 0 {
			ir.Width, ir.Height = parseImageDimensions(r.Content)
		}

		imageResults = append(imageResults, ir)
	}

	return imageResults
}

func parseResolution(s string) (int, int) {
	if s == "" {
		return 0, 0
	}
	norm := strings.ToLower(s)
	norm = strings.ReplaceAll(norm, "×", "x")
	parts := strings.SplitN(norm, "x", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0
	}
	return w, h
}

func parseImageDimensions(content string) (width, height int) {
	patterns := []struct {
		sep string
	}{
		{"x"},
		{"×"},
		{" x "},
	}

	for _, p := range patterns {
		parts := splitOnce(content, p.sep)
		if len(parts) == 2 {
			w, err1 := strconv.Atoi(extractNumber(parts[0]))
			h, err2 := strconv.Atoi(extractNumber(parts[1]))
			if err1 == nil && err2 == nil && w > 0 && h > 0 {
				return w, h
			}
		}
	}
	return 0, 0
}

func splitOnce(s, sep string) []string {
	idx := indexOf(s, sep)
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func extractNumber(s string) string {
	var num []byte
	started := false
	for _, c := range s {
		if c >= '0' && c <= '9' {
			num = append(num, byte(c))
			started = true
		} else if started {
			break
		}
	}
	return string(num)
}
