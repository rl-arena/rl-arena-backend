package models

import "time"

type MatchStatus string

const (
	MatchStatusPending   MatchStatus = "pending"
	MatchStatusRunning   MatchStatus = "running"
	MatchStatusCompleted MatchStatus = "completed"
	MatchStatusFailed    MatchStatus = "failed"
)

type Match struct {
	ID              string      `json:"id" db:"id"`
	EnvironmentID   string      `json:"environmentId" db:"environment_id"`
	Agent1ID        string      `json:"agent1Id" db:"agent1_id"`
	Agent2ID        string      `json:"agent2Id" db:"agent2_id"`
	Status          MatchStatus `json:"status" db:"status"`
	WinnerID        *string     `json:"winnerId,omitempty" db:"winner_id"`
	Agent1Score     *float64    `json:"agent1Score,omitempty" db:"agent1_score"`
	Agent2Score     *float64    `json:"agent2Score,omitempty" db:"agent2_score"`
	Agent1ELOChange *int        `json:"agent1EloChange,omitempty" db:"agent1_elo_change"`
	Agent2ELOChange *int        `json:"agent2EloChange,omitempty" db:"agent2_elo_change"`
	IsPublic        bool        `json:"isPublic" db:"is_public"`
	ReplayURL       *string     `json:"replayUrl,omitempty" db:"replay_url"`
	ReplayHTMLURL   *string     `json:"replayHtmlUrl,omitempty" db:"replay_html_url"`
	ErrorMessage    *string     `json:"errorMessage,omitempty" db:"error_message"`
	StartedAt       *time.Time  `json:"startedAt,omitempty" db:"started_at"`
	CompletedAt     *time.Time  `json:"completedAt,omitempty" db:"completed_at"`
	CreatedAt       time.Time   `json:"createdAt" db:"created_at"`
}

type MatchResult struct {
	MatchID     string  `json:"matchId"`
	WinnerID    *string `json:"winnerId"`
	Agent1Score float64 `json:"agent1Score"`
	Agent2Score float64 `json:"agent2Score"`
	ReplayURL   string  `json:"replayUrl"`
}
