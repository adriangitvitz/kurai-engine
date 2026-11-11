package server

import (
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

const sameSiteTitleMinWords = 4

func siteOf(rawURL string) string {
	host := urlutil.ExtractDomain(rawURL)
	if site, err := publicsuffix.EffectiveTLDPlusOne(host); err == nil {
		return site
	}
	return host
}

func collapseSameSiteTitles(ranked []ranker.ScoredResult) []ranker.ScoredResult {
	seen := make(map[string]struct{}, len(ranked))
	out := make([]ranker.ScoredResult, 0, len(ranked))
	for _, r := range ranked {
		words := strings.Fields(strings.ToLower(r.Title))
		if len(words) >= sameSiteTitleMinWords {
			key := siteOf(r.URL) + "\x00" + strings.Join(words, " ")
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
		}
		r.FinalRank = len(out) + 1
		out = append(out, r)
	}
	return out
}
