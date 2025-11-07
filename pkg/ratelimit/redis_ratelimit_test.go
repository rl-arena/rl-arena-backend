package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRedisRateLimiter 테스트용 Redis Rate Limiter 설정
// 주의: 실제 Redis 서버가 필요합니다 (localhost:6379)
func setupRedisRateLimiter(t *testing.T) *RedisRateLimiter {
	config := RedisRateLimiterConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           15, // 테스트용 DB
		KeyPrefix:    "test:ratelimit:",
		DefaultLimit: 5,
		DefaultTTL:   time.Minute,
	}

	limiter := NewRedisRateLimiter(config)

	// Redis 연결 확인
	ctx := context.Background()
	err := limiter.Ping(ctx)
	if err != nil {
		t.Skipf("Redis server not available: %v", err)
	}

	return limiter
}

// cleanupRedis 테스트 후 정리
func cleanupRedis(t *testing.T, limiter *RedisRateLimiter, keys ...string) {
	ctx := context.Background()
	for _, key := range keys {
		limiter.Reset(ctx, key)
	}
}

func TestRedisRateLimiter_Allow(t *testing.T) {
	limiter := setupRedisRateLimiter(t)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:123"
	defer cleanupRedis(t, limiter, key)

	t.Run("첫 요청은 항상 허용", func(t *testing.T) {
		allowed, err := limiter.Allow(ctx, key, 5, time.Minute)
		require.NoError(t, err)
		assert.True(t, allowed)
	})
}

func TestRedisRateLimiter_TokenBucket(t *testing.T) {
	limiter := setupRedisRateLimiter(t)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:456"
	defer cleanupRedis(t, limiter, key)

	limit := 3
	window := time.Minute

	t.Run("제한 내 요청은 모두 허용", func(t *testing.T) {
		for i := 0; i < limit; i++ {
			allowed, err := limiter.Allow(ctx, key, limit, window)
			require.NoError(t, err)
			assert.True(t, allowed, "Request %d should be allowed", i+1)
		}
	})

	t.Run("제한 초과 요청은 거부", func(t *testing.T) {
		allowed, err := limiter.Allow(ctx, key, limit, window)
		require.NoError(t, err)
		assert.False(t, allowed, "Request over limit should be denied")
	})
}

func TestRedisRateLimiter_AllowWithInfo(t *testing.T) {
	limiter := setupRedisRateLimiter(t)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:789"
	defer cleanupRedis(t, limiter, key)

	limit := 5
	window := time.Minute

	t.Run("상세 정보 반환", func(t *testing.T) {
		allowed, info, err := limiter.AllowWithInfo(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, allowed)
		assert.NotNil(t, info)
		assert.Equal(t, limit, info.Limit)
		assert.Equal(t, limit-1, info.Remaining)
		assert.False(t, info.ResetTime.IsZero())
	})

	t.Run("Remaining 감소 확인", func(t *testing.T) {
		// 2번 더 요청
		limiter.Allow(ctx, key, limit, window)
		limiter.Allow(ctx, key, limit, window)

		allowed, info, err := limiter.AllowWithInfo(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, limit-4, info.Remaining)
	})
}

func TestRedisRateLimiter_TokenRefill(t *testing.T) {
	limiter := setupRedisRateLimiter(t)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:refill"
	defer cleanupRedis(t, limiter, key)

	limit := 2
	window := 2 * time.Second // 짧은 윈도우로 테스트

	t.Run("토큰 소진 후 리필 테스트", func(t *testing.T) {
		// 토큰 2개 소비
		allowed1, _ := limiter.Allow(ctx, key, limit, window)
		allowed2, _ := limiter.Allow(ctx, key, limit, window)
		assert.True(t, allowed1)
		assert.True(t, allowed2)

		// 토큰 소진 확인
		allowed3, _ := limiter.Allow(ctx, key, limit, window)
		assert.False(t, allowed3, "Should be denied when tokens exhausted")

		// 1초 대기 (토큰 1개 리필 예상)
		time.Sleep(1 * time.Second)

		// 리필된 토큰으로 요청 가능
		allowed4, _ := limiter.Allow(ctx, key, limit, window)
		assert.True(t, allowed4, "Should be allowed after token refill")
	})
}

