package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/config"
)

const geoOaxaca = `{"results":[{"name":"Oaxaca City","latitude":17.06,"longitude":-96.72,"country_code":"MX","country":"Mexico","admin1":"Oaxaca","timezone":"America/Mexico_City"}]}`
const geoReykjavik = `{"results":[{"name":"Reykjavik","latitude":64.14,"longitude":-21.94,"country_code":"IS","country":"Iceland","admin1":"Capital Region","timezone":"Atlantic/Reykjavik"}]}`
const geoSpringfield = `{"results":[
	{"name":"Springfield","latitude":37.22,"longitude":-93.30,"country_code":"US","country":"United States","admin1":"Missouri","timezone":"America/Chicago"},
	{"name":"Springfield","latitude":39.80,"longitude":-89.64,"country_code":"US","country":"United States","admin1":"Illinois","timezone":"America/Chicago"},
	{"name":"Springfield","latitude":42.10,"longitude":-72.59,"country_code":"US","country":"United States","admin1":"Massachusetts","timezone":"America/New_York"}
]}`

func freshForecastHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		tz := r.URL.Query().Get("timezone")
		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.UTC
		}
		now := time.Now().In(loc).Format("2006-01-02T15:04")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"current": {"time": %q, "temperature_2m": 24.1, "weather_code": 2},
			"daily": {
				"time": ["2026-07-17","2026-07-18","2026-07-19","2026-07-20","2026-07-21","2026-07-22","2026-07-23","2026-07-24","2026-07-25","2026-07-26"],
				"temperature_2m_min": [15.2,15.0,14.8,15.5,16.0,15.1,14.9,15.3,15.6,15.8],
				"temperature_2m_max": [27.8,28.1,27.5,28.4,29.0,27.9,27.6,28.2,28.5,28.8],
				"weather_code": [2,0,3,61,2,0,2,3,61,2]
			}
		}`, now)
	}
}

type weatherFakes struct {
	geoCalls  []string
	geoBody   string
	fcHandler http.HandlerFunc
	geoUA     string
}

func newWeatherServer(t *testing.T, searxBody string, fakes *weatherFakes) *Server {
	t.Helper()
	geo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fakes.geoCalls = append(fakes.geoCalls, r.URL.String())
		fakes.geoUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fakes.geoBody))
	}))
	t.Cleanup(geo.Close)

	fc := httptest.NewServer(fakes.fcHandler)
	t.Cleanup(fc.Close)

	return newAnswersServerCfg(t, searxBody, func(cfg *config.Config) {
		cfg.Answers.Weather = config.WeatherAnswerConfig{
			Enabled: true, GeocodingURL: geo.URL, ForecastURL: fc.URL, Timeout: 2 * time.Second,
		}
	})
}

func TestWeather_EnglishHappyPath(t *testing.T) {
	fakes := &weatherFakes{geoBody: geoOaxaca, fcHandler: freshForecastHandler(t)}
	s := newWeatherServer(t, searxFixture("", "", "", "", ""), fakes)

	_, m := doSearch(t, s, url.QueryEscape("weather in oaxaca"), "")
	require.Contains(t, m, "answers")
	w := m["answers"].(map[string]any)["weather"].(map[string]any)
	assert.Equal(t, "Oaxaca City, Mexico", w["place"])
	assert.Equal(t, 24.1, w["temperature_c"])
	assert.Equal(t, "Partly cloudy", w["condition"])
	assert.Equal(t, "open-meteo", w["source"])
	assert.Equal(t, "America/Mexico_City", w["timezone"])
	assert.Equal(t, "MX", w["country_code"])
	assert.NotEmpty(t, w["as_of"])
}

func TestWeather_WorldwideCoverage(t *testing.T) {
	fakes := &weatherFakes{geoBody: geoReykjavik, fcHandler: freshForecastHandler(t)}
	s := newWeatherServer(t, searxFixture("", "", "", "", ""), fakes)

	_, m := doSearch(t, s, url.QueryEscape("weather in reykjavik"), "")
	require.Contains(t, m, "answers")
	w := m["answers"].(map[string]any)["weather"].(map[string]any)
	assert.Equal(t, "Reykjavik, Iceland", w["place"], "single in-country candidate → country qualifier only")
}

func TestWeather_UnresolvablePlaceNoCard(t *testing.T) {
	fakes := &weatherFakes{geoBody: `{}`, fcHandler: freshForecastHandler(t)}
	s := newWeatherServer(t, searxFixture("", "", "", "", ""), fakes)

	_, m := doSearch(t, s, url.QueryEscape("weather in zzqxjkw"), "")
	assert.NotContains(t, m, "answers")
	assert.NotContains(t, m, "error")
	assert.Len(t, m["results"], 3)
}

func TestWeather_OutboundCarriesOnlyQueryDerivedLocation(t *testing.T) {
	fakes := &weatherFakes{geoBody: geoOaxaca, fcHandler: freshForecastHandler(t)}
	s := newWeatherServer(t, searxFixture("", "", "", "", ""), fakes)

	req := httptest.NewRequest(http.MethodGet, "/search?q="+url.QueryEscape("weather in oaxaca"), nil)
	req.RemoteAddr = "203.0.113.77:4444"
	req.Header.Set("X-Forwarded-For", "203.0.113.77")
	rec := httptest.NewRecorder()
	s.handleSearch(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.NotEmpty(t, fakes.geoCalls)
	outbound := fakes.geoCalls[0]
	u, err := url.Parse(outbound)
	require.NoError(t, err)
	q := u.Query()
	assert.Equal(t, "oaxaca", q.Get("name"), "place derived from the query text only")
	for param := range q {
		assert.Contains(t, []string{"name", "count"}, param, "no extra parameters on the outbound request")
	}
	assert.NotContains(t, outbound, "203.0.113.77", "client address never leaks")
	assert.NotEmpty(t, fakes.geoUA)
	assert.False(t, strings.HasPrefix(fakes.geoUA, "Go-http-client"),
		"AnonymizeRequest applied — rotating browser UA, not the Go default")
}

func TestWeather_SpringfieldBindingExemplar(t *testing.T) {
	fakes := &weatherFakes{geoBody: geoSpringfield, fcHandler: freshForecastHandler(t)}
	s := newWeatherServer(t, searxFixture("", "", "", "", ""), fakes)

	_, m := doSearch(t, s, url.QueryEscape("weather in springfield"), "")
	require.Contains(t, m, "answers")
	w := m["answers"].(map[string]any)["weather"].(map[string]any)
	assert.Equal(t, "Springfield, Missouri, United States", w["place"],
		"binding exemplar: region qualifier required for intra-country homonyms")
}

func TestWeather_UniquePlaceCountryOnly(t *testing.T) {
	fakes := &weatherFakes{geoBody: geoReykjavik, fcHandler: freshForecastHandler(t)}
	s := newWeatherServer(t, searxFixture("", "", "", "", ""), fakes)

	_, m := doSearch(t, s, url.QueryEscape("weather in reykjavik"), "")
	w := m["answers"].(map[string]any)["weather"].(map[string]any)
	assert.NotContains(t, w["place"], "Capital Region", "no region qualifier for a unique in-country name")
}

func TestWeather_FreshDataCarriesMachineReadableAsOf(t *testing.T) {
	fakes := &weatherFakes{geoBody: geoOaxaca, fcHandler: freshForecastHandler(t)}
	s := newWeatherServer(t, searxFixture("", "", "", "", ""), fakes)

	_, m := doSearch(t, s, url.QueryEscape("weather in oaxaca"), "")
	w := m["answers"].(map[string]any)["weather"].(map[string]any)

	asOf, err := time.Parse(time.RFC3339, w["as_of"].(string))
	require.NoError(t, err, "as_of is machine-readable ISO 8601")
	assert.LessOrEqual(t, time.Since(asOf), 60*time.Minute,
		"provider data within the freshness bar at serve time")

	forecast := w["forecast"].([]any)
	assert.LessOrEqual(t, len(forecast), 8)
}

func TestWeather_SourceFailureDegradesSilently(t *testing.T) {
	fakes := &weatherFakes{geoBody: geoOaxaca, fcHandler: func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}}
	s := newWeatherServer(t, searxFixture("", "", "", "", ""), fakes)

	_, m := doSearch(t, s, url.QueryEscape("weather in oaxaca"), "")
	assert.NotContains(t, m, "answers")
	assert.NotContains(t, m, "error")
	assert.Len(t, m["results"], 3)
}

func TestAnswers_SourceIdentifierAcrossAllClasses(t *testing.T) {
	fx := fakeFrankfurter(t, datedRateHandler(17.4251, "2026-07-17"))
	sCur := newCurrencyServer(t, searxFixture("", "", "", "", ""), fx.URL)
	_, mCur := doSearch(t, sCur, "100+usd+to+mxn", "")
	assert.Equal(t, "frankfurter", mCur["answers"].(map[string]any)["currency"].(map[string]any)["source"])

	fakes := &weatherFakes{geoBody: geoOaxaca, fcHandler: freshForecastHandler(t)}
	sWx := newWeatherServer(t, searxFixture("", "", "", "", ""), fakes)
	_, mWx := doSearch(t, sWx, url.QueryEscape("weather in oaxaca"), "")
	assert.Equal(t, "open-meteo", mWx["answers"].(map[string]any)["weather"].(map[string]any)["source"])

	sIb := newAnswersServer(t, searxFixture("", "", fixtureInfoboxE1, "", ""))
	_, mIb := doSearch(t, sIb, "einstein", "")
	assert.Equal(t, "wikipedia", mIb["answers"].(map[string]any)["infobox"].(map[string]any)["source"])
}
