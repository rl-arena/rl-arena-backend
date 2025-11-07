package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
	"github.com/rl-arena/rl-arena-backend/pkg/storage"
	"go.uber.org/zap"
)

const (
	MaxRetryCount = 3 // 최대 재시도 횟수
)

var (
	ErrMaxRetriesExceeded = errors.New("maximum retry count exceeded")
)

type SubmissionService struct {
	submissionRepo *repository.SubmissionRepository
	agentRepo      *repository.AgentRepository
	storage        *storage.Storage
	builderService *BuilderService
	logger         *zap.Logger
}

func NewSubmissionService(
	submissionRepo *repository.SubmissionRepository,
	agentRepo *repository.AgentRepository,
	storage *storage.Storage,
	builderService *BuilderService,
) *SubmissionService {
	logger, _ := zap.NewProduction()
	return &SubmissionService{
		submissionRepo: submissionRepo,
		agentRepo:      agentRepo,
		storage:        storage,
		builderService: builderService,
		logger:         logger,
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

	// 일일 제출 쿼터 체크 (5회/일)
	todayCount, err := s.submissionRepo.CountTodaySubmissions(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check daily quota: %w", err)
	}
	if todayCount >= 5 {
		return nil, ErrDailyQuotaExceeded
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

	// 일일 제출 쿼터 체크 (5회/일)
	todayCount, err := s.submissionRepo.CountTodaySubmissions(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check daily quota: %w", err)
	}
	if todayCount >= 5 {
		return nil, ErrDailyQuotaExceeded
	}

	// Submission 생성
	submission, err := s.submissionRepo.Create(agentID, codeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	// Docker 이미지 빌드 시작 (비동기)
	if s.builderService != nil {
		go func() {
			ctx := context.Background()
			
			// 상태를 'building'으로 업데이트
			status := models.SubmissionStatusBuilding
			if err := s.submissionRepo.UpdateStatus(submission.ID, status, nil, nil); err != nil {
				s.logger.Error("Failed to update submission status to building",
					zap.String("submissionId", submission.ID),
					zap.Error(err))
				return
			}
			
			s.logger.Info("Starting Docker image build",
				zap.String("submissionId", submission.ID),
				zap.String("agentId", agentID),
				zap.String("codeUrl", codeURL))
			
			// Kaniko 빌드 실행
			if err := s.builderService.BuildAgentImage(ctx, submission); err != nil {
				s.logger.Error("Docker image build failed",
					zap.String("submissionId", submission.ID),
					zap.Error(err))
				
				// 빌드 실패 상태로 업데이트
				errMsg := err.Error()
				failedStatus := models.SubmissionStatusBuildFailed
				if updateErr := s.submissionRepo.UpdateStatus(submission.ID, failedStatus, nil, &errMsg); updateErr != nil {
					s.logger.Error("Failed to update submission status to build_failed",
						zap.String("submissionId", submission.ID),
						zap.Error(updateErr))
				}
				return
			}
			
			s.logger.Info("Docker image build job created successfully",
				zap.String("submissionId", submission.ID),
				zap.Stringp("jobName", submission.BuildJobName),
				zap.Stringp("imageUrl", submission.DockerImageURL))
			
			// 빌드 Job이 생성되었으므로, 모니터링은 BuildMonitor가 담당
			// (TODO #9에서 구현 예정)
		}()
	}

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

// RebuildSubmission 제출 재빌드
func (s *SubmissionService) RebuildSubmission(submissionID, userID string) error {
	// 제출 조회
	submission, err := s.submissionRepo.FindByID(submissionID)
	if err != nil {
		return fmt.Errorf("failed to find submission: %w", err)
	}
	if submission == nil {
		return ErrSubmissionNotFound
	}

	// 에이전트 조회 및 소유자 확인
	agent, err := s.agentRepo.FindByID(submission.AgentID)
	if err != nil {
		return fmt.Errorf("failed to find agent: %w", err)
	}
	if agent == nil {
		return ErrAgentNotFound
	}
	if agent.UserID != userID {
		return ErrUnauthorized
	}

	// 재시도 횟수 확인
	if submission.RetryCount >= MaxRetryCount {
		return ErrMaxRetriesExceeded
	}

	// 재시도 횟수 증가 및 타임스탬프 업데이트
	now := time.Now()
	newRetryCount := submission.RetryCount + 1
	
	// UpdateRetryInfo 메서드 호출 (아래에서 repository에 추가할 예정)
	if err := s.submissionRepo.UpdateRetryInfo(submissionID, newRetryCount, &now); err != nil {
		return fmt.Errorf("failed to update retry info: %w", err)
	}

	// 상태를 pending으로 리셋
	if err := s.submissionRepo.UpdateStatus(submissionID, models.SubmissionStatusPending, nil, nil); err != nil {
		return fmt.Errorf("failed to reset status: %w", err)
	}

	// Docker 이미지 빌드 재시작 (비동기)
	if s.builderService != nil {
		go func() {
			ctx := context.Background()
			
			// 상태를 'building'으로 업데이트
			status := models.SubmissionStatusBuilding
			if err := s.submissionRepo.UpdateStatus(submissionID, status, nil, nil); err != nil {
				s.logger.Error("Failed to update submission status to building",
					zap.String("submissionId", submissionID),
					zap.Error(err))
				return
			}
			
			// 최신 submission 정보를 다시 가져오기
			updatedSubmission, err := s.submissionRepo.FindByID(submissionID)
			if err != nil || updatedSubmission == nil {
				s.logger.Error("Failed to get updated submission",
					zap.String("submissionId", submissionID),
					zap.Error(err))
				return
			}
			
			s.logger.Info("Starting Docker image rebuild",
				zap.String("submissionId", submissionID),
				zap.String("codeUrl", updatedSubmission.CodeURL),
				zap.Int("retryCount", newRetryCount))
			
			if err := s.builderService.BuildAgentImage(ctx, updatedSubmission); err != nil {
				s.logger.Error("Failed to rebuild Docker image",
					zap.String("submissionId", submissionID),
					zap.Error(err))
			}
		}()
	}

	s.logger.Info("Submission rebuild initiated",
		zap.String("submissionId", submissionID),
		zap.Int("retryCount", newRetryCount))

	return nil
}