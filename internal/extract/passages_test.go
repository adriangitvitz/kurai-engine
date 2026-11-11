package extract

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const passagesMD = `Intro paragraph before any heading. Talks about the
document's purpose in general terms.

## Authorization Server Discovery

Servers advertise their authorization server via protected resource metadata.
Clients fetch /.well-known/oauth-protected-resource to discover it.

## Refresh Tokens

Authorization servers MAY issue refresh tokens. Clients MUST store them
securely and rotate on use. Refresh token rotation prevents replay.

### Token Expiry

Access tokens SHOULD be short-lived. One hour is a common default.

## Error Handling

Servers return 401 with WWW-Authenticate on invalid tokens.
`

func TestSplitPassages_SplitsByHeading(t *testing.T) {
	got := SplitPassages(passagesMD)
	require.Len(t, got, 5, "preamble + 4 headed sections")

	assert.Equal(t, "", got[0].Heading, "preamble has no heading")
	assert.Contains(t, got[0].Text, "Intro paragraph")

	assert.Equal(t, "Authorization Server Discovery", got[1].Heading)
	assert.Contains(t, got[1].Text, "well-known")

	assert.Equal(t, "Refresh Tokens", got[2].Heading)
	assert.Contains(t, got[2].Text, "rotation prevents replay")

	assert.Equal(t, "Token Expiry", got[3].Heading)
	assert.Equal(t, "Error Handling", got[4].Heading)
}

func TestSplitPassages_NoHeadings(t *testing.T) {
	got := SplitPassages("Just one block of text.\n\nWith two paragraphs.")
	require.Len(t, got, 1)
	assert.Equal(t, "", got[0].Heading)
	assert.Contains(t, got[0].Text, "two paragraphs")
}

func TestSplitPassages_EmptyAndWhitespace(t *testing.T) {
	assert.Empty(t, SplitPassages(""))
	assert.Empty(t, SplitPassages("   \n\n  "))
	got := SplitPassages("## Lonely Heading")
	require.Len(t, got, 1)
	assert.Equal(t, "Lonely Heading", got[0].Heading)
}

func TestSplitPassages_CodeFenceHashIsNotHeading(t *testing.T) {
	md := "## Real Heading\n\nText.\n\n```bash\n# this is a comment, not a heading\necho hi\n```\n\nMore text."
	got := SplitPassages(md)
	require.Len(t, got, 1, "the # inside the code fence must not split the passage")
	assert.Contains(t, got[0].Text, "echo hi")
	assert.Contains(t, got[0].Text, "More text")
}

func TestSplitPassages_WindowsLongHeadinglessText(t *testing.T) {
	para := strings.Repeat("Sentence about attention mechanisms. ", 20)
	doc := strings.Join([]string{para, para, para, para, para}, "\n\n")

	got := SplitPassages(doc)
	require.Greater(t, len(got), 1, "a long heading-less document must be addressable")
	for i, p := range got {
		assert.LessOrEqual(t, len(p.Text), maxPassageChars+len(para),
			"passage %d overflows the window budget", i)
		assert.Empty(t, p.Heading)
	}
}

func TestSplitPassages_WindowsKeepTheirHeading(t *testing.T) {
	para := strings.Repeat("Body text that goes on. ", 40)
	doc := "## Real Heading\n\n" + strings.Join([]string{para, para, para}, "\n\n")

	got := SplitPassages(doc)
	require.Greater(t, len(got), 1)
	for _, p := range got {
		assert.Equal(t, "Real Heading", p.Heading,
			"a windowed chunk must still say which section it came from")
	}
}

func TestSplitPassages_ShortTextStaysOnePassage(t *testing.T) {
	got := SplitPassages("Just one block of text.\n\nWith two paragraphs.")
	require.Len(t, got, 1)
}

func TestSplitPassages_OversizedSingleBlockIsNotCutMidSentence(t *testing.T) {
	huge := strings.Repeat("word ", 1000)
	got := SplitPassages(huge)
	require.Len(t, got, 1)
	assert.Equal(t, strings.TrimSpace(huge), got[0].Text)
}
