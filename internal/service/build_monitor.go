package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
	"github.com/rl-arena/rl-arena-backend/internal/websocket"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch_api "k8s.io/apimachinery/pkg/watch"
)

// BuildMonitor K8s Job을 Watch하여 빌드 상태를 실시간 추적
type BuildMonitor struct {
	builderService     *BuilderService
	submissionRepo     *repository.SubmissionRepository
	wsHub              *websocket.Hub
	matchmakingService *MatchmakingService
	logger             *zap.Logger
	stopChan           chan struct{}
	wg                 sync.WaitGroup
	running            bool
	mu                 sync.Mutex
	
	// Watch 관련
	watcher        watch_api.Interface
	watchCtx       context.Context
	watchCancel    context.CancelFunc
}

// NewBuildMonitor BuildMonitor 생성
func NewBuildMonitor(
	builderService *BuilderService,
	submissionRepo *repository.SubmissionRepository,
	wsHub *websocket.Hub,
	matchmakingService *MatchmakingService,
	checkInterval time.Duration, // 호환성을 위해 유지하지만 사용하지 않음
) *BuildMonitor {
	logger, _ := zap.NewProduction()

	return &BuildMonitor{
		builderService:     builderService,
		submissionRepo:     submissionRepo,
		wsHub:              wsHub,
		matchmakingService: matchmakingService,
		logger:             logger,
		stopChan:           make(chan struct{}),
	}
}

// Start 모니터링 시작 (K8s Watch API 사용)
func (m *BuildMonitor) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	m.logger.Info("Starting BuildMonitor with K8s Watch API")

	// Watch context 생성
	m.watchCtx, m.watchCancel = context.WithCancel(context.Background())

	// 시작 시 pending submission 처리
	go m.processPendingSubmissions()

	m.wg.Add(1)
	go m.watchLoop()
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
	
	// Watch 취소
	if m.watchCancel != nil {
		m.watchCancel()
	}
	
	// Watcher 종료
	if m.watcher != nil {
		m.watcher.Stop()
	}
	
	close(m.stopChan)
	m.wg.Wait()
	m.logger.Info("BuildMonitor stopped")
}

// watchLoop K8s Watch API를 사용하여 Job 이벤트 실시간 감지
func (m *BuildMonitor) watchLoop() {
	defer m.wg.Done()

	// 재연결 로직을 위한 백오프
	backoff := time.Second
	maxBackoff := time.Minute

	for {
		select {
		case <-m.stopChan:
			return
		case <-m.watchCtx.Done():
			return
		default:
		}

		// Watch 시작
		if err := m.startWatch(); err != nil {
			m.logger.Error("Failed to start watch", zap.Error(err))
			
			// 백오프 후 재시도
			select {
			case <-time.After(backoff):
				// 백오프 증가 (최대 1분)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			case <-m.stopChan:
				return
			case <-m.watchCtx.Done():
				return
			}
			continue
		}

		// Watch 성공적으로 시작됨, 백오프 리셋
		backoff = time.Second
	}
}

// startWatch K8s Job Watch 시작
func (m *BuildMonitor) startWatch() error {
	// 빌드 전용 레이블로 필터링
	labelSelector := "app=rl-arena,type=agent-build"
	
	m.logger.Info("Starting K8s Job watch", zap.String("labelSelector", labelSelector))

	// Watch 생성
	watcher, err := m.builderService.k8sClient.BatchV1().Jobs(m.builderService.namespace).Watch(
		m.watchCtx,
		metav1.ListOptions{
			LabelSelector: labelSelector,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	m.watcher = watcher
	defer func() {
		if m.watcher != nil {
			m.watcher.Stop()
			m.watcher = nil
		}
	}()

	// 이벤트 처리
	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				m.logger.Warn("Watch channel closed, reconnecting...")
				return fmt.Errorf("watch channel closed")
			}

			m.handleWatchEvent(event)

		case <-m.stopChan:
			return nil
		case <-m.watchCtx.Done():
			return nil
		}
	}
}

// handleWatchEvent Watch 이벤트 처리
func (m *BuildMonitor) handleWatchEvent(event watch_api.Event) {
	job, ok := event.Object.(*batchv1.Job)
	if !ok {
		m.logger.Warn("Unexpected object type in watch event")
		return
	}

	m.logger.Debug("Received Job event",
		zap.String("type", string(event.Type)),
		zap.String("jobName", job.Name),
		zap.Int32("active", job.Status.Active),
		zap.Int32("succeeded", job.Status.Succeeded),
		zap.Int32("failed", job.Status.Failed))

	// ADDED, MODIFIED 이벤트만 처리
	if event.Type != watch_api.Added && event.Type != watch_api.Modified {
		return
	}

	// Job에서 submissionId 추출
	submissionID, ok := job.Labels["submission-id"]
	if !ok {
		m.logger.Warn("Job has no submission-id label", zap.String("jobName", job.Name))
		return
	}

	// Submission 조회
	ctx := context.Background()
	submission, err := m.submissionRepo.FindByID(submissionID)
	if err != nil {
		m.logger.Error("Failed to find submission",
			zap.String("submissionId", submissionID),
			zap.Error(err))
		return
	}

	// Submission이 없으면 스킵
	if submission == nil {
		m.logger.Warn("Submission not found",
			zap.String("submissionId", submissionID))
		return
	}

	// 이미 처리된 상태면 스킵
	if submission.Status != models.SubmissionStatusBuilding {
		return
	}

	// Job 상태에 따라 처리
	if job.Status.Succeeded > 0 {
		m.handleBuildSuccess(ctx, submission)
	} else if job.Status.Failed > 0 {
		m.handleBuildFailure(ctx, submission)
	}
	// Active > 0이면 아직 빌드 중
}

