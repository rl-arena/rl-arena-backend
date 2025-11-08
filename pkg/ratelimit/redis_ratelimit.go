package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter Redis 기반 분산 Rate Limiter (Token Bucket 알고리즘)
type RedisRateLimiter struct {
	client       *redis.Client
	keyPrefix    string
	defaultLimit int
	defaultTTL   time.Duration
}

// RedisRateLimiterConfig Redis Rate Limiter 설정
type RedisRateLimiterConfig struct {
	Addr         string        // Redis 서버 주소 (예: "localhost:6379")
	Password     string        // Redis 비밀번호
	DB           int           // Redis DB 번호
	KeyPrefix    string        // 키 접두사 (예: "ratelimit:")
	DefaultLimit int           // 기본 요청 제한
	DefaultTTL   time.Duration // 기본 TTL (윈도우 크기)
}

// NewRedisRateLimiter Redis 기반 Rate Limiter 생성
func NewRedisRateLimiter(config RedisRateLimiterConfig) *RedisRateLimiter {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	if config.KeyPrefix == "" {
		config.KeyPrefix = "ratelimit:"
	}
	if config.DefaultLimit <= 0 {
		config.DefaultLimit = 60
	}
	if config.DefaultTTL <= 0 {
		config.DefaultTTL = time.Minute
	}

	return &RedisRateLimiter{
		client:       client,
		keyPrefix:    config.KeyPrefix,
		defaultLimit: config.DefaultLimit,
		defaultTTL:   config.DefaultTTL,
	}
}

// Allow 요청 허용 여부 확인 (Token Bucket 알고리즘)
// key: Rate Limit 대상 식별자 (예: userID, IP)
// limit: 윈도우 내 최대 요청 수
// window: 윈도우 크기 (시간)
func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 {
		limit = r.defaultLimit
	}
	if window <= 0 {
		window = r.defaultTTL
	}

	redisKey := r.keyPrefix + key
	now := time.Now().Unix()

	// Lua 스크립트로 원자적 연산 (Token Bucket 알고리즘)
	// 1. 현재 토큰 수 조회
	// 2. 마지막 리필 시간 확인
	// 3. 경과 시간에 따라 토큰 리필
	// 4. 토큰 소비 (1개)
	// 5. 결과 반환
	script := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		-- Redis에서 토큰과 마지막 업데이트 시간 가져오기
		local tokens_key = key .. ":tokens"
		local timestamp_key = key .. ":timestamp"
		
		local tokens = tonumber(redis.call('GET', tokens_key))
		local last_update = tonumber(redis.call('GET', timestamp_key))
		
		-- 초기화 (첫 요청)
		if tokens == nil then
			tokens = limit
			last_update = now
		end
		
		-- 경과 시간 계산 및 토큰 리필
		local elapsed = now - last_update
		local refill_rate = limit / window  -- 초당 리필 속도
		local new_tokens = math.min(limit, tokens + (elapsed * refill_rate))
		
		-- 토큰 소비
		local allowed = 0
		if new_tokens >= 1 then
			new_tokens = new_tokens - 1
			allowed = 1
		end
		
		-- Redis에 업데이트
		redis.call('SET', tokens_key, new_tokens, 'EX', window * 2)
		redis.call('SET', timestamp_key, now, 'EX', window * 2)
		
		-- 결과 반환 (allowed, tokens_remaining, reset_time)
		return {allowed, math.floor(new_tokens), last_update + window}
	`)

	result, err := script.Run(ctx, r.client, []string{redisKey}, limit, int(window.Seconds()), now).Result()
	if err != nil {
		return false, fmt.Errorf("redis script execution failed: %w", err)
	}

	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) < 1 {
		return false, fmt.Errorf("invalid script result")
	}

	allowed, ok := resultSlice[0].(int64)
	if !ok {
		return false, fmt.Errorf("invalid allowed value")
	}

	return allowed == 1, nil
}

// AllowWithInfo 요청 허용 여부와 상세 정보 반환
func (r *RedisRateLimiter) AllowWithInfo(ctx context.Context, key string, limit int, window time.Duration) (bool, *RateLimitInfo, error) {
	if limit <= 0 {
		limit = r.defaultLimit
	}
	if window <= 0 {
		window = r.defaultTTL
	}

	redisKey := r.keyPrefix + key
	now := time.Now().Unix()

	script := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		local tokens_key = key .. ":tokens"
		local timestamp_key = key .. ":timestamp"
		
		local tokens = tonumber(redis.call('GET', tokens_key))
		local last_update = tonumber(redis.call('GET', timestamp_key))
		
		if tokens == nil then
			tokens = limit
			last_update = now
		end
		
		local elapsed = now - last_update
		local refill_rate = limit / window
		local new_tokens = math.min(limit, tokens + (elapsed * refill_rate))
		
		local allowed = 0
		if new_tokens >= 1 then
			new_tokens = new_tokens - 1
			allowed = 1
		end
		
		redis.call('SET', tokens_key, new_tokens, 'EX', window * 2)
		redis.call('SET', timestamp_key, now, 'EX', window * 2)
		
		return {allowed, math.floor(new_tokens), last_update + window}
	`)

	result, err := script.Run(ctx, r.client, []string{redisKey}, limit, int(window.Seconds()), now).Result()
	if err != nil {
		return false, nil, fmt.Errorf("redis script execution failed: %w", err)
	}

	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) < 3 {
		return false, nil, fmt.Errorf("invalid script result")
	}

	allowed, _ := resultSlice[0].(int64)
	remaining, _ := resultSlice[1].(int64)
	resetTime, _ := resultSlice[2].(int64)

	info := &RateLimitInfo{
		Limit:     limit,
		Remaining: int(remaining),
		ResetTime: time.Unix(resetTime, 0),
	}

	return allowed == 1, info, nil
}

// Reset 특정 키의 Rate Limit 초기화
func (r *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	redisKey := r.keyPrefix + key
	tokensKey := redisKey + ":tokens"
	timestampKey := redisKey + ":timestamp"

	pipe := r.client.Pipeline()
	pipe.Del(ctx, tokensKey)
	pipe.Del(ctx, timestampKey)
	_, err := pipe.Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}

	return nil
}

// Ping Redis 연결 확인
func (r *RedisRateLimiter) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close Redis 연결 종료
func (r *RedisRateLimiter) Close() error {
	return r.client.Close()
}

// GetClient Redis 클라이언트 반환 (다른 패키지에서 재사용)
func (r *RedisRateLimiter) GetClient() *redis.Client {
	return r.client
}

// RateLimitInfo Rate Limit 상세 정보
type RateLimitInfo struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	ResetTime time.Time `json:"reset_time"`
}
