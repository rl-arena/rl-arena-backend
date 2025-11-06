package repository

import (
	"database/sql"
	"fmt"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/pkg/database"
)

type MatchRepository struct {
	db *database.DB
}

func NewMatchRepository(db *database.DB) *MatchRepository {
	return &MatchRepository{db: db}
}

// Create 새 매치 생성
func (r *MatchRepository) Create(environmentID, agent1ID, agent2ID string) (*models.Match, error) {
	query := `
		INSERT INTO matches (environment_id, agent1_id, agent2_id, status)
		VALUES ($1, $2, $3, 'pending')
		RETURNING id, environment_id, agent1_id, agent2_id, status, created_at
	`

	match := &models.Match{}
	err := r.db.QueryRow(query, environmentID, agent1ID, agent2ID).Scan(
		&match.ID,
		&match.EnvironmentID,
		&match.Agent1ID,
		&match.Agent2ID,
		&match.Status,
		&match.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create match: %w", err)
	}

	return match, nil
}

// UpdateResult 매치 결과 업데이트
func (r *MatchRepository) UpdateResult(
	matchID string,
	winnerID *string,
	agent1Score, agent2Score float64,
	agent1ELOChange, agent2ELOChange int,
	replayURL string,
) error {
	query := `
		UPDATE matches
		SET status = 'completed',
		    winner_id = $1,
		    agent1_score = $2,
		    agent2_score = $3,
		    agent1_elo_change = $4,
		    agent2_elo_change = $5,
		    replay_url = $6,
		    completed_at = NOW()
		WHERE id = $7
	`

	_, err := r.db.Exec(query,
		winnerID,
		agent1Score,
		agent2Score,
		agent1ELOChange,
		agent2ELOChange,
		replayURL,
		matchID,
	)

	if err != nil {
		return fmt.Errorf("failed to update match result: %w", err)
	}

	return nil
}

// FindByID ID로 매치 찾기
func (r *MatchRepository) FindByID(id string) (*models.Match, error) {
	query := `
		SELECT id, environment_id, agent1_id, agent2_id, status,
		       winner_id, agent1_score, agent2_score,
		       agent1_elo_change, agent2_elo_change,
		       replay_url, error_message,
		       started_at, completed_at, created_at
		FROM matches
		WHERE id = $1
	`

	match := &models.Match{}
	err := r.db.QueryRow(query, id).Scan(
		&match.ID,
		&match.EnvironmentID,
		&match.Agent1ID,
		&match.Agent2ID,
		&match.Status,
		&match.WinnerID,
		&match.Agent1Score,
		&match.Agent2Score,
		&match.Agent1ELOChange,
		&match.Agent2ELOChange,
		&match.ReplayURL,
		&match.ErrorMessage,
		&match.StartedAt,
		&match.CompletedAt,
		&match.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find match: %w", err)
	}

	return match, nil
}

// FindByAgentID 에이전트의 매치 목록
func (r *MatchRepository) FindByAgentID(agentID string, limit, offset int) ([]*models.Match, error) {
	query := `
		SELECT id, environment_id, agent1_id, agent2_id, status,
		       winner_id, agent1_score, agent2_score,
		       agent1_elo_change, agent2_elo_change,
		       replay_url, created_at
		FROM matches
		WHERE agent1_id = $1 OR agent2_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, agentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query matches: %w", err)
	}
	defer rows.Close()

	var matches []*models.Match
	for rows.Next() {
		match := &models.Match{}
		err := rows.Scan(
			&match.ID,
			&match.EnvironmentID,
			&match.Agent1ID,
			&match.Agent2ID,
			&match.Status,
			&match.WinnerID,
			&match.Agent1Score,
			&match.Agent2Score,
			&match.Agent1ELOChange,
			&match.Agent2ELOChange,
			&match.ReplayURL,
			&match.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan match: %w", err)
		}
		matches = append(matches, match)
	}

	return matches, nil
}