// checkAllBuilds는 더 이상 사용하지 않지만 초기 상태 복구를 위해 유지
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

	m.logger.Info("Recovering building submissions",
		zap.Int("count", len(submissions)))

	for _, submission := range submissions {
		m.checkBuild(ctx, submission)
	}
}

// checkBuild 개별 Submission의 빌드 상태 확인 (복구용)
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

	// Submission 상태를 'active'로 업데이트 (빌드 성공 시 즉시 활성화)
	if err := m.submissionRepo.UpdateStatus(
		submission.ID,
		models.SubmissionStatusActive,
		buildLog,
		nil,
	); err != nil {
	m.logger.Error("Failed to update submission status to active",
		zap.String("submissionId", submission.ID),
		zap.Error(err))
	return
}

// 자동으로 active submission으로 설정
if err := m.submissionRepo.SetActive(submission.ID, submission.AgentID); err != nil {
	m.logger.Warn("Failed to set submission as active",
		zap.String("submissionId", submission.ID),
		zap.Error(err))
	// 실패해도 계속 진행 (중요하지 않음)
}	// WebSocket으로 실시간 알림 전송
	if m.wsHub != nil {
		imageURL := ""
		if submission.DockerImageURL != nil {
			imageURL = *submission.DockerImageURL
		}
		m.wsHub.SendBuildStatus(
			submission.AgentID,
			submission.ID,
			string(models.SubmissionStatusActive),
			"Build completed successfully",
			imageURL,
		)
	}

	// 자동 매칭 큐에 등록 (EnvironmentID가 있는 경우)
	if m.matchmakingService != nil && submission.EnvironmentID != nil {
		if err := m.matchmakingService.EnqueueAgent(
			submission.AgentID,
			*submission.EnvironmentID,
		); err != nil {
			m.logger.Error("Failed to enqueue agent for matchmaking",
				zap.String("agentId", submission.AgentID),
				zap.String("environmentId", *submission.EnvironmentID),
				zap.Error(err))
		} else {
			m.logger.Info("Agent auto-enqueued for matchmaking",
				zap.String("agentId", submission.AgentID),
				zap.String("environmentId", *submission.EnvironmentID))
		}
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
		return
	}

	// WebSocket으로 실시간 알림 전송
	if m.wsHub != nil {
		m.wsHub.SendBuildStatus(
			submission.AgentID,
			submission.ID,
			string(models.SubmissionStatusBuildFailed),
			errorMsg,
			"",
		)
	}
}

// processPendingSubmissions 시작 시 pending 상태인 submission들을 처리
func (m *BuildMonitor) processPendingSubmissions() {
	ctx := context.Background()
	
	// pending 상태인 모든 Submission 조회
	submissions, err := m.submissionRepo.FindByStatus(models.SubmissionStatusPending)
	if err != nil {
		m.logger.Error("Failed to find pending submissions", zap.Error(err))
		return
	}

	if len(submissions) == 0 {
		m.logger.Info("No pending submissions to process")
		return
	}

	m.logger.Info("Processing pending submissions",
		zap.Int("count", len(submissions)))

	for _, submission := range submissions {
		// 상태를 'building'으로 업데이트
		status := models.SubmissionStatusBuilding
		if err := m.submissionRepo.UpdateStatus(submission.ID, status, nil, nil); err != nil {
			m.logger.Error("Failed to update submission status to building",
				zap.String("submissionId", submission.ID),
				zap.Error(err))
			continue
		}
		
		m.logger.Info("Starting Docker image build for pending submission",
			zap.String("submissionId", submission.ID),
			zap.String("agentId", submission.AgentID))
		
		// Kaniko 빌드 실행
		if err := m.builderService.BuildAgentImage(ctx, submission); err != nil {
			m.logger.Error("Docker image build failed",
				zap.String("submissionId", submission.ID),
				zap.Error(err))
			
			// 빌드 실패 상태로 업데이트
			errMsg := err.Error()
			failedStatus := models.SubmissionStatusBuildFailed
			if updateErr := m.submissionRepo.UpdateStatus(submission.ID, failedStatus, nil, &errMsg); updateErr != nil {
				m.logger.Error("Failed to update submission status to build_failed",
					zap.String("submissionId", submission.ID),
					zap.Error(updateErr))
			}
			continue
		}
		
		// Build 정보 데이터베이스에 저장
		if err := m.submissionRepo.UpdateBuildInfo(
			submission.ID,
			submission.BuildJobName,
			submission.DockerImageURL,
			submission.BuildPodName,
		); err != nil {
			m.logger.Error("Failed to update build info",
				zap.String("submissionId", submission.ID),
				zap.Error(err))
		}
		
		m.logger.Info("Docker image build job created for pending submission",
			zap.String("submissionId", submission.ID),
			zap.Stringp("jobName", submission.BuildJobName))
	}
}
