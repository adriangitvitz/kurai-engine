package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type MockResult struct {
	URL   string
	Title string
	Score float64
}

func TestNewSearchCache(t *testing.T) {
	cache := NewSearchCache(5 * time.Minute)
	if cache == nil {
		t.Fatal("NewSearchCache returned nil")
	}
	if cache.ttl != 5*time.Minute {
		t.Errorf("TTL not set correctly: got %v, want %v", cache.ttl, 5*time.Minute)
	}
}

func TestCacheSetAndGet(t *testing.T) {
	cache := NewSearchCache(5 * time.Minute)

	results := []any{
		MockResult{URL: "https://example.com/1", Title: "Result 1", Score: 100},
		MockResult{URL: "https://example.com/2", Title: "Result 2", Score: 90},
		MockResult{URL: "https://example.com/3", Title: "Result 3", Score: 80},
	}

	cache.Set("alice", "test query", results)

	got, ok := cache.Get("alice", "test query")
	if !ok {
		t.Fatal("Get returned false for cached query")
	}
	if len(got) != len(results) {
		t.Errorf("Got %d results, want %d", len(got), len(results))
	}
}

func TestCacheGetPage(t *testing.T) {
	cache := NewSearchCache(5 * time.Minute)

	var results []any
	for i := range 50 {
		results = append(results, MockResult{
			URL:   fmt.Sprintf("https://example.com/%d", i),
			Title: "Result",
			Score: float64(100 - i),
		})
	}

	cache.Set("alice", "paginated query", results)

	page1, total, ok := cache.GetPage("alice", "paginated query", 1, 20)
	if !ok {
		t.Fatal("GetPage returned false for cached query")
	}
	if len(page1) != 20 {
		t.Errorf("Page 1: got %d results, want 20", len(page1))
	}
	if total != 50 {
		t.Errorf("Total: got %d, want 50", total)
	}

	page2, _, ok := cache.GetPage("alice", "paginated query", 2, 20)
	if !ok {
		t.Fatal("GetPage returned false for page 2")
	}
	if len(page2) != 20 {
		t.Errorf("Page 2: got %d results, want 20", len(page2))
	}

	page3, _, ok := cache.GetPage("alice", "paginated query", 3, 20)
	if !ok {
		t.Fatal("GetPage returned false for page 3")
	}
	if len(page3) != 10 {
		t.Errorf("Page 3: got %d results, want 10", len(page3))
	}

	page4, _, ok := cache.GetPage("alice", "paginated query", 4, 20)
	if ok {
		t.Error("GetPage should return false for page beyond results")
	}
	if len(page4) != 0 {
		t.Errorf("Page 4: got %d results, want 0", len(page4))
	}
}

