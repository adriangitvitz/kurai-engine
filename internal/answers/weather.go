package answers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/adriangitvitz/kurai/internal/apierror"
	"github.com/adriangitvitz/kurai/internal/privacy"
)

var weatherTriggerRe = regexp.MustCompile(`^\s*(?:weather\s+in|clima\s+en|tiempo\s+en)\s+(\S.{0,78}?)\s*$`)

func ParseWeatherTrigger(query string) (place string, ok bool) {
	m := weatherTriggerRe.FindStringSubmatch(strings.ToLower(query))
	if m == nil {
		return "", false
	}
	return m[1], true
}

type openMeteoClient struct {
	geocodingURL string
	forecastURL  string
	httpClient   *http.Client
}

func newOpenMeteoClient(geocodingURL, forecastURL string, timeout time.Duration) *openMeteoClient {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &openMeteoClient{
		geocodingURL: geocodingURL,
		forecastURL:  forecastURL,
		httpClient:   &http.Client{Timeout: timeout},
	}
}

type geoCandidate struct {
	Name        string  `json:"name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	CountryCode string  `json:"country_code"`
	Country     string  `json:"country"`
	Admin1      string  `json:"admin1"`
	Timezone    string  `json:"timezone"`
}

func (c *openMeteoClient) geocode(ctx context.Context, place string) []geoCandidate {
	u := fmt.Sprintf("%s/v1/search?name=%s&count=10", c.geocodingURL, url.QueryEscape(place))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil
	}
	privacy.AnonymizeRequest(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("answers: weather geocode failed", "code", apierror.CodeUpstreamUnavailable, "error", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Warn("answers: weather geocoder returned non-200", "code", apierror.CodeUpstreamUnavailable, "status", resp.StatusCode)
		return nil
	}

	var out struct {
		Results []geoCandidate `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		slog.Warn("answers: weather geocode payload malformed", "code", apierror.CodeUpstreamUnavailable, "error", err)
		return nil
	}
	return out.Results
}

func fold(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, strings.ToLower(strings.TrimSpace(s)))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(s))
	}
	return out
}

func matchesHints(c geoCandidate, hints []string) bool {
	haystack := fold(strings.Join([]string{c.Admin1, c.Country, c.CountryCode}, " "))
	for _, h := range hints {
		if !strings.Contains(haystack, fold(h)) {
			return false
		}
	}
	return true
}

const maxPrefixProbes = 4

func namesHead(c geoCandidate, head string) bool {
	return strings.Contains(fold(c.Name), fold(head))
}

