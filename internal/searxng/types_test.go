package searxng

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const p1LegacyAnswer = `[{
	"url": "https://duckduckgo.com",
	"engine": "currency",
	"template": "answer/legacy.html",
	"answer": "300.0 EUR = 5983.53404958 MXN (1 euro : 19.9451134986 peso)"
}]`

const w1TypedWeather = `[{
	"engine": "duckduckgo weather",
	"template": "answer/weather.html",
	"current": {"temperature": {"val": 24.1, "unit": "°C"}},
	"forecasts": [{"time": "2026-07-17T15:00:00"}]
}]`

func TestLegacyAnswers_FlatStringPreserved(t *testing.T) {
	var resp Response
	require.NoError(t, json.Unmarshal([]byte(`{"query":"x","results":[],"answers":`+p1LegacyAnswer+`}`), &resp))

	got := resp.LegacyAnswers()
	require.Len(t, got, 1)
	assert.Equal(t, "300.0 EUR = 5983.53404958 MXN (1 euro : 19.9451134986 peso)", got[0].Answer)
	assert.Equal(t, "currency", got[0].Engine)
	assert.Equal(t, "https://duckduckgo.com", got[0].URL)
}

func TestLegacyAnswers_TypedWeatherNeverCorrupts(t *testing.T) {
	var resp Response
	require.NoError(t, json.Unmarshal([]byte(`{"query":"x","results":[],"answers":`+w1TypedWeather+`}`), &resp))

	assert.Empty(t, resp.LegacyAnswers())
}

func TestLegacyAnswers_UnknownAndMalformedShapesDroppedSilently(t *testing.T) {
	mixed := `[
		{"template": "answer/translations.html", "translations": ["hola"]},
		"not even an object",
		{"answer": "kept", "engine": "plugin: hash_plugin"},
		12345
	]`
	var resp Response
	require.NoError(t, json.Unmarshal([]byte(`{"query":"x","results":[],"answers":`+mixed+`}`), &resp))

	got := resp.LegacyAnswers()
	require.Len(t, got, 1, "only the well-formed legacy entry survives")
	assert.Equal(t, "kept", got[0].Answer)
}

func TestLegacyAnswers_MalformedArrayIsNilNotError(t *testing.T) {
	resp := Response{Answers: json.RawMessage(`"the whole field is a string"`)}
	assert.Nil(t, resp.LegacyAnswers())

	empty := Response{}
	assert.Nil(t, empty.LegacyAnswers())
}

func TestInfobox_ParsesEngineAndImgSrc(t *testing.T) {
	raw := `{
		"infobox": "Albert Einstein",
		"id": "https://en.wikipedia.org/wiki/Albert_Einstein",
		"content": "German-born theoretical physicist",
		"img_src": "https://upload.wikimedia.org/einstein.jpg",
		"engine": "wikipedia",
		"urls": [{"title": "Wikipedia", "url": "https://en.wikipedia.org/wiki/Albert_Einstein"}]
	}`
	var ib Infobox
	require.NoError(t, json.Unmarshal([]byte(raw), &ib))
	assert.Equal(t, "wikipedia", ib.Engine)
	assert.Equal(t, "https://upload.wikimedia.org/einstein.jpg", ib.ImgSrc)
}
