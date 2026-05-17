package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// TokenBucketLimiter is an in-memory, per-key token bucket rate limiter backed
// by golang.org/x/time/rate.  A background goroutine periodically evicts stale
// entries so memory does not grow without bound.
//
// Not suitable for multi-instance deployments without a shared store — use it
// per-instance only.
type TokenBucketLimiter struct {
	mu       sync.Mutex // guards entries map during eviction
	entries  sync.Map   // key(string) → *entry
	rate     rate.Limit
	burst    int
	ttl      time.Duration
	stopOnce sync.Once
	stopCh   chan struct{}
}

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
	mu       sync.Mutex
}

func (e *entry) touch() {
	e.mu.Lock()
	e.lastSeen = time.Now()
	e.mu.Unlock()
}

// NewTokenBucketLimiter creates a per-key token bucket limiter.
//   - r: sustained requests per second (use rate.Inf for unlimited)
//   - burst: maximum burst size
//
// Stale entries (not accessed for 10 minutes) are evicted every 5 minutes.
func NewTokenBucketLimiter(r rate.Limit, burst int) *TokenBucketLimiter {
	l := &TokenBucketLimiter{
		rate:   r,
		burst:  burst,
		ttl:    10 * time.Minute,
		stopCh: make(chan struct{}),
	}
	go l.cleanupLoop(5 * time.Minute)
	return l
}

// Allow reports whether the request identified by key is within the rate limit.
func (l *TokenBucketLimiter) Allow(key string) bool {
	e := l.getOrCreate(key)
	e.touch()
	return e.limiter.Allow()
}

// Stop shuts down the background cleanup goroutine.  Safe to call multiple times.
func (l *TokenBucketLimiter) Stop() {
	l.stopOnce.Do(func() { close(l.stopCh) })
}

func (l *TokenBucketLimiter) getOrCreate(key string) *entry {
	if v, ok := l.entries.Load(key); ok {
		return v.(*entry)
	}
	e := &entry{
		limiter:  rate.NewLimiter(l.rate, l.burst),
		lastSeen: time.Now(),
	}
	// Use LoadOrStore to handle the race where two goroutines both miss the
	// initial Load.
	actual, _ := l.entries.LoadOrStore(key, e)
	return actual.(*entry)
}

func (l *TokenBucketLimiter) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-l.stopCh:
			return
		case <-ticker.C:
			l.evictStale()
		}
	}
}

func (l *TokenBucketLimiter) evictStale() {
	cutoff := time.Now().Add(-l.ttl)
	l.entries.Range(func(k, v any) bool {
		e := v.(*entry)
		e.mu.Lock()
		stale := e.lastSeen.Before(cutoff)
		e.mu.Unlock()
		if stale {
			l.entries.Delete(k)
		}
		return true
	})
}
