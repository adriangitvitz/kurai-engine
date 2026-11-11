package cache

import (
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

const numShards = 256

type ContentType string

const (
	ContentTypeWeb      ContentType = "web"
	ContentTypeImages   ContentType = "images"
	ContentTypeNews     ContentType = "news"
	ContentTypeTrending ContentType = "trending"
	ContentTypeExtract  ContentType = "extract"
	ContentTypeFX       ContentType = "fx"
	ContentTypeWeather  ContentType = "wx"
)

type CachedSearch struct {
	Results     []any
	FetchedAt   time.Time
	ContentType ContentType
}

type shard struct {
	mu      sync.RWMutex
	entries map[string]*CachedSearch
}

type SearchCache struct {
	shards    [numShards]shard
	ttl       time.Duration
	ttlByType map[ContentType]time.Duration
}

func NewSearchCache(ttl time.Duration) *SearchCache {
	cache := &SearchCache{
		ttl: ttl,
		ttlByType: map[ContentType]time.Duration{
			ContentTypeWeb:      ttl,
			ContentTypeImages:   1 * time.Hour,
			ContentTypeNews:     5 * time.Minute,
			ContentTypeTrending: 2 * time.Minute,
			ContentTypeExtract:  24 * time.Hour,
			ContentTypeFX:       1 * time.Hour,
			ContentTypeWeather:  30 * time.Minute,
		},
	}
	for i := range cache.shards {
		cache.shards[i].entries = make(map[string]*CachedSearch)
	}

	go cache.cleanupLoop()
	return cache
}

func NewSearchCacheWithTTLs(defaultTTL time.Duration, ttlByType map[ContentType]time.Duration) *SearchCache {
	cache := &SearchCache{
		ttl:       defaultTTL,
		ttlByType: ttlByType,
	}
	for i := range cache.shards {
		cache.shards[i].entries = make(map[string]*CachedSearch)
	}
	go cache.cleanupLoop()
	return cache
}

func (c *SearchCache) SetTTL(contentType ContentType, ttl time.Duration) {
	c.ttlByType[contentType] = ttl
}

func cacheKeyRaw(userID, query string) [32]byte {
	return blake3.Sum256([]byte(userID + ":" + query))
}

func cacheKey(userID, query string) string {
	h := cacheKeyRaw(userID, query)
	return hex.EncodeToString(h[:16])
}

func shardIndex(raw [32]byte) int {
	return int(raw[0])
}

func (c *SearchCache) getShard(userID, query string) (*shard, string) {
	raw := cacheKeyRaw(userID, query)
	key := hex.EncodeToString(raw[:16])
	return &c.shards[shardIndex(raw)], key
}

func (c *SearchCache) getTTLForKey(query string) time.Duration {
	var contentType ContentType
	switch {
	case strings.HasPrefix(query, "images:"):
		contentType = ContentTypeImages
	case strings.HasPrefix(query, "news:"):
		contentType = ContentTypeNews
	case strings.HasPrefix(query, "trending:"):
		contentType = ContentTypeTrending
	case strings.HasPrefix(query, "extract:"):
		contentType = ContentTypeExtract
	case strings.HasPrefix(query, "fx:"):
		contentType = ContentTypeFX
	case strings.HasPrefix(query, "wx:"):
		contentType = ContentTypeWeather
	default:
		contentType = ContentTypeWeb
	}

	if ttl, ok := c.ttlByType[contentType]; ok {
		return ttl
	}
	return c.ttl
}

func (c *SearchCache) getTTLForEntry(entry *CachedSearch) time.Duration {
	if ttl, ok := c.ttlByType[entry.ContentType]; ok {
		return ttl
	}
	return c.ttl
}

func (c *SearchCache) Get(userID, query string) ([]any, bool) {
	s, key := c.getShard(userID, query)

	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[key]
	if !ok {
		return nil, false
	}

	ttl := c.getTTLForKey(query)
	if time.Since(entry.FetchedAt) > ttl {
		return nil, false
	}

	return entry.Results, true
}

func (c *SearchCache) GetPage(userID, query string, page, pageSize int) ([]any, int, bool) {
	results, ok := c.Get(userID, query)
	if !ok {
		return nil, 0, false
	}

	total := len(results)

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return nil, total, false
	}
	if end > total {
		end = total
	}

	return results[start:end], total, true
}

func (c *SearchCache) Set(userID, query string, results []any) {
	s, key := c.getShard(userID, query)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[key] = &CachedSearch{
		Results:   results,
		FetchedAt: time.Now(),
	}
}

func (c *SearchCache) TotalPages(userID, query string, pageSize int) int {
	results, ok := c.Get(userID, query)
	if !ok || pageSize <= 0 {
		return 0
	}
	return (len(results) + pageSize - 1) / pageSize
}

func (c *SearchCache) ClearUser(_ string) {}

func (c *SearchCache) Clear() {
	for i := range c.shards {
		c.shards[i].mu.Lock()
		c.shards[i].entries = make(map[string]*CachedSearch)
		c.shards[i].mu.Unlock()
	}
}

func (c *SearchCache) Size() int {
	total := 0
	for i := range c.shards {
		c.shards[i].mu.RLock()
		total += len(c.shards[i].entries)
		c.shards[i].mu.RUnlock()
	}
	return total
}

func (c *SearchCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *SearchCache) cleanup() {
	now := time.Now()
	for i := range c.shards {
		s := &c.shards[i]
		s.mu.Lock()
		for key, entry := range s.entries {
			ttl := c.getTTLForEntry(entry)
			if now.Sub(entry.FetchedAt) > ttl {
				delete(s.entries, key)
			}
		}
		s.mu.Unlock()
	}
}
