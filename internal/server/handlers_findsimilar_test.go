package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveQueryFromTitle(t *testing.T) {
	cases := map[string]string{
		"What Is Ownership? - The Rust Programming Language": "what is ownership rust programming language",
		"How to Fix a TypeError in Python":                   "how fix typeerror python",
		"":                                                   "",
		"the the the":                                        "",
		"OpenAI's antitrust lawsuit hits a snag":             "openai antitrust lawsuit hits snag",
	}
	for in, want := range cases {
		got := deriveQueryFromTitle(in)
		assert.Equal(t, want, got, "title=%q", in)
	}
}
