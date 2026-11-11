package tavily

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/adriangitvitz/kurai/internal/config"
)

func testConfig(t *testing.T, url string) *config.TavilyConfig {
	t.Helper()
	t.Setenv("TAVILY_TEST_KEY", "tvly-test")
	return &config.TavilyConfig{
		URL:        url,
		Enabled:    true,
		APIKeyEnv:  "TAVILY_TEST_KEY",
		MaxResults: 5,
		Timeout:    2 * time.Second,
	}
}

func usageJSON(usage, limit, planUsage, planLimit int) string {
	return `{"key":{"usage":` + strconv.Itoa(usage) + `,"limit":` + strconv.Itoa(limit) +
		`},"account":{"plan_usage":` + strconv.Itoa(planUsage) + `,"plan_limit":` + strconv.Itoa(planLimit) +
		`,"paygo_usage":0,"paygo_limit":0}}`
}

func TestSearchSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tvly-test" {
			t.Errorf("missing bearer auth, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[{"title":"T","url":"https://x.com","content":"c","score":0.9,"published_date":"2025-01-01T00:00:00"}]}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(t, srv.URL), nil)
	results, err := c.Search(context.Background(), "q", SearchParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].URL != "https://x.com" || results[0].Score != 0.9 {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestSearchPassesFilters(t *testing.T) {
	var got searchRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"results":[]}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(t, srv.URL), nil)
	_, err := c.Search(context.Background(), "q", SearchParams{
		IncludeDomains: []string{"go.dev"},
		ExcludeDomains: []string{"spam.com"},
		StartDate:      "2025-01-01",
		EndDate:        "2025-06-30",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.IncludeDomains) != 1 || got.IncludeDomains[0] != "go.dev" {
		t.Errorf("include_domains not forwarded: %+v", got.IncludeDomains)
	}
	if len(got.ExcludeDomains) != 1 || got.ExcludeDomains[0] != "spam.com" {
		t.Errorf("exclude_domains not forwarded: %+v", got.ExcludeDomains)
	}
	if got.StartDate != "2025-01-01" || got.EndDate != "2025-06-30" {
		t.Errorf("date range not forwarded: %q..%q", got.StartDate, got.EndDate)
	}
}

func TestAvailableUsesUsageEndpoint(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usage" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		hits++
		w.Write([]byte(usageJSON(150, 1000, 500, 15000)))
	}))
	defer srv.Close()

	c := NewClient(testConfig(t, srv.URL), nil)
	if !c.Available(context.Background()) {
		t.Fatal("expected available with credits remaining")
	}
	if !c.Available(context.Background()) {
		t.Fatal("expected available on second call")
	}
	if hits != 1 {
		t.Fatalf("expected usage endpoint cached (1 hit), got %d", hits)
	}
}

func TestUnavailableWhenExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(usageJSON(1000, 1000, 15000, 15000)))
	}))
	defer srv.Close()

	c := NewClient(testConfig(t, srv.URL), nil)
	if c.Available(context.Background()) {
		t.Fatal("expected unavailable when key and plan limits reached")
	}
}

func TestQuotaTripsBreaker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/usage":
			w.Write([]byte(usageJSON(0, 1000, 0, 15000)))
		case "/search":
			w.WriteHeader(http.StatusTooManyRequests)
		}
	}))
	defer srv.Close()

	c := NewClient(testConfig(t, srv.URL), nil)
	if !c.Available(context.Background()) {
		t.Fatal("expected available before first search")
	}
	if _, err := c.Search(context.Background(), "q", SearchParams{}); err == nil {
		t.Fatal("expected quota error from 429")
	}
	if c.Available(context.Background()) {
		t.Fatal("expected breaker cooldown to mark client unavailable")
	}
}

func TestUnavailableWithoutKey(t *testing.T) {
	c := NewClient(&config.TavilyConfig{URL: "https://api.tavily.com", APIKeyEnv: "TAVILY_MISSING_KEY"}, nil)
	if c.Available(context.Background()) {
		t.Fatal("expected unavailable when api key env is unset")
	}
}

func TestUsageHasCredits(t *testing.T) {
	cases := []struct {
		name string
		u    Usage
		want bool
	}{
		{"room on plan", Usage{UsageKey{150, 1000}, UsageAccount{500, 15000, 0, 0}}, true},
		{"key capped", Usage{UsageKey{1000, 1000}, UsageAccount{500, 15000, 0, 0}}, false},
		{"plan capped no paygo", Usage{UsageKey{10, 1000}, UsageAccount{15000, 15000, 0, 0}}, false},
		{"plan capped paygo available", Usage{UsageKey{10, 1000}, UsageAccount{15000, 15000, 25, 100}}, true},
		{"unlimited key and plan", Usage{UsageKey{999, 0}, UsageAccount{999, 0, 0, 0}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := usageHasCredits(&tc.u); got != tc.want {
				t.Errorf("usageHasCredits = %v, want %v", got, tc.want)
			}
		})
	}
}
