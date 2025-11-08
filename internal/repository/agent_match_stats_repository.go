package repository

import (
	"fmt"
	"time"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/pkg/database"
)

type AgentMatchStatsRepository struct {
	db *database.DB
}

func NewAgentMatchStatsRepository(db *database.DB) *AgentMatchStatsRepository {
	return &AgentMatchStatsRepository{db: db}
}

// GetOrCreate retrieves agent match stats or creates a new record
func (r *AgentMatchStatsRepository) GetOrCreate(agentID string) (*models.AgentMatchStats, error) {
	stats := &models.AgentMatchStats{}
	
	query := `
		SELECT agent_id, last_match_at, matches_today, daily_reset_at, total_matches, created_at, updated_at
		FROM agent_match_stats
		WHERE agent_id = $1
	`
	
	err := r.db.QueryRow(query, agentID).Scan(
		&stats.AgentID,
		&stats.LastMatchAt,
		&stats.MatchesToday,
		&stats.DailyResetAt,
		&stats.TotalMatches,
		&stats.CreatedAt,
		&stats.UpdatedAt,
	)
	
	if err != nil {
		// Create new stats record
		insertQuery := `
			INSERT INTO agent_match_stats (agent_id)
			VALUES ($1)
			RETURNING agent_id, last_match_at, matches_today, daily_reset_at, total_matches, created_at, updated_at
		`
		err = r.db.QueryRow(insertQuery, agentID).Scan(
			&stats.AgentID,
			&stats.LastMatchAt,
			&stats.MatchesToday,
			&stats.DailyResetAt,
			&stats.TotalMatches,
			&stats.CreatedAt,
			&stats.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent match stats: %w", err)
		}
	}
	
	return stats, nil
}

// IncrementMatchCount increments match counters and updates last match time
// NOTE: Always use UTC timestamps to avoid timezone comparison issues with database NOW()
func (r *AgentMatchStatsRepository) IncrementMatchCount(agentID string) error {
	now := time.Now().UTC() // Force UTC to match database NOW() function
	
	query := `
		UPDATE agent_match_stats
		SET 
			last_match_at = $2,
			matches_today = CASE 
				WHEN daily_reset_at <= $2 THEN 1
				ELSE matches_today + 1
			END,
			daily_reset_at = CASE
				WHEN daily_reset_at <= $2 THEN (CURRENT_DATE + INTERVAL '1 day')
				ELSE daily_reset_at
			END,
			total_matches = total_matches + 1,
			updated_at = $2
		WHERE agent_id = $1
	`
	
	result, err := r.db.Exec(query, agentID, now)
	if err != nil {
		return fmt.Errorf("failed to increment match count: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		// Create new record if doesn't exist
		insertQuery := `
			INSERT INTO agent_match_stats (agent_id, last_match_at, matches_today, total_matches)
			VALUES ($1, $2, 1, 1)
		`
		_, err = r.db.Exec(insertQuery, agentID, now) // now is already UTC from above
		if err != nil {
			return fmt.Errorf("failed to create agent match stats: %w", err)
		}
	}
	
	return nil
}

// CanMatch checks if an agent is allowed to participate in a match based on rate limits
func (r *AgentMatchStatsRepository) CanMatch(agentID string, config models.MatchRateLimitConfig) (bool, string, error) {
	stats, err := r.GetOrCreate(agentID)
	if err != nil {
		return false, "", err
	}
	
	now := time.Now().UTC() // Use UTC for consistent comparison
	
	// Check daily limit
	if stats.DailyResetAt.After(now) && stats.MatchesToday >= config.DailyMatchLimit {
		resetTime := stats.DailyResetAt.Format("15:04:05")
		return false, fmt.Sprintf("Daily match limit reached (%d/%d). Resets at %s", 
			stats.MatchesToday, config.DailyMatchLimit, resetTime), nil
	}
	
	// Check cooldown
	if stats.LastMatchAt != nil {
		timeSinceLastMatch := now.Sub(*stats.LastMatchAt)
		if timeSinceLastMatch < config.MatchCooldown {
			remainingCooldown := config.MatchCooldown - timeSinceLastMatch
			return false, fmt.Sprintf("Match cooldown active. Wait %d more seconds", 
				int(remainingCooldown.Seconds())), nil
	}
	}
	
	return true, "", nil
}

// ResetDailyStats manually resets daily statistics for testing purposes
func (r *AgentMatchStatsRepository) ResetDailyStats(agentID string) error {
	query := `
		UPDATE agent_match_stats
		SET 
			matches_today = 0,
			daily_reset_at = (CURRENT_DATE + INTERVAL '1 day'),
			updated_at = NOW()
		WHERE agent_id = $1
	`
	
	_, err := r.db.Exec(query, agentID)
	return err
}

// GetStats retrieves current stats for an agent
func (r *AgentMatchStatsRepository) GetStats(agentID string) (*models.AgentMatchStats, error) {
	return r.GetOrCreate(agentID)
}