func TestRedisRateLimiter_Reset(t *testing.T) {
	limiter := setupRedisRateLimiter(t)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:reset"

	limit := 2
	window := time.Minute

	t.Run("리셋 후 토큰 복구", func(t *testing.T) {
		// 토큰 모두 소비
		limiter.Allow(ctx, key, limit, window)
		limiter.Allow(ctx, key, limit, window)

		// 소진 확인
		allowed, _ := limiter.Allow(ctx, key, limit, window)
		assert.False(t, allowed)

		// 리셋
		err := limiter.Reset(ctx, key)
		require.NoError(t, err)

		// 리셋 후 다시 허용
		allowed, _ = limiter.Allow(ctx, key, limit, window)
		assert.True(t, allowed)
	})
}

func TestRedisRateLimiter_MultipleKeys(t *testing.T) {
	limiter := setupRedisRateLimiter(t)
	defer limiter.Close()

	ctx := context.Background()
	key1 := "user:multi1"
	key2 := "user:multi2"
	defer cleanupRedis(t, limiter, key1, key2)

	limit := 2
	window := time.Minute

	t.Run("키별 독립적인 Rate Limit", func(t *testing.T) {
		// key1 토큰 소진
		limiter.Allow(ctx, key1, limit, window)
		limiter.Allow(ctx, key1, limit, window)
		allowed1, _ := limiter.Allow(ctx, key1, limit, window)
		assert.False(t, allowed1, "key1 should be limited")

		// key2는 여전히 허용
		allowed2, _ := limiter.Allow(ctx, key2, limit, window)
		assert.True(t, allowed2, "key2 should be allowed")
	})
}

func TestRedisRateLimiter_ConcurrentRequests(t *testing.T) {
	limiter := setupRedisRateLimiter(t)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:concurrent"
	defer cleanupRedis(t, limiter, key)

	limit := 10
	window := time.Minute

	t.Run("동시 요청 처리", func(t *testing.T) {
		concurrency := 20
		results := make(chan bool, concurrency)

		// 동시 요청
		for i := 0; i < concurrency; i++ {
			go func() {
				allowed, _ := limiter.Allow(ctx, key, limit, window)
				results <- allowed
			}()
		}

		// 결과 수집
		allowedCount := 0
		for i := 0; i < concurrency; i++ {
			if <-results {
				allowedCount++
			}
		}

		// limit 만큼만 허용되어야 함
		assert.Equal(t, limit, allowedCount, "Only %d requests should be allowed", limit)
	})
}

func TestRedisRateLimiter_Ping(t *testing.T) {
	limiter := setupRedisRateLimiter(t)
	defer limiter.Close()

	ctx := context.Background()

	t.Run("Redis 연결 확인", func(t *testing.T) {
		err := limiter.Ping(ctx)
		assert.NoError(t, err)
	})
}

func TestRedisRateLimiter_InvalidRedis(t *testing.T) {
	t.Run("잘못된 Redis 주소", func(t *testing.T) {
		config := RedisRateLimiterConfig{
			Addr:      "invalid:9999",
			KeyPrefix: "test:",
		}
		limiter := NewRedisRateLimiter(config)
		defer limiter.Close()

		ctx := context.Background()
		err := limiter.Ping(ctx)
		assert.Error(t, err)
	})
}

// BenchmarkRedisRateLimiter_Allow Redis Rate Limiter 성능 벤치마크
func BenchmarkRedisRateLimiter_Allow(b *testing.B) {
	config := RedisRateLimiterConfig{
		Addr:      "localhost:6379",
		DB:        15,
		KeyPrefix: "bench:ratelimit:",
	}
	limiter := NewRedisRateLimiter(config)
	defer limiter.Close()

	ctx := context.Background()
	// Redis 연결 확인
	if err := limiter.Ping(ctx); err != nil {
		b.Skipf("Redis not available: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "user:bench"
		limiter.Allow(ctx, key, 1000, time.Minute)
	}
}
