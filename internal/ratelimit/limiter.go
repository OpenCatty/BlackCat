package ratelimit

import (
	"sync"
	"time"
)

type windowEntry struct {
	count       int
	windowStart time.Time
}

// Limiter implements a fixed-window rate limiter with per-key tracking.
type Limiter struct {
	mu          sync.Mutex
	entries     map[string]*windowEntry
	maxRequests int
	window      time.Duration
}

// NewLimiter creates a new rate limiter.
func NewLimiter(maxRequests int, windowSeconds int) *Limiter {
	return &Limiter{
		entries:     make(map[string]*windowEntry),
		maxRequests: maxRequests,
		window:      time.Duration(windowSeconds) * time.Second,
	}
}

// Allow checks if the given key is within rate limits.
// Key format: "channelType:userID" (composite for cross-channel isolation).
// Returns true if allowed, false if rate limited.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Inline cleanup: remove expired entries.
	for k, e := range l.entries {
		if now.Sub(e.windowStart) > l.window {
			delete(l.entries, k)
		}
	}

	entry, exists := l.entries[key]
	if !exists || now.Sub(entry.windowStart) > l.window {
		l.entries[key] = &windowEntry{count: 1, windowStart: now}
		return true
	}

	entry.count++
	return entry.count <= l.maxRequests
}
