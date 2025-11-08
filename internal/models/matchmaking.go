package models

import "time"

type MatchmakingQueueStatus string

const (
	QueueStatusWaiting MatchmakingQueueStatus = "waiting"
	QueueStatusMatched MatchmakingQueueStatus = "matched"
	QueueStatusExpired MatchmakingQueueStatus = "expired"
)

type MatchmakingQueue struct {
	ID            string                 `db:"id" json:"id"`
	AgentID       string                 `db:"agent_id" json:"agentId"`
	EnvironmentID string                 `db:"environment_id" json:"environmentId"`
	EloRating     int                    `db:"elo_rating" json:"eloRating"`
	Priority      int                    `db:"priority" json:"priority"`
	QueuedAt      time.Time              `db:"queued_at" json:"queuedAt"`
	Status        MatchmakingQueueStatus `db:"status" json:"status"`
	MatchedAt     *time.Time             `db:"matched_at" json:"matchedAt,omitempty"`
}

type MatchmakingHistory struct {
	ID            string     `db:"id" json:"id"`
	Agent1ID      string     `db:"agent1_id" json:"agent1Id"`
	Agent2ID      string     `db:"agent2_id" json:"agent2Id"`
	EnvironmentID string     `db:"environment_id" json:"environmentId"`
	MatchID       *string    `db:"match_id" json:"matchId,omitempty"`
	EloDifference int        `db:"elo_difference" json:"eloDifference"`
	MatchedAt     time.Time  `db:"matched_at" json:"matchedAt"`
}
