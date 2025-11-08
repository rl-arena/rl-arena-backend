package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket implements the token bucket algorithm for rate limiting
type TokenBucket struct {
	mu           sync.Mutex
	capacity     int64         // Maximum number of tokens
	tokens       int64         // Current number of tokens
	refillRate   int64         // Tokens added per second
	lastRefill   time.Time     // Last refill timestamp
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token if so
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN checks if n requests are allowed and consumes n tokens if so
func (tb *TokenBucket) AllowN(n int64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}

	return false
}

// refill adds tokens based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	
	// Calculate tokens to add
	tokensToAdd := int64(elapsed.Seconds()) * tb.refillRate
	
	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}
}

// RateLimiter manages rate limits for multiple keys (e.g., user IDs, IP addresses)
type RateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*TokenBucket
	capacity int64
	refillRate int64
	cleanupInterval time.Duration
	lastCleanup time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(capacity, refillRate int64) *RateLimiter {
	rl := &RateLimiter{
		buckets:    make(map[string]*TokenBucket),
		capacity:   capacity,
		refillRate: refillRate,
		cleanupInterval: 10 * time.Minute,
		lastCleanup: time.Now(),
	}
	
	// Start background cleanup
	go rl.cleanupLoop()
	
	return rl
}

// Allow checks if a request from the given key is allowed
func (rl *RateLimiter) Allow(key string) bool {
	return rl.AllowN(key, 1)
}

// AllowN checks if n requests from the given key are allowed
func (rl *RateLimiter) AllowN(key string, n int64) bool {
	bucket := rl.getBucket(key)
	return bucket.AllowN(n)
}

// getBucket gets or creates a token bucket for the given key
func (rl *RateLimiter) getBucket(key string) *TokenBucket {
	rl.mu.RLock()
	bucket, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	bucket, exists = rl.buckets[key]
	if exists {
		return bucket
	}

	bucket = NewTokenBucket(rl.capacity, rl.refillRate)
	rl.buckets[key] = bucket
	return bucket
}

// cleanupLoop periodically removes inactive buckets to prevent memory leaks
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes buckets that are full (haven't been used recently)
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, bucket := range rl.buckets {
		bucket.mu.Lock()
		
		// Remove if bucket is full and hasn't been used for a while
		if bucket.tokens == bucket.capacity && 
		   now.Sub(bucket.lastRefill) > rl.cleanupInterval {
			delete(rl.buckets, key)
		}
		
		bucket.mu.Unlock()
	}

	rl.lastCleanup = now
}

// Reset resets the rate limit for a given key
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.buckets, key)
}

// GetStats returns statistics about the rate limiter
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"active_buckets": len(rl.buckets),
		"capacity":       rl.capacity,
		"refill_rate":    rl.refillRate,
		"last_cleanup":   rl.lastCleanup,
	}
}
