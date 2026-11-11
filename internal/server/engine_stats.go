package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/adriangitvitz/kurai/internal/ranker"
)

type engineStats struct {
	mu      sync.Mutex
	since   time.Time
	queries int
	engines map[string]ranker.EngineTally
}

func newEngineStats() *engineStats {
	return &engineStats{since: time.Now().UTC(), engines: map[string]ranker.EngineTally{}}
}

func (es *engineStats) record(stats *ranker.RankStats) {
	if es == nil || stats == nil || len(stats.PerEngine) == 0 {
		return
	}
	es.mu.Lock()
	defer es.mu.Unlock()
	es.queries++
	for name, t := range stats.PerEngine {
		cur := es.engines[name]
		cur.Contributed += t.Contributed
		cur.Blocked += t.Blocked
		cur.Irrelevant += t.Irrelevant
		cur.Kept += t.Kept
		es.engines[name] = cur
	}
}

type EngineStatsRow struct {
	Engine         string  `json:"engine"`
	Contributed    int     `json:"contributed"`
	Kept           int     `json:"kept"`
	Blocked        int     `json:"blocked"`
	Irrelevant     int     `json:"irrelevant"`
	KeptRate       float64 `json:"kept_rate"`
	BlockedRate    float64 `json:"blocked_rate"`
	IrrelevantRate float64 `json:"irrelevant_rate"`
}

type EngineStatsResponse struct {
	Since   time.Time        `json:"since"`
	Queries int              `json:"queries"`
	Engines []EngineStatsRow `json:"engines"`
	Note    string           `json:"note"`
}

const engineStatsNote = "blocked = kurai's ad/risk blocklist (policy); irrelevant = " +
	"zero query-term overlap (quality). Only a high irrelevant_rate argues for removing an " +
	"engine from searxng/settings.yml - a high blocked_rate means the blocklist is doing its " +
	"job on an engine that may still be useful. Results returned by several engines are " +
	"credited to each, so columns overlap across rows."

func (es *engineStats) snapshot() EngineStatsResponse {
	es.mu.Lock()
	defer es.mu.Unlock()

	rows := make([]EngineStatsRow, 0, len(es.engines))
	for name, t := range es.engines {
		r := EngineStatsRow{
			Engine: name, Contributed: t.Contributed,
			Kept: t.Kept, Blocked: t.Blocked, Irrelevant: t.Irrelevant,
		}
		if t.Contributed > 0 {
			d := float64(t.Contributed)
			r.KeptRate = round3(float64(t.Kept) / d)
			r.BlockedRate = round3(float64(t.Blocked) / d)
			r.IrrelevantRate = round3(float64(t.Irrelevant) / d)
		}
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].IrrelevantRate != rows[j].IrrelevantRate {
			return rows[i].IrrelevantRate > rows[j].IrrelevantRate
		}
		return rows[i].Contributed > rows[j].Contributed
	})
	return EngineStatsResponse{
		Since: es.since, Queries: es.queries, Engines: rows, Note: engineStatsNote,
	}
}

func (es *engineStats) reset() {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.since = time.Now().UTC()
	es.queries = 0
	es.engines = map[string]ranker.EngineTally{}
}

func round3(f float64) float64 {
	return float64(int(f*1000+0.5)) / 1000
}

func (s *Server) handleEngineStats(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.engineStats.snapshot())
	case http.MethodDelete:
		s.engineStats.reset()
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
