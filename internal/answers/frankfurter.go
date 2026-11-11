package answers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/adriangitvitz/kurai/internal/apierror"
	"github.com/adriangitvitz/kurai/internal/privacy"
)

type frankfurterClient struct {
	baseURL    string
	httpClient *http.Client
}

func newFrankfurterClient(baseURL string, timeout time.Duration) *frankfurterClient {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &frankfurterClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

type frankfurterRate struct {
	Date  string  `json:"date"`
	Base  string  `json:"base"`
	Quote string  `json:"quote"`
	Rate  float64 `json:"rate"`
}

func (c *frankfurterClient) rate(ctx context.Context, base, quote string) *frankfurterRate {
	url := fmt.Sprintf("%s/v2/rate/%s/%s", c.baseURL, base, quote)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	privacy.AnonymizeRequest(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("answers: currency rate fetch failed", "code", apierror.CodeUpstreamUnavailable, "error", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("answers: currency rate source returned non-200", "code", apierror.CodeUpstreamUnavailable, "status", resp.StatusCode)
		return nil
	}

	var out frankfurterRate
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		slog.Warn("answers: currency rate payload malformed", "code", apierror.CodeUpstreamUnavailable, "error", err)
		return nil
	}
	if out.Rate <= 0 {
		return nil
	}
	if out.Date == "" {
		slog.Warn("answers: dateless currency rate suppressed at forwarding")
		return nil
	}
	return &out
}

func (r *Resolver) resolveCurrencyCard(ctx context.Context, userID string, trig *CurrencyTrigger) *Conversion {
	cacheKey := "fx:" + trig.Base + ":" + trig.Quote

	var rate *frankfurterRate
	if r.cache != nil {
		if items, ok := r.cache.Get(userID, cacheKey); ok && len(items) > 0 {
			if cached, ok := items[0].(frankfurterRate); ok {
				rate = &cached
			}
		}
	}
	if rate == nil {
		if r.fx == nil {
			return nil
		}
		rate = r.fx.rate(ctx, trig.Base, trig.Quote)
		if rate == nil {
			return nil
		}
		if r.cache != nil {
			r.cache.Set(userID, cacheKey, []any{*rate})
		}
	}

	return &Conversion{
		Base:       trig.Base,
		Quote:      trig.Quote,
		Amount:     trig.Amount,
		Converted:  trig.Amount * rate.Rate,
		Rate:       rate.Rate,
		RateDate:   rate.Date,
		Source:     "frankfurter",
		MinorUnits: MinorUnits(trig.Quote),
	}
}
