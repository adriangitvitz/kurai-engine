package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const focusMD = `## Installation

Run the installer and follow the prompts. Supports macOS, Linux, Windows.

## Refresh Tokens

Authorization servers MAY issue refresh tokens. Clients MUST rotate refresh
tokens on use; rotation prevents token replay attacks.

## Pricing

The service costs ten dollars per month for the basic tier.
`

func TestScorePassages_RelevantSectionFirst(t *testing.T) {
	got := scorePassages(focusMD, "refresh token rotation", 5)
	require.NotEmpty(t, got)
	assert.Equal(t, "Refresh Tokens", got[0].Heading,
		"the refresh-token section must outrank installation and pricing")
	assert.Greater(t, got[0].Score, 0.0)
	for i := 1; i < len(got); i++ {
		assert.GreaterOrEqual(t, got[i-1].Score, got[i].Score)
	}
}

func TestScorePassages_TopNCap(t *testing.T) {
	got := scorePassages(focusMD, "tokens", 2)
	assert.LessOrEqual(t, len(got), 2)
}

func TestScorePassages_EmptyInputs(t *testing.T) {
	assert.Empty(t, scorePassages("", "query", 5))
	assert.Empty(t, scorePassages(focusMD, "", 5))
}
