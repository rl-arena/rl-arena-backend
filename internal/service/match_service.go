package service

import (
	"errors"
	"fmt"

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
	matchRepo      *repository.MatchRepository
	agentRepo      *repository.AgentRepository
	submissionRepo *repository.SubmissionRepository
	eloService     *ELOService
	executorClient *executor.Client
}

func NewMatchService(
	matchRepo *repository.MatchRepository,
	agentRepo *repository.AgentRepository,
	submissionRepo *repository.SubmissionRepository,
	eloService *ELOService,
	executorClient *executor.Client,
) *MatchService {
	return &MatchService{
		matchRepo:      matchRepo,
		agentRepo:      agentRepo,
		submissionRepo: submissionRepo,
		eloService:     eloService,
		executorClient: executorClient,
	}
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
	// 1. Docker 이미지가 빌드되었으면 우선 사용
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

	// 3. 빌드 실패했거나 Docker 이미지가 없으면 코드 URL 사용
	// (fallback for backward compatibility)
	return submission.CodeURL, false
}
