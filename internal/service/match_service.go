package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
	"github.com/rl-arena/rl-arena-backend/pkg/executor"
	"github.com/rl-arena/rl-arena-backend/pkg/logger"
)

var (
	ErrAgentNotReady        = errors.New("agent does not have active submission")
	ErrSameAgent            = errors.New("cannot match agent against itself")
	ErrDifferentEnvironment = errors.New("agents must be in the same environment")
)

type MatchService struct {
	matchRepo           *repository.MatchRepository
	agentRepo           *repository.AgentRepository
	submissionRepo      *repository.SubmissionRepository
	statsRepo           *repository.AgentMatchStatsRepository
	eloService          *ELOService
	executorClient      *executor.Client
	matchmakingService  *MatchmakingService
}

func NewMatchService(
	matchRepo *repository.MatchRepository,
	agentRepo *repository.AgentRepository,
	submissionRepo *repository.SubmissionRepository,
	statsRepo *repository.AgentMatchStatsRepository,
	eloService *ELOService,
	executorClient *executor.Client,
) *MatchService {
	return &MatchService{
		matchRepo:      matchRepo,
		agentRepo:      agentRepo,
		submissionRepo: submissionRepo,
		statsRepo:      statsRepo,
		eloService:     eloService,
		executorClient: executorClient,
	}
}

// SetMatchmakingService sets the matchmaking service (to avoid circular dependency)
func (s *MatchService) SetMatchmakingService(matchmakingService *MatchmakingService) {
	s.matchmakingService = matchmakingService
}

// CreateAndExecute 매치 생성 및 실행
func (s *MatchService) CreateAndExecute(agent1ID, agent2ID string) (*models.Match, error) {
	// 같은 에이전트 체크
	if agent1ID == agent2ID {
		return nil, ErrSameAgent
	}

	// 에이전트 조회
	agent1, err := s.agentRepo.FindByID(agent1ID)
	if err != nil || agent1 == nil {
		return nil, fmt.Errorf("agent1 not found: %w", err)
	}

	agent2, err := s.agentRepo.FindByID(agent2ID)
	if err != nil || agent2 == nil {
		return nil, fmt.Errorf("agent2 not found: %w", err)
	}

	// 같은 환경 체크
	if agent1.EnvironmentID != agent2.EnvironmentID {
		return nil, ErrDifferentEnvironment
	}

	// Active submission 확인
	if agent1.ActiveSubmissionID == nil {
		return nil, fmt.Errorf("agent1 %s: %w", agent1.Name, ErrAgentNotReady)
	}
	if agent2.ActiveSubmissionID == nil {
		return nil, fmt.Errorf("agent2 %s: %w", agent2.Name, ErrAgentNotReady)
	}

	// Submission 조회
	sub1, err := s.submissionRepo.FindByID(*agent1.ActiveSubmissionID)
	if err != nil || sub1 == nil {
		return nil, fmt.Errorf("agent1 submission not found")
	}

	sub2, err := s.submissionRepo.FindByID(*agent2.ActiveSubmissionID)
	if err != nil || sub2 == nil {
		return nil, fmt.Errorf("agent2 submission not found")
	}

	// Docker 이미지 또는 코드 URL 결정
	agent1Source, agent1IsDocker := s.getAgentSource(sub1)
	agent2Source, agent2IsDocker := s.getAgentSource(sub2)
	
	// 로컬 파일 경로면 절대 경로로 변환
	if !agent1IsDocker {
		agent1Source = s.resolveCodePath(agent1Source)
	}
	if !agent2IsDocker {
		agent2Source = s.resolveCodePath(agent2Source)
	}

	// 빌드 중인 Submission 확인
	if agent1Source == "" {
		return nil, fmt.Errorf("agent1 submission is not ready (status: %s)", sub1.Status)
	}
	if agent2Source == "" {
		return nil, fmt.Errorf("agent2 submission is not ready (status: %s)", sub2.Status)
	}

	// 매치 생성
	match, err := s.matchRepo.Create(agent1.EnvironmentID, agent1ID, agent2ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create match: %w", err)
	}

	logger.Info("Match created",
		"matchId", match.ID,
		"agent1", agent1.Name,
		"agent1Source", agent1Source,
		"agent1IsDocker", agent1IsDocker,
		"agent2", agent2.Name,
		"agent2Source", agent2Source,
		"agent2IsDocker", agent2IsDocker,
		"environment", agent1.EnvironmentID,
	)

	// Executor에 요청
	execReq := executor.ExecuteMatchRequest{
		MatchID:       match.ID,
		EnvironmentID: agent1.EnvironmentID,
		Agent1: executor.AgentInfo{
			ID:      agent1.ID,
			Name:    agent1.Name,
			CodeURL: agent1Source,
		},
		Agent2: executor.AgentInfo{
			ID:      agent2.ID,
			Name:    agent2.Name,
			CodeURL: agent2Source,
		},
	}

	// 실행 (동기 또는 비동기)
	result, err := s.executorClient.ExecuteMatch(execReq)
	if err != nil {
		logger.Error("Match execution failed",
			"matchId", match.ID,
			"error", err,
		)
		return match, fmt.Errorf("executor failed: %w", err)
	}

	// 결과 처리
	if result.Status == "success" {
		s.processMatchResult(match.ID, agent1, agent2, result)
	}

	return match, nil
}

