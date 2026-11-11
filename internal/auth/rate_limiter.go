package auth

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   time.Minute,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	if rl == nil || rl.limit <= 0 {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)
	recent := rl.prune(ip, cutoff)

	if len(recent) >= rl.limit {
		rl.attempts[ip] = recent
		return false
	}

	rl.attempts[ip] = append(recent, now)
	return true
}

func (rl *RateLimiter) Limited(ip string) bool {
	if rl == nil || rl.limit <= 0 {
		return false
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.window)
	recent := rl.prune(ip, cutoff)
	return len(recent) >= rl.limit
}

func (rl *RateLimiter) RecordFailure(ip string) {
	if rl == nil || rl.limit <= 0 {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.window)
	recent := rl.prune(ip, cutoff)
	rl.attempts[ip] = append(recent, time.Now())
}

func (rl *RateLimiter) prune(ip string, cutoff time.Time) []time.Time {
	src := rl.attempts[ip]
	out := src[:0]
	for _, t := range src {
		if t.After(cutoff) {
			out = append(out, t)
		}
	}
	return out
}
