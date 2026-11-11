package answers

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const (
	maxForecastPeriods = 8
	maxSummaryChars    = 500
	maxAttributes      = 8
	maxLinks           = 3
	maxCorrections     = 3
	maxTotalBytes      = 4096
)

const ellipsis = "…"

func truncateAtWordBoundary(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	if limit < 2 {
		return ellipsis
	}
	runes := []rune(s)
	cut := string(runes[:limit-1])
	if idx := strings.LastIndex(cut, " "); idx > 0 {
		cut = cut[:idx]
	}
	return strings.TrimRight(cut, " ") + ellipsis
}

func boundInfobox(ib *Infobox) *Infobox {
	if ib == nil {
		return nil
	}
	ib.Summary = truncateAtWordBoundary(ib.Summary, maxSummaryChars)
	if len(ib.Attributes) > maxAttributes {
		ib.Attributes = ib.Attributes[:maxAttributes]
	}
	if len(ib.Links) > maxLinks {
		ib.Links = ib.Links[:maxLinks]
	}
	return ib
}

func boundWeather(w *Weather) *Weather {
	if w == nil {
		return nil
	}
	if len(w.Forecast) > maxForecastPeriods {
		w.Forecast = w.Forecast[:maxForecastPeriods]
	}
	return w
}

func boundCorrections(cs []string) []string {
	if len(cs) > maxCorrections {
		return cs[:maxCorrections]
	}
	return cs
}

func serializedSize(b *Block) int {
	data, err := json.Marshal(b)
	if err != nil {
		return maxTotalBytes + 1
	}
	return len(data)
}

func Finalize(b *Block) *Block {
	if b.IsEmpty() {
		return nil
	}
	if serializedSize(b) <= maxTotalBytes {
		return b
	}

	if b.Infobox != nil {
		for len(b.Infobox.Attributes) > 0 && serializedSize(b) > maxTotalBytes {
			b.Infobox.Attributes = b.Infobox.Attributes[:len(b.Infobox.Attributes)-1]
		}
	}
	if b.Infobox != nil && len(b.Infobox.Links) > 1 && serializedSize(b) > maxTotalBytes {
		b.Infobox.Links = b.Infobox.Links[:1]
	}
	if b.Infobox != nil {
		for limit := maxSummaryChars / 2; limit >= 16 && serializedSize(b) > maxTotalBytes; limit /= 2 {
			b.Infobox.Summary = truncateAtWordBoundary(b.Infobox.Summary, limit)
		}
	}
	if b.Infobox != nil && serializedSize(b) > maxTotalBytes {
		b.Infobox = nil
	}
	if len(b.Corrections) > 1 && serializedSize(b) > maxTotalBytes {
		b.Corrections = b.Corrections[:1]
	}
	if len(b.Corrections) > 0 && serializedSize(b) > maxTotalBytes {
		b.Corrections = nil
	}
	if serializedSize(b) > maxTotalBytes {
		return nil
	}
	if b.IsEmpty() {
		return nil
	}
	return b
}
