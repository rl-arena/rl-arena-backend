package service

import (
	"errors"
	"fmt"
	"mime/multipart"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
	"github.com/rl-arena/rl-arena-backend/pkg/storage"
)

var (
	ErrInvalidFile = errors.New("invalid file")
)

type SubmissionService struct {
	submissionRepo *repository.SubmissionRepository
	agentRepo      *repository.AgentRepository
	storage        *storage.Storage
	builderService *BuilderService
}

func NewSubmissionService(
	submissionRepo *repository.SubmissionRepository,
	agentRepo *repository.AgentRepository,
	storage *storage.Storage,
	builderService *BuilderService,
) *SubmissionService {
	return &SubmissionService{
		submissionRepo: submissionRepo,
		agentRepo:      agentRepo,
		storage:        storage,
		builderService: builderService,
	}
}

// Create 새 제출 생성
func (s *SubmissionService) Create(agentID, userID string, file *multipart.FileHeader) (*models.Submission, error) {
	// 에이전트 존재 및 소유자 확인
	agent, err := s.agentRepo.FindByID(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find agent: %w", err)
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	if agent.UserID != userID {
		return nil, ErrUnauthorized
	}

	// 파일 저장
	filePath, err := s.storage.SaveFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Python 파일 검증
	if err := s.storage.ValidatePythonFile(filePath); err != nil {
		s.storage.DeleteFile(filePath) // 실패 시 파일 삭제
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	// 파일 URL
	codeURL := s.storage.GetFileURL(filePath)

	// Submission 생성
	submission, err := s.submissionRepo.Create(agentID, codeURL)
	if err != nil {
		s.storage.DeleteFile(filePath) // 실패 시 파일 삭제
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	// TODO: 빌드 큐에 추가 (나중에 구현)
	// buildQueue.Enqueue(submission.ID)

	return submission, nil
}

// CreateFromURL URL로 제출 생성 (GitHub 등)
func (s *SubmissionService) CreateFromURL(agentID, userID, codeURL string) (*models.Submission, error) {
	// 에이전트 존재 및 소유자 확인
	agent, err := s.agentRepo.FindByID(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find agent: %w", err)
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	if agent.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Submission 생성
	submission, err := s.submissionRepo.Create(agentID, codeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	// TODO: Docker 이미지 빌드 (Kaniko)
	// BuilderService를 통해 비동기 빌드 시작
	// if s.builderService != nil {
	//     go func() {
	//         ctx := context.Background()
	//         if err := s.builderService.BuildAgentImage(ctx, submission); err != nil {
	//             // 빌드 실패 처리 (로그, 상태 업데이트 등)
	//         }
	//     }()
	// }

	return submission, nil
}

// GetByID ID로 제출 조회
func (s *SubmissionService) GetByID(id string) (*models.Submission, error) {
	submission, err := s.submissionRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}
	if submission == nil {
		return nil, ErrSubmissionNotFound
	}

	return submission, nil
}

// GetByAgentID 에이전트의 모든 제출 조회
func (s *SubmissionService) GetByAgentID(agentID string) ([]*models.Submission, error) {
	submissions, err := s.submissionRepo.FindByAgentID(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get submissions: %w", err)
	}

	return submissions, nil
}

// SetActive 제출을 활성화
func (s *SubmissionService) SetActive(submissionID, userID string) error {
	// 제출 조회
	submission, err := s.submissionRepo.FindByID(submissionID)
	if err != nil {
		return fmt.Errorf("failed to find submission: %w", err)
	}
	if submission == nil {
		return ErrSubmissionNotFound
	}

	// 에이전트 소유자 확인
	agent, err := s.agentRepo.FindByID(submission.AgentID)
	if err != nil {
		return fmt.Errorf("failed to find agent: %w", err)
	}
	if agent.UserID != userID {
		return ErrUnauthorized
	}

	// 활성화
	err = s.submissionRepo.SetActive(submissionID, submission.AgentID)
	if err != nil {
		return fmt.Errorf("failed to activate submission: %w", err)
	}

	return nil
}

// UpdateStatus 제출 상태 업데이트 (빌드 결과)
func (s *SubmissionService) UpdateStatus(submissionID string, status models.SubmissionStatus, buildLog, errorMessage *string) error {
	err := s.submissionRepo.UpdateStatus(submissionID, status, buildLog, errorMessage)
	if err != nil {
		return fmt.Errorf("failed to update submission status: %w", err)
	}

	// 빌드 성공 시 자동 활성화
	if status == models.SubmissionStatusActive {
		submission, _ := s.submissionRepo.FindByID(submissionID)
		if submission != nil {
			s.submissionRepo.SetActive(submissionID, submission.AgentID)
		}
	}

	return nil
}
