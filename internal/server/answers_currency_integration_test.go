package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/config"
)

const fixtureC1UpstreamCurrency = `[{
	"url": "https://duckduckgo.com",
	"engine": "currency",
	"template": "answer/legacy.html",
	"answer": "300.0 EUR = 5983.53404958 MXN (1 euro : 19.9451134986 peso)"
}]`

func fakeFrankfurter(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func datedRateHandler(rate float64, date string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"date":%q,"base":"X","quote":"Y","rate":%g}`, date, rate)
	}
}

func newCurrencyServer(t *testing.T, searxBody string, fxURL string) *Server {
	t.Helper()
	return newAnswersServerCfg(t, searxBody, func(cfg *config.Config) {
		cfg.Answers.Currency = config.CurrencyAnswerConfig{Enabled: true, BaseURL: fxURL, Timeout: 2 * time.Second}
	})
}

func TestCurrency_ISOCodeConversion(t *testing.T) {
	fx := fakeFrankfurter(t, datedRateHandler(17.4251, "2026-07-17"))
	s := newCurrencyServer(t, searxFixture("", "", "", "", ""), fx.URL)

	_, m := doSearch(t, s, "100+usd+to+mxn", "")
	require.Contains(t, m, "answers")
	c := m["answers"].(map[string]any)["currency"].(map[string]any)
	assert.Equal(t, "USD", c["base"])
	assert.Equal(t, "MXN", c["quote"])
	assert.Equal(t, 100.0, c["amount"])
	assert.InDelta(t, 1742.51, c["converted"].(float64), 0.001)
	assert.Equal(t, 17.4251, c["rate"])
	assert.Equal(t, "2026-07-17", c["rate_date"])
	assert.Equal(t, "frankfurter", c["source"])
	assert.Equal(t, 2.0, c["minor_units"])
}

func TestCurrency_CompleteEnrichedConversionInCompact(t *testing.T) {
	fx := fakeFrankfurter(t, datedRateHandler(17.4251, "2026-07-17"))
	s := newCurrencyServer(t, searxFixture("", "", "", "", ""), fx.URL)

	_, m := doSearch(t, s, "100+usd+to+mxn", "&response_format=compact")
	require.Contains(t, m, "answers")
	c := m["answers"].(map[string]any)["currency"].(map[string]any)
	for _, k := range []string{"base", "quote", "amount", "converted", "rate", "rate_date", "source"} {
		assert.Contains(t, c, k)
	}
}

func TestCurrency_NameConversionThroughAliasTable(t *testing.T) {
	fx := fakeFrankfurter(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v2/rate/EUR/MXN")
		datedRateHandler(19.9471, "2026-07-17")(w, r)
	})
	s := newCurrencyServer(t, searxFixture("", "", "", "", ""), fx.URL)

	_, m := doSearch(t, s, "300+euros+to+pesos", "")
	require.Contains(t, m, "answers")
	c := m["answers"].(map[string]any)["currency"].(map[string]any)
	assert.Equal(t, "EUR", c["base"])
	assert.Equal(t, "MXN", c["quote"])
}

func TestCurrency_DatelessUpstreamSuppressedAtForwarding(t *testing.T) {
	fx := fakeFrankfurter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	s := newCurrencyServer(t, searxFixture("", fixtureC1UpstreamCurrency, "", "", ""), fx.URL)

	for _, params := range []string{"", "&response_format=compact"} {
		raw, m := doSearch(t, s, "300+euros+to+mxn", params)
		assert.NotContains(t, m, "answers", "params=%q", params)
		assert.NotContains(t, string(raw), "5983.53404958",
			"the raw upstream string must leave no trace in any format (round-1 critical C-1)")
		assert.Len(t, m["results"], 3)
	}
}

func TestCurrency_DatelessRateSuppressedEndToEnd(t *testing.T) {
	fx := fakeFrankfurter(t, datedRateHandler(17.4251, ""))
	s := newCurrencyServer(t, searxFixture("", "", "", "", ""), fx.URL)

	for _, params := range []string{"", "&response_format=compact"} {
		_, m := doSearch(t, s, "100+usd+to+mxn", params)
		assert.NotContains(t, m, "answers", "a dateless conversion exists in NO response format")
	}
}

func TestCurrency_RateSourceUnavailableDegradesSilently(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	s := newCurrencyServer(t, searxFixture("", "", "", "", ""), deadURL)
	_, m := doSearch(t, s, "100+usd+to+mxn", "")
	assert.NotContains(t, m, "answers")
	assert.NotContains(t, m, "error")
	assert.Len(t, m["results"], 3)
}

func TestCurrency_UpstreamDarkIsDesignedBehavior(t *testing.T) {
	fx := fakeFrankfurter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	s := newCurrencyServer(t, searxFixture("", `[]`, "", "", ""), fx.URL)

	_, m := doSearch(t, s, "100+usd+to+eur", "")
	assert.NotContains(t, m, "answers")
	assert.NotContains(t, m, "error")
	assert.Len(t, m["results"], 3)
}

func TestCurrency_SlowSourceDegradesAtDeadline(t *testing.T) {
	fx := fakeFrankfurter(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(4 * time.Second)
		datedRateHandler(17.4251, "2026-07-17")(w, r)
	})
	s := newCurrencyServer(t, searxFixture("", "", "", "", ""), fx.URL)

	start := time.Now()
	raw, m := doSearch(t, s, "100+usd+to+mxn", "")
	elapsed := time.Since(start)

	assert.NotContains(t, m, "answers", "deadline expiry ships without answer data")
	assert.NotContains(t, string(raw), `"error"`)
	assert.Len(t, m["results"], 3)
	assert.Less(t, elapsed, 3*time.Second,
		"added latency attributable to the answer path must be bounded by the budget")
}

func TestCurrency_RateCachedOneHour(t *testing.T) {
	calls := 0
	fx := fakeFrankfurter(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		datedRateHandler(17.4251, "2026-07-17")(w, r)
	})
	s := newCurrencyServer(t, searxFixture("", "", "", "", ""), fx.URL)

	_, first := doSearch(t, s, "100+usd+to+mxn", "")
	require.Contains(t, first, "answers")
	_, second := doSearch(t, s, "250+usd+to+mxn", "")
	require.Contains(t, second, "answers")

	assert.Equal(t, 1, calls, "fx:<base>:<quote> entry collapses repeat fetches")
	c := second["answers"].(map[string]any)["currency"].(map[string]any)
	assert.InDelta(t, 250*17.4251, c["converted"].(float64), 0.01,
		"cached rate recomputes per-request amounts")
}
