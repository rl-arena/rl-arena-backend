package distributed

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrLockNotAcquired = errors.New("lock not acquired")
	ErrLockNotHeld     = errors.New("lock not held")
)

// RedisLock Redis 기반 분산 락
type RedisLock struct {
	client *redis.Client
	key    string
	value  string
	ttl    time.Duration
}

// RedisLockManager Redis 분산 락 관리자
type RedisLockManager struct {
	client *redis.Client
}

// NewRedisLockManager Redis Lock Manager 생성
func NewRedisLockManager(client *redis.Client) *RedisLockManager {
	return &RedisLockManager{
		client: client,
	}
}

// AcquireLock 분산 락 획득 시도
func (m *RedisLockManager) AcquireLock(ctx context.Context, key, value string, ttl time.Duration) (*RedisLock, error) {
	// SET NX (Not Exists) 명령으로 원자적 락 획득
	success, err := m.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, ErrLockNotAcquired
	}

	return &RedisLock{
		client: m.client,
		key:    key,
		value:  value,
		ttl:    ttl,
	}, nil
}

// TryLockWithRetry 재시도를 통한 락 획득
func (m *RedisLockManager) TryLockWithRetry(
	ctx context.Context,
	key, value string,
	ttl time.Duration,
	maxRetries int,
	retryInterval time.Duration,
) (*RedisLock, error) {
	for i := 0; i < maxRetries; i++ {
		lock, err := m.AcquireLock(ctx, key, value, ttl)
		if err == nil {
			return lock, nil
		}

		if err != ErrLockNotAcquired {
			return nil, err
		}

		// 재시도 전 대기
		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryInterval):
			}
		}
	}

	return nil, ErrLockNotAcquired
}

// Release 락 해제 (Lua 스크립트로 안전하게)
func (l *RedisLock) Release(ctx context.Context) error {
	// Lua 스크립트: 자신이 획득한 락만 해제
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, l.client, []string{l.key}, l.value).Int()
	if err != nil {
		return err
	}

	if result == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// Extend 락 TTL 연장
func (l *RedisLock) Extend(ctx context.Context, extension time.Duration) error {
	// Lua 스크립트: 자신이 획득한 락만 TTL 연장
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`)

	ttlMs := extension.Milliseconds()
	result, err := script.Run(ctx, l.client, []string{l.key}, l.value, ttlMs).Int()
	if err != nil {
		return err
	}

	if result == 0 {
		return ErrLockNotHeld
	}

	l.ttl = extension
	return nil
}

// IsHeld 락이 현재 유효한지 확인
func (l *RedisLock) IsHeld(ctx context.Context) (bool, error) {
	value, err := l.client.Get(ctx, l.key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return value == l.value, nil
}
