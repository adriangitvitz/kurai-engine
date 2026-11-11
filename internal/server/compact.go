package server

import (
	"github.com/adriangitvitz/kurai/internal/answers"
	"github.com/adriangitvitz/kurai/internal/ranker"
)

type CompactResult struct {
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Snippet       string   `json:"snippet"`
	Score         float64  `json:"score"`
	FinalRank     int      `json:"final_rank"`
	RRFRank       int      `json:"rrf_rank"`
	Engines       []string `json:"engines"`
	Domain        string   `json:"domain,omitempty"`
	ContentType   string   `json:"content_type,omitempty"`
	PublishedDate string   `json:"published_date,omitempty"`
	Content       string   `json:"content,omitempty"`
}

type CompactSearchResponse struct {
	Query   string          `json:"query"`
	Results []CompactResult `json:"results"`
	Answers *answers.Block  `json:"answers,omitempty"`
	Meta    SearchMeta      `json:"meta"`
	Envelope
}

func ToCompactSearchResponse(full *SearchResponse) *CompactSearchResponse {
	out := &CompactSearchResponse{
		Query:    full.Query,
		Results:  make([]CompactResult, 0, len(full.Results)),
		Answers:  full.Answers,
		Meta:     full.Meta,
		Envelope: full.Envelope,
	}
	for i := range full.Results {
		out.Results = append(out.Results, toCompactResult(&full.Results[i]))
	}
	return out
}

func toCompactResult(sr *ranker.ScoredResult) CompactResult {
	return CompactResult{
		URL:           sr.URL,
		Title:         sr.Title,
		Snippet:       sr.Snippet,
		Score:         sr.FinalScore,
		FinalRank:     sr.FinalRank,
		RRFRank:       sr.RRFRank,
		Engines:       sr.Engines,
		Domain:        string(sr.PrimaryDomain),
		ContentType:   string(sr.ContentType),
		PublishedDate: sr.PublishedDate,
		Content:       sr.ExtractedContent,
	}
}
