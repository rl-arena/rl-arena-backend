package models

import "time"

// AgentMatchStats tracks match statistics for rate limiting
type AgentMatchStats struct {
	AgentID       string     `json:"agentId" db:"agent_id"`
	LastMatchAt   *time.Time `json:"lastMatchAt" db:"last_match_at"`
	MatchesToday  int        `json:"matchesToday" db:"matches_today"`
	DailyResetAt  time.Time  `json:"dailyResetAt" db:"daily_reset_at"`
	TotalMatches  int        `json:"totalMatches" db:"total_matches"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"updated_at"`
}

// MatchRateLimitConfig configuration for match rate limiting
type MatchRateLimitConfig struct {
	DailyMatchLimit    int           // Maximum matches per day per agent
	MatchCooldown      time.Duration // Minimum time between matches
	MaxConcurrentMatch int           // Maximum concurrent matches per agent (already enforced by query)
}

// DefaultMatchRateLimitConfig returns default rate limit configuration
func DefaultMatchRateLimitConfig() MatchRateLimitConfig {
	return MatchRateLimitConfig{
		DailyMatchLimit:    100,         // 100 matches per day (Kaggle-style)
		MatchCooldown:      5 * time.Minute, // 5 minutes between matches
		MaxConcurrentMatch: 1,           // Only 1 match at a time per agent
	}
}
