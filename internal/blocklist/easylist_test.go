package blocklist

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEasyList(t *testing.T) {
	store := NewDomainStore(1000)

	testdataPath := filepath.Join("testdata", "sample_easylist.txt")
	data, err := os.ReadFile(testdataPath)
	require.NoError(t, err)

	err = ParseEasyList(bytes.NewReader(data), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("ads.example.com"))
	assert.True(t, store.Contains("tracker.net"))
	assert.True(t, store.Contains("spam.io"))
	assert.True(t, store.Contains("doubleclick.net"))
	assert.True(t, store.Contains("adserver.com"))

	assert.False(t, store.Contains("example.com"))
	assert.False(t, store.Contains("site.com"))
	assert.False(t, store.Contains("good.com"))

	assert.Equal(t, 5, store.Count())
}

func TestParseEasyList_DomainRules(t *testing.T) {
	store := NewDomainStore(1000)

	input := `||simple.com^
||another-domain.net^
||sub.domain.org^
`
	err := ParseEasyList(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("simple.com"))
	assert.True(t, store.Contains("another-domain.net"))
	assert.True(t, store.Contains("sub.domain.org"))
	assert.Equal(t, 3, store.Count())
}

func TestParseEasyList_SkipsComments(t *testing.T) {
	store := NewDomainStore(1000)

	input := `! This is a comment
! Another comment
[Adblock Plus 2.0]
||valid.com^
! Comment after rule
||another.com^
`
	err := ParseEasyList(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("valid.com"))
	assert.True(t, store.Contains("another.com"))
	assert.Equal(t, 2, store.Count())
}

func TestParseEasyList_SkipsPathRules(t *testing.T) {
	store := NewDomainStore(1000)

	input := `||domain.com/path/to/ads^
||site.net/tracking/*
||example.org/analytics.js^
||valid.com^
`
	err := ParseEasyList(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("valid.com"))
	assert.False(t, store.Contains("domain.com"))
	assert.False(t, store.Contains("site.net"))
	assert.False(t, store.Contains("example.org"))
	assert.Equal(t, 1, store.Count())
}

func TestParseEasyList_SkipsWhitelist(t *testing.T) {
	store := NewDomainStore(1000)

	input := `||block.com^
@@||allow.com^
@@||whitelist.net^
||another-block.org^
`
	err := ParseEasyList(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("block.com"))
	assert.True(t, store.Contains("another-block.org"))
	assert.False(t, store.Contains("allow.com"))
	assert.False(t, store.Contains("whitelist.net"))
	assert.Equal(t, 2, store.Count())
}

func TestParseEasyList_SkipsComplexRules(t *testing.T) {
	store := NewDomainStore(1000)

	input := `||simple.com^
||*.wildcard.com^
||domain.com^$third-party
||site.net^|
||valid.org^
/regex-rule/
||another-valid.io^
`
	err := ParseEasyList(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("simple.com"))
	assert.True(t, store.Contains("valid.org"))
	assert.True(t, store.Contains("another-valid.io"))
}

func TestParseEasyList_EmptyInput(t *testing.T) {
	store := NewDomainStore(1000)

	err := ParseEasyList(bytes.NewReader([]byte("")), store)
	require.NoError(t, err)
	assert.Equal(t, 0, store.Count())
}

func TestParseEasyList_OnlyComments(t *testing.T) {
	store := NewDomainStore(1000)

	input := `! Comment 1
! Comment 2
[Adblock Plus 2.0]
! More comments
`
	err := ParseEasyList(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)
	assert.Equal(t, 0, store.Count())
}

func TestParseEasyList_MixedContent(t *testing.T) {
	store := NewDomainStore(1000)

	input := `[Adblock Plus 2.0]
! EasyList

! Advertising
||ad.doubleclick.net^
||ads.facebook.com^

! Tracking
||google-analytics.com^
||tracking.example.com^

! Exception rules (should be skipped)
@@||ads.google.com^

! Cosmetic filters (should be skipped)
##.ad-banner
###advertisement

! Path rules (should be skipped)
||example.com/ads/*
`
	err := ParseEasyList(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("ad.doubleclick.net"))
	assert.True(t, store.Contains("ads.facebook.com"))
	assert.True(t, store.Contains("google-analytics.com"))
	assert.True(t, store.Contains("tracking.example.com"))
	assert.False(t, store.Contains("ads.google.com"))
	assert.Equal(t, 4, store.Count())
}
