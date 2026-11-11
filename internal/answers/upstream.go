package answers

import (
	"strings"

	"github.com/adriangitvitz/kurai/internal/searxng"
)

const (
	engineWikidata  = "wikidata"
	engineWikipedia = "wikipedia"
)

func ExtractUpstream(resp *searxng.Response, infoboxEnabled bool) *Block {
	if resp == nil {
		return nil
	}
	b := &Block{
		Corrections: boundCorrections(cloneStrings(resp.Corrections)),
	}
	if infoboxEnabled {
		if selected := selectInfobox(resp.Infoboxes); selected != nil {
			b.Infobox = boundInfobox(convertInfobox(selected))
		}
	}
	if b.IsEmpty() {
		return nil
	}
	return b
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func selectInfobox(boxes []searxng.Infobox) *searxng.Infobox {
	var best *searxng.Infobox
	for i := range boxes {
		ib := &boxes[i]
		if !allowedInfoboxEngine(ib.Engine) {
			continue
		}
		if best == nil || infoboxRicher(ib, best) {
			best = ib
		}
	}
	return best
}

func allowedInfoboxEngine(engine string) bool {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case engineWikidata, engineWikipedia:
		return true
	}
	return false
}

func infoboxRicher(candidate, current *searxng.Infobox) bool {
	cp, bp := enginePreference(candidate.Engine), enginePreference(current.Engine)
	if cp != bp {
		return cp > bp
	}
	return len(candidate.Attributes) > len(current.Attributes)
}

func enginePreference(engine string) int {
	if strings.ToLower(strings.TrimSpace(engine)) == engineWikidata {
		return 2
	}
	return 1
}

func convertInfobox(ib *searxng.Infobox) *Infobox {
	out := &Infobox{
		Title:    ib.Infobox,
		Summary:  normalizeContent(ib.Content),
		ImageURL: ib.ImgSrc,
		Source:   strings.ToLower(strings.TrimSpace(ib.Engine)),
	}
	for _, a := range ib.Attributes {
		out.Attributes = append(out.Attributes, Attribute{Label: a.Label, Value: a.Value})
	}
	for _, u := range ib.URLs {
		out.Links = append(out.Links, Link{Title: u.Title, URL: u.URL})
	}
	return out
}

func normalizeContent(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	default:
		return ""
	}
}
