package models

import "time"

type SubmissionStatus string

const (
	SubmissionStatusPending  SubmissionStatus = "pending"
	SubmissionStatusBuilding SubmissionStatus = "building"
	SubmissionStatusActive   SubmissionStatus = "active"
	SubmissionStatusFailed   SubmissionStatus = "failed"
	SubmissionStatusInactive SubmissionStatus = "inactive"
)

type Submission struct {
	ID           string           `json:"id" db:"id"`
	AgentID      string           `json:"agentId" db:"agent_id"`
	Version      int              `json:"version" db:"version"`
	Status       SubmissionStatus `json:"status" db:"status"`
	CodeURL      string           `json:"codeUrl" db:"code_url"`
	
	// Docker Build 관련 필드
	DockerImageURL *string `json:"dockerImageUrl,omitempty" db:"docker_image_url"`
	BuildJobName   *string `json:"buildJobName,omitempty" db:"build_job_name"`
	BuildPodName   *string `json:"buildPodName,omitempty" db:"build_pod_name"`
	BuildLog       *string `json:"buildLog,omitempty" db:"build_log"`
	
	ErrorMessage *string   `json:"errorMessage,omitempty" db:"error_message"`
	IsActive     bool      `json:"isActive" db:"is_active"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

type CreateSubmissionRequest struct {
	AgentID string `json:"agentId" binding:"required"`
	CodeURL string `json:"codeUrl" binding:"required"`
}
