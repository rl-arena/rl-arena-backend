package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
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

	s.logger.Info("Starting MatchmakingService", zap.Duration("interval", s.interval))

	s.wg.Add(1)
	go s.matchmakingLoop()
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
	close(s.stopChan)
	s.wg.Wait()
	s.logger.Info("MatchmakingService stopped")
}

// matchmakingLoop 주기적 매칭 실행
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

	return s.matchmakingRepo.EnqueueAgent(
		agentID,
		environmentID,
		agent.ELO,
		5,
	)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
