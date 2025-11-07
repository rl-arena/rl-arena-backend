package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// MatchmakingEvent 매칭 이벤트
type MatchmakingEvent struct {
	Type          string    `json:"type"`           // "agent_enqueued", "matching_requested"
	EnvironmentID string    `json:"environment_id"`
	AgentID       string    `json:"agent_id,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// MatchmakingCoordinator Redis Pub/Sub 기반 분산 매칭 조정자
type MatchmakingCoordinator struct {
	client      *redis.Client
	lockManager *RedisLockManager
	logger      *zap.Logger
	instanceID  string // 인스턴스 고유 ID
	
	// Channels
	eventChannel    string
	stopChan        chan struct{}
	subscriptionCtx context.Context
	cancelSub       context.CancelFunc
}

// NewMatchmakingCoordinator 분산 매칭 조정자 생성
func NewMatchmakingCoordinator(client *redis.Client, logger *zap.Logger) *MatchmakingCoordinator {
	return &MatchmakingCoordinator{
		client:       client,
		lockManager:  NewRedisLockManager(client),
		logger:       logger,
		instanceID:   uuid.New().String(),
		eventChannel: "matchmaking:events",
		stopChan:     make(chan struct{}),
	}
}

// Start 이벤트 수신 시작
func (c *MatchmakingCoordinator) Start(ctx context.Context, handler func(event MatchmakingEvent) error) error {
	c.subscriptionCtx, c.cancelSub = context.WithCancel(ctx)

	// Redis Pub/Sub 구독
	pubsub := c.client.Subscribe(c.subscriptionCtx, c.eventChannel)
	defer pubsub.Close()

	// 구독 확인
	_, err := pubsub.Receive(c.subscriptionCtx)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	c.logger.Info("Matchmaking coordinator started",
		zap.String("instance_id", c.instanceID),
		zap.String("channel", c.eventChannel))

	// 메시지 수신 루프
	ch := pubsub.Channel()
	for {
		select {
		case msg := <-ch:
			if msg == nil {
				continue
			}

			var event MatchmakingEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				c.logger.Error("Failed to unmarshal event", zap.Error(err))
				continue
			}

			c.logger.Debug("Received matchmaking event",
				zap.String("type", event.Type),
				zap.String("environment", event.EnvironmentID))

			// 이벤트 처리 (분산 락 사용)
			if err := c.handleEventWithLock(event, handler); err != nil {
				c.logger.Error("Failed to handle event", zap.Error(err))
			}

		case <-c.stopChan:
			c.logger.Info("Matchmaking coordinator stopped")
			return nil

		case <-c.subscriptionCtx.Done():
			return c.subscriptionCtx.Err()
		}
	}
}

// Stop 이벤트 수신 중지
func (c *MatchmakingCoordinator) Stop() {
	close(c.stopChan)
	if c.cancelSub != nil {
		c.cancelSub()
	}
}

// PublishEvent 매칭 이벤트 발행
func (c *MatchmakingCoordinator) PublishEvent(ctx context.Context, event MatchmakingEvent) error {
	event.Timestamp = time.Now()
	
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := c.client.Publish(ctx, c.eventChannel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	c.logger.Debug("Published matchmaking event",
		zap.String("type", event.Type),
		zap.String("environment", event.EnvironmentID))

	return nil
}

// handleEventWithLock 분산 락을 사용한 이벤트 처리
func (c *MatchmakingCoordinator) handleEventWithLock(event MatchmakingEvent, handler func(event MatchmakingEvent) error) error {
	// 환경별 분산 락 획득 (동시에 하나의 인스턴스만 매칭 처리)
	lockKey := fmt.Sprintf("matchmaking:lock:%s", event.EnvironmentID)
	lockValue := c.instanceID

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	lock, err := c.lockManager.TryLockWithRetry(
		ctx,
		lockKey,
		lockValue,
		5*time.Second,  // Lock TTL
		3,              // Max retries
		500*time.Millisecond, // Retry interval
	)

	if err == ErrLockNotAcquired {
		// 다른 인스턴스가 이미 처리 중
		c.logger.Debug("Lock already acquired by another instance",
			zap.String("environment", event.EnvironmentID))
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	defer func() {
		if err := lock.Release(context.Background()); err != nil {
			c.logger.Error("Failed to release lock", zap.Error(err))
		}
	}()

	c.logger.Debug("Lock acquired, processing event",
		zap.String("instance_id", c.instanceID),
		zap.String("environment", event.EnvironmentID))

	// 실제 매칭 로직 실행
	return handler(event)
}

// NotifyAgentEnqueued Agent가 큐에 추가됨을 알림
func (c *MatchmakingCoordinator) NotifyAgentEnqueued(ctx context.Context, environmentID, agentID string) error {
	return c.PublishEvent(ctx, MatchmakingEvent{
		Type:          "agent_enqueued",
		EnvironmentID: environmentID,
		AgentID:       agentID,
	})
}

// NotifyMatchingRequested 매칭 요청 알림 (주기적 트리거용)
func (c *MatchmakingCoordinator) NotifyMatchingRequested(ctx context.Context, environmentID string) error {
	return c.PublishEvent(ctx, MatchmakingEvent{
		Type:          "matching_requested",
		EnvironmentID: environmentID,
	})
}
