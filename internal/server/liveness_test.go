package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/ranker"
)

func testProber() *livenessProber {
	return newLivenessProber(config.ProbeConfig{
		Enabled:              true,
		TopN:                 10,
		Timeout:              2 * time.Second,
		Concurrency:          8,
		CacheTTL:             time.Minute,
		DropUnreachable:      true,
		AllowPrivateNetworks: true,
	})
}

func TestClassifyStatus(t *testing.T) {
	cases := map[int]string{
		200: availReachable,
		206: availReachable,
		301: availReachable,
		401: availReachable,
		403: availReachable,
		405: availReachable,
		429: availReachable,
		404: availUnreachable,
		410: availUnreachable,
		500: availUnreachable,
		503: availUnreachable,
	}
	for code, want := range cases {
		if got := classifyStatus(code); got != want {
			t.Errorf("classifyStatus(%d) = %q, want %q", code, got, want)
		}
	}
}

func TestProbe_StatusVerdicts(t *testing.T) {
	p := testProber()
	ctx := context.Background()

	for _, tc := range []struct {
		name   string
		status int
		want   string
	}{
		{"ok", 200, availReachable},
		{"forbidden", 403, availReachable},
		{"notfound", 404, availUnreachable},
		{"gone", 410, availUnreachable},
		{"server-error", 500, availUnreachable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()
			avail, status := p.probe(ctx, srv.URL)
			if avail != tc.want {
				t.Errorf("avail = %q, want %q", avail, tc.want)
			}
			if status != tc.status {
				t.Errorf("status = %d, want %d", status, tc.status)
			}
		})
	}
}

func TestProbe_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	avail, status := testProber().probe(context.Background(), url)
	if avail != availUnreachable {
		t.Errorf("closed server: avail = %q, want %q", avail, availUnreachable)
	}
	if status != 0 {
		t.Errorf("transport error should yield status 0, got %d", status)
	}
}

func TestProbe_InternalURLIsUnknown(t *testing.T) {
	safe := newLivenessProber(config.ProbeConfig{TopN: 10, Timeout: time.Second})
	for _, u := range []string{"http://localhost:8080/x", "http://kurai:8080/y", "http://10.0.0.1/z"} {
		if avail, _ := safe.probe(context.Background(), u); avail != availUnknown {
			t.Errorf("probe(%q) = %q, want %q", u, avail, availUnknown)
		}
	}
}

func TestProbeAndFilter_DropsDeadKeepsRest(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer ok.Close()
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) }))
	defer dead.Close()

	results := []ranker.ScoredResult{
		{URL: ok.URL + "/a"},
		{URL: dead.URL + "/b"},
		{URL: "http://bad host/with spaces"},
	}

	got := testProber().probeAndFilter(context.Background(), results, true)
	if len(got) != 2 {
		t.Fatalf("expected 2 survivors (reachable + unknown), got %d", len(got))
	}
	for _, r := range got {
		if r.Availability == availUnreachable {
			t.Errorf("unreachable result survived: %+v", r)
		}
	}
}

func TestProbeAndFilter_AnnotateOnly(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }))
	defer dead.Close()

	results := []ranker.ScoredResult{{URL: dead.URL}}
	got := testProber().probeAndFilter(context.Background(), results, false)
	if len(got) != 1 {
		t.Fatalf("annotate-only must keep all results, got %d", len(got))
	}
	if got[0].Availability != availUnreachable {
		t.Errorf("availability = %q, want %q", got[0].Availability, availUnreachable)
	}
}

func TestProbeAndFilter_UsesCache(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	p := testProber()
	r1 := []ranker.ScoredResult{{URL: srv.URL}}
	p.probeAndFilter(context.Background(), r1, true)
	r2 := []ranker.ScoredResult{{URL: srv.URL}}
	p.probeAndFilter(context.Background(), r2, true)

	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("expected 1 network hit (second from cache), got %d", got)
	}
}
