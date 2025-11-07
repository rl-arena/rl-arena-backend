package distributed

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRedisQueue(t *testing.T) (*redis.Client, *RedisQueue) {
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

	queue := NewRedisQueue(client, "test_queue", 0)
	return client, queue
}

func TestRedisQueue_EnqueueDequeue(t *testing.T) {
	client, queue := setupRedisQueue(t)
	defer client.Close()

	ctx := context.Background()

	// Enqueue
	item := &QueueItem{
		ID:         uuid.New().String(),
		Data:       map[string]interface{}{"agent_id": "agent1", "opponent_id": "agent2"},
		Priority:   100,
		MaxRetries: 3,
	}

	err := queue.Enqueue(ctx, item)
	require.NoError(t, err)

	// 큐 크기 확인
	size, err := queue.Size(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), size)

	// Dequeue
	dequeued, err := queue.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, item.ID, dequeued.ID)
	assert.Equal(t, item.Priority, dequeued.Priority)

	// 큐 비어있음
	size, err = queue.Size(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), size)

	// Processing에 있음
	processingCount, err := queue.ProcessingCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), processingCount)
}

func TestRedisQueue_Priority(t *testing.T) {
	client, queue := setupRedisQueue(t)
	defer client.Close()

	ctx := context.Background()

	// 다양한 우선순위로 Enqueue
	items := []*QueueItem{
		{ID: "item1", Priority: 50, MaxRetries: 3},
		{ID: "item2", Priority: 100, MaxRetries: 3}, // 가장 높은 우선순위
		{ID: "item3", Priority: 10, MaxRetries: 3},
		{ID: "item4", Priority: 75, MaxRetries: 3},
	}

	for _, item := range items {
		err := queue.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	// Dequeue 순서 확인 (우선순위 높은 순)
	expectedOrder := []string{"item2", "item4", "item1", "item3"}
	
	for _, expectedID := range expectedOrder {
		dequeued, err := queue.Dequeue(ctx)
		require.NoError(t, err)
		assert.Equal(t, expectedID, dequeued.ID, "Priority order incorrect")
	}
}

func TestRedisQueue_Complete(t *testing.T) {
	client, queue := setupRedisQueue(t)
	defer client.Close()

	ctx := context.Background()

	// Enqueue & Dequeue
	item := &QueueItem{
		ID:         uuid.New().String(),
		Priority:   100,
		MaxRetries: 3,
	}

	err := queue.Enqueue(ctx, item)
	require.NoError(t, err)

	dequeued, err := queue.Dequeue(ctx)
	require.NoError(t, err)

	// Processing에 있음
	processingCount, err := queue.ProcessingCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), processingCount)

	// Complete
	err = queue.Complete(ctx, dequeued.ID)
	assert.NoError(t, err)

	// Processing에서 제거됨
	processingCount, err = queue.ProcessingCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), processingCount)
}

func TestRedisQueue_Retry(t *testing.T) {
	client, queue := setupRedisQueue(t)
	defer client.Close()

	ctx := context.Background()

	// Enqueue & Dequeue
	item := &QueueItem{
		ID:         uuid.New().String(),
		Priority:   100,
		MaxRetries: 3,
	}

	err := queue.Enqueue(ctx, item)
	require.NoError(t, err)

	dequeued, err := queue.Dequeue(ctx)
	require.NoError(t, err)

	// Retry
	err = queue.Retry(ctx, dequeued)
	assert.NoError(t, err)

	// 다시 큐에 추가됨 (우선순위 낮아짐)
	size, err := queue.Size(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), size)

	// Dequeue 다시
	retried, err := queue.Dequeue(ctx)
	assert.NoError(t, err)
	assert.Equal(t, item.ID, retried.ID)
	assert.Equal(t, 1, retried.Retries)
	assert.Equal(t, item.Priority-10, retried.Priority) // 우선순위 낮아짐
}

