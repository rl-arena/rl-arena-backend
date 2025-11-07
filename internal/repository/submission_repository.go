package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/pkg/database"
)

type SubmissionRepository struct {
	db *database.DB
}

func NewSubmissionRepository(db *database.DB) *SubmissionRepository {
	return &SubmissionRepository{db: db}
}

// Create 새 제출 생성
func (r *SubmissionRepository) Create(agentID, codeURL string) (*models.Submission, error) {
	// 다음 버전 번호 계산
	var nextVersion int
	err := r.db.QueryRow(`
		SELECT COALESCE(MAX(version), 0) + 1 
		FROM submissions 
		WHERE agent_id = $1
	`, agentID).Scan(&nextVersion)

	if err != nil {
		return nil, fmt.Errorf("failed to get next version: %w", err)
	}

	// Submission 생성
	query := `
		INSERT INTO submissions (agent_id, version, status, code_url, retry_count)
		VALUES ($1, $2, 'pending', $3, 0)
		RETURNING id, agent_id, version, status, code_url, docker_image_url, 
		          build_job_name, build_pod_name, build_log, error_message, 
		          retry_count, last_retry_at, is_active, created_at, updated_at
	`

	submission := &models.Submission{}
	err = r.db.QueryRow(query, agentID, nextVersion, codeURL).Scan(
		&submission.ID,
		&submission.AgentID,
		&submission.Version,
		&submission.Status,
		&submission.CodeURL,
		&submission.DockerImageURL,
		&submission.BuildJobName,
		&submission.BuildPodName,
		&submission.BuildLog,
		&submission.ErrorMessage,
		&submission.RetryCount,
		&submission.LastRetryAt,
		&submission.IsActive,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	return submission, nil
}

// FindByID ID로 제출 찾기
func (r *SubmissionRepository) FindByID(id string) (*models.Submission, error) {
	query := `
		SELECT id, agent_id, version, status, code_url, docker_image_url,
		       build_job_name, build_pod_name, build_log, error_message,
		       retry_count, last_retry_at,
		       is_active, created_at, updated_at
		FROM submissions
		WHERE id = $1
	`

	submission := &models.Submission{}
	err := r.db.QueryRow(query, id).Scan(
		&submission.ID,
		&submission.AgentID,
		&submission.Version,
		&submission.Status,
		&submission.CodeURL,
		&submission.DockerImageURL,
		&submission.BuildJobName,
		&submission.BuildPodName,
		&submission.BuildLog,
		&submission.ErrorMessage,
		&submission.RetryCount,
		&submission.LastRetryAt,
		&submission.IsActive,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find submission: %w", err)
	}

	return submission, nil
}

// FindByAgentID 에이전트의 모든 제출 조회
func (r *SubmissionRepository) FindByAgentID(agentID string) ([]*models.Submission, error) {
	query := `
		SELECT id, agent_id, version, status, code_url, docker_image_url,
		       build_job_name, build_pod_name, build_log, error_message,
		       retry_count, last_retry_at,
		       is_active, created_at, updated_at
		FROM submissions
		WHERE agent_id = $1
		ORDER BY version DESC
	`

	rows, err := r.db.Query(query, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query submissions: %w", err)
	}
	defer rows.Close()

	var submissions []*models.Submission
	for rows.Next() {
		submission := &models.Submission{}
		err := rows.Scan(
			&submission.ID,
			&submission.AgentID,
			&submission.Version,
			&submission.Status,
			&submission.CodeURL,
			&submission.DockerImageURL,
			&submission.BuildJobName,
			&submission.BuildPodName,
			&submission.BuildLog,
			&submission.ErrorMessage,
			&submission.RetryCount,
			&submission.LastRetryAt,
			&submission.IsActive,
			&submission.CreatedAt,
			&submission.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}
		submissions = append(submissions, submission)
	}

	return submissions, nil
}

// FindByStatus 특정 상태의 모든 Submission 조회
func (r *SubmissionRepository) FindByStatus(status models.SubmissionStatus) ([]*models.Submission, error) {
	query := `
		SELECT id, agent_id, version, status, code_url, docker_image_url,
		       build_job_name, build_pod_name, build_log, error_message,
		       retry_count, last_retry_at,
		       is_active, created_at, updated_at
		FROM submissions
		WHERE status = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query submissions by status: %w", err)
	}
	defer rows.Close()

	var submissions []*models.Submission
	for rows.Next() {
		submission := &models.Submission{}
		err := rows.Scan(
			&submission.ID,
			&submission.AgentID,
			&submission.Version,
			&submission.Status,
			&submission.CodeURL,
			&submission.DockerImageURL,
			&submission.BuildJobName,
			&submission.BuildPodName,
			&submission.BuildLog,
			&submission.ErrorMessage,
			&submission.RetryCount,
			&submission.LastRetryAt,
			&submission.IsActive,
			&submission.CreatedAt,
			&submission.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}
		submissions = append(submissions, submission)
	}

	return submissions, nil
}

// GetActiveSubmission 에이전트의 활성 제출 가져오기
func (r *SubmissionRepository) GetActiveSubmission(agentID string) (*models.Submission, error) {
	query := `
		SELECT id, agent_id, version, status, code_url, docker_image_url,
		       build_job_name, build_pod_name, build_log, error_message,
		       retry_count, last_retry_at,
		       is_active, created_at, updated_at
		FROM submissions
		WHERE agent_id = $1 AND is_active = true
		LIMIT 1
	`

	submission := &models.Submission{}
	err := r.db.QueryRow(query, agentID).Scan(
		&submission.ID,
		&submission.AgentID,
		&submission.Version,
		&submission.Status,
		&submission.CodeURL,
		&submission.DockerImageURL,
		&submission.BuildJobName,
		&submission.BuildPodName,
		&submission.BuildLog,
		&submission.ErrorMessage,
		&submission.RetryCount,
		&submission.LastRetryAt,
		&submission.IsActive,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get active submission: %w", err)
	}

	return submission, nil
}

// UpdateStatus 제출 상태 업데이트
func (r *SubmissionRepository) UpdateStatus(id string, status models.SubmissionStatus, buildLog, errorMessage *string) error {
	query := `
		UPDATE submissions
		SET status = $1, build_log = $2, error_message = $3
		WHERE id = $4
	`

	_, err := r.db.Exec(query, status, buildLog, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to update submission status: %w", err)
	}

	return nil
}

// SetActive 제출을 활성화 (기존 활성 제출은 비활성화)
func (r *SubmissionRepository) SetActive(submissionID, agentID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 기존 활성 제출 비활성화
	_, err = tx.Exec(`
		UPDATE submissions
		SET is_active = false, status = 'inactive'
		WHERE agent_id = $1 AND is_active = true
	`, agentID)
	if err != nil {
		return fmt.Errorf("failed to deactivate old submissions: %w", err)
	}

	// 새 제출 활성화
	_, err = tx.Exec(`
		UPDATE submissions
		SET is_active = true, status = 'active'
		WHERE id = $1
	`, submissionID)
	if err != nil {
		return fmt.Errorf("failed to activate submission: %w", err)
	}

	// Agent의 active_submission_id 업데이트
	_, err = tx.Exec(`
		UPDATE agents
		SET active_submission_id = $1
		WHERE id = $2
	`, submissionID, agentID)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete 제출 삭제
func (r *SubmissionRepository) Delete(id string) error {
	query := `DELETE FROM submissions WHERE id = $1`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete submission: %w", err)
	}

	return nil
}

// UpdateRetryInfo 재시도 정보 업데이트
func (r *SubmissionRepository) UpdateRetryInfo(id string, retryCount int, lastRetryAt *time.Time) error {
	query := `
		UPDATE submissions
		SET retry_count = $1, last_retry_at = $2, updated_at = NOW()
		WHERE id = $3
	`

	_, err := r.db.Exec(query, retryCount, lastRetryAt, id)
	if err != nil {
		return fmt.Errorf("failed to update retry info: %w", err)
	}

	return nil
}

// CountTodaySubmissions 오늘 제출한 submission 개수 조회 (일일 쿼터 체크용)
func (r *SubmissionRepository) CountTodaySubmissions(agentID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM submissions
		WHERE agent_id = $1
			AND created_at >= CURRENT_DATE
			AND created_at < CURRENT_DATE + INTERVAL '1 day'
	`

	var count int
	err := r.db.QueryRow(query, agentID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count today submissions: %w", err)
	}

	return count, nil
}
