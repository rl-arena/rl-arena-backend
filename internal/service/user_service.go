package service

import (
	"fmt"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
)

type UserService struct {
	userRepo       *repository.UserRepository
	agentRepo      *repository.AgentRepository
	submissionRepo *repository.SubmissionRepository
}

func NewUserService(userRepo *repository.UserRepository, agentRepo *repository.AgentRepository, submissionRepo *repository.SubmissionRepository) *UserService {
	return &UserService{
		userRepo:       userRepo,
		agentRepo:      agentRepo,
		submissionRepo: submissionRepo,
	}
}

// Register 새 사용자 등록
func (s *UserService) Register(username, email, password, fullName string) (*models.User, error) {
	// 입력 검증
	if username == "" || email == "" || password == "" {
		return nil, ErrInvalidInput
	}

	// 이메일 중복 확인
	existingUser, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// 사용자명 중복 확인
	existingUser, err = s.userRepo.FindByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing username: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// 비밀번호 해싱
	passwordHash, err := models.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 사용자 생성
	user, err := s.userRepo.Create(username, email, passwordHash, fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login 로그인
func (s *UserService) Login(email, password string) (*models.User, error) {
	// 사용자 찾기
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// 비밀번호 확인
	if !user.CheckPassword(password) {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// GetByID ID로 사용자 조회
func (s *UserService) GetByID(id string) (*models.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// Update 사용자 정보 업데이트
func (s *UserService) Update(id string, fullName string, avatarURL *string) error {
	// 사용자 존재 확인
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("failed to check user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	// 업데이트
	err = s.userRepo.Update(id, fullName, avatarURL)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete 사용자 삭제
func (s *UserService) Delete(id string) error {
	err := s.userRepo.Delete(id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// UserStats 사용자 통계
type UserStats struct {
	TotalSubmissions int `json:"totalSubmissions"`
	ActiveAgents     int `json:"activeAgents"`
	TotalMatches     int `json:"totalMatches"`
	TotalWins        int `json:"totalWins"`
	BestRank         int `json:"bestRank"`
}

// GetUserStats 사용자 통계 조회
func (s *UserService) GetUserStats(userID string) (*UserStats, error) {
	// 제출 개수
	totalSubmissions, err := s.submissionRepo.CountByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count submissions: %w", err)
	}

	// 활성 에이전트 개수
	activeAgents, err := s.agentRepo.CountActiveByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count active agents: %w", err)
	}

	// 매치 통계
	totalMatches, totalWins, err := s.agentRepo.GetUserMatchStats(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get match stats: %w", err)
	}

	// 최고 순위
	bestRank, err := s.agentRepo.GetUserBestRank(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get best rank: %w", err)
	}

	return &UserStats{
		TotalSubmissions: totalSubmissions,
		ActiveAgents:     activeAgents,
		TotalMatches:     totalMatches,
		TotalWins:        totalWins,
		BestRank:         bestRank,
	}, nil
}
