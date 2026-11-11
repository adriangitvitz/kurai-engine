package blocklist

import (
	"strings"
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
)

type DomainStore struct {
	mu    sync.RWMutex
	bloom *bloom.BloomFilter
	exact map[string]struct{}
}

func NewDomainStore(expectedItems int) *DomainStore {
	if expectedItems < 1 {
		expectedItems = 1000
	}
	bf := bloom.NewWithEstimates(uint(expectedItems), 0.01)
	return &DomainStore{
		bloom: bf,
		exact: make(map[string]struct{}, expectedItems),
	}
}

func (s *DomainStore) Add(domain string) {
	domain = normalizeDomain(domain)
	if domain == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.bloom.Add([]byte(domain))
	s.exact[domain] = struct{}{}
}

func (s *DomainStore) Contains(domain string) bool {
	domain = normalizeDomain(domain)
	if domain == "" {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.bloom.Test([]byte(domain)) {
		return false
	}

	_, exists := s.exact[domain]
	return exists
}

func (s *DomainStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.exact)
}

func (s *DomainStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bloom.ClearAll()
	s.exact = make(map[string]struct{})
}

func normalizeDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "www.")
	return domain
}
