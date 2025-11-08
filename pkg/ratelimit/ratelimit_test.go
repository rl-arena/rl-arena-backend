package ratelimit

import (
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	bucket := NewTokenBucket(5, 1) // 5 capacity, 1 refill per second

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		if !bucket.Allow() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if bucket.Allow() {
		t.Error("6th request should be denied")
	}

	// Wait 1 second for refill
	time.Sleep(1100 * time.Millisecond)

	// Should allow 1 more request
	if !bucket.Allow() {
		t.Error("Request after refill should be allowed")
	}
}

func TestTokenBucket_AllowN(t *testing.T) {
	bucket := NewTokenBucket(10, 2) // 10 capacity, 2 refill per second

	// Should allow 10 tokens
	if !bucket.AllowN(10) {
		t.Error("AllowN(10) should be allowed")
	}

	// Should deny 1 more
	if bucket.AllowN(1) {
		t.Error("AllowN(1) should be denied after consuming all tokens")
	}

	// Wait 1 second (should refill 2 tokens)
	time.Sleep(1100 * time.Millisecond)

	// Should allow 2 tokens
	if !bucket.AllowN(2) {
		t.Error("AllowN(2) should be allowed after refill")
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(3, 1) // 3 capacity, 1 refill per second

	// Test for key "user1"
	for i := 0; i < 3; i++ {
		if !limiter.Allow("user1") {
			t.Errorf("Request %d for user1 should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if limiter.Allow("user1") {
		t.Error("4th request for user1 should be denied")
	}

	// Different key should have separate bucket
	if !limiter.Allow("user2") {
		t.Error("First request for user2 should be allowed")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	limiter := NewRateLimiter(5, 2) // 5 capacity, 2 refill per second

	// Consume all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow("test")
	}

	// Should be denied
	if limiter.Allow("test") {
		t.Error("Request should be denied after consuming all tokens")
	}

	// Wait 1 second (should refill 2 tokens)
	time.Sleep(1100 * time.Millisecond)

	// Should allow 2 requests
	if !limiter.Allow("test") || !limiter.Allow("test") {
		t.Error("Should allow 2 requests after refill")
	}

	// 3rd should be denied
	if limiter.Allow("test") {
		t.Error("3rd request should be denied")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	limiter := NewRateLimiter(2, 1)

	// Consume all tokens
	limiter.Allow("test")
	limiter.Allow("test")

	// Should be denied
	if limiter.Allow("test") {
		t.Error("Request should be denied")
	}

	// Reset
	limiter.Reset("test")

	// Should be allowed again
	if !limiter.Allow("test") {
		t.Error("Request should be allowed after reset")
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewRateLimiter(100, 10)
	done := make(chan bool)

	// Simulate concurrent requests
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				limiter.Allow("concurrent")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Stats should be consistent
	stats := limiter.GetStats()
	if stats["active_buckets"].(int) != 1 {
		t.Errorf("Expected 1 active bucket, got %d", stats["active_buckets"])
	}
}

func BenchmarkTokenBucket_Allow(b *testing.B) {
	bucket := NewTokenBucket(1000000, 100000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bucket.Allow()
	}
}

func BenchmarkRateLimiter_Allow(b *testing.B) {
	limiter := NewRateLimiter(1000000, 100000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow("test")
	}
}
