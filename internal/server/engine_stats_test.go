package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/ranker"
)

func TestEngineStats_AccumulatesAcrossQueries(t *testing.T) {
	es := newEngineStats()
	es.record(&ranker.RankStats{PerEngine: map[string]ranker.EngineTally{
		"bing":  {Contributed: 10, Irrelevant: 9, Kept: 1},
		"mwmbl": {Contributed: 4, Kept: 4},
	}})
	es.record(&ranker.RankStats{PerEngine: map[string]ranker.EngineTally{
		"bing": {Contributed: 10, Irrelevant: 8, Kept: 2},
	}})

	snap := es.snapshot()
	assert.Equal(t, 2, snap.Queries)
	require.Len(t, snap.Engines, 2)

	assert.Equal(t, "bing", snap.Engines[0].Engine)
	assert.Equal(t, 20, snap.Engines[0].Contributed)
	assert.Equal(t, 17, snap.Engines[0].Irrelevant)
	assert.InDelta(t, 0.85, snap.Engines[0].IrrelevantRate, 0.001)
	assert.InDelta(t, 0.15, snap.Engines[0].KeptRate, 0.001)

	assert.Equal(t, "mwmbl", snap.Engines[1].Engine)
	assert.Zero(t, snap.Engines[1].IrrelevantRate)
	assert.InDelta(t, 1.0, snap.Engines[1].KeptRate, 0.001)
}

func TestEngineStats_SeparatesPolicyFromQuality(t *testing.T) {
	es := newEngineStats()
	es.record(&ranker.RankStats{PerEngine: map[string]ranker.EngineTally{
		"adheavy":  {Contributed: 10, Blocked: 9, Kept: 1},
		"offtopic": {Contributed: 10, Irrelevant: 9, Kept: 1},
	}})

	snap := es.snapshot()
	byName := map[string]EngineStatsRow{}
	for _, r := range snap.Engines {
		byName[r.Engine] = r
	}
	assert.InDelta(t, 0.9, byName["adheavy"].BlockedRate, 0.001)
	assert.Zero(t, byName["adheavy"].IrrelevantRate,
		"an ad-heavy engine must not read as a noisy one")
	assert.InDelta(t, 0.9, byName["offtopic"].IrrelevantRate, 0.001)
	assert.Zero(t, byName["offtopic"].BlockedRate)

	assert.Equal(t, "offtopic", snap.Engines[0].Engine)
	assert.Contains(t, snap.Note, "blocklist is doing its job")
}

func TestEngineStats_ResetStartsANewWindow(t *testing.T) {
	es := newEngineStats()
	es.record(&ranker.RankStats{PerEngine: map[string]ranker.EngineTally{"bing": {Contributed: 3}}})
	before := es.snapshot().Since

	es.reset()
	snap := es.snapshot()
	assert.Zero(t, snap.Queries)
	assert.Empty(t, snap.Engines)
	assert.False(t, snap.Since.Before(before))
}

func TestEngineStats_IgnoresEmptyRecords(t *testing.T) {
	es := newEngineStats()
	es.record(nil)
	es.record(&ranker.RankStats{})
	assert.Zero(t, es.snapshot().Queries)
}

func TestEngineStats_ConcurrentRecords(t *testing.T) {
	es := newEngineStats()
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			es.record(&ranker.RankStats{PerEngine: map[string]ranker.EngineTally{
				"bing": {Contributed: 1, Irrelevant: 1},
			}})
		}()
	}
	wg.Wait()
	snap := es.snapshot()
	assert.Equal(t, 50, snap.Queries)
	assert.Equal(t, 50, snap.Engines[0].Contributed)
}

func TestHandleEngineStats(t *testing.T) {
	s := newExtractorServer(t)
	s.engineStats.record(&ranker.RankStats{PerEngine: map[string]ranker.EngineTally{
		"bing": {Contributed: 5, Irrelevant: 4, Kept: 1},
	}})

	t.Run("GET returns the tally", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleEngineStats(w, httptest.NewRequest(http.MethodGet, "/stats/engines", nil))
		require.Equal(t, http.StatusOK, w.Code)

		var got EngineStatsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.Len(t, got.Engines, 1)
		assert.Equal(t, "bing", got.Engines[0].Engine)
		assert.InDelta(t, 0.8, got.Engines[0].IrrelevantRate, 0.001)
	})

	t.Run("DELETE resets", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleEngineStats(w, httptest.NewRequest(http.MethodDelete, "/stats/engines", nil))
		require.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, s.engineStats.snapshot().Engines)
	})

	t.Run("other methods are rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleEngineStats(w, httptest.NewRequest(http.MethodPost, "/stats/engines", nil))
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		assert.Equal(t, "GET, DELETE", w.Header().Get("Allow"))
	})
}
