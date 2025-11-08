package middleware

import (
	"context"
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

// RedisRateLimitConfig Redis 기반 Rate Limit 설정
type RedisRateLimitConfig struct {
	Limiter    *ratelimit.RedisRateLimiter      // Redis Rate Limiter
	Limit      int                               // 윈도우 내 최대 요청 수
	Window     time.Duration                     // 윈도우 크기
	KeyFunc    func(*gin.Context) string         // 키 추출 함수
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

// RedisRateLimitMiddleware Redis 기반 분산 Rate Limiting 미들웨어
func RedisRateLimitMiddleware(config RedisRateLimitConfig) gin.HandlerFunc {
	if config.KeyFunc == nil {
		config.KeyFunc = DefaultKeyFunc
	}
	if config.Limit <= 0 {
		config.Limit = 60
	}
	if config.Window <= 0 {
		config.Window = time.Minute
	}

	return func(c *gin.Context) {
		key := config.KeyFunc(c)
		
		if key == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required for rate limiting",
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		allowed, info, err := config.Limiter.AllowWithInfo(ctx, key, config.Limit, config.Window)
		
		if err != nil {
			// Redis 오류 시 로깅하고 요청 허용 (Fail-open)
			fmt.Printf("Redis rate limit error for key %s: %v\n", key, err)
			c.Next()
			return
		}

		// Rate Limit 헤더 추가
		c.Header("X-RateLimit-Limit", strconv.Itoa(info.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(info.ResetTime.Unix(), 10))

		if !allowed {
			retryAfter := int(time.Until(info.ResetTime).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests. Limit: %d per %v", config.Limit, config.Window),
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Redis 기반 Rate Limit 프리셋 (Limiter가 주입되어야 사용 가능)

// RedisSubmissionRateLimit Redis 기반 제출 Rate Limit (5회/분)
func RedisSubmissionRateLimit(limiter *ratelimit.RedisRateLimiter) gin.HandlerFunc {
	return RedisRateLimitMiddleware(RedisRateLimitConfig{
		Limiter: limiter,
		Limit:   5,
		Window:  time.Minute,
		KeyFunc: UserKeyFunc,
	})
}

// RedisMatchCreationRateLimit Redis 기반 매치 생성 Rate Limit (10회/분)
func RedisMatchCreationRateLimit(limiter *ratelimit.RedisRateLimiter) gin.HandlerFunc {
	return RedisRateLimitMiddleware(RedisRateLimitConfig{
		Limiter: limiter,
		Limit:   10,
		Window:  time.Minute,
		KeyFunc: UserKeyFunc,
	})
}

// RedisAuthRateLimit Redis 기반 인증 Rate Limit (5회/분)
func RedisAuthRateLimit(limiter *ratelimit.RedisRateLimiter) gin.HandlerFunc {
	return RedisRateLimitMiddleware(RedisRateLimitConfig{
		Limiter: limiter,
		Limit:   5,
		Window:  time.Minute,
		KeyFunc: IPKeyFunc, // IP 기반 (인증 전이므로)
	})
}

// RedisReplayDownloadRateLimit Redis 기반 리플레이 다운로드 Rate Limit (20회/분)
func RedisReplayDownloadRateLimit(limiter *ratelimit.RedisRateLimiter) gin.HandlerFunc {
	return RedisRateLimitMiddleware(RedisRateLimitConfig{
		Limiter: limiter,
		Limit:   20,
		Window:  time.Minute,
		KeyFunc: DefaultKeyFunc,
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
