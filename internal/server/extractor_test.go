package server

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/extract"
	"github.com/adriangitvitz/kurai/internal/ranker"
)

type fakeExtractor struct {
	mu      sync.Mutex
	calls   []string
	article *extract.Article
	err     error
}

func (f *fakeExtractor) Fetch(_ context.Context, rawURL string, _ extract.FetchOpts) (*extract.Article, error) {
	f.mu.Lock()
	f.calls = append(f.calls, rawURL)
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	a := *f.article
	a.URL, a.FinalURL = rawURL, rawURL
	return &a, nil
}

func (f *fakeExtractor) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}

func newExtractorServer(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 0, ReadTimeout: time.Second, WriteTimeout: time.Second,
			PageSize: 10, PrefetchPages: 1, CacheTTL: time.Minute,
		},
		Ranking:    config.RankingConfig{},
		Blocklists: config.BlocklistConfig{},
		Extract:    config.ExtractConfig{Enabled: true, Timeout: time.Second, MaxBytes: 1 << 20},
	}
	bl := blocklist.NewManager(&cfg.Blocklists, nil)
	return New(cfg, nil, ranker.New(&cfg.Ranking, bl), bl)
}

func TestSetExtractor_SwapsTheBackend(t *testing.T) {
	fake := &fakeExtractor{article: &extract.Article{
		Title:           "From the fake",
		ContentMarkdown: "## Body\n\nsubstituted content",
		ContentText:     "substituted content",
		WordCount:       2,
	}}
	s := newExtractorServer(t)
	s.SetExtractor(fake)

	resp, err := s.ServiceExtract(context.Background(), ServiceExtractReq{
		UserID: "u", URL: "https://example.com/doc",
	})
	require.NoError(t, err)
	assert.Equal(t, "From the fake", resp.Result.Title)
	assert.Equal(t, []string{"https://example.com/doc"}, fake.called())
}

func TestSetExtractor_ErrorsPropagate(t *testing.T) {
	s := newExtractorServer(t)
	s.SetExtractor(&fakeExtractor{err: extract.ErrUnsupportedType})

	_, err := s.ServiceExtract(context.Background(), ServiceExtractReq{
		UserID: "u", URL: "https://example.com/file.bin",
	})
	require.Error(t, err)
	var se *ServiceError
	require.True(t, errors.As(err, &se), "got %T: %v", err, err)
	assert.Equal(t, "unsupported_content_type", se.Code)
}

func TestSetExtractor_NilDisablesExtraction(t *testing.T) {
	s := newExtractorServer(t)
	s.SetExtractor(nil)

	_, err := s.ServiceExtract(context.Background(), ServiceExtractReq{
		UserID: "u", URL: "https://example.com/doc",
	})
	require.Error(t, err)
	var se *ServiceError
	require.True(t, errors.As(err, &se))
	assert.Equal(t, "not_found", se.Code)
}

func TestFetcherSatisfiesExtractor(t *testing.T) {
	var _ Extractor = extract.NewFetcher(extract.Config{})
}
