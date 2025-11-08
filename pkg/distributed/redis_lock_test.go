package distributed

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRedisClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 테스트용 DB
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available:", err)
	}

	// 테스트 전 DB 초기화
	client.FlushDB(ctx)

	return client
}

func TestRedisLock_AcquireAndRelease(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	manager := NewRedisLockManager(client)
	ctx := context.Background()

	// Lock 획득
	lock, err := manager.AcquireLock(ctx, "test:lock", "instance1", 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, lock)

	// 동일한 키로 다시 Lock 획득 시도 (실패해야 함)
	lock2, err := manager.AcquireLock(ctx, "test:lock", "instance2", 5*time.Second)
	assert.Error(t, err)
	assert.Equal(t, ErrLockNotAcquired, err)
	assert.Nil(t, lock2)

	// Lock 해제
	err = lock.Release(ctx)
	assert.NoError(t, err)

	// 해제 후 다시 획득 가능
	lock3, err := manager.AcquireLock(ctx, "test:lock", "instance3", 5*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, lock3)
	defer lock3.Release(ctx)
}

func TestRedisLock_AutoExpire(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	manager := NewRedisLockManager(client)
	ctx := context.Background()

	// 1초 TTL로 Lock 획득
	lock, err := manager.AcquireLock(ctx, "test:expire", "instance1", 1*time.Second)
	require.NoError(t, err)
	require.NotNil(t, lock)

	// 즉시는 Lock 유지
	held, err := lock.IsHeld(ctx)
	assert.NoError(t, err)
	assert.True(t, held)

	// 1.5초 대기 (TTL 만료)
	time.Sleep(1500 * time.Millisecond)

	// Lock 자동 만료 확인
	held, err = lock.IsHeld(ctx)
	assert.NoError(t, err)
	assert.False(t, held)

	// 새로운 인스턴스가 Lock 획득 가능
	lock2, err := manager.AcquireLock(ctx, "test:expire", "instance2", 5*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, lock2)
	defer lock2.Release(ctx)
}

func TestRedisLock_ExtendTTL(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	manager := NewRedisLockManager(client)
	ctx := context.Background()

	// 2초 TTL로 Lock 획득
	lock, err := manager.AcquireLock(ctx, "test:extend", "instance1", 2*time.Second)
	require.NoError(t, err)
	require.NotNil(t, lock)
	defer lock.Release(ctx)

	// 1초 대기
	time.Sleep(1 * time.Second)

	// TTL 10초로 연장
	err = lock.Extend(ctx, 10*time.Second)
	assert.NoError(t, err)

	// 2초 더 대기 (원래 TTL이면 만료됨)
	time.Sleep(2 * time.Second)

	// 여전히 Lock 유지
	held, err := lock.IsHeld(ctx)
	assert.NoError(t, err)
	assert.True(t, held)
}

func TestRedisLock_TryLockWithRetry(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	manager := NewRedisLockManager(client)
	ctx := context.Background()

	// 먼저 Lock 획득
	lock1, err := manager.AcquireLock(ctx, "test:retry", "instance1", 500*time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, lock1)

	// 다른 고루틴에서 500ms 후 Lock 해제
	go func() {
		time.Sleep(500 * time.Millisecond)
		lock1.Release(context.Background())
	}()

	// Retry로 Lock 획득 시도 (3번 시도, 300ms 간격)
	start := time.Now()
	lock2, err := manager.TryLockWithRetry(
		ctx,
		"test:retry",
		"instance2",
		5*time.Second,
		3,                       // 최대 3번 시도
		300*time.Millisecond,    // 재시도 간격
	)
	elapsed := time.Since(start)

	// Lock 획득 성공 (500ms 후 해제되었으므로)
	assert.NoError(t, err)
	assert.NotNil(t, lock2)
	
	// 최소 500ms 이상 걸렸어야 함
	assert.Greater(t, elapsed, 400*time.Millisecond)
	
	defer lock2.Release(ctx)
}

func TestRedisLock_SafeRelease(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	manager := NewRedisLockManager(client)
	ctx := context.Background()

	// Instance1이 Lock 획득
	lock1, err := manager.AcquireLock(ctx, "test:safe", "instance1", 1*time.Second)
	require.NoError(t, err)

	// Lock 만료 대기
	time.Sleep(1100 * time.Millisecond)

	// Instance2가 Lock 획득
	lock2, err := manager.AcquireLock(ctx, "test:safe", "instance2", 5*time.Second)
	require.NoError(t, err)
	defer lock2.Release(ctx)

	// Instance1이 Release 시도 (다른 인스턴스 Lock이므로 실패)
	err = lock1.Release(ctx)
	assert.Error(t, err)
	assert.Equal(t, ErrLockNotHeld, err)

	// Instance2의 Lock은 여전히 유효
	held, err := lock2.IsHeld(ctx)
	assert.NoError(t, err)
	assert.True(t, held)
}

func TestRedisLock_ConcurrentAcquire(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	manager := NewRedisLockManager(client)
	
	const numGoroutines = 10
	successChan := make(chan string, numGoroutines)
	
	// 10개의 고루틴이 동시에 Lock 획득 시도
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			ctx := context.Background()
			instanceID := fmt.Sprintf("instance%d", id)
			
			lock, err := manager.AcquireLock(ctx, "test:concurrent", instanceID, 2*time.Second)
			if err == nil {
				successChan <- instanceID
				time.Sleep(100 * time.Millisecond) // 약간 대기
				lock.Release(ctx)
			}
		}(i)
	}

	// 결과 수집
	time.Sleep(3 * time.Second)
	close(successChan)

	winners := []string{}
	for winner := range successChan {
		winners = append(winners, winner)
	}

	// 정확히 1개 인스턴스만 Lock을 획득해야 함
	assert.Equal(t, 1, len(winners), "Only one instance should acquire the lock")
	t.Logf("Winner: %s", winners[0])
}

func BenchmarkRedisLock_AcquireRelease(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skip("Redis not available")
	}
	defer client.Close()

	manager := NewRedisLockManager(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lockKey := fmt.Sprintf("bench:lock:%d", i)
		lock, err := manager.AcquireLock(ctx, lockKey, "bench", 5*time.Second)
		if err != nil {
			b.Fatal(err)
		}
		lock.Release(ctx)
	}
}
