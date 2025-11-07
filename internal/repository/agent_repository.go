package repository

import (
	"database/sql"
	"fmt"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/pkg/database"
)

type AgentRepository struct {
	db *database.DB
}

func NewAgentRepository(db *database.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

// Create 새 에이전트 생성
func (r *AgentRepository) Create(userID, name, description, environmentID string) (*models.Agent, error) {
	query := `
		INSERT INTO agents (user_id, name, description, environment_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, name, description, environment_id, elo, wins, losses, draws, 
		          total_matches, active_submission_id, created_at, updated_at
	`

	agent := &models.Agent{}
	err := r.db.QueryRow(query, userID, name, description, environmentID).Scan(
		&agent.ID,
		&agent.UserID,
		&agent.Name,
		&agent.Description,
		&agent.EnvironmentID,
		&agent.ELO,
		&agent.Wins,
		&agent.Losses,
		&agent.Draws,
		&agent.TotalMatches,
		&agent.ActiveSubmissionID,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agent, nil
}

// FindByID ID로 에이전트 찾기
func (r *AgentRepository) FindByID(id string) (*models.Agent, error) {
	query := `
		SELECT id, user_id, name, description, environment_id, elo, wins, losses, draws,
		       total_matches, active_submission_id, created_at, updated_at
		FROM agents
		WHERE id = $1
	`

	agent := &models.Agent{}
	err := r.db.QueryRow(query, id).Scan(
		&agent.ID,
		&agent.UserID,
		&agent.Name,
		&agent.Description,
		&agent.EnvironmentID,
		&agent.ELO,
		&agent.Wins,
		&agent.Losses,
		&agent.Draws,
		&agent.TotalMatches,
		&agent.ActiveSubmissionID,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find agent: %w", err)
	}

	return agent, nil
}

// FindByUserID 사용자의 모든 에이전트 조회
func (r *AgentRepository) FindByUserID(userID string) ([]*models.Agent, error) {
	query := `
		SELECT id, user_id, name, description, environment_id, elo, wins, losses, draws,
		       total_matches, active_submission_id, created_at, updated_at
		FROM agents
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		agent := &models.Agent{}
		err := rows.Scan(
			&agent.ID,
			&agent.UserID,
			&agent.Name,
			&agent.Description,
			&agent.EnvironmentID,
			&agent.ELO,
			&agent.Wins,
			&agent.Losses,
			&agent.Draws,
			&agent.TotalMatches,
			&agent.ActiveSubmissionID,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// FindAll 모든 에이전트 조회 (페이지네이션)
func (r *AgentRepository) FindAll(limit, offset int) ([]*models.Agent, error) {
	query := `
		SELECT id, user_id, name, description, environment_id, elo, wins, losses, draws,
		       total_matches, active_submission_id, created_at, updated_at
		FROM agents
		ORDER BY elo DESC, created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		agent := &models.Agent{}
		err := rows.Scan(
			&agent.ID,
			&agent.UserID,
			&agent.Name,
			&agent.Description,
			&agent.EnvironmentID,
			&agent.ELO,
			&agent.Wins,
			&agent.Losses,
			&agent.Draws,
			&agent.TotalMatches,
			&agent.ActiveSubmissionID,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// FindByEnvironmentID 특정 환경의 에이전트 조회 (리더보드용)
func (r *AgentRepository) FindByEnvironmentID(environmentID string, limit, offset int) ([]*models.Agent, error) {
	query := `
		SELECT id, user_id, name, description, environment_id, elo, wins, losses, draws,
		       total_matches, active_submission_id, created_at, updated_at
		FROM agents
		WHERE environment_id = $1
		ORDER BY elo DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, environmentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		agent := &models.Agent{}
		err := rows.Scan(
			&agent.ID,
			&agent.UserID,
			&agent.Name,
			&agent.Description,
			&agent.EnvironmentID,
			&agent.ELO,
			&agent.Wins,
			&agent.Losses,
			&agent.Draws,
			&agent.TotalMatches,
			&agent.ActiveSubmissionID,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// FindByEnvironmentIDWithPublicity 특정 환경의 에이전트 조회 (Public/Private 리더보드 분리)
func (r *AgentRepository) FindByEnvironmentIDWithPublicity(environmentID string, isPublic *bool, limit, offset int) ([]*models.Agent, error) {
	// Base query with agent stats
	baseQuery := `
		SELECT DISTINCT a.id, a.user_id, a.name, a.description, a.environment_id, 
		       a.elo, a.wins, a.losses, a.draws, a.total_matches, 
		       a.active_submission_id, a.created_at, a.updated_at
		FROM agents a
		INNER JOIN matches m ON (m.agent1_id = a.id OR m.agent2_id = a.id)
		WHERE a.environment_id = $1
	`

	var args []interface{}
	args = append(args, environmentID)

	// Add is_public filter if specified
	if isPublic != nil {
		baseQuery += " AND m.is_public = $2"
		args = append(args, *isPublic)
	}

	// Group by and order
	baseQuery += `
		GROUP BY a.id, a.user_id, a.name, a.description, a.environment_id,
		         a.elo, a.wins, a.losses, a.draws, a.total_matches,
		         a.active_submission_id, a.created_at, a.updated_at
		ORDER BY a.elo DESC
	`

	// Add limit and offset
	limitOffset := len(args) + 1
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", limitOffset, limitOffset+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents with publicity: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		agent := &models.Agent{}
		err := rows.Scan(
			&agent.ID,
			&agent.UserID,
			&agent.Name,
			&agent.Description,
			&agent.EnvironmentID,
			&agent.ELO,
			&agent.Wins,
			&agent.Losses,
			&agent.Draws,
			&agent.TotalMatches,
			&agent.ActiveSubmissionID,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// Update 에이전트 정보 업데이트
func (r *AgentRepository) Update(id, name, description string) error {
	query := `
		UPDATE agents
		SET name = $1, description = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(query, name, description, id)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	return nil
}

// UpdateStats 에이전트 통계 업데이트 (매치 후)
func (r *AgentRepository) UpdateStats(id string, eloChange int, won, lost, draw bool) error {
	query := `
		UPDATE agents
		SET elo = elo + $1,
		    wins = wins + $2,
		    losses = losses + $3,
		    draws = draws + $4,
		    total_matches = total_matches + 1
		WHERE id = $5
	`

	winsInc := 0
	lossesInc := 0
	drawsInc := 0

	if won {
		winsInc = 1
	} else if lost {
		lossesInc = 1
	} else if draw {
		drawsInc = 1
	}

	_, err := r.db.Exec(query, eloChange, winsInc, lossesInc, drawsInc, id)
	if err != nil {
		return fmt.Errorf("failed to update agent stats: %w", err)
	}

	return nil
}

// SetActiveSubmission active submission 설정
func (r *AgentRepository) SetActiveSubmission(agentID, submissionID string) error {
	query := `
		UPDATE agents
		SET active_submission_id = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(query, submissionID, agentID)
	if err != nil {
		return fmt.Errorf("failed to set active submission: %w", err)
	}

	return nil
}

// Delete 에이전트 삭제
func (r *AgentRepository) Delete(id string) error {
	query := `DELETE FROM agents WHERE id = $1`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

// Count 전체 에이전트 수
func (r *AgentRepository) Count() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM agents`

	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count agents: %w", err)
	}

	return count, nil
}