func TestCacheKeyHashing(t *testing.T) {
	key1 := cacheKey("alice", "test query")
	key2 := cacheKey("alice", "test query")
	if key1 != key2 {
		t.Error("Same user+query should produce same cache key")
	}

	key3 := cacheKey("alice", "different query")
	if key1 == key3 {
		t.Error("Different queries should produce different cache keys")
	}

	key4 := cacheKey("bob", "test query")
	if key1 == key4 {
		t.Error("Different users should produce different cache keys")
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := NewSearchCache(50 * time.Millisecond)

	results := []any{MockResult{URL: "test", Title: "test", Score: 100}}
	cache.Set("alice", "expiring query", results)

	_, ok := cache.Get("alice", "expiring query")
	if !ok {
		t.Fatal("Cache entry should exist immediately after Set")
	}

	time.Sleep(100 * time.Millisecond)

	_, ok = cache.Get("alice", "expiring query")
	if ok {
		t.Error("Cache entry should be expired")
	}
}

func TestCacheMiss(t *testing.T) {
	cache := NewSearchCache(5 * time.Minute)

	_, ok := cache.Get("alice", "nonexistent query")
	if ok {
		t.Error("Get should return false for uncached query")
	}

	_, _, ok = cache.GetPage("alice", "nonexistent query", 1, 20)
	if ok {
		t.Error("GetPage should return false for uncached query")
	}
}

func TestCacheTotalPages(t *testing.T) {
	cache := NewSearchCache(5 * time.Minute)

	tests := []struct {
		resultCount int
		pageSize    int
		wantPages   int
	}{
		{0, 20, 0},
		{1, 20, 1},
		{20, 20, 1},
		{21, 20, 2},
		{40, 20, 2},
		{41, 20, 3},
		{100, 20, 5},
	}

	for _, tt := range tests {
		var results []any
		for range tt.resultCount {
			results = append(results, MockResult{URL: "test", Title: "test", Score: 100})
		}
		cache.Set("alice", "test", results)

		pages := cache.TotalPages("alice", "test", tt.pageSize)
		if pages != tt.wantPages {
			t.Errorf("TotalPages(%d results, pageSize %d) = %d, want %d",
				tt.resultCount, tt.pageSize, pages, tt.wantPages)
		}
	}
}

func TestUserIsolation(t *testing.T) {
	cache := NewSearchCache(5 * time.Minute)

	results := []any{MockResult{URL: "test", Title: "test", Score: 100}}
	cache.Set("alice", "shared query", results)

	_, ok := cache.Get("alice", "shared query")
	if !ok {
		t.Error("Alice should see her own cached results")
	}

	_, ok = cache.Get("bob", "shared query")
	if ok {
		t.Error("Bob should NOT see Alice's cached results")
	}
}

func TestUserIsolation_SameUser(t *testing.T) {
	cache := NewSearchCache(5 * time.Minute)

	results := []any{MockResult{URL: "test", Title: "test", Score: 100}}
	cache.Set("alice", "my query", results)

	got, ok := cache.Get("alice", "my query")
	if !ok {
		t.Fatal("Same user should hit their own cache")
	}
	if len(got) != 1 {
		t.Errorf("Expected 1 result, got %d", len(got))
	}
}

func TestAnonymousFallback(t *testing.T) {
	cache := NewSearchCache(5 * time.Minute)

	results := []any{MockResult{URL: "test", Title: "test", Score: 100}}
	cache.Set("anonymous", "query", results)

	got, ok := cache.Get("anonymous", "query")
	if !ok {
		t.Fatal("Anonymous user cache should work")
	}
	if len(got) != 1 {
		t.Errorf("Expected 1 result, got %d", len(got))
	}
}

func TestShardDistribution(t *testing.T) {
	seen := make(map[int]bool)
	for i := range 1000 {
		raw := cacheKeyRaw(fmt.Sprintf("user%d", i), fmt.Sprintf("query%d", i))
		seen[shardIndex(raw)] = true
	}

	if len(seen) < 200 {
		t.Errorf("Poor shard distribution: only %d/256 shards used with 1000 keys", len(seen))
	}
}

func TestConcurrentAccess(t *testing.T) {
	cache := NewSearchCache(5 * time.Minute)

	var wg sync.WaitGroup
	for u := range 10 {
		for q := range 10 {
			wg.Add(1)
			go func(user, query int) {
				defer wg.Done()
				userID := fmt.Sprintf("user%d", user)
				queryStr := fmt.Sprintf("query%d", query)

				results := []any{MockResult{
					URL:   fmt.Sprintf("https://example.com/%d/%d", user, query),
					Title: "Result",
					Score: 100,
				}}

				cache.Set(userID, queryStr, results)
				cache.Get(userID, queryStr)
				cache.GetPage(userID, queryStr, 1, 10)
			}(u, q)
		}
	}

	wg.Wait()

	size := cache.Size()
	if size != 100 {
		t.Errorf("Expected 100 entries, got %d", size)
	}
}

func TestClear(t *testing.T) {
	cache := NewSearchCache(5 * time.Minute)

	cache.Set("alice", "q1", []any{MockResult{URL: "test"}})
	cache.Set("bob", "q2", []any{MockResult{URL: "test"}})

	if cache.Size() != 2 {
		t.Fatalf("Expected 2 entries, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected 0 entries after Clear, got %d", cache.Size())
	}
}
