package extract

import "strings"

const maxPassageChars = 2000

type Passage struct {
	Heading string `json:"heading,omitempty"`
	Text    string `json:"text"`
}

func SplitPassages(md string) []Passage {
	if strings.TrimSpace(md) == "" {
		return nil
	}

	var passages []Passage
	var heading string
	var body []string
	inFence := false

	flush := func() {
		text := strings.TrimSpace(strings.Join(body, "\n"))
		if text != "" || heading != "" {
			passages = append(passages, Passage{Heading: heading, Text: text})
		}
		body = body[:0]
	}

	for _, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			body = append(body, line)
			continue
		}

		if !inFence && isATXHeading(trimmed) {
			flush()
			heading = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			continue
		}

		body = append(body, line)
	}
	flush()

	if len(passages) == 1 && passages[0].Heading == "" && passages[0].Text == "" {
		return nil
	}
	return windowLongPassages(passages)
}

func windowLongPassages(in []Passage) []Passage {
	out := make([]Passage, 0, len(in))
	for _, p := range in {
		if len(p.Text) <= maxPassageChars {
			out = append(out, p)
			continue
		}
		for _, chunk := range windowText(p.Text) {
			out = append(out, Passage{Heading: p.Heading, Text: chunk})
		}
	}
	return out
}

func windowText(text string) []string {
	blocks := strings.Split(text, "\n\n")
	var chunks []string
	var cur strings.Builder

	flush := func() {
		if t := strings.TrimSpace(cur.String()); t != "" {
			chunks = append(chunks, t)
		}
		cur.Reset()
	}
	for _, b := range blocks {
		if strings.TrimSpace(b) == "" {
			continue
		}
		if cur.Len() > 0 && cur.Len()+len(b) > maxPassageChars {
			flush()
		}
		if cur.Len() > 0 {
			cur.WriteString("\n\n")
		}
		cur.WriteString(b)
		if cur.Len() >= maxPassageChars {
			flush()
		}
	}
	flush()
	return chunks
}

func isATXHeading(line string) bool {
	if line == "" || line[0] != '#' {
		return false
	}
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level > 6 {
		return false
	}
	return level < len(line) && line[level] == ' '
}
