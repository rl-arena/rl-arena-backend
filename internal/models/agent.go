package models

import "time"

type Agent struct {
	ID                 string    `json:"id" db:"id"`
	UserID             string    `json:"userId" db:"user_id"`
	Name               string    `json:"name" db:"name"`
	Description        string    `json:"description" db:"description"`
	EnvironmentID      string    `json:"environmentId" db:"environment_id"`
	ELO                int       `json:"elo" db:"elo"`
	Wins               int       `json:"wins" db:"wins"`
	Losses             int       `json:"losses" db:"losses"`
	Draws              int       `json:"draws" db:"draws"`
	TotalMatches       int       `json:"totalMatches" db:"total_matches"`
	ActiveSubmissionID *string   `json:"activeSubmissionId,omitempty" db:"active_submission_id"`
	CreatedAt          time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt          time.Time `json:"updatedAt" db:"updated_at"`
}

type CreateAgentRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	EnvironmentID string `json:"environmentId" binding:"required"`
}

type UpdateAgentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
