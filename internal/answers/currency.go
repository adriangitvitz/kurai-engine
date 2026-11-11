package answers

import (
	"regexp"
	"strconv"
	"strings"
)

var currencyTriggerRe = regexp.MustCompile(
	`^\s*(\d+(?:[.,]\d+)?)\s+([\p{L}$€£¥]{2,20})\s+(?:to|in|a|en)\s+([\p{L}$€£¥]{2,20})\s*$`,
)

type CurrencyTrigger struct {
	Amount float64
	Base   string
	Quote  string
}

func ParseCurrencyTrigger(query string) *CurrencyTrigger {
	m := currencyTriggerRe.FindStringSubmatch(strings.ToLower(query))
	if m == nil {
		return nil
	}
	amount, ok := parseAmount(m[1])
	if !ok || amount <= 0 {
		return nil
	}
	base, ok := resolveCurrency(m[2])
	if !ok {
		return nil
	}
	quote, ok := resolveCurrency(m[3])
	if !ok {
		return nil
	}
	return &CurrencyTrigger{Amount: amount, Base: base, Quote: quote}
}

func parseAmount(s string) (float64, bool) {
	if strings.Contains(s, ".") && strings.Contains(s, ",") {
		return 0, false
	}
	normalized := strings.ReplaceAll(s, ",", ".")
	v, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func resolveCurrency(token string) (string, bool) {
	upper := strings.ToUpper(token)
	if _, known := currencyMinorUnits[upper]; known && len(token) == 3 {
		return upper, true
	}
	iso, ok := currencyAliases[token]
	return iso, ok
}

func MinorUnits(iso string) int {
	if n, ok := currencyMinorUnits[iso]; ok {
		return n
	}
	return 2
}

var currencyAliases = map[string]string{
	"dollar": "USD", "dollars": "USD", "dolar": "USD", "dolares": "USD",
	"dólar": "USD", "dólares": "USD",
	"peso": "MXN", "pesos": "MXN",
	"euro": "EUR", "euros": "EUR",
	"pound": "GBP", "pounds": "GBP", "libra": "GBP", "libras": "GBP",
	"sterling": "GBP",
	"yen":      "JPY", "yenes": "JPY",
	"yuan": "CNY", "yuanes": "CNY", "renminbi": "CNY",
	"franc": "CHF", "francs": "CHF", "franco": "CHF", "francos": "CHF",
	"rupee": "INR", "rupees": "INR", "rupia": "INR", "rupias": "INR",
	"real": "BRL", "reales": "BRL", "reais": "BRL",
	"won": "KRW", "wones": "KRW",
	"krona": "SEK", "kronor": "SEK", "corona": "SEK", "coronas": "SEK",
	"krone": "NOK", "kroner": "NOK",
	"zloty": "PLN", "zlotys": "PLN",
	"rand":  "ZAR",
	"ruble": "RUB", "rubles": "RUB", "rublo": "RUB", "rublos": "RUB",
	"lira": "TRY", "liras": "TRY",
	"shekel": "ILS", "shekels": "ILS",
	"dirham": "AED", "dirhams": "AED",
	"baht": "THB", "bahts": "THB",
	"ringgit": "MYR",
	"rupiah":  "IDR",
	"forint":  "HUF", "forints": "HUF",
	"leu": "RON", "lei": "RON",
	"quetzal": "GTQ", "quetzales": "GTQ",
	"sol": "PEN", "soles": "PEN",
	"bolivar": "VES", "bolivares": "VES", "bolívar": "VES", "bolívares": "VES",
	"colon": "CRC", "colones": "CRC", "colón": "CRC",
	"guarani": "PYG", "guaranies": "PYG", "guaraní": "PYG", "guaraníes": "PYG",
	"$": "USD", "€": "EUR", "£": "GBP", "¥": "JPY",
}

var currencyMinorUnits = map[string]int{
	"USD": 2, "EUR": 2, "GBP": 2, "JPY": 0, "CNY": 2, "CHF": 2, "CAD": 2,
	"AUD": 2, "NZD": 2, "MXN": 2, "BRL": 2, "ARS": 2, "CLP": 0, "COP": 2,
	"PEN": 2, "UYU": 2, "PYG": 0, "BOB": 2, "VES": 2, "GTQ": 2, "CRC": 2,
	"DOP": 2, "HNL": 2, "NIO": 2, "PAB": 2, "SVC": 2, "INR": 2, "KRW": 0,
	"SEK": 2, "NOK": 2, "DKK": 2, "PLN": 2, "CZK": 2, "HUF": 2, "RON": 2,
	"BGN": 2, "TRY": 2, "RUB": 2, "UAH": 2, "ILS": 2, "AED": 2, "SAR": 2,
	"THB": 2, "MYR": 2, "IDR": 2, "PHP": 2, "SGD": 2, "HKD": 2, "TWD": 2,
	"ZAR": 2, "EGP": 2, "NGN": 2, "KES": 2, "ISK": 0, "VND": 0,
}
