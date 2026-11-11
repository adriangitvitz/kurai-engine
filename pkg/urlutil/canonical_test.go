package urlutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalize_LowercasesHostStripsWWWAndDefaultPort(t *testing.T) {
	got := Canonicalize("https://WWW.Example.COM:443/Foo")
	assert.Equal(t, "https://example.com/Foo", got)
}

func TestCanonicalize_UpgradesHTTPToHTTPS(t *testing.T) {
	assert.Equal(t, "https://example.com/foo", Canonicalize("http://example.com/foo"))
}

func TestCanonicalize_DropsFragmentAndTrailingSlash(t *testing.T) {
	assert.Equal(t, "https://example.com/foo", Canonicalize("https://example.com/foo/#section-2"))
}

func TestCanonicalize_StripsAMPAndMobilePrefixes(t *testing.T) {
	assert.Equal(t, Canonicalize("https://m.example.com/foo"), Canonicalize("https://example.com/foo"))
	assert.Equal(t, Canonicalize("https://amp.example.com/foo"), Canonicalize("https://example.com/foo"))
}

func TestCanonicalize_StripsTrackingParamsAndSortsRest(t *testing.T) {
	got := Canonicalize("https://example.com/foo?utm_source=x&fbclid=y&page=2&id=42&gclid=z")
	assert.Equal(t, "https://example.com/foo?id=42&page=2", got)
}

func TestCanonicalize_StripsTrackingPrefixes(t *testing.T) {
	got := Canonicalize("https://example.com/x?utm_source=a&utm_brand=b&hsa_acc=c&sc_channel=d&id=42")
	assert.Equal(t, "https://example.com/x?id=42", got)
}

func TestCanonicalize_ParamOrderInvariant(t *testing.T) {
	assert.Equal(t,
		Canonicalize("https://example.com/foo?b=2&a=1"),
		Canonicalize("https://example.com/foo?a=1&b=2"))
}

func TestCanonicalize_CollapsesSlashesAndCleansPath(t *testing.T) {
	got := Canonicalize("https://example.com//foo///bar/../baz/")
	assert.Equal(t, "https://example.com/foo/baz", got)
}

func TestCanonicalize_RootPathKeepsSlash(t *testing.T) {
	got := Canonicalize("https://example.com/")
	assert.Equal(t, "https://example.com/", got)
}

func TestCanonicalize_LeavesUnknownParams(t *testing.T) {
	got := Canonicalize("https://example.com/foo?id=42&category=tech")
	assert.Equal(t, "https://example.com/foo?category=tech&id=42", got)
}

func TestCanonicalize_HandlesGarbageGracefully(t *testing.T) {
	assert.Equal(t, "not a url", Canonicalize("Not a URL"))
}

func TestCanonicalKey_StripsIndexPages(t *testing.T) {
	assert.Equal(t,
		CanonicalKey("https://example.com/foo/index.html"),
		CanonicalKey("https://example.com/foo/"))
	assert.Equal(t,
		CanonicalKey("https://example.com/index.php"),
		CanonicalKey("https://example.com/"))
	assert.Equal(t,
		CanonicalKey("https://example.com/page/default.aspx"),
		CanonicalKey("https://example.com/page/"))
}

func TestCanonicalKey_GroupsArticleVariants(t *testing.T) {
	cases := []string{
		"https://www.example.com/article?utm_source=twitter",
		"http://example.com/article/",
		"https://amp.example.com/article",
		"https://example.com/article?fbclid=xxx&utm_medium=email",
	}
	first := CanonicalKey(cases[0])
	for _, c := range cases[1:] {
		assert.Equal(t, first, CanonicalKey(c), "url=%s", c)
	}
}
