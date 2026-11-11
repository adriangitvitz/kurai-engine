package blocklist

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const trancoCSVSample = "1,google.com\n2,facebook.com\n3,youtube.com\n4,twitter.com\n5,instagram.com\n"

func makeTrancoZip(t *testing.T, csv string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("top-1m.csv")
	require.NoError(t, err)
	_, err = f.Write([]byte(csv))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestParseAuto_PlainCSV(t *testing.T) {
	store := NewTrancoStore()
	require.NoError(t, store.ParseAuto(strings.NewReader(trancoCSVSample)))
	assert.Equal(t, 5, store.Count())
	assert.Equal(t, 1, store.GetRank("google.com"))
	assert.Equal(t, 5, store.GetRank("instagram.com"))
}

func TestParseAuto_ZipContainingCSV(t *testing.T) {
	store := NewTrancoStore()
	zipBytes := makeTrancoZip(t, trancoCSVSample)
	require.Equal(t, byte('P'), zipBytes[0])
	require.Equal(t, byte('K'), zipBytes[1])

	require.NoError(t, store.ParseAuto(bytes.NewReader(zipBytes)))
	assert.Equal(t, 5, store.Count(),
		"ZIP-wrapped Tranco file must extract and load all entries")
	assert.Equal(t, 1, store.GetRank("google.com"))
}

func TestParseAuto_EmptyInput(t *testing.T) {
	store := NewTrancoStore()
	assert.NoError(t, store.ParseAuto(strings.NewReader("")))
	assert.Equal(t, 0, store.Count())
}

func TestParseAuto_GarbageInput(t *testing.T) {
	store := NewTrancoStore()
	junk := []byte{0xFF, 0xFE, 0x00, 0x00, 0xDE, 0xAD, 0xBE, 0xEF}
	err := store.ParseAuto(bytes.NewReader(junk))
	if err == nil {
		assert.Equal(t, 0, store.Count(), "garbage input must not populate the store")
	}
}

func TestParseAuto_RealCachedFile(t *testing.T) {
	store := NewTrancoStore()
	const path = "../../blocklists/cache/tranco.csv"

	f, _ := openIfExists(t, path)
	if f == nil {
		t.Skipf("real cache file %s not present; skipping integration check", path)
	}
	defer f.Close()

	require.NoError(t, store.ParseAuto(f))
	t.Logf("loaded %d entries from %s", store.Count(), path)
	assert.Greater(t, store.Count(), 100000,
		"real Tranco cache should yield ~1M entries, not 4 - got %d", store.Count())
	assert.NotZero(t, store.GetRank("google.com"))
	assert.NotZero(t, store.GetRank("youtube.com"))
}