// processMatchResult 매치 결과 처리
func (s *MatchService) processMatchResult(
	matchID string,
	agent1, agent2 *models.Agent,
	result *executor.ExecuteMatchResponse,
) {
	logger.Info("Processing match result",
		"matchId", matchID,
		"agent1", agent1.ID,
		"agent2", agent2.ID,
		"resultStatus", result.Status)
	
	// 승자 결정
	var winnerID *string
	if result.WinnerID != nil {
		winnerID = result.WinnerID
	}

	// ELO 계산
	var matchResult float64
	if winnerID == nil {
		matchResult = 0.5 // 무승부
	} else if *winnerID == agent1.ID {
		matchResult = 1.0 // agent1 승
	} else {
		matchResult = 0.0 // agent2 승
	}

	// Use provisional ELO rating system with dynamic K-factors based on match count
	_, _, agent1ELOChange, agent2ELOChange := s.eloService.CalculateNewRatingsWithMatchCounts(
		agent1.ELO,
		agent2.ELO,
		agent1.TotalMatches,
		agent2.TotalMatches,
		matchResult,
	)

	// 매치 결과 저장
	err := s.matchRepo.UpdateResult(
		matchID,
		winnerID,
		result.Agent1Score,
		result.Agent2Score,
		agent1ELOChange,
		agent2ELOChange,
		result.ReplayURL,
		result.ReplayHTMLURL,
	)
	if err != nil {
		logger.Error("Failed to update match result", "error", err)
		return
	}

	// Agent 통계 업데이트
	agent1Won := winnerID != nil && *winnerID == agent1.ID
	agent1Lost := winnerID != nil && *winnerID == agent2.ID
	agent1Draw := winnerID == nil

	s.agentRepo.UpdateStats(agent1.ID, agent1ELOChange, agent1Won, agent1Lost, agent1Draw)
	s.agentRepo.UpdateStats(agent2.ID, agent2ELOChange, !agent1Won && !agent1Draw, agent1Won, agent1Draw)

	// 매치 통계 업데이트 (rate limiting용)
	if err := s.statsRepo.IncrementMatchCount(agent1.ID); err != nil {
		logger.Error("Failed to update agent1 match stats", "agentId", agent1.ID, "error", err)
	}
	if err := s.statsRepo.IncrementMatchCount(agent2.ID); err != nil {
		logger.Error("Failed to update agent2 match stats", "agentId", agent2.ID, "error", err)
	}

	// 매치 완료 후 agent들을 다시 매칭 큐에 등록 (waiting 상태로 복귀)
	if s.matchmakingService != nil {
		logger.Info("Re-enqueueing agents after match completion",
			"matchId", matchID,
			"agent1Id", agent1.ID,
			"agent2Id", agent2.ID)
		
		// agent1 재등록
		if err := s.matchmakingService.EnqueueAgent(agent1.ID, agent1.EnvironmentID); err != nil {
			logger.Error("Failed to re-enqueue agent1 after match",
				"agentId", agent1.ID,
				"error", err)
		} else {
			logger.Info("Agent1 re-enqueued successfully", "agentId", agent1.ID)
		}
		
		// agent2 재등록
		if err := s.matchmakingService.EnqueueAgent(agent2.ID, agent2.EnvironmentID); err != nil {
			logger.Error("Failed to re-enqueue agent2 after match",
				"agentId", agent2.ID,
				"error", err)
		} else {
			logger.Info("Agent2 re-enqueued successfully", "agentId", agent2.ID)
		}
	} else {
		logger.Warn("MatchmakingService is nil, cannot re-enqueue agents after match")
	}

	logger.Info("Match result processed",
		"matchId", matchID,
		"winner", winnerID,
		"agent1ELO", agent1ELOChange,
		"agent2ELO", agent2ELOChange,
	)
}

// GetByID 매치 조회
func (s *MatchService) GetByID(id string) (*models.Match, error) {
	match, err := s.matchRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get match: %w", err)
	}
	if match == nil {
		return nil, ErrMatchNotFound
	}

	return match, nil
}

// GetByAgentID 에이전트의 매치 목록
func (s *MatchService) GetByAgentID(agentID string, page, pageSize int) ([]*models.Match, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	matches, err := s.matchRepo.FindByAgentID(agentID, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get matches: %w", err)
	}

	return matches, nil
}

// getAgentSource Docker 이미지 URL 또는 코드 URL 결정
// Returns: (source, isDockerImage)
func (s *MatchService) getAgentSource(submission *models.Submission) (string, bool) {
	// TEMPORARY: Use code_url first for local testing
	// Until executor supports Docker images
	if submission.CodeURL != "" {
		return submission.CodeURL, false
	}
	
	// 1. Docker 이미지가 빌드되었으면 사용
	if submission.DockerImageURL != nil && *submission.DockerImageURL != "" {
		// 빌드가 성공한 경우에만 사용
		if submission.Status == models.SubmissionStatusSuccess ||
			submission.Status == models.SubmissionStatusActive {
			return *submission.DockerImageURL, true
		}
	}

	// 2. 빌드 중이면 대기 필요
	if submission.Status == models.SubmissionStatusBuilding {
		return "", false
	}

	// 3. Fallback
	return "", false
}

// resolveCodePath /storage/... 경로를 절대 경로로 변환 (파일 경로 그대로 유지)
func (s *MatchService) resolveCodePath(codePath string) string {
	// /storage/로 시작하면 현재 작업 디렉토리 기준 상대 경로로 변환
	if strings.HasPrefix(codePath, "/storage/") {
		// 현재 작업 디렉토리 가져오기
		cwd, err := os.Getwd()
		if err != nil {
			logger.Error("Failed to get current directory", "error", err)
			return codePath
		}
		
		// /storage/를 제거하고 현재 디렉토리와 결합
		relativePath := strings.TrimPrefix(codePath, "/")
		absolutePath := filepath.Join(cwd, relativePath)
		
		return absolutePath
	}
	
	return codePath
}
