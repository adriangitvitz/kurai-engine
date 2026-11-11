package blocklist

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainStore_AddAndContains(t *testing.T) {
	store := NewDomainStore(1000)

	store.Add("ads.example.com")
	store.Add("spam.net")
	store.Add("tracker.io")

	assert.True(t, store.Contains("ads.example.com"))
	assert.True(t, store.Contains("spam.net"))
	assert.True(t, store.Contains("tracker.io"))
	assert.False(t, store.Contains("example.com"))
	assert.False(t, store.Contains("good.org"))
}

func TestDomainStore_Normalization(t *testing.T) {
	store := NewDomainStore(1000)

	store.Add("www.example.com")

	assert.True(t, store.Contains("example.com"))
	assert.True(t, store.Contains("www.example.com"))

	store.Add("test.org")

	assert.True(t, store.Contains("test.org"))
	assert.True(t, store.Contains("www.test.org"))
}

func TestDomainStore_CaseInsensitive(t *testing.T) {
	store := NewDomainStore(1000)

	store.Add("EXAMPLE.COM")

	assert.True(t, store.Contains("example.com"))
	assert.True(t, store.Contains("Example.Com"))
	assert.True(t, store.Contains("EXAMPLE.COM"))
}

func TestDomainStore_Count(t *testing.T) {
	store := NewDomainStore(1000)

	assert.Equal(t, 0, store.Count())

	store.Add("example.com")
	assert.Equal(t, 1, store.Count())

	store.Add("test.org")
	assert.Equal(t, 2, store.Count())

	store.Add("example.com")
	assert.Equal(t, 2, store.Count())
}

func TestDomainStore_ConcurrentAccess(t *testing.T) {
	store := NewDomainStore(10000)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				store.Add("domain" + string(rune('a'+n)) + ".com")
			}
		}(i % 26)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = store.Contains("example.com")
			}
		}()
	}

	wg.Wait()
}

func TestDomainStore_EmptyDomain(t *testing.T) {
	store := NewDomainStore(1000)

	store.Add("")
	assert.False(t, store.Contains(""))
}

func TestDomainStore_BloomFilterFalsePositiveRate(t *testing.T) {
	store := NewDomainStore(10000)

	for i := 0; i < 5000; i++ {
		store.Add("blocked" + string(rune(i)) + ".com")
	}

	falsePositives := 0
	for i := 0; i < 5000; i++ {
		if store.Contains("notblocked" + string(rune(i)) + ".org") {
			falsePositives++
		}
	}

	falsePositiveRate := float64(falsePositives) / 5000.0
	assert.Less(t, falsePositiveRate, 0.01, "False positive rate too high: %f", falsePositiveRate)
}

func TestDomainStore_Clear(t *testing.T) {
	store := NewDomainStore(1000)

	store.Add("example.com")
	store.Add("test.org")
	assert.Equal(t, 2, store.Count())

	store.Clear()
	assert.Equal(t, 0, store.Count())
	assert.False(t, store.Contains("example.com"))
}

func BenchmarkDomainStore_Contains(b *testing.B) {
	store := NewDomainStore(1000000)

	for i := 0; i < 1000000; i++ {
		store.Add("domain" + string(rune(i%26)) + string(rune(i%1000)) + ".com")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Contains("domain500000.com")
	}
}

func BenchmarkDomainStore_Add(b *testing.B) {
	store := NewDomainStore(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Add("domain" + string(rune(i)) + ".com")
	}
}
