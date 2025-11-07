package distributed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrQueueEmpty = errors.New("queue is empty")
	ErrQueueFull  = errors.New("queue is full")
)

// QueueItem Redis Queue의 아이템
type QueueItem struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	Priority  int                    `json:"priority"`  // 높을수록 우선순위 높음
	Retries   int                    `json:"retries"`   // 재시도 횟수
	MaxRetries int                   `json:"max_retries"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// RedisQueue Redis 기반 우선순위 큐
type RedisQueue struct {
	client       *redis.Client
	queueKey     string // 메인 큐 (Sorted Set)
	processingKey string // 처리 중 아이템 (Hash)
	dlqKey       string // Dead Letter Queue (List)
	maxSize      int    // 최대 큐 크기 (0 = 무제한)
}

// NewRedisQueue Redis Queue 생성
func NewRedisQueue(client *redis.Client, queueName string, maxSize int) *RedisQueue {
	return &RedisQueue{
		client:        client,
		queueKey:      fmt.Sprintf("queue:%s", queueName),
		processingKey: fmt.Sprintf("queue:%s:processing", queueName),
		dlqKey:        fmt.Sprintf("queue:%s:dlq", queueName),
		maxSize:       maxSize,
	}
}

// Enqueue 큐에 아이템 추가 (우선순위 기반)
func (q *RedisQueue) Enqueue(ctx context.Context, item *QueueItem) error {
	// 큐 크기 제한 확인
	if q.maxSize > 0 {
		size, err := q.client.ZCard(ctx, q.queueKey).Result()
		if err != nil {
			return fmt.Errorf("failed to get queue size: %w", err)
		}
		
		if int(size) >= q.maxSize {
			return ErrQueueFull
		}
	}

	// 타임스탬프 설정
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now

	// 아이템 직렬화
	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	// Sorted Set에 추가 (priority를 score로 사용, 높을수록 먼저 처리)
	// 음수로 저장하여 높은 priority가 낮은 score를 가지도록 (ZPOPMIN 사용 위해)
	score := float64(-item.Priority)
	
	if err := q.client.ZAdd(ctx, q.queueKey, redis.Z{
		Score:  score,
		Member: data,
	}).Err(); err != nil {
		return fmt.Errorf("failed to enqueue: %w", err)
	}

	return nil
}

// Dequeue 큐에서 아이템 가져오기 (우선순위 높은 것부터)
func (q *RedisQueue) Dequeue(ctx context.Context) (*QueueItem, error) {
	// Lua 스크립트로 원자적 Pop + Processing Set 추가
	script := redis.NewScript(`
		local queue_key = KEYS[1]
		local processing_key = KEYS[2]
		local timestamp = ARGV[1]
		
		-- Pop highest priority item (lowest score)
		local items = redis.call('ZPOPMIN', queue_key, 1)
		if #items == 0 then
			return nil
		end
		
		local item_data = items[1]
		local item_id = cjson.decode(item_data).id
		
		-- Add to processing set with timestamp
		redis.call('HSET', processing_key, item_id, item_data)
		redis.call('HSET', processing_key, item_id .. ':timestamp', timestamp)
		
		return item_data
	`)

	result, err := script.Run(ctx, q.client, []string{q.queueKey, q.processingKey}, time.Now().Unix()).Result()
	if err == redis.Nil || result == nil {
		return nil, ErrQueueEmpty
	}
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue: %w", err)
	}

	// 역직렬화
	var item QueueItem
	if err := json.Unmarshal([]byte(result.(string)), &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal item: %w", err)
	}

	return &item, nil
}

// Complete 아이템 처리 완료 (processing에서 제거)
func (q *RedisQueue) Complete(ctx context.Context, itemID string) error {
	pipe := q.client.Pipeline()
	pipe.HDel(ctx, q.processingKey, itemID)
	pipe.HDel(ctx, q.processingKey, itemID+":timestamp")
	_, err := pipe.Exec(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to complete item: %w", err)
	}

	return nil
}

// Retry 아이템 재시도 (다시 큐에 추가)
func (q *RedisQueue) Retry(ctx context.Context, item *QueueItem) error {
	// 재시도 횟수 증가
	item.Retries++
	item.UpdatedAt = time.Now()

	// 최대 재시도 초과 시 DLQ로 이동
	if item.Retries >= item.MaxRetries {
		return q.MoveToDLQ(ctx, item, "max retries exceeded")
	}

	// Processing에서 제거
	if err := q.Complete(ctx, item.ID); err != nil {
		return err
	}

	// 다시 큐에 추가 (우선순위 낮춤)
	item.Priority = item.Priority - 10 // 재시도 시 우선순위 낮춤
	
	return q.Enqueue(ctx, item)
}

// MoveToDLQ Dead Letter Queue로 이동
func (q *RedisQueue) MoveToDLQ(ctx context.Context, item *QueueItem, reason string) error {
	// DLQ 아이템 구조
	dlqItem := map[string]interface{}{
		"item":       item,
		"reason":     reason,
		"moved_at":   time.Now(),
		"final_retry": item.Retries,
	}

	data, err := json.Marshal(dlqItem)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ item: %w", err)
	}

	// DLQ에 추가 (List)
	if err := q.client.LPush(ctx, q.dlqKey, data).Err(); err != nil {
		return fmt.Errorf("failed to move to DLQ: %w", err)
	}

	// Processing에서 제거
	if err := q.Complete(ctx, item.ID); err != nil {
		return err
	}

	return nil
}

// RecoverStale 일정 시간 이상 처리 중인 아이템 복구
func (q *RedisQueue) RecoverStale(ctx context.Context, staleTimeout time.Duration) (int, error) {
	// Processing에서 모든 아이템 조회
	items, err := q.client.HGetAll(ctx, q.processingKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get processing items: %w", err)
	}

	recovered := 0
	now := time.Now().Unix()

	for key, value := range items {
		// Timestamp 키는 스킵
		if len(key) > 10 && key[len(key)-10:] == ":timestamp" {
			continue
		}

		// Timestamp 확인
		timestampKey := key + ":timestamp"
		timestampStr, exists := items[timestampKey]
		if !exists {
			continue
		}

		var timestamp int64
		if _, err := fmt.Sscanf(timestampStr, "%d", &timestamp); err != nil {
			continue
		}

		// Stale 체크
		if now-timestamp > int64(staleTimeout.Seconds()) {
			// 아이템 복구
			var item QueueItem
			if err := json.Unmarshal([]byte(value), &item); err != nil {
				continue
			}

			// 재시도
			if err := q.Retry(ctx, &item); err != nil {
				continue
			}

			recovered++
		}
	}

	return recovered, nil
}

// Size 큐 크기 조회
func (q *RedisQueue) Size(ctx context.Context) (int64, error) {
	return q.client.ZCard(ctx, q.queueKey).Result()
}

// ProcessingCount 처리 중 아이템 개수
func (q *RedisQueue) ProcessingCount(ctx context.Context) (int64, error) {
	count, err := q.client.HLen(ctx, q.processingKey).Result()
	if err != nil {
		return 0, err
	}
	// Timestamp 키 제외 (실제 아이템 수의 2배)
	return count / 2, nil
}

// DLQSize DLQ 크기
func (q *RedisQueue) DLQSize(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.dlqKey).Result()
}

// PeekDLQ DLQ에서 아이템 확인 (제거하지 않음)
func (q *RedisQueue) PeekDLQ(ctx context.Context, count int64) ([]map[string]interface{}, error) {
	items, err := q.client.LRange(ctx, q.dlqKey, 0, count-1).Result()
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		var dlqItem map[string]interface{}
		if err := json.Unmarshal([]byte(item), &dlqItem); err != nil {
			continue
		}
		result = append(result, dlqItem)
	}

	return result, nil
}

// ClearDLQ DLQ 비우기
func (q *RedisQueue) ClearDLQ(ctx context.Context) error {
	return q.client.Del(ctx, q.dlqKey).Err()
}

// Stats 큐 통계
type QueueStats struct {
	QueueSize      int64 `json:"queue_size"`
	ProcessingCount int64 `json:"processing_count"`
	DLQSize        int64 `json:"dlq_size"`
}

// GetStats 큐 통계 조회
func (q *RedisQueue) GetStats(ctx context.Context) (*QueueStats, error) {
	queueSize, err := q.Size(ctx)
	if err != nil {
		return nil, err
	}

	processingCount, err := q.ProcessingCount(ctx)
	if err != nil {
		return nil, err
	}

	dlqSize, err := q.DLQSize(ctx)
	if err != nil {
		return nil, err
	}

	return &QueueStats{
		QueueSize:      queueSize,
		ProcessingCount: processingCount,
		DLQSize:        dlqSize,
	}, nil
}
