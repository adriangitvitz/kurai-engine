package tavily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/privacy"
)

const (
	quotaCooldown = 15 * time.Minute
	usageTTL      = 5 * time.Minute
)

type Client struct {
	baseURL     string
	apiKey      string
	searchDepth string
	maxResults  int
	httpClient  *http.Client

	mu             sync.Mutex
	unavailableTil time.Time
	usageCheckedAt time.Time
	usageOK        bool
}

type Result struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	Content       string  `json:"content"`
	Score         float64 `json:"score"`
	PublishedDate string  `json:"published_date"`
}

type Usage struct {
	Key     UsageKey     `json:"key"`
	Account UsageAccount `json:"account"`
}

type UsageKey struct {
	Usage int `json:"usage"`
	Limit int `json:"limit"`
}

type UsageAccount struct {
	PlanUsage  int `json:"plan_usage"`
	PlanLimit  int `json:"plan_limit"`
	PaygoUsage int `json:"paygo_usage"`
	PaygoLimit int `json:"paygo_limit"`
}

type SearchParams struct {
	IncludeDomains []string
	ExcludeDomains []string
	StartDate      string
	EndDate        string
}

type searchRequest struct {
	Query          string   `json:"query"`
	SearchDepth    string   `json:"search_depth,omitempty"`
	MaxResults     int      `json:"max_results,omitempty"`
	IncludeDomains []string `json:"include_domains,omitempty"`
	ExcludeDomains []string `json:"exclude_domains,omitempty"`
	StartDate      string   `json:"start_date,omitempty"`
	EndDate        string   `json:"end_date,omitempty"`
}

type searchResponse struct {
	Results []Result `json:"results"`
}

func NewClient(cfg *config.TavilyConfig, privacyCfg *config.PrivacyConfig) *Client {
	var httpClient *http.Client

	if privacyCfg != nil && privacyCfg.DoH.Enabled {
		if dohClient, err := privacy.NewDoHClient(privacyCfg.DoH.ResolverURL); err == nil {
			httpClient = dohClient
		}
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	httpClient.Timeout = cfg.Timeout

	apiKey := ""
	if cfg.APIKeyEnv != "" {
		apiKey = os.Getenv(cfg.APIKeyEnv)
	}

	baseURL := strings.TrimSuffix(cfg.URL, "/")
	if baseURL == "" {
		baseURL = "https://api.tavily.com"
	}

	depth := cfg.SearchDepth
	if depth == "" {
		depth = "basic"
	}

	maxResults := cfg.MaxResults
	if maxResults <= 0 {
		maxResults = 5
	}

	return &Client{
		baseURL:     baseURL,
		apiKey:      apiKey,
		searchDepth: depth,
		maxResults:  maxResults,
		httpClient:  httpClient,
	}
}

func (c *Client) Available(ctx context.Context) bool {
	if c.apiKey == "" {
		return false
	}

	c.mu.Lock()
	if time.Now().Before(c.unavailableTil) {
		c.mu.Unlock()
		return false
	}
	if time.Since(c.usageCheckedAt) < usageTTL {
		ok := c.usageOK
		c.mu.Unlock()
		return ok
	}
	c.mu.Unlock()

	usage, err := c.GetUsage(ctx)
	if err != nil {
		return false
	}
	ok := usageHasCredits(usage)

	c.mu.Lock()
	c.usageCheckedAt = time.Now()
	c.usageOK = ok
	c.mu.Unlock()

	return ok
}

func (c *Client) GetUsage(ctx context.Context) (*Usage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/usage", nil)
	if err != nil {
		return nil, err
	}
	privacy.AnonymizeRequest(req)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily usage status: %d", resp.StatusCode)
	}

	var usage Usage
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, err
	}
	return &usage, nil
}

func usageHasCredits(u *Usage) bool {
	keyOK := u.Key.Limit <= 0 || u.Key.Usage < u.Key.Limit
	planOK := u.Account.PlanLimit <= 0 || u.Account.PlanUsage < u.Account.PlanLimit
	paygoOK := u.Account.PaygoLimit > 0 && u.Account.PaygoUsage < u.Account.PaygoLimit
	return keyOK && (planOK || paygoOK)
}

func (c *Client) inCooldown() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Now().Before(c.unavailableTil)
}

func (c *Client) markUnavailable() {
	c.mu.Lock()
	c.unavailableTil = time.Now().Add(quotaCooldown)
	c.usageOK = false
	c.mu.Unlock()
}

func (c *Client) Search(ctx context.Context, query string, params SearchParams) ([]Result, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("tavily: no api key")
	}
	if c.inCooldown() {
		return nil, fmt.Errorf("tavily unavailable")
	}

	body, err := json.Marshal(searchRequest{
		Query:          query,
		SearchDepth:    c.searchDepth,
		MaxResults:     c.maxResults,
		IncludeDomains: params.IncludeDomains,
		ExcludeDomains: params.ExcludeDomains,
		StartDate:      params.StartDate,
		EndDate:        params.EndDate,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	privacy.AnonymizeRequest(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests, http.StatusUnauthorized, http.StatusForbidden, http.StatusPaymentRequired:
		c.markUnavailable()
		return nil, fmt.Errorf("tavily quota or auth exhausted: %d", resp.StatusCode)
	default:
		return nil, fmt.Errorf("tavily unexpected status: %d", resp.StatusCode)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	return sr.Results, nil
}
