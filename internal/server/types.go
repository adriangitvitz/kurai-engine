package server

import (
	"time"

	"github.com/adriangitvitz/kurai/internal/answers"
	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/ranker"
)

type Envelope struct {
	RequestID  string      `json:"request_id,omitempty"`
	CachedAt   *time.Time  `json:"cached_at,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	Total    int  `json:"total"`
	HasMore  bool `json:"has_more"`
	NextPage *int `json:"next_page,omitempty"`
}

func buildPagination(total, page, pageSize int) *Pagination {
	if total == 0 || pageSize <= 0 {
		return nil
	}
	hasMore := page*pageSize < total
	p := &Pagination{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		HasMore:  hasMore,
	}
	if hasMore {
		next := page + 1
		p.NextPage = &next
	}
	return p
}

type SearchOptions struct {
	Page     int
	Debug    bool
	Language string
}

type SearchResponse struct {
	Query    string                `json:"query"`
	Results  []ranker.ScoredResult `json:"results"`
	Answers  *answers.Block        `json:"answers,omitempty"`
	Meta     SearchMeta            `json:"meta"`
	Intent   *IntentOutput         `json:"intent,omitempty"`
	Envelope Envelope              `json:"envelope,omitempty"`
}

type IntentOutput struct {
	PrimaryDomain string             `json:"primary_domain"`
	Category      string             `json:"category"`
	Confidence    float64            `json:"confidence"`
	Distribution  map[string]float64 `json:"distribution,omitempty"`
	Modifiers     *IntentModifiers   `json:"modifiers,omitempty"`
}

type IntentModifiers struct {
	IsQuestion   bool `json:"is_question,omitempty"`
	IsTutorial   bool `json:"is_tutorial,omitempty"`
	IsRecent     bool `json:"is_recent,omitempty"`
	IsComparison bool `json:"is_comparison,omitempty"`
}

type SearchMeta struct {
	Total              int            `json:"total"`
	Page               int            `json:"page"`
	PageSize           int            `json:"page_size"`
	TotalPages         int            `json:"total_pages"`
	BlocklistFiltered  int            `json:"blocklist_filtered"`
	IrrelevantFiltered int            `json:"irrelevant_filtered"`
	ProcessingTimeMs   int64          `json:"processing_time_ms"`
	FromCache          bool           `json:"from_cache"`
	Engines            []string       `json:"engines_used,omitempty"`
	Facets             map[string]int `json:"facets,omitempty"`
	OffTopicFiltered   int            `json:"off_topic_filtered,omitempty"`
	UndatedFiltered    int            `json:"undated_filtered,omitempty"`
	PhraseFiltered     int            `json:"phrase_filtered,omitempty"`
	PhraseRelaxed      bool           `json:"phrase_relaxed,omitempty"`
	TimeRange          string         `json:"time_range,omitempty"`
}

type HealthResponse struct {
	Status     string           `json:"status"`
	Uptime     string           `json:"uptime"`
	Blocklists blocklist.Status `json:"blocklists"`
}

type ReloadResponse struct {
	Status  string           `json:"status"`
	Updated blocklist.Status `json:"updated"`
}
