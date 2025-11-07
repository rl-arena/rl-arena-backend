package service

import (
	"fmt"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
)

type AgentService struct {
	agentRepo *repository.AgentRepository
}

func NewAgentService(agentRepo *repository.AgentRepository) *AgentService {
	return &AgentService{
		agentRepo: agentRepo,
	}
}

// Create 새 에이전트 생성
func (s *AgentService) Create(userID, name, description, environmentID string) (*models.Agent, error) {
	// 입력 검증
	if name == "" {
		return nil, ErrInvalidInput
	}

	// TODO: Environment 존재 확인
	// env, err := s.envRepo.FindByID(environmentID)
	// if env == nil {
	//     return nil, ErrInvalidEnvironment
	// }

	// 에이전트 생성
	agent, err := s.agentRepo.Create(userID, name, description, environmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agent, nil
}

// GetByID ID로 에이전트 조회
func (s *AgentService) GetByID(id string) (*models.Agent, error) {
	agent, err := s.agentRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}

	return agent, nil
}

// GetByUserID 사용자의 모든 에이전트 조회
func (s *AgentService) GetByUserID(userID string) ([]*models.Agent, error) {
	agents, err := s.agentRepo.FindByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user agents: %w", err)
	}

	return agents, nil
}

// List 모든 에이전트 조회 (페이지네이션)
func (s *AgentService) List(page, pageSize int) ([]*models.Agent, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	agents, err := s.agentRepo.FindAll(pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list agents: %w", err)
	}

	total, err := s.agentRepo.Count()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count agents: %w", err)
	}

	return agents, total, nil
}

// GetLeaderboard 리더보드 조회
func (s *AgentService) GetLeaderboard(environmentID string, limit int) ([]*models.Agent, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var agents []*models.Agent
	var err error

	if environmentID == "" {
		// 전체 리더보드
		agents, err = s.agentRepo.FindAll(limit, 0)
	} else {
		// 특정 환경 리더보드
		agents, err = s.agentRepo.FindByEnvironmentID(environmentID, limit, 0)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}

	return agents, nil
}

// GetLeaderboardWithType Public/Private 리더보드 조회
func (s *AgentService) GetLeaderboardWithType(environmentID, leaderboardType string, limit int) ([]*models.Agent, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// 참고: leaderboardType 파라미터는 향후 Public/Private 분리 기능을 위해 보존
	// 현재는 active_submission_id가 있는 모든 agent를 표시

	var agents []*models.Agent
	var err error

	if environmentID == "" {
		// 전체 리더보드 - 기존 메서드 사용 (is_public 필터링 없음)
		agents, err = s.agentRepo.FindAll(limit, 0)
	} else {
		// 특정 환경 리더보드 - 매치 기록 없어도 active submission이 있으면 표시
		// Public/Private 필터링은 일단 사용하지 않음 (matches 테이블 JOIN 불필요)
		agents, err = s.agentRepo.FindByEnvironmentID(environmentID, limit, 0)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}

	return agents, nil
}

// Update 에이전트 정보 업데이트
func (s *AgentService) Update(id, userID, name, description string) error {
	// 에이전트 존재 확인
	agent, err := s.agentRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("failed to find agent: %w", err)
	}
	if agent == nil {
		return ErrAgentNotFound
	}

	// 소유자 확인
	if agent.UserID != userID {
		return ErrUnauthorized
	}

	// 업데이트
	err = s.agentRepo.Update(id, name, description)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	return nil
}

// Delete 에이전트 삭제
func (s *AgentService) Delete(id, userID string) error {
	// 에이전트 존재 확인
	agent, err := s.agentRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("failed to find agent: %w", err)
	}
	if agent == nil {
		return ErrAgentNotFound
	}

	// 소유자 확인
	if agent.UserID != userID {
		return ErrUnauthorized
	}

	// 삭제
	err = s.agentRepo.Delete(id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

// GetOpponentStats 에이전트의 상대별 전적 통계 조회
func (s *AgentService) GetOpponentStats(agentID string) ([]*repository.OpponentStats, error) {
	// 에이전트 존재 확인
	agent, err := s.agentRepo.FindByID(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find agent: %w", err)
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}

	// 상대별 전적 조회
	stats, err := s.agentRepo.GetOpponentStats(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get opponent stats: %w", err)
	}

	return stats, nil
}

//
