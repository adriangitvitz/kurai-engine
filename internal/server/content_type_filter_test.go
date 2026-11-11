package server

import (
	"testing"

	"github.com/adriangitvitz/kurai/internal/ranker"
)

func TestFilterByContentType(t *testing.T) {
	results := []ranker.ScoredResult{
		{URL: "a", ContentType: ranker.ContentArticle},
		{URL: "b", ContentType: ranker.ContentGuide},
		{URL: "c", ContentType: ranker.ContentVideo},
		{URL: "d", ContentType: ""},
		{URL: "e", ContentType: ranker.ContentGuide},
	}

	if got := filterByContentType(results, ""); len(got) != 5 {
		t.Fatalf(`ct="" must disable the filter; got %d`, len(got))
	}

	guides := filterByContentType(results, "guide")
	if len(guides) != 2 {
		t.Errorf("guide: want 2, got %d", len(guides))
	}

	articles := filterByContentType(results, "article")
	if len(articles) != 2 {
		t.Errorf("article (incl. unclassified): want 2, got %d", len(articles))
	}

	facets := computeFacets(results)
	if facets["article"] != len(articles) {
		t.Errorf("facet count %d != filtered article count %d", facets["article"], len(articles))
	}
	if facets["guide"] != len(guides) {
		t.Errorf("facet count %d != filtered guide count %d", facets["guide"], len(guides))
	}
}
