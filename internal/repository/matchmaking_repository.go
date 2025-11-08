package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/pkg/database"
)

type MatchmakingRepository struct {
	db *database.DB
}

func NewMatchmakingRepository(db *database.DB) *MatchmakingRepository {
	return &MatchmakingRepository{db: db}
}

// EnqueueAgent 매칭 큐에 Agent 추가
func (r *MatchmakingRepository) EnqueueAgent(agentID, environmentID string, eloRating, priority int) error {
	query := `
		INSERT INTO matchmaking_queue (agent_id, environment_id, elo_rating, priority)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (agent_id, environment_id) 
		DO UPDATE SET 
			elo_rating = EXCLUDED.elo_rating,
			priority = EXCLUDED.priority,
			queued_at = NOW(),
			status = 'waiting'
	`
	_, err := r.db.Exec(query, agentID, environmentID, eloRating, priority)
	return err
}

// FindOpponent ELO 기반 상대 찾기
func (r *MatchmakingRepository) FindOpponent(agentID, environmentID string, eloRating int, eloRange int) (*models.MatchmakingQueue, error) {
	query := `
		SELECT id, agent_id, environment_id, elo_rating, priority, queued_at, status, matched_at
		FROM matchmaking_queue
		WHERE environment_id = $1
		  AND agent_id != $2
		  AND status = 'waiting'
		  AND elo_rating BETWEEN $3 AND $4
		ORDER BY 
			ABS(elo_rating - $5) ASC,
			priority DESC,
			queued_at ASC
		LIMIT 1
	`
	
	queue := &models.MatchmakingQueue{}
	err := r.db.QueryRow(query, 
		environmentID, 
		agentID, 
		eloRating-eloRange, 
		eloRating+eloRange,
		eloRating,
	).Scan(
		&queue.ID,
		&queue.AgentID,
		&queue.EnvironmentID,
		&queue.EloRating,
		&queue.Priority,
		&queue.QueuedAt,
		&queue.Status,
		&queue.MatchedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to find opponent: %w", err)
	}
	
	return queue, nil
}

// MarkAsMatched 매칭 완료로 표시
func (r *MatchmakingRepository) MarkAsMatched(queueIDs ...string) error {
	if len(queueIDs) == 0 {
		return nil
	}
	
	query := `
		UPDATE matchmaking_queue
		SET status = 'matched', matched_at = NOW()
		WHERE id = ANY($1)
	`
	_, err := r.db.Exec(query, pq.Array(queueIDs))
	return err
}

// GetWaitingAgents 대기 중인 Agent 목록 (현재 매치 진행 중이 아니고 rate limit을 통과한 에이전트만)
// cooldownMinutes: 마지막 매치 후 최소 대기 시간 (분)
// dailyLimit: 하루 최대 매치 횟수
func (r *MatchmakingRepository) GetWaitingAgents(environmentID string, cooldownMinutes int, dailyLimit int) ([]models.MatchmakingQueue, error) {
	query := `
		SELECT mq.id, mq.agent_id, mq.environment_id, mq.elo_rating, mq.priority, mq.queued_at, mq.status, mq.matched_at
		FROM matchmaking_queue mq
		LEFT JOIN agent_match_stats ams ON mq.agent_id = ams.agent_id
		WHERE mq.environment_id = $1 
		  AND mq.status = 'waiting'
		  AND NOT EXISTS (
		    SELECT 1 FROM matches m
		    WHERE m.status IN ('pending', 'running')
		      AND (m.agent1_id = mq.agent_id OR m.agent2_id = mq.agent_id)
		  )
		  -- Cooldown check: last match must be older than cooldown period
		  AND (ams.last_match_at IS NULL OR ams.last_match_at < NOW() - INTERVAL '1 minute' * $2)
		  -- Daily limit check: reset if daily_reset_at has passed
		  AND (
		    ams.agent_id IS NULL OR 
		    ams.daily_reset_at <= NOW() OR 
		    ams.matches_today < $3
		  )
		ORDER BY mq.priority DESC, mq.queued_at ASC
	`
	
	rows, err := r.db.Query(query, environmentID, cooldownMinutes, dailyLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to get waiting agents: %w", err)
	}
	defer rows.Close()
	
	var queues []models.MatchmakingQueue
	for rows.Next() {
		var queue models.MatchmakingQueue
		if err := rows.Scan(
			&queue.ID,
			&queue.AgentID,
			&queue.EnvironmentID,
			&queue.EloRating,
			&queue.Priority,
			&queue.QueuedAt,
			&queue.Status,
			&queue.MatchedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan queue: %w", err)
		}
		queues = append(queues, queue)
	}
	
	return queues, nil
}

// RecordMatch 매칭 기록 저장
func (r *MatchmakingRepository) RecordMatch(agent1ID, agent2ID, environmentID, matchID string, eloDiff int) error {
	query := `
		INSERT INTO matchmaking_history (agent1_id, agent2_id, environment_id, match_id, elo_difference)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(query, agent1ID, agent2ID, environmentID, matchID, eloDiff)
	return err
}

// CleanupExpired 오래된 대기 삭제
func (r *MatchmakingRepository) CleanupExpired(duration time.Duration) error {
	query := `
		UPDATE matchmaking_queue
		SET status = 'expired'
		WHERE status = 'waiting' AND queued_at < NOW() - $1::interval
	`
	_, err := r.db.Exec(query, fmt.Sprintf("%d seconds", int(duration.Seconds())))
	return err
}

// RemoveFromQueue 큐에서 제거
func (r *MatchmakingRepository) RemoveFromQueue(agentID, environmentID string) error {
	query := `
		DELETE FROM matchmaking_queue
		WHERE agent_id = $1 AND environment_id = $2
	`
	_, err := r.db.Exec(query, agentID, environmentID)
	return err
}
