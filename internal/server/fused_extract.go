package server

import (
	"context"
	"log/slog"
	"sync"

	"github.com/adriangitvitz/kurai/internal/extract"
	"github.com/adriangitvitz/kurai/internal/ranker"
)

const maxExtractTop = 5

func (s *Server) attachExtractedContent(ctx context.Context, results []ranker.ScoredResult, n int) {
	if s.extractor == nil || n <= 0 || len(results) == 0 {
		return
	}
	if n > maxExtractTop {
		n = maxExtractTop
	}
	if n > len(results) {
		n = len(results)
	}

	if s.fusedExtractBudget > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.fusedExtractBudget)
		defer cancel()
	}

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			article, err := s.extractor.Fetch(ctx, results[idx].URL, extract.FetchOpts{
				Format: extract.FormatMarkdown,
			})
			if err != nil {
				slog.Debug("fused extract failed", "url", results[idx].URL, "error", err)
				return
			}
			results[idx].ExtractedContent = article.ContentMarkdown
		}(i)
	}
	wg.Wait()
}
