package searxng

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/privacy"
)

func isInternalURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := u.Hostname()

	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return true
		}
		return false
	}

	if !strings.Contains(host, ".") {
		return true
	}

	internalSuffixes := []string{".local", ".internal", ".docker", ".localhost"}
	for _, suffix := range internalSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}

	return false
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg *config.SearXNGConfig, privacyCfg *config.PrivacyConfig) *Client {
	var httpClient *http.Client

	useDoH := privacyCfg != nil && privacyCfg.DoH.Enabled && !isInternalURL(cfg.URL)

	if useDoH {
		dohClient, err := privacy.NewDoHClient(privacyCfg.DoH.ResolverURL)
		if err == nil {
			httpClient = dohClient
			httpClient.Timeout = cfg.Timeout
		}
	}

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	}

	return &Client{
		baseURL:    strings.TrimSuffix(cfg.URL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) Search(ctx context.Context, query string, opts *SearchOptions) (*Response, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	searchURL, err := c.buildSearchURL(query, opts)
	if err != nil {
		return nil, fmt.Errorf("building search URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	privacy.AnonymizeRequest(req)

	if _, hasDeadline := ctx.Deadline(); !hasDeadline && c.httpClient.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.httpClient.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var searchResp Response
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &searchResp, nil
}

func (c *Client) buildSearchURL(query string, opts *SearchOptions) (string, error) {
	u, err := url.Parse(c.baseURL + "/search")
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")

	if opts != nil {
		if opts.Page > 0 {
			params.Set("pageno", strconv.Itoa(opts.Page))
		}
		if opts.Language != "" {
			params.Set("language", opts.Language)
		}
		if opts.SafeSearch > 0 {
			params.Set("safesearch", strconv.Itoa(opts.SafeSearch))
		}
		if opts.TimeRange != "" {
			params.Set("time_range", opts.TimeRange)
		}
		if len(opts.Categories) > 0 {
			params.Set("categories", strings.Join(opts.Categories, ","))
		}
		if len(opts.Engines) > 0 {
			params.Set("engines", strings.Join(opts.Engines, ","))
		}
	}

	u.RawQuery = params.Encode()
	return u.String(), nil
}

func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}
