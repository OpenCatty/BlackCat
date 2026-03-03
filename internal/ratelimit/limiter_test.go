package ratelimit

import (
	"testing"
	"time"
)

func TestRateLimiter_Basic(t *testing.T) {
	limiter := NewLimiter(10, 60)

	for i := 0; i < 10; i++ {
		if !limiter.Allow("user:123") {
			t.Fatalf("expected allow on call %d", i+1)
		}
	}

	if limiter.Allow("user:123") {
		t.Fatal("expected deny on call 11")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	limiter := NewLimiter(2, 1) // 2 requests per 1 second

	if !limiter.Allow("user:456") {
		t.Fatal("expected allow on call 1")
	}
	if !limiter.Allow("user:456") {
		t.Fatal("expected allow on call 2")
	}
	if limiter.Allow("user:456") {
		t.Fatal("expected deny on call 3")
	}

	// Manually expire the window.
	limiter.mu.Lock()
	limiter.entries["user:456"].windowStart = time.Now().Add(-2 * time.Second)
	limiter.mu.Unlock()

	if !limiter.Allow("user:456") {
		t.Fatal("expected allow after window reset")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	limiter := NewLimiter(1, 60)

	if !limiter.Allow("whatsapp:user1") {
		t.Fatal("expected allow for user1")
	}
	if limiter.Allow("whatsapp:user1") {
		t.Fatal("expected deny for user1")
	}
	if !limiter.Allow("telegram:user1") {
		t.Fatal("expected allow for telegram:user1 (different key)")
	}
	if !limiter.Allow("whatsapp:user2") {
		t.Fatal("expected allow for user2 (different key)")
	}
}
