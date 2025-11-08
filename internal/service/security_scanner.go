package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecurityScanner Trivy를 사용한 보안 스캔
type SecurityScanner struct {
	k8sClient      *kubernetes.Clientset
	namespace      string
	submissionRepo *repository.SubmissionRepository
	wsHub          interface{ SendBuildStatus(userID, submissionID, status, message, imageURL string) }
	logger         *zap.Logger
}

// TrivyResult Trivy 스캔 결과
type TrivyResult struct {
	Results []struct {
		Target          string `json:"Target"`
		Vulnerabilities []struct {
			VulnerabilityID  string `json:"VulnerabilityID"`
			PkgName          string `json:"PkgName"`
			InstalledVersion string `json:"InstalledVersion"`
			Severity         string `json:"Severity"`
			Title            string `json:"Title"`
			Description      string `json:"Description"`
		} `json:"Vulnerabilities"`
	} `json:"Results"`
}

// NewSecurityScanner SecurityScanner 생성
func NewSecurityScanner(
	k8sClient *kubernetes.Clientset,
	namespace string,
	submissionRepo *repository.SubmissionRepository,
	wsHub interface{ SendBuildStatus(userID, submissionID, status, message, imageURL string) },
) *SecurityScanner {
	logger, _ := zap.NewProduction()
	return &SecurityScanner{
		k8sClient:      k8sClient,
		namespace:      namespace,
		submissionRepo: submissionRepo,
		wsHub:          wsHub,
		logger:         logger,
	}
}

// ScanImage Docker 이미지 보안 스캔
func (s *SecurityScanner) ScanImage(ctx context.Context, submission *models.Submission) error {
	if submission.DockerImageURL == nil || *submission.DockerImageURL == "" {
		return fmt.Errorf("no docker image URL")
	}

	imageURL := *submission.DockerImageURL
	jobName := fmt.Sprintf("trivy-scan-%s", submission.ID[:8])

	s.logger.Info("Starting security scan",
		zap.String("submissionId", submission.ID),
		zap.String("imageURL", imageURL),
		zap.String("jobName", jobName))

	// Trivy 스캔 Job 생성
	job := s.createTrivyJob(jobName, imageURL, submission.ID)

	createdJob, err := s.k8sClient.BatchV1().Jobs(s.namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Trivy job: %w", err)
	}

	s.logger.Info("Trivy job created",
		zap.String("submissionId", submission.ID),
		zap.String("jobName", createdJob.Name))

	return nil
}

// createTrivyJob Trivy 스캔 Job 생성
func (s *SecurityScanner) createTrivyJob(jobName, imageURL, submissionID string) *batchv1.Job {
	backoffLimit := int32(0)
	ttlSecondsAfterFinished := int32(300) // 5분 후 자동 삭제

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"app":           "trivy-scanner",
				"submission-id": submissionID,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":           "trivy-scanner",
						"submission-id": submissionID,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "trivy",
							Image: "aquasec/trivy:latest",
							Args: []string{
								"image",
								"--format", "json",
								"--severity", "CRITICAL,HIGH,MEDIUM,LOW",
								"--exit-code", "0", // 취약점이 있어도 성공으로 처리
								imageURL,
							},
							// Note: 실제 환경에서는 스캔 결과를 ConfigMap이나 PVC에 저장해야 함
						},
					},
				},
			},
		},
	}
}

// ProcessScanResult Trivy 스캔 결과 처리
func (s *SecurityScanner) ProcessScanResult(ctx context.Context, submission *models.Submission, scanResult string) error {
	var trivyResult TrivyResult
	if err := json.Unmarshal([]byte(scanResult), &trivyResult); err != nil {
		return fmt.Errorf("failed to parse Trivy result: %w", err)
	}

	// 취약점 개수 집계
	counts := map[string]int{
		"CRITICAL": 0,
		"HIGH":     0,
		"MEDIUM":   0,
		"LOW":      0,
	}

	for _, result := range trivyResult.Results {
		for _, vuln := range result.Vulnerabilities {
			counts[vuln.Severity]++
		}
	}

	// 요약 생성
	summary := fmt.Sprintf(
		"Security Scan Complete: %d CRITICAL, %d HIGH, %d MEDIUM, %d LOW vulnerabilities found",
		counts["CRITICAL"], counts["HIGH"], counts["MEDIUM"], counts["LOW"],
	)

	s.logger.Info("Security scan completed",
		zap.String("submissionId", submission.ID),
		zap.Int("critical", counts["CRITICAL"]),
		zap.Int("high", counts["HIGH"]),
		zap.Int("medium", counts["MEDIUM"]),
		zap.Int("low", counts["LOW"]))

	// WebSocket 알림
	if s.wsHub != nil {
		status := "scan_complete"
		if counts["CRITICAL"] > 0 {
			status = "scan_complete_critical"
		}
		s.wsHub.SendBuildStatus(
			submission.AgentID,
			submission.ID,
			status,
			summary,
			"",
		)
	}

	return nil
}

// MonitorSecurityScans Trivy Job Watch (BuildMonitor와 유사)
func (s *SecurityScanner) MonitorSecurityScans(ctx context.Context) {
	s.logger.Info("Starting security scan monitor")

	// TODO: K8s Watch API를 사용하여 Trivy Job 모니터링
	// BuildMonitor와 유사한 패턴으로 구현
}
