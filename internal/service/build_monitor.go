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

// BuildMonitor K8s Job을 모니터링하여 빌드 상태를 추적
type BuildMonitor struct {
	builderService *BuilderService
	submissionRepo *repository.SubmissionRepository
	logger         *zap.Logger
	checkInterval  time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
	running        bool
	mu             sync.Mutex
}

// NewBuildMonitor BuildMonitor 생성
func NewBuildMonitor(
	builderService *BuilderService,
	submissionRepo *repository.SubmissionRepository,
	checkInterval time.Duration,
) *BuildMonitor {
	logger, _ := zap.NewProduction()
	
	if checkInterval == 0 {
		checkInterval = 10 * time.Second // 기본 10초
	}

	return &BuildMonitor{
		builderService: builderService,
		submissionRepo: submissionRepo,
		logger:         logger,
		checkInterval:  checkInterval,
		stopChan:       make(chan struct{}),
	}
}

// Start 모니터링 시작
func (m *BuildMonitor) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	m.logger.Info("Starting BuildMonitor",
		zap.Duration("checkInterval", m.checkInterval))

	m.wg.Add(1)
	go m.monitorLoop()
}

// Stop 모니터링 중지
func (m *BuildMonitor) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()

	m.logger.Info("Stopping BuildMonitor")
	close(m.stopChan)
	m.wg.Wait()
	m.logger.Info("BuildMonitor stopped")
}

// monitorLoop 주기적으로 빌드 상태 확인
func (m *BuildMonitor) monitorLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAllBuilds()
		case <-m.stopChan:
			return
		}
	}
}

// checkAllBuilds 모든 'building' 상태의 Submission 확인
func (m *BuildMonitor) checkAllBuilds() {
	ctx := context.Background()

	// 'building' 상태인 모든 Submission 조회
	submissions, err := m.submissionRepo.FindByStatus(models.SubmissionStatusBuilding)
	if err != nil {
		m.logger.Error("Failed to find building submissions", zap.Error(err))
		return
	}

	if len(submissions) == 0 {
		return
	}

	m.logger.Debug("Checking builds",
		zap.Int("count", len(submissions)))

	for _, submission := range submissions {
		m.checkBuild(ctx, submission)
	}
}

// checkBuild 개별 Submission의 빌드 상태 확인
func (m *BuildMonitor) checkBuild(ctx context.Context, submission *models.Submission) {
	if submission.BuildJobName == nil || *submission.BuildJobName == "" {
		m.logger.Warn("Submission has no build job name",
			zap.String("submissionId", submission.ID))
		return
	}

	jobName := *submission.BuildJobName

	// K8s Job 상태 확인
	status, err := m.builderService.GetBuildStatus(ctx, jobName)
	if err != nil {
		m.logger.Error("Failed to get build status",
			zap.String("submissionId", submission.ID),
			zap.String("jobName", jobName),
			zap.Error(err))
		return
	}

	m.logger.Debug("Build status",
		zap.String("submissionId", submission.ID),
		zap.String("jobName", jobName),
		zap.String("status", status))

	// 상태에 따라 처리
	switch status {
	case "succeeded":
		m.handleBuildSuccess(ctx, submission)
	case "failed":
		m.handleBuildFailure(ctx, submission)
	case "building", "pending":
		// 아직 진행 중이므로 아무 작업도 하지 않음
	default:
		m.logger.Warn("Unknown build status",
			zap.String("submissionId", submission.ID),
			zap.String("status", status))
	}
}

// handleBuildSuccess 빌드 성공 처리
func (m *BuildMonitor) handleBuildSuccess(ctx context.Context, submission *models.Submission) {
	m.logger.Info("Build succeeded",
		zap.String("submissionId", submission.ID),
		zap.Stringp("jobName", submission.BuildJobName),
		zap.Stringp("imageUrl", submission.DockerImageURL))

	// 빌드 로그 가져오기
	var buildLog *string
	if submission.BuildPodName != nil && *submission.BuildPodName != "" {
		logs, err := m.builderService.GetBuildLogs(ctx, *submission.BuildPodName)
		if err != nil {
			m.logger.Warn("Failed to get build logs",
				zap.String("submissionId", submission.ID),
				zap.Error(err))
		} else {
			buildLog = &logs
		}
	}

	// Submission 상태를 'success'로 업데이트
	if err := m.submissionRepo.UpdateStatus(
		submission.ID,
		models.SubmissionStatusSuccess,
		buildLog,
		nil,
	); err != nil {
		m.logger.Error("Failed to update submission status to success",
			zap.String("submissionId", submission.ID),
			zap.Error(err))
	}
}

// handleBuildFailure 빌드 실패 처리
func (m *BuildMonitor) handleBuildFailure(ctx context.Context, submission *models.Submission) {
	m.logger.Warn("Build failed",
		zap.String("submissionId", submission.ID),
		zap.Stringp("jobName", submission.BuildJobName))

	// 빌드 로그 가져오기
	var buildLog *string
	var errorMsg string

	if submission.BuildPodName != nil && *submission.BuildPodName != "" {
		logs, err := m.builderService.GetBuildLogs(ctx, *submission.BuildPodName)
		if err != nil {
			m.logger.Warn("Failed to get build logs",
				zap.String("submissionId", submission.ID),
				zap.Error(err))
			errorMsg = fmt.Sprintf("Build failed: unable to retrieve logs (%v)", err)
		} else {
			buildLog = &logs
			// 로그에서 에러 메시지 추출 (마지막 부분)
			if len(logs) > 500 {
				errorMsg = fmt.Sprintf("Build failed. Last 500 chars: %s", logs[len(logs)-500:])
			} else {
				errorMsg = fmt.Sprintf("Build failed: %s", logs)
			}
		}
	} else {
		errorMsg = "Build failed: no pod information available"
	}

	// Submission 상태를 'build_failed'로 업데이트
	if err := m.submissionRepo.UpdateStatus(
		submission.ID,
		models.SubmissionStatusBuildFailed,
		buildLog,
		&errorMsg,
	); err != nil {
		m.logger.Error("Failed to update submission status to build_failed",
			zap.String("submissionId", submission.ID),
			zap.Error(err))
	}
}
