package blocklist

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrancoStore_SetAndGetRank(t *testing.T) {
	store := NewTrancoStore()

	store.Set("google.com", 1)
	store.Set("github.com", 100)
	store.Set("example.com", 1000)

	assert.Equal(t, 1, store.GetRank("google.com"))
	assert.Equal(t, 100, store.GetRank("github.com"))
	assert.Equal(t, 1000, store.GetRank("example.com"))
}

func TestTrancoStore_GetRank_UnknownDomain(t *testing.T) {
	store := NewTrancoStore()

	store.Set("google.com", 1)

	assert.Equal(t, 0, store.GetRank("unknown.com"))
	assert.Equal(t, 0, store.GetRank(""))
}

func TestTrancoStore_Normalization(t *testing.T) {
	store := NewTrancoStore()

	store.Set("www.Example.COM", 42)

	assert.Equal(t, 42, store.GetRank("example.com"))
	assert.Equal(t, 42, store.GetRank("www.example.com"))
	assert.Equal(t, 42, store.GetRank("EXAMPLE.COM"))
}

func TestTrancoStore_Count(t *testing.T) {
	store := NewTrancoStore()

	assert.Equal(t, 0, store.Count())

	store.Set("google.com", 1)
	assert.Equal(t, 1, store.Count())

	store.Set("github.com", 2)
	assert.Equal(t, 2, store.Count())

	store.Set("google.com", 5)
	assert.Equal(t, 2, store.Count())
}

func TestTrancoStore_ConcurrentAccess(t *testing.T) {
	store := NewTrancoStore()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				store.Set("domain"+string(rune('a'+n))+".com", n*100+j)
			}
		}(i % 26)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = store.GetRank("google.com")
			}
		}()
	}

	wg.Wait()
}

func TestTrancoStore_Clear(t *testing.T) {
	store := NewTrancoStore()

	store.Set("google.com", 1)
	store.Set("github.com", 2)
	assert.Equal(t, 2, store.Count())

	store.Clear()
	assert.Equal(t, 0, store.Count())
	assert.Equal(t, 0, store.GetRank("google.com"))
}

func TestParseTrancoCSV(t *testing.T) {
	store := NewTrancoStore()

	testdataPath := filepath.Join("testdata", "sample_tranco.csv")
	data, err := os.ReadFile(testdataPath)
	require.NoError(t, err)

	err = store.ParseCSV(bytes.NewReader(data))
	require.NoError(t, err)

	assert.Equal(t, 1, store.GetRank("google.com"))
	assert.Equal(t, 2, store.GetRank("youtube.com"))
	assert.Equal(t, 100, store.GetRank("github.com"))
	assert.Equal(t, 10000, store.GetRank("test.org"))
	assert.Equal(t, 10, store.Count())
}

func TestParseTrancoCSV_InvalidFormat(t *testing.T) {
	store := NewTrancoStore()

	data := "google.com\nyoutube.com\n"
	err := store.ParseCSV(bytes.NewReader([]byte(data)))
	require.NoError(t, err)

	data = "abc,google.com\n"
	err = store.ParseCSV(bytes.NewReader([]byte(data)))
	require.NoError(t, err)
}

func TestParseTrancoZip(t *testing.T) {
	store := NewTrancoStore()

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	csvContent := "1,google.com\n2,github.com\n3,example.com\n"
	writer, err := zipWriter.Create("top-1m.csv")
	require.NoError(t, err)
	_, err = writer.Write([]byte(csvContent))
	require.NoError(t, err)
	zipWriter.Close()

	err = store.ParseZip(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)

	assert.Equal(t, 1, store.GetRank("google.com"))
	assert.Equal(t, 2, store.GetRank("github.com"))
	assert.Equal(t, 3, store.GetRank("example.com"))
	assert.Equal(t, 3, store.Count())
}

func TestParseTrancoZip_NoCSV(t *testing.T) {
	store := NewTrancoStore()

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	writer, err := zipWriter.Create("readme.txt")
	require.NoError(t, err)
	_, err = writer.Write([]byte("no csv here"))
	require.NoError(t, err)
	zipWriter.Close()

	err = store.ParseZip(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no CSV file found")
}

func BenchmarkTrancoStore_GetRank(b *testing.B) {
	store := NewTrancoStore()

	for i := 1; i <= 1000000; i++ {
		store.Set("domain"+string(rune(i%65536))+".com", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetRank("domain500000.com")
	}
}
