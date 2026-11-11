package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

type TEIClient struct {
	URL        string
	HTTPClient *http.Client
	Timeout    time.Duration
}

func NewTEIClient(url string, timeout time.Duration) *TEIClient {
	if timeout <= 0 {
		timeout = 200 * time.Millisecond
	}
	return &TEIClient{
		URL:        url,
		HTTPClient: &http.Client{Timeout: timeout * 2},
		Timeout:    timeout,
	}
}

type teiRequest struct {
	Query     string   `json:"query"`
	Texts     []string `json:"texts"`
	RawScores bool     `json:"raw_scores"`
}

type teiResultItem struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
}

func (c *TEIClient) Rerank(ctx context.Context, query string, docs []Doc) ([]Score, error) {
	if len(docs) == 0 {
		return nil, nil
	}
	texts := make([]string, len(docs))
	for i, d := range docs {
		texts[i] = d.Text
	}

	body, err := json.Marshal(teiRequest{Query: query, Texts: texts, RawScores: true})
	if err != nil {
		return nil, fmt.Errorf("tei: marshal request: %w", err)
	}

	if _, ok := ctx.Deadline(); !ok && c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL+"/rerank", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("tei: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tei: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("tei: status %d: %s", resp.StatusCode, raw)
	}

	var items []teiResultItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("tei: decode response: %w", err)
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Index < items[j].Index })

	out := make([]Score, len(docs))
	for i := range docs {
		out[i] = Score{ID: docs[i].ID, Score: 0}
	}
	for _, it := range items {
		if it.Index < 0 || it.Index >= len(out) {
			continue
		}
		out[it.Index].Score = it.Score
	}
	return out, nil
}
