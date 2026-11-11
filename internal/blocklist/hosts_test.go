package blocklist

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHosts(t *testing.T) {
	store := NewDomainStore(1000)

	testdataPath := filepath.Join("testdata", "sample_hosts.txt")
	data, err := os.ReadFile(testdataPath)
	require.NoError(t, err)

	err = ParseHosts(bytes.NewReader(data), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("ads.example.com"))
	assert.True(t, store.Contains("tracker.net"))
	assert.True(t, store.Contains("spam.io"))
	assert.True(t, store.Contains("doubleclick.net"))
	assert.True(t, store.Contains("adserver.com"))

	assert.False(t, store.Contains("localhost"))
	assert.False(t, store.Contains("localhost.localdomain"))
	assert.False(t, store.Contains("broadcasthost"))

	assert.Equal(t, 5, store.Count())
}

func TestParseHosts_ZeroAddress(t *testing.T) {
	store := NewDomainStore(1000)

	input := `0.0.0.0 ads.com
0.0.0.0 tracker.org
0.0.0.0 spam.net
`
	err := ParseHosts(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("ads.com"))
	assert.True(t, store.Contains("tracker.org"))
	assert.True(t, store.Contains("spam.net"))
	assert.Equal(t, 3, store.Count())
}

func TestParseHosts_LoopbackAddress(t *testing.T) {
	store := NewDomainStore(1000)

	input := `127.0.0.1 blocked1.com
127.0.0.1 blocked2.org
`
	err := ParseHosts(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("blocked1.com"))
	assert.True(t, store.Contains("blocked2.org"))
	assert.Equal(t, 2, store.Count())
}

func TestParseHosts_SkipsComments(t *testing.T) {
	store := NewDomainStore(1000)

	input := `# This is a comment
0.0.0.0 valid.com
# Another comment
# 0.0.0.0 commented-out.com
0.0.0.0 another.com # inline comment
`
	err := ParseHosts(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("valid.com"))
	assert.True(t, store.Contains("another.com"))
	assert.False(t, store.Contains("commented-out.com"))
	assert.Equal(t, 2, store.Count())
}

func TestParseHosts_SkipsLocalhost(t *testing.T) {
	store := NewDomainStore(1000)

	input := `127.0.0.1 localhost
127.0.0.1 localhost.localdomain
255.255.255.255 broadcasthost
::1 localhost
0.0.0.0 0.0.0.0
0.0.0.0 blocked.com
`
	err := ParseHosts(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.False(t, store.Contains("localhost"))
	assert.False(t, store.Contains("localhost.localdomain"))
	assert.False(t, store.Contains("broadcasthost"))
	assert.False(t, store.Contains("0.0.0.0"))
	assert.True(t, store.Contains("blocked.com"))
	assert.Equal(t, 1, store.Count())
}

func TestParseHosts_MultipleDomainsPerLine(t *testing.T) {
	store := NewDomainStore(1000)

	input := `0.0.0.0 domain1.com domain2.com domain3.com
127.0.0.1 site1.org site2.org
`
	err := ParseHosts(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("domain1.com"))
	assert.True(t, store.Contains("domain2.com"))
	assert.True(t, store.Contains("domain3.com"))
	assert.True(t, store.Contains("site1.org"))
	assert.True(t, store.Contains("site2.org"))
	assert.Equal(t, 5, store.Count())
}

func TestParseHosts_EmptyInput(t *testing.T) {
	store := NewDomainStore(1000)

	err := ParseHosts(bytes.NewReader([]byte("")), store)
	require.NoError(t, err)
	assert.Equal(t, 0, store.Count())
}

func TestParseHosts_OnlyComments(t *testing.T) {
	store := NewDomainStore(1000)

	input := `# Comment 1
# Comment 2
# Comment 3
`
	err := ParseHosts(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)
	assert.Equal(t, 0, store.Count())
}

func TestParseHosts_TabSeparated(t *testing.T) {
	store := NewDomainStore(1000)

	input := "0.0.0.0\tads.com\n127.0.0.1\ttracker.net\n"
	err := ParseHosts(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("ads.com"))
	assert.True(t, store.Contains("tracker.net"))
	assert.Equal(t, 2, store.Count())
}

func TestParseHosts_IPv6(t *testing.T) {
	store := NewDomainStore(1000)

	input := `::1 localhost
:: blocked-ipv6.com
0.0.0.0 blocked.com
`
	err := ParseHosts(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.False(t, store.Contains("localhost"))
	assert.True(t, store.Contains("blocked-ipv6.com"))
	assert.True(t, store.Contains("blocked.com"))
}

func TestParseHosts_MalformedLines(t *testing.T) {
	store := NewDomainStore(1000)

	input := `0.0.0.0
only-domain.com
0.0.0.0 valid.com
`
	err := ParseHosts(bytes.NewReader([]byte(input)), store)
	require.NoError(t, err)

	assert.True(t, store.Contains("valid.com"))
	assert.Equal(t, 1, store.Count())
}
