package answers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCurrencyTrigger_Negatives(t *testing.T) {
	for _, q := range []string{
		"how to convert currencies",
		"usd to eur",
		"0 usd to eur",
		"-100 usd to eur",
		"100 usd",
		"100 usd eur",
		"weather in oaxaca",
		"100 zorkmids to eur",
		"convert 100 usd euro",
		"100.000,50 usd to eur",
	} {
		assert.Nil(t, ParseCurrencyTrigger(q), "must not trigger: %q", q)
	}
}

func TestParseCurrencyTrigger_SpanishConnectorAndDecimalComma(t *testing.T) {
	trig := ParseCurrencyTrigger("300 euros a pesos")
	require.NotNil(t, trig)
	assert.Equal(t, "EUR", trig.Base)
	assert.Equal(t, "MXN", trig.Quote)
	assert.Equal(t, 300.0, trig.Amount)

	comma := ParseCurrencyTrigger("100,50 euros a pesos")
	require.NotNil(t, comma)
	assert.Equal(t, 100.50, comma.Amount, "decimal comma parses as the decimal separator — a wrong parse is a wrong number")

	en := ParseCurrencyTrigger("100 usd en eur")
	require.NotNil(t, en, "Spanish connector 'en'")
	assert.Equal(t, "USD", en.Base)
	assert.Equal(t, "EUR", en.Quote)
}

func TestParseCurrencyTrigger_NormativeMapping(t *testing.T) {
	trig := ParseCurrencyTrigger("100 dollars to euros")
	require.NotNil(t, trig)
	assert.Equal(t, "USD", trig.Base)
	assert.Equal(t, "EUR", trig.Quote)

	for name, iso := range map[string]string{
		"dolares": "USD", "dólares": "USD", "pesos": "MXN", "euros": "EUR",
	} {
		got := ParseCurrencyTrigger("50 " + name + " to gbp")
		require.NotNil(t, got, name)
		assert.Equal(t, iso, got.Base, name)
	}

	first := ParseCurrencyTrigger("100 pesos to dollars")
	for range 5 {
		assert.Equal(t, first, ParseCurrencyTrigger("100 pesos to dollars"))
	}
}

func TestParseCurrencyTrigger_LowercaseISOCodes(t *testing.T) {
	trig := ParseCurrencyTrigger("300 eur to mxn")
	require.NotNil(t, trig)
	assert.Equal(t, "EUR", trig.Base)
	assert.Equal(t, "MXN", trig.Quote)
}

func TestParseCurrencyTrigger_UnknownCodeNoTrigger(t *testing.T) {
	assert.Nil(t, ParseCurrencyTrigger("100 usd to xyz"))
	assert.Nil(t, ParseCurrencyTrigger("100 xyz to usd"))
}

func TestMinorUnits(t *testing.T) {
	assert.Equal(t, 2, MinorUnits("MXN"))
	assert.Equal(t, 0, MinorUnits("JPY"))
	assert.Equal(t, 2, MinorUnits("UNKNOWN"), "defaults to 2")
}
