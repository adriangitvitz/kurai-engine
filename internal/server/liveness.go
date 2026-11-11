package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/extract"
	"github.com/adriangitvitz/kurai/internal/privacy"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

const (
	availReachable   = "reachable"
	availUnreachable = "unreachable"
	availUnknown     = "unknown"
)

const maxLivenessCache = 20000

type livenessVerdict struct {
	availability string
	status       int
	at           time.Time
}

type livenessProber struct {
	client       *http.Client
	timeout      time.Duration
	topN         int
	concurrency  int
	cacheTTL     time.Duration
	allowPrivate bool

	mu    sync.Mutex
	cache map[string]livenessVerdict
}

func newLivenessProber(cfg config.ProbeConfig) *livenessProber {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 12
	}
	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("redirect cap exceeded")
			}
			return nil
		},
	}
	return &livenessProber{
		client:       client,
		timeout:      timeout,
		topN:         cfg.TopN,
		concurrency:  concurrency,
		cacheTTL:     ttl,
		allowPrivate: cfg.AllowPrivateNetworks,
		cache:        make(map[string]livenessVerdict),
	}
}

func (p *livenessProber) probeAndFilter(ctx context.Context, results []ranker.ScoredResult, drop bool) []ranker.ScoredResult {
	n := min(p.topN, len(results))
	if n <= 0 {
		return results
	}

	waveCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	sem := make(chan struct{}, p.concurrency)
	var wg sync.WaitGroup
	for i := range n {
		key := urlutil.CanonicalKey(results[i].URL)
		if v, ok := p.cached(key); ok {
			results[i].Availability = v.availability
			results[i].HTTPStatus = v.status
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, k string) {
			defer wg.Done()
			defer func() { <-sem }()
			avail, status := p.probe(waveCtx, results[idx].URL)
			results[idx].Availability = avail
			results[idx].HTTPStatus = status
			p.store(k, avail, status)
		}(i, key)
	}
	wg.Wait()

	if !drop {
		return results
	}
	filtered := make([]ranker.ScoredResult, 0, len(results))
	for _, r := range results {
		if r.Availability == availUnreachable {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func (p *livenessProber) probe(ctx context.Context, rawURL string) (string, int) {
	if !p.allowPrivate && extract.IsInternalURL(rawURL) {
		return availUnknown, 0
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return availUnknown, 0
	}
	privacy.AnonymizeRequest(req)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.1")

	resp, err := p.client.Do(req)
	if err != nil {
		return availUnreachable, 0
	}
	_ = resp.Body.Close()
	return classifyStatus(resp.StatusCode), resp.StatusCode
}

func classifyStatus(code int) string {
	switch {
	case code >= 500:
		return availUnreachable
	case code == http.StatusNotFound, code == http.StatusGone:
		return availUnreachable
	default:
		return availReachable
	}
}

func (p *livenessProber) cached(key string) (livenessVerdict, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.cache[key]
	if !ok {
		return livenessVerdict{}, false
	}
	if time.Since(v.at) > p.cacheTTL {
		delete(p.cache, key)
		return livenessVerdict{}, false
	}
	return v, true
}

func (p *livenessProber) store(key, avail string, status int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.cache) >= maxLivenessCache {
		p.pruneLocked()
	}
	p.cache[key] = livenessVerdict{availability: avail, status: status, at: time.Now()}
}

func (p *livenessProber) pruneLocked() {
	for k, v := range p.cache {
		if time.Since(v.at) > p.cacheTTL {
			delete(p.cache, k)
		}
	}
	for k := range p.cache {
		if len(p.cache) < maxLivenessCache {
			break
		}
		delete(p.cache, k)
	}
}
