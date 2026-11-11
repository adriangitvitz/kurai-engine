package answers

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/config"
)

func newGeoServer(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestParseWeatherTrigger_Spanish(t *testing.T) {
	place, ok := ParseWeatherTrigger("clima en oaxaca")
	require.True(t, ok)
	assert.Equal(t, "oaxaca", place)

	place, ok = ParseWeatherTrigger("tiempo en buenos aires")
	require.True(t, ok)
	assert.Equal(t, "buenos aires", place)

	place, ok = ParseWeatherTrigger("weather in springfield")
	require.True(t, ok)
	assert.Equal(t, "springfield", place)
}

func TestParseWeatherTrigger_BareQueryNoTrigger(t *testing.T) {
	for _, q := range []string{"weather", "clima", "tiempo", "weather in", "the weather in oaxaca today"} {
		_, ok := ParseWeatherTrigger(q)
		assert.False(t, ok, "must not trigger: %q", q)
	}

	r := NewResolver(&config.AnswersConfig{
		Enabled: true,
		Weather: config.WeatherAnswerConfig{Enabled: true, GeocodingURL: "http://unreachable.invalid", ForecastURL: "http://unreachable.invalid"},
	}, nil)
	assert.Nil(t, r.ResolveCards(t.Context(), "u", "weather"))
}

func TestResolvePlace_Deterministic(t *testing.T) {
	candidates := []geoCandidate{
		{Name: "Springfield", CountryCode: "US", Country: "United States", Admin1: "Missouri", Timezone: "America/Chicago"},
		{Name: "Springfield", CountryCode: "US", Country: "United States", Admin1: "Illinois", Timezone: "America/Chicago"},
		{Name: "Springfield", CountryCode: "US", Country: "United States", Admin1: "Massachusetts", Timezone: "America/New_York"},
	}

	first, display1, ok := ResolvePlace(candidates, candidates)
	require.True(t, ok)
	second, display2, _ := ResolvePlace(candidates, candidates)
	assert.Equal(t, first, second, "same candidate set → same pick")
	assert.Equal(t, display1, display2)
	assert.Equal(t, "Missouri", first.Admin1, "prominence ranking: results[0] wins")
}

func TestResolvePlace_QualifierRule(t *testing.T) {
	springfield := []geoCandidate{
		{Name: "Springfield", CountryCode: "US", Country: "United States", Admin1: "Missouri"},
		{Name: "Springfield", CountryCode: "US", Country: "United States", Admin1: "Illinois"},
	}
	_, display, ok := ResolvePlace(springfield, springfield)
	require.True(t, ok)
	assert.Equal(t, "Springfield, Missouri, United States", display)

	reykjavik := []geoCandidate{
		{Name: "Reykjavik", CountryCode: "IS", Country: "Iceland", Admin1: "Capital Region"},
	}
	_, display, ok = ResolvePlace(reykjavik, reykjavik)
	require.True(t, ok)
	assert.Equal(t, "Reykjavik, Iceland", display, "unique in country → no region qualifier")

	_, _, ok = ResolvePlace(nil, nil)
	assert.False(t, ok)
}

const zamoraCandidates = `{"results":[
	{"name":"Zamora","latitude":19.98,"longitude":-102.28,"country_code":"MX","country":"Mexico","admin1":"Michoacán","timezone":"America/Mexico_City"},
	{"name":"Zamora","latitude":41.50,"longitude":-5.75,"country_code":"ES","country":"Spain","admin1":"Castille and León","timezone":"Europe/Madrid"},
	{"name":"Zamora","latitude":-4.06,"longitude":-78.95,"country_code":"EC","country":"Ecuador","admin1":"Zamora Chinchipe","timezone":"America/Guayaquil"},
	{"name":"Zamora","latitude":38.79,"longitude":-121.86,"country_code":"US","country":"United States","admin1":"California","timezone":"America/Los_Angeles"},
	{"name":"Zamora","latitude":15.42,"longitude":-86.65,"country_code":"HN","country":"Honduras","admin1":"Colón Department","timezone":"America/Tegucigalpa"},
	{"name":"Zamora","latitude":19.31,"longitude":-96.61,"country_code":"MX","country":"Mexico","admin1":"Veracruz","timezone":"America/Mexico_City"},
	{"name":"Zamora","latitude":27.49,"longitude":-109.93,"country_code":"MX","country":"Mexico","admin1":"Sonora","timezone":"America/Hermosillo"},
	{"name":"Zamora","latitude":19.58,"longitude":-88.05,"country_code":"MX","country":"Mexico","admin1":"Quintana Roo","timezone":"America/Cancun"},
	{"name":"Zamora","latitude":21.99,"longitude":-100.71,"country_code":"MX","country":"Mexico","admin1":"San Luis Potosí","timezone":"America/Mexico_City"}
]}`

func geoServerNameOnly(t *testing.T, bareName, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("name") == bareName {
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func newGeocodeResolver(t *testing.T, geoURL string) *Resolver {
	t.Helper()
	return NewResolver(&config.AnswersConfig{
		Enabled: true,
		Weather: config.WeatherAnswerConfig{Enabled: true, GeocodingURL: geoURL, ForecastURL: "http://127.0.0.1:1", Timeout: 2 * time.Second},
	}, nil)
}

func TestGeocodeTypedPlace_QualifiersNarrowTheCandidates(t *testing.T) {
	r := newGeocodeResolver(t, geoServerNameOnly(t, "zamora", zamoraCandidates))

	selected, source := r.geocodeTypedPlace(t.Context(), "zamora michoacan mexico")
	require.Len(t, selected, 1, "hints must select exactly the Mexican Zamora")
	assert.Equal(t, "MX", selected[0].CountryCode)
	assert.Len(t, source, 9, "the source view keeps every Zamora the geocoder knows")

	_, display, ok := ResolvePlace(selected, source)
	require.True(t, ok)
	assert.Equal(t, "Zamora, Michoacán, Mexico", display)
}

func TestGeocodeTypedPlace_QualifierSelectsNonTopCandidate(t *testing.T) {
	r := newGeocodeResolver(t, geoServerNameOnly(t, "zamora", zamoraCandidates))

	for _, tc := range []struct{ typed, wantCC string }{
		{"zamora spain", "ES"},
		{"zamora ecuador", "EC"},
		{"zamora mexico", "MX"},
		{"zamora es", "ES"},
	} {
		selected, _ := r.geocodeTypedPlace(t.Context(), tc.typed)
		require.NotEmpty(t, selected, tc.typed)
		assert.Equal(t, tc.wantCC, selected[0].CountryCode, "%q must not resolve to the top hit blindly", tc.typed)
	}
}

func TestGeocodeTypedPlace_ImpossibleQualifierYieldsNoCard(t *testing.T) {
	r := newGeocodeResolver(t, geoServerNameOnly(t, "zamora", zamoraCandidates))

	selected, _ := r.geocodeTypedPlace(t.Context(), "zamora france")
	assert.Empty(t, selected)
	selected, _ = r.geocodeTypedPlace(t.Context(), "zzqxjkw")
	assert.Empty(t, selected)
	selected, _ = r.geocodeTypedPlace(t.Context(), "")
	assert.Empty(t, selected)
}

func TestGeocodeTypedPlace_MultiWordPlaceNamePrefersLongestMatch(t *testing.T) {
	body := `{"results":[{"name":"San Luis Potosí","latitude":22.15,"longitude":-100.98,"country_code":"MX","country":"Mexico","admin1":"San Luis Potosí","timezone":"America/Mexico_City"}]}`
	r := newGeocodeResolver(t, geoServerNameOnly(t, "san luis potosi", body))

	selected, _ := r.geocodeTypedPlace(t.Context(), "san luis potosi")
	require.Len(t, selected, 1)
	assert.Equal(t, "San Luis Potosí", selected[0].Name)
}

func TestGeocodeTypedPlace_RejectsFuzzyAliasMatches(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("name") {
		case "ciudad de":
			_, _ = w.Write([]byte(`{"results":[
				{"name":"Los Angeles","country_code":"US","country":"United States","admin1":"California","timezone":"America/Los_Angeles"},
				{"name":"Mérida","country_code":"MX","country":"Mexico","admin1":"Yucatán","timezone":"America/Merida"},
				{"name":"Panama City","country_code":"PA","country":"Panama","admin1":"Provincia de Panamá","timezone":"America/Panama"}
			]}`))
		case "ciudad":
			_, _ = w.Write([]byte(`{"results":[
				{"name":"La Ciudad","country_code":"MX","country":"Mexico","admin1":"Durango","timezone":"America/Mexico_City"},
				{"name":"Santiago","country_code":"CL","country":"Chile","admin1":"Santiago Metropolitan","timezone":"America/Santiago"}
			]}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	r := newGeocodeResolver(t, srv.URL)

	selected, _ := r.geocodeTypedPlace(t.Context(), "ciudad de mexico")
	assert.Empty(t, selected, "an alias match must never satisfy a different typed place")
}

func TestGeocodeTypedPlace_AcceptsSuffixedNameVariants(t *testing.T) {
	body := `{"results":[{"name":"San Luis Potosí City","latitude":22.15,"longitude":-100.98,"country_code":"MX","country":"Mexico","admin1":"San Luis Potosí","timezone":"America/Mexico_City"}]}`
	r := newGeocodeResolver(t, geoServerNameOnly(t, "san luis potosi", body))

	selected, _ := r.geocodeTypedPlace(t.Context(), "san luis potosi")
	require.Len(t, selected, 1)
	assert.Equal(t, "San Luis Potosí City", selected[0].Name)
}

func TestGeocodeTypedPlace_ProbesConcurrently(t *testing.T) {
	const delay = 150 * time.Millisecond
	var mu sync.Mutex
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("name") == "zamora" {
			_, _ = w.Write([]byte(zamoraCandidates))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	r := newGeocodeResolver(t, srv.URL)

	start := time.Now()
	selected, _ := r.geocodeTypedPlace(t.Context(), "zamora michoacan mexico")
	elapsed := time.Since(start)

	require.NotEmpty(t, selected)
	assert.Equal(t, 3, calls, "one probe per prefix")
	assert.Less(t, elapsed, 3*delay,
		"probes must run concurrently: sequential would cost 3×%v and blow the answer budget on cache hits", delay)
}

func TestFold_StripsAccentsForMatching(t *testing.T) {
	assert.Equal(t, "michoacan", fold("Michoacán"))
	assert.Equal(t, "castille and leon", fold("Castille and León"))
	assert.Equal(t, "mexico", fold("México"))
	assert.Equal(t, "sao paulo", fold("São Paulo"))
	assert.Equal(t, "zurich", fold("Zürich"))
}

func TestWeatherCard_ForecastBoundedInPayload(t *testing.T) {
	w := &Weather{Place: "Oaxaca City, Mexico"}
	for i := 0; i < 229; i++ {
		w.Forecast = append(w.Forecast, ForecastPeriod{Date: "2026-07-17"})
	}
	assert.Len(t, boundWeather(w).Forecast, 8)
}

type fakeCache struct{ entries map[string][]any }

func (f *fakeCache) Get(_, key string) ([]any, bool) { v, ok := f.entries[key]; return v, ok }
func (f *fakeCache) Set(_, key string, results []any) {
	if f.entries == nil {
		f.entries = map[string][]any{}
	}
	f.entries[key] = results
}

func TestWeather_StaleDataSuppressedViaClockSeam(t *testing.T) {
	fixed := time.Date(2026, 7, 17, 15, 0, 0, 0, time.UTC)
	staleProvider := fixed.Add(-90 * time.Minute)

	cache := &fakeCache{}
	cache.Set("u", "wx:17.06,-96.72", []any{cachedWeather{
		Card:         Weather{Place: "Oaxaca City, Mexico", TemperatureC: 24.1, Source: "open-meteo", AsOf: staleProvider.Format(time.RFC3339)},
		ProviderTime: staleProvider,
		FetchedAt:    staleProvider,
	}})

	r := NewResolver(&config.AnswersConfig{
		Enabled: true,
		Weather: config.WeatherAnswerConfig{Enabled: true, GeocodingURL: "http://127.0.0.1:1", ForecastURL: "http://127.0.0.1:1", Timeout: 200 * time.Millisecond},
	}, cache)
	r.now = func() time.Time { return fixed }

	got := r.resolveWeatherCard(t.Context(), "u", "oaxaca")
	assert.Nil(t, got, "provider data older than 60 minutes must never render")
}

func TestWeather_FreshCacheEntryServedWithoutRefetch(t *testing.T) {
	fixed := time.Date(2026, 7, 17, 15, 0, 0, 0, time.UTC)
	fresh := fixed.Add(-10 * time.Minute)

	cache := &fakeCache{}
	cache.Set("u", "wx:17.06,-96.72", []any{cachedWeather{
		Card:         Weather{Place: "Oaxaca City, Mexico", TemperatureC: 24.1, Source: "open-meteo", AsOf: fresh.Format(time.RFC3339)},
		ProviderTime: fresh,
		FetchedAt:    fresh,
	}})

	geo := newGeoServer(t, `{"results":[{"name":"Oaxaca City","latitude":17.06,"longitude":-96.72,"country_code":"MX","country":"Mexico","admin1":"Oaxaca","timezone":"America/Mexico_City"}]}`)
	r := NewResolver(&config.AnswersConfig{
		Enabled: true,
		Weather: config.WeatherAnswerConfig{Enabled: true, GeocodingURL: geo, ForecastURL: "http://127.0.0.1:1", Timeout: 500 * time.Millisecond},
	}, cache)
	r.now = func() time.Time { return fixed }

	got := r.resolveWeatherCard(t.Context(), "u", "oaxaca")
	require.NotNil(t, got)
	assert.Equal(t, 24.1, got.TemperatureC)
}

func TestSelectWeatherSource_Deterministic(t *testing.T) {
	two := []string{"duckduckgo weather", "open-meteo"}
	first := SelectWeatherSource(two)
	assert.Equal(t, "open-meteo", first)
	assert.Equal(t, first, SelectWeatherSource(two), "same input → same selection")

	noPreferred := []string{"wttr.in", "duckduckgo weather"}
	assert.Equal(t, "duckduckgo weather", SelectWeatherSource(noPreferred), "lexicographic fallback")
	assert.Equal(t, "", SelectWeatherSource(nil))
}

func TestUtilityCardPrecedence_CurrencyWins(t *testing.T) {
	r := NewResolver(&config.AnswersConfig{
		Enabled:  true,
		Currency: config.CurrencyAnswerConfig{Enabled: true, BaseURL: "http://127.0.0.1:1", Timeout: 100 * time.Millisecond},
		Weather:  config.WeatherAnswerConfig{Enabled: true, GeocodingURL: "http://127.0.0.1:1", ForecastURL: "http://127.0.0.1:1", Timeout: 100 * time.Millisecond},
	}, nil)

	got := r.ResolveCards(t.Context(), "u", "100 usd en eur")
	assert.Nil(t, got, "currency pipeline failed (dead fetcher) and weather was not consulted")

	place, ok := ParseWeatherTrigger("weather in oaxaca")
	require.True(t, ok)
	assert.Equal(t, "oaxaca", place)
	assert.NotNil(t, ParseCurrencyTrigger("100 usd en eur"), "the overlap query is a real currency trigger")
}
