package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/pkg/ratelimit"
)

// RateLimitConfig holds rate limit configuration
type RateLimitConfig struct {
	Capacity   int64  // Maximum number of requests
	RefillRate int64  // Requests per second
	KeyFunc    func(*gin.Context) string // Function to extract rate limit key
}

// DefaultKeyFunc uses user ID if authenticated, otherwise IP address
func DefaultKeyFunc(c *gin.Context) string {
	// Try to get user ID from context (set by auth middleware)
	if userID, exists := c.Get("userID"); exists {
		return fmt.Sprintf("user:%v", userID)
	}
	
	// Fall back to IP address
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// IPKeyFunc uses only IP address (for public endpoints)
func IPKeyFunc(c *gin.Context) string {
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// UserKeyFunc uses only user ID (requires authentication)
func UserKeyFunc(c *gin.Context) string {
	if userID, exists := c.Get("userID"); exists {
		return fmt.Sprintf("user:%v", userID)
	}
	return ""
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	limiter := ratelimit.NewRateLimiter(config.Capacity, config.RefillRate)
	
	if config.KeyFunc == nil {
		config.KeyFunc = DefaultKeyFunc
	}

	return func(c *gin.Context) {
		key := config.KeyFunc(c)
		
		if key == "" {
			// No key available (e.g., user not authenticated for UserKeyFunc)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required for rate limiting",
			})
			c.Abort()
			return
		}

		// Check if request is allowed
		if !limiter.Allow(key) {
			// Add rate limit headers
			c.Header("X-RateLimit-Limit", strconv.FormatInt(config.Capacity, 10))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))
			c.Header("Retry-After", "1")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests. Limit: %d requests per second", config.RefillRate),
			})
			c.Abort()
			return
		}

		// Add rate limit headers for successful requests
		c.Header("X-RateLimit-Limit", strconv.FormatInt(config.Capacity, 10))
		
		c.Next()
	}
}

// Common rate limit configurations

// SubmissionRateLimit - 5 requests per minute per user
func SubmissionRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(RateLimitConfig{
		Capacity:   5,
		RefillRate: 1,  // 1 token per second = 60 per minute
		KeyFunc:    UserKeyFunc,
	})
}

// MatchCreationRateLimit - 10 requests per minute per user
func MatchCreationRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(RateLimitConfig{
		Capacity:   10,
		RefillRate: 1,  // 1 token per second
		KeyFunc:    UserKeyFunc,
	})
}

// GeneralAPIRateLimit - 100 requests per minute per IP/user
func GeneralAPIRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(RateLimitConfig{
		Capacity:   100,
		RefillRate: 10,  // 10 tokens per second = 600 per minute
		KeyFunc:    DefaultKeyFunc,
	})
}

// ReplayDownloadRateLimit - 20 downloads per minute per IP/user
func ReplayDownloadRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(RateLimitConfig{
		Capacity:   20,
		RefillRate: 2,  // 2 tokens per second = 120 per minute
		KeyFunc:    DefaultKeyFunc,
	})
}

// AuthRateLimit - 5 login/register attempts per minute per IP
func AuthRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(RateLimitConfig{
		Capacity:   5,
		RefillRate: 1,  // 1 token per second
		KeyFunc:    IPKeyFunc,
	})
}