func (r *Resolver) geocodeTypedPlace(ctx context.Context, typed string) (selected, source []geoCandidate) {
	tokens := strings.Fields(typed)
	if len(tokens) == 0 {
		return nil, nil
	}
	probes := min(len(tokens), maxPrefixProbes)
	results := make([][]geoCandidate, probes)

	var wg sync.WaitGroup
	for i := range probes {
		cut := len(tokens) - i
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = r.om.geocode(ctx, strings.Join(tokens[:cut], " "))
		}()
	}
	wg.Wait()

	for i := range probes {
		cut := len(tokens) - i
		head := strings.Join(tokens[:cut], " ")
		candidates := make([]geoCandidate, 0, len(results[i]))
		for _, c := range results[i] {
			if namesHead(c, head) {
				candidates = append(candidates, c)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		hints := tokens[cut:]
		if len(hints) == 0 {
			return candidates, candidates
		}
		filtered := make([]geoCandidate, 0, len(candidates))
		for _, c := range candidates {
			if matchesHints(c, hints) {
				filtered = append(filtered, c)
			}
		}
		return filtered, candidates
	}
	return nil, nil
}

func ResolvePlace(selected, source []geoCandidate) (geoCandidate, string, bool) {
	if len(selected) == 0 {
		return geoCandidate{}, "", false
	}
	top := selected[0]
	if len(source) == 0 {
		source = selected
	}

	sameNameInCountry := 0
	for _, c := range source {
		if strings.EqualFold(c.Name, top.Name) && c.CountryCode == top.CountryCode {
			sameNameInCountry++
		}
	}

	parts := []string{top.Name}
	if sameNameInCountry > 1 && top.Admin1 != "" {
		parts = append(parts, top.Admin1)
	}
	if top.Country != "" {
		parts = append(parts, top.Country)
	}
	return top, strings.Join(parts, ", "), true
}

type forecastResponse struct {
	Current struct {
		Time          string  `json:"time"`
		Temperature2M float64 `json:"temperature_2m"`
		WeatherCode   int     `json:"weather_code"`
	} `json:"current"`
	Daily struct {
		Time             []string  `json:"time"`
		Temperature2MMin []float64 `json:"temperature_2m_min"`
		Temperature2MMax []float64 `json:"temperature_2m_max"`
		WeatherCode      []int     `json:"weather_code"`
	} `json:"daily"`
}

func (c *openMeteoClient) forecast(ctx context.Context, cand geoCandidate) *forecastResponse {
	u := fmt.Sprintf(
		"%s/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,weather_code&daily=temperature_2m_min,temperature_2m_max,weather_code&forecast_days=8&timezone=%s",
		c.forecastURL, cand.Latitude, cand.Longitude, url.QueryEscape(cand.Timezone),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil
	}
	privacy.AnonymizeRequest(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("answers: weather forecast failed", "code", apierror.CodeUpstreamUnavailable, "error", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Warn("answers: weather forecast returned non-200", "code", apierror.CodeUpstreamUnavailable, "status", resp.StatusCode)
		return nil
	}

	var out forecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		slog.Warn("answers: weather forecast payload malformed", "code", apierror.CodeUpstreamUnavailable, "error", err)
		return nil
	}
	return &out
}

type cachedWeather struct {
	Card         Weather
	ProviderTime time.Time
	FetchedAt    time.Time
}

const weatherFreshnessBar = 60 * time.Minute

func SelectWeatherSource(sources []string) string {
	if len(sources) == 0 {
		return ""
	}
	for _, s := range sources {
		if s == "open-meteo" {
			return s
		}
	}
	sorted := append([]string(nil), sources...)
	sort.Strings(sorted)
	return sorted[0]
}

func (r *Resolver) resolveWeatherCard(ctx context.Context, userID, place string) *Weather {
	if r.om == nil {
		return nil
	}
	selected, source := r.geocodeTypedPlace(ctx, place)
	cand, display, ok := ResolvePlace(selected, source)
	if !ok {
		return nil
	}

	cacheKey := fmt.Sprintf("wx:%.2f,%.2f", cand.Latitude, cand.Longitude)
	if r.cache != nil {
		if items, found := r.cache.Get(userID, cacheKey); found && len(items) > 0 {
			if entry, isWx := items[0].(cachedWeather); isWx {
				if r.now().Sub(entry.ProviderTime) <= weatherFreshnessBar {
					card := entry.Card
					return &card
				}
			}
		}
	}

	fc := r.om.forecast(ctx, cand)
	if fc == nil {
		return nil
	}
	providerTime, err := parsePlaceLocal(fc.Current.Time, cand.Timezone)
	if err != nil {
		slog.Warn("answers: weather provider timestamp unparseable, suppressing", "error", err)
		return nil
	}
	if r.now().Sub(providerTime) > weatherFreshnessBar {
		slog.Warn("answers: weather data older than the freshness bar, suppressing")
		return nil
	}

	card := Weather{
		Place:        display,
		CountryCode:  cand.CountryCode,
		Timezone:     cand.Timezone,
		TemperatureC: fc.Current.Temperature2M,
		Condition:    weatherCondition(fc.Current.WeatherCode),
		Source:       SelectWeatherSource([]string{"open-meteo"}),
		AsOf:         providerTime.Format(time.RFC3339),
	}
	for i := range fc.Daily.Time {
		if len(card.Forecast) >= maxForecastPeriods {
			break
		}
		p := ForecastPeriod{Date: fc.Daily.Time[i]}
		if i < len(fc.Daily.Temperature2MMin) {
			p.MinC = fc.Daily.Temperature2MMin[i]
		}
		if i < len(fc.Daily.Temperature2MMax) {
			p.MaxC = fc.Daily.Temperature2MMax[i]
		}
		if i < len(fc.Daily.WeatherCode) {
			p.Condition = weatherCondition(fc.Daily.WeatherCode[i])
		}
		card.Forecast = append(card.Forecast, p)
	}
	boundWeather(&card)

	if r.cache != nil {
		r.cache.Set(userID, cacheKey, []any{cachedWeather{Card: card, ProviderTime: providerTime, FetchedAt: r.now()}})
	}
	return &card
}

func parsePlaceLocal(value, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	return time.ParseInLocation("2006-01-02T15:04", value, loc)
}

func weatherCondition(code int) string {
	switch {
	case code == 0:
		return "Clear sky"
	case code <= 2:
		return "Partly cloudy"
	case code == 3:
		return "Overcast"
	case code == 45 || code == 48:
		return "Fog"
	case code >= 51 && code <= 57:
		return "Drizzle"
	case code >= 61 && code <= 67:
		return "Rain"
	case code >= 71 && code <= 77:
		return "Snow"
	case code >= 80 && code <= 82:
		return "Rain showers"
	case code == 85 || code == 86:
		return "Snow showers"
	case code >= 95:
		return "Thunderstorm"
	default:
		return "Mixed conditions"
	}
}