func TestRedisQueue_MaxRetries_MoveToDLQ(t *testing.T) {
	client, queue := setupRedisQueue(t)
	defer client.Close()

	ctx := context.Background()

	// Enqueue with MaxRetries=2
	item := &QueueItem{
		ID:         uuid.New().String(),
		Priority:   100,
		MaxRetries: 2,
	}

	err := queue.Enqueue(ctx, item)
	require.NoError(t, err)

	// 첫 번째 시도
	dequeued, err := queue.Dequeue(ctx)
	require.NoError(t, err)

	// 첫 번째 재시도 (Retries=1)
	err = queue.Retry(ctx, dequeued)
	assert.NoError(t, err)

	retried1, err := queue.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, retried1.Retries)

	// 두 번째 재시도 (Retries=2, MaxRetries 도달 → DLQ로 이동)
	err = queue.Retry(ctx, retried1)
	assert.NoError(t, err)

	// 큐 비어있음
	size, err := queue.Size(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), size)

	// DLQ에 추가됨
	dlqSize, err := queue.DLQSize(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), dlqSize)

	// DLQ 확인
	dlqItems, err := queue.PeekDLQ(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, dlqItems, 1)
	assert.Equal(t, "max retries exceeded", dlqItems[0]["reason"])
}

func TestRedisQueue_RecoverStale(t *testing.T) {
	client, queue := setupRedisQueue(t)
	defer client.Close()

	ctx := context.Background()

	// Enqueue & Dequeue (처리 시작)
	item := &QueueItem{
		ID:         uuid.New().String(),
		Priority:   100,
		MaxRetries: 3,
	}

	err := queue.Enqueue(ctx, item)
	require.NoError(t, err)

	dequeued, err := queue.Dequeue(ctx)
	require.NoError(t, err)

	// Processing에 있음
	processingCount, err := queue.ProcessingCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), processingCount)

	// Stale 복구 (1초 타임아웃)
	time.Sleep(1100 * time.Millisecond)

	recovered, err := queue.RecoverStale(ctx, 1*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 1, recovered)

	// 다시 큐에 추가됨
	size, err := queue.Size(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), size)

	// Processing에서 제거됨
	processingCount, err = queue.ProcessingCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), processingCount)
}

func TestRedisQueue_MaxSize(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	client.FlushDB(ctx)

	// 최대 크기 3인 큐
	queue := NewRedisQueue(client, "limited_queue", 3)

	// 3개까지 추가 가능
	for i := 0; i < 3; i++ {
		item := &QueueItem{
			ID:         fmt.Sprintf("item%d", i),
			Priority:   100,
			MaxRetries: 3,
		}
		err := queue.Enqueue(ctx, item)
		assert.NoError(t, err)
	}

	// 4번째 추가 시 ErrQueueFull
	item := &QueueItem{
		ID:         "item4",
		Priority:   100,
		MaxRetries: 3,
	}
	err := queue.Enqueue(ctx, item)
	assert.Error(t, err)
	assert.Equal(t, ErrQueueFull, err)
}

func TestRedisQueue_Stats(t *testing.T) {
	client, queue := setupRedisQueue(t)
	defer client.Close()

	ctx := context.Background()

	// 초기 통계
	stats, err := queue.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.QueueSize)
	assert.Equal(t, int64(0), stats.ProcessingCount)
	assert.Equal(t, int64(0), stats.DLQSize)

	// Enqueue 2개
	for i := 0; i < 2; i++ {
		item := &QueueItem{
			ID:         fmt.Sprintf("item%d", i),
			Priority:   100,
			MaxRetries: 3,
		}
		err := queue.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	// Dequeue 1개 (Processing으로 이동)
	_, err = queue.Dequeue(ctx)
	require.NoError(t, err)

	// DLQ에 1개 추가
	dlqItem := &QueueItem{
		ID:         "dlq_item",
		Priority:   100,
		MaxRetries: 0,
		Retries:    3,
	}
	err = queue.MoveToDLQ(ctx, dlqItem, "test")
	require.NoError(t, err)

	// 통계 확인
	stats, err = queue.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.QueueSize)       // 1개 큐 대기
	assert.Equal(t, int64(1), stats.ProcessingCount) // 1개 처리 중
	assert.Equal(t, int64(1), stats.DLQSize)         // 1개 DLQ
}

func BenchmarkRedisQueue_EnqueueDequeue(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skip("Redis not available")
	}
	defer client.Close()

	queue := NewRedisQueue(client, "bench_queue", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := &QueueItem{
			ID:         uuid.New().String(),
			Priority:   100,
			MaxRetries: 3,
		}

		if err := queue.Enqueue(ctx, item); err != nil {
			b.Fatal(err)
		}

		if _, err := queue.Dequeue(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
