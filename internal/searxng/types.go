package searxng

import "encoding/json"

type Response struct {
	Query               string          `json:"query"`
	NumberOfResults     int64           `json:"number_of_results"`
	Results             []Result        `json:"results"`
	Answers             json.RawMessage `json:"answers,omitempty"`
	Corrections         []string        `json:"corrections,omitempty"`
	Infoboxes           []Infobox       `json:"infoboxes,omitempty"`
	Suggestions         []string        `json:"suggestions,omitempty"`
	UnresponsiveEngines [][]string      `json:"unresponsive_engines,omitempty"`
}

func (r *Response) LegacyAnswers() []Answer {
	if len(r.Answers) == 0 {
		return nil
	}
	var raws []json.RawMessage
	if err := json.Unmarshal(r.Answers, &raws); err != nil {
		return nil
	}
	out := make([]Answer, 0, len(raws))
	for _, raw := range raws {
		var a Answer
		if err := json.Unmarshal(raw, &a); err != nil {
			continue
		}
		if a.Answer == "" {
			continue
		}
		out = append(out, a)
	}
	return out
}

type Result struct {
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Engines       []string `json:"engines"`
	Score         float64  `json:"score"`
	Positions     []int    `json:"positions,omitempty"`
	PublishedDate string   `json:"publishedDate,omitempty"`
	ImgSrc        string   `json:"img_src,omitempty"`
	Thumbnail     string   `json:"thumbnail,omitempty"`
	Category      string   `json:"category,omitempty"`
	Template      string   `json:"template,omitempty"`
	Author        string   `json:"author,omitempty"`
	Metadata      string   `json:"metadata,omitempty"`

	ThumbnailSrc string `json:"thumbnail_src,omitempty"`
	Resolution   string `json:"resolution,omitempty"`
	ImgFormat    string `json:"img_format,omitempty"`
	Filesize     string `json:"filesize,omitempty"`
}

type Answer struct {
	URL    string `json:"url,omitempty"`
	Answer string `json:"answer"`
	Engine string `json:"engine,omitempty"`
	Title  string `json:"title,omitempty"`
}

type Infobox struct {
	ID         string             `json:"id,omitempty"`
	Infobox    string             `json:"infobox,omitempty"`
	Content    interface{}        `json:"content,omitempty"`
	URLs       []InfoboxURL       `json:"urls,omitempty"`
	Attributes []InfoboxAttribute `json:"attributes,omitempty"`
	ImgSrc     string             `json:"img_src,omitempty"`
	Engine     string             `json:"engine,omitempty"`
}

type InfoboxURL struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type InfoboxAttribute struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type SearchOptions struct {
	Page       int      `json:"page,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Engines    []string `json:"engines,omitempty"`
	Language   string   `json:"language,omitempty"`
	SafeSearch int      `json:"safesearch,omitempty"`
	TimeRange  string   `json:"time_range,omitempty"`
}
