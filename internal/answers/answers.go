package answers

type Block struct {
	Currency    *Conversion `json:"currency,omitempty"`
	Weather     *Weather    `json:"weather,omitempty"`
	Infobox     *Infobox    `json:"infobox,omitempty"`
	Corrections []string    `json:"corrections,omitempty"`
}

type Conversion struct {
	Base       string  `json:"base"`
	Quote      string  `json:"quote"`
	Amount     float64 `json:"amount"`
	Converted  float64 `json:"converted"`
	Rate       float64 `json:"rate"`
	RateDate   string  `json:"rate_date"`
	Source     string  `json:"source"`
	MinorUnits int     `json:"minor_units"`
}

type Weather struct {
	Place        string           `json:"place"`
	CountryCode  string           `json:"country_code,omitempty"`
	Timezone     string           `json:"timezone,omitempty"`
	TemperatureC float64          `json:"temperature_c"`
	Condition    string           `json:"condition,omitempty"`
	Forecast     []ForecastPeriod `json:"forecast,omitempty"`
	Source       string           `json:"source"`
	AsOf         string           `json:"as_of"`
}

type ForecastPeriod struct {
	Date      string  `json:"date"`
	MinC      float64 `json:"min_c"`
	MaxC      float64 `json:"max_c"`
	Condition string  `json:"condition,omitempty"`
}

type Infobox struct {
	Title      string      `json:"title"`
	Summary    string      `json:"summary,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
	Links      []Link      `json:"links,omitempty"`
	ImageURL   string      `json:"image_url,omitempty"`
	Source     string      `json:"source"`
}

type Attribute struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (b *Block) IsEmpty() bool {
	return b == nil ||
		(b.Currency == nil && b.Weather == nil && b.Infobox == nil && len(b.Corrections) == 0)
}

func Merge(cards, extras *Block) *Block {
	if cards.IsEmpty() && extras.IsEmpty() {
		return nil
	}
	out := &Block{}
	if cards != nil {
		out.Currency = cards.Currency
		out.Weather = cards.Weather
	}
	if extras != nil {
		out.Infobox = extras.Infobox
		out.Corrections = extras.Corrections
	}
	if out.IsEmpty() {
		return nil
	}
	return out
}
