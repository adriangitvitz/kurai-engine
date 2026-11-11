package server

import (
	"context"

	"github.com/adriangitvitz/kurai/internal/extract"
)

type Extractor interface {
	Fetch(ctx context.Context, rawURL string, opts extract.FetchOpts) (*extract.Article, error)
}

func (s *Server) SetExtractor(e Extractor) {
	s.extractor = e
}
