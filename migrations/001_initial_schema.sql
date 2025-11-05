-- 기존 테이블 삭제 (순서 중요!)
DROP TABLE IF EXISTS matches CASCADE;
DROP TABLE IF EXISTS submissions CASCADE;
DROP TABLE IF EXISTS agents CASCADE;
DROP TABLE IF EXISTS environments CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Updated_at 자동 업데이트 트리거 함수
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
RETURN NEW;
END;
$$ language 'plpgsql';

-- Users 테이블
CREATE TABLE IF NOT EXISTS users (
                                     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100),
    avatar_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
    );

-- Environments 테이블 (id를 VARCHAR로 변경)
CREATE TABLE IF NOT EXISTS environments (
                                            id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    version VARCHAR(20) NOT NULL,
    config JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
    );

-- Agents 테이블
CREATE TABLE IF NOT EXISTS agents (
                                      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    environment_id VARCHAR(50) NOT NULL REFERENCES environments(id),
    elo INTEGER NOT NULL DEFAULT 1500,
    wins INTEGER NOT NULL DEFAULT 0,
    losses INTEGER NOT NULL DEFAULT 0,
    draws INTEGER NOT NULL DEFAULT 0,
    total_matches INTEGER NOT NULL DEFAULT 0,
    active_submission_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
    );

-- Submissions 테이블
CREATE TABLE IF NOT EXISTS submissions (
                                           id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'building', 'active', 'failed', 'inactive')),
    code_url TEXT NOT NULL,
    build_log TEXT,
    error_message TEXT,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(agent_id, version)
    );

-- Agents의 active_submission_id 외래 키 추가
ALTER TABLE agents
    ADD CONSTRAINT fk_active_submission
        FOREIGN KEY (active_submission_id)
            REFERENCES submissions(id)
            ON DELETE SET NULL;

-- Matches 테이블
CREATE TABLE IF NOT EXISTS matches (
                                       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id VARCHAR(50) NOT NULL REFERENCES environments(id),
    agent1_id UUID NOT NULL REFERENCES agents(id),
    agent2_id UUID NOT NULL REFERENCES agents(id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    winner_id UUID REFERENCES agents(id),
    agent1_score DOUBLE PRECISION,
    agent2_score DOUBLE PRECISION,
    agent1_elo_change INTEGER,
    agent2_elo_change INTEGER,
    replay_url TEXT,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
    );

-- 인덱스 생성
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_agents_user_id ON agents(user_id);
CREATE INDEX idx_agents_environment_id ON agents(environment_id);
CREATE INDEX idx_agents_elo ON agents(elo DESC);
CREATE INDEX idx_submissions_agent_id ON submissions(agent_id);
CREATE INDEX idx_submissions_status ON submissions(status);
CREATE INDEX idx_matches_agent1_id ON matches(agent1_id);
CREATE INDEX idx_matches_agent2_id ON matches(agent2_id);
CREATE INDEX idx_matches_environment_id ON matches(environment_id);
CREATE INDEX idx_matches_status ON matches(status);
CREATE INDEX idx_matches_created_at ON matches(created_at DESC);

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_environments_updated_at BEFORE UPDATE ON environments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_submissions_updated_at BEFORE UPDATE ON submissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 기본 환경 데이터 삽입
INSERT INTO environments (id, name, description, version, config)
VALUES
    (
        'tic-tac-toe',
        'Tic-Tac-Toe',
        'Classic 3x3 Tic-Tac-Toe game',
        '1.0.0',
        '{"gridSize": 3, "winCondition": 3, "timeLimit": 1000}'::jsonb
    ),
    (
        'connect-four',
        'Connect Four',
        '6x7 Connect Four game',
        '1.0.0',
        '{"rows": 6, "cols": 7, "winCondition": 4, "timeLimit": 2000}'::jsonb
    ),
    (
        'chess',
        'Chess',
        'Standard chess game',
        '1.0.0',
        '{"timeLimit": 300000, "incrementPerMove": 5000}'::jsonb
    );