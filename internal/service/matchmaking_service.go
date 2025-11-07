package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
	"github.com/rl-arena/rl-arena-backend/pkg/distributed"
	"go.uber.org/zap"
)

type MatchmakingService struct {
	matchmakingRepo *repository.MatchmakingRepository
	agentRepo       *repository.AgentRepository
	matchService    *MatchService
	logger          *zap.Logger
	interval        time.Duration
	eloRange        int
	maxEloRange     int
	stopChan        chan struct{}
	wg              sync.WaitGroup
	running         bool
	mu              sync.Mutex
	
	// Redis 분산 처리
	coordinator     *distributed.MatchmakingCoordinator
	useDistributed  bool
}

func NewMatchmakingService(
	matchmakingRepo *repository.MatchmakingRepository,
	agentRepo *repository.AgentRepository,
	matchService *MatchService,
	interval time.Duration,
) *MatchmakingService {
	logger, _ := zap.NewProduction()
	
	return &MatchmakingService{
		matchmakingRepo: matchmakingRepo,
		agentRepo:       agentRepo,
		matchService:    matchService,
		logger:          logger,
		interval:        interval,
		eloRange:        100,
		maxEloRange:     500,
		stopChan:        make(chan struct{}),
		useDistributed:  false, // 기본값은 단일 고루틴 모드
	}
}

// NewMatchmakingServiceWithRedis Redis 분산 처리를 사용하는 Matchmaking Service 생성
func NewMatchmakingServiceWithRedis(
	matchmakingRepo *repository.MatchmakingRepository,
	agentRepo *repository.AgentRepository,
	matchService *MatchService,
	interval time.Duration,
	redisClient *redis.Client,
) *MatchmakingService {
	logger, _ := zap.NewProduction()
	
	coordinator := distributed.NewMatchmakingCoordinator(redisClient, logger)
	
	return &MatchmakingService{
		matchmakingRepo: matchmakingRepo,
		agentRepo:       agentRepo,
		matchService:    matchService,
		logger:          logger,
		interval:        interval,
		eloRange:        100,
		maxEloRange:     500,
		stopChan:        make(chan struct{}),
		coordinator:     coordinator,
		useDistributed:  true, // 분산 모드 활성화
	}
}

// Start 매칭 시스템 시작
func (s *MatchmakingService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	if s.useDistributed {
		s.logger.Info("Starting MatchmakingService with Redis Pub/Sub",
			zap.Duration("interval", s.interval))
		
		// Redis Pub/Sub 기반 분산 처리
		s.wg.Add(2)
		go s.distributedMatchmakingLoop()
		go s.periodicMatchingTrigger()
	} else {
		s.logger.Info("Starting MatchmakingService (single-instance mode)",
			zap.Duration("interval", s.interval))
		
		// 기존 단일 고루틴 모드
		s.wg.Add(1)
		go s.matchmakingLoop()
	}
}

// Stop 매칭 시스템 중지
func (s *MatchmakingService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	s.logger.Info("Stopping MatchmakingService")
	
	if s.useDistributed && s.coordinator != nil {
		s.coordinator.Stop()
	}
	
	close(s.stopChan)
	s.wg.Wait()
	s.logger.Info("MatchmakingService stopped")
}

// matchmakingLoop 주기적 매칭 실행 (단일 인스턴스 모드)
func (s *MatchmakingService) matchmakingLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// 시작 시 한번 실행
	s.runMatchmaking()

	for {
		select {
		case <-ticker.C:
			s.runMatchmaking()
		case <-s.stopChan:
			return
		}
	}
}

// distributedMatchmakingLoop Redis Pub/Sub 기반 분산 매칭 루프
func (s *MatchmakingService) distributedMatchmakingLoop() {
	defer s.wg.Done()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 매칭 이벤트 핸들러
	handler := func(event distributed.MatchmakingEvent) error {
		switch event.Type {
		case "agent_enqueued", "matching_requested":
			// 해당 환경에서 매칭 시도
			s.matchEnvironment(ctx, event.EnvironmentID)
		default:
			s.logger.Warn("Unknown event type", zap.String("type", event.Type))
		}
		return nil
	}

	// Redis Pub/Sub 수신 시작
	if err := s.coordinator.Start(ctx, handler); err != nil {
		s.logger.Error("Coordinator stopped with error", zap.Error(err))
	}
}

// periodicMatchingTrigger 주기적으로 매칭 이벤트 발행
func (s *MatchmakingService) periodicMatchingTrigger() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	ctx := context.Background()

	// 시작 시 한번 실행
	s.triggerMatching(ctx)

	for {
		select {
		case <-ticker.C:
			s.triggerMatching(ctx)
		case <-s.stopChan:
			return
		}
	}
}

// triggerMatching 모든 환경에 대해 매칭 이벤트 발행
func (s *MatchmakingService) triggerMatching(ctx context.Context) {
	// 만료된 큐 정리
	if err := s.matchmakingRepo.CleanupExpired(24 * time.Hour); err != nil {
		s.logger.Error("Failed to cleanup expired queue", zap.Error(err))
	}

	// 각 환경별로 매칭 요청 이벤트 발행
	environments := []string{"pong"} // TODO: DB에서 활성 환경 목록 가져오기
	
	for _, env := range environments {
		if err := s.coordinator.NotifyMatchingRequested(ctx, env); err != nil {
			s.logger.Error("Failed to notify matching requested",
				zap.String("environment", env),
				zap.Error(err))
		}
	}
}

// runMatchmaking 매칭 실행
func (s *MatchmakingService) runMatchmaking() {
	ctx := context.Background()

	// 1. 만료된 큐 정리 (24시간)
	if err := s.matchmakingRepo.CleanupExpired(24 * time.Hour); err != nil {
		s.logger.Error("Failed to cleanup expired queue", zap.Error(err))
	}

	// 2. 각 환경별로 매칭 (일단 pong만)
	s.matchEnvironment(ctx, "pong")
}

// matchEnvironment 특정 환경에서 매칭
func (s *MatchmakingService) matchEnvironment(ctx context.Context, environmentName string) {
	waiting, err := s.matchmakingRepo.GetWaitingAgents(environmentName)
	if err != nil {
		s.logger.Error("Failed to get waiting agents", zap.Error(err))
		return
	}

	if len(waiting) < 2 {
		if len(waiting) > 0 {
			s.logger.Debug("Not enough agents for matching", 
				zap.String("env", environmentName),
				zap.Int("count", len(waiting)))
		}
		return
	}

	s.logger.Info("Starting matchmaking", 
		zap.String("env", environmentName),
		zap.Int("waiting", len(waiting)))

	matched := 0
	processed := make(map[string]bool)
	
	for _, queue := range waiting {
		if processed[queue.AgentID] {
			continue
		}

		opponent := s.findOpponentWithExpandingRange(queue, environmentName, processed)
		
		if opponent == nil {
			continue
		}

		if err := s.createMatch(ctx, &queue, opponent, environmentName); err != nil {
			s.logger.Error("Failed to create match",
				zap.String("agent1", queue.AgentID),
				zap.String("agent2", opponent.AgentID),
				zap.Error(err))
			continue
		}

		processed[queue.AgentID] = true
		processed[opponent.AgentID] = true
		matched += 2
	}

	if matched > 0 {
		s.logger.Info("Matchmaking completed",
			zap.String("env", environmentName),
			zap.Int("matched", matched/2),
			zap.Int("matches_created", matched/2))
	}
}

// findOpponentWithExpandingRange ELO 범위를 넓혀가며 상대 찾기
func (s *MatchmakingService) findOpponentWithExpandingRange(
	queue models.MatchmakingQueue, 
	environmentID string,
	processed map[string]bool,
) *models.MatchmakingQueue {
	for eloRange := s.eloRange; eloRange <= s.maxEloRange; eloRange += 100 {
		opponent, err := s.matchmakingRepo.FindOpponent(
			queue.AgentID,
			environmentID,
			queue.EloRating,
			eloRange,
		)
		
		if err == nil && opponent != nil && !processed[opponent.AgentID] {
			s.logger.Debug("Found opponent",
				zap.String("agent", queue.AgentID),
				zap.Int("agentElo", queue.EloRating),
				zap.String("opponent", opponent.AgentID),
				zap.Int("opponentElo", opponent.EloRating),
				zap.Int("eloRange", eloRange))
			return opponent
		}
	}
	
	return nil
}

// createMatch Match 생성 및 실행
func (s *MatchmakingService) createMatch(ctx context.Context, queue1, queue2 *models.MatchmakingQueue, environmentID string) error {
	agent1, err := s.agentRepo.FindByID(queue1.AgentID)
	if err != nil {
		return fmt.Errorf("failed to find agent1: %w", err)
	}

	agent2, err := s.agentRepo.FindByID(queue2.AgentID)
	if err != nil {
		return fmt.Errorf("failed to find agent2: %w", err)
	}

	// Match 생성 및 실행
	match, err := s.matchService.CreateAndExecute(agent1.ID, agent2.ID)
	if err != nil {
		return fmt.Errorf("failed to create and execute match: %w", err)
	}

	// 큐에서 제거
	if err := s.matchmakingRepo.MarkAsMatched(queue1.ID, queue2.ID); err != nil {
		s.logger.Error("Failed to mark as matched", zap.Error(err))
	}

	// 매칭 기록
	eloDiff := abs(agent1.ELO - agent2.ELO)
	if err := s.matchmakingRepo.RecordMatch(
		agent1.ID,
		agent2.ID,
		environmentID,
		match.ID,
		eloDiff,
	); err != nil {
		s.logger.Error("Failed to record match", zap.Error(err))
	}

	s.logger.Info("Match created automatically",
		zap.String("matchId", match.ID),
		zap.String("agent1", agent1.Name),
		zap.String("agent2", agent2.Name),
		zap.Int("eloDiff", eloDiff))

	return nil
}

// EnqueueAgent Agent를 매칭 큐에 추가
func (s *MatchmakingService) EnqueueAgent(agentID, environmentID string) error {
	agent, err := s.agentRepo.FindByID(agentID)
	if err != nil {
		return fmt.Errorf("failed to find agent: %w", err)
	}

	if err := s.matchmakingRepo.EnqueueAgent(
		agentID,
		environmentID,
		agent.ELO,
		5,
	); err != nil {
		return err
	}

	// Redis 분산 모드인 경우 이벤트 발행
	if s.useDistributed && s.coordinator != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		
		if err := s.coordinator.NotifyAgentEnqueued(ctx, environmentID, agentID); err != nil {
			s.logger.Error("Failed to notify agent enqueued",
				zap.String("agent", agentID),
				zap.String("environment", environmentID),
				zap.Error(err))
			// 이벤트 발행 실패해도 큐 추가는 성공으로 처리
		}
	}

	return nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
