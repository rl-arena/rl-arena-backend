-- Complete Database Reset Script
-- WARNING: This will delete ALL data!

-- Drop all tables in correct order (reverse of dependencies)
DROP TABLE IF EXISTS agent_match_stats CASCADE;
DROP TABLE IF EXISTS matches CASCADE;
DROP TABLE IF EXISTS matchmaking_queue CASCADE;
DROP TABLE IF EXISTS submissions CASCADE;
DROP TABLE IF EXISTS agents CASCADE;
DROP TABLE IF EXISTS environments CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Recreate schema from scratch
-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100),
    avatar_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Environments table
CREATE TABLE environments (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    version VARCHAR(20) NOT NULL,
    config JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Agents table
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    environment_id VARCHAR(50) NOT NULL REFERENCES environments(id),
    elo INTEGER NOT NULL DEFAULT 1000,
    wins INTEGER NOT NULL DEFAULT 0,
    losses INTEGER NOT NULL DEFAULT 0,
    draws INTEGER NOT NULL DEFAULT 0,
    total_matches INTEGER NOT NULL DEFAULT 0,
    active_submission_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Submissions table
CREATE TABLE submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    environment_id VARCHAR(50) NOT NULL REFERENCES environments(id),
    version INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'building', 'active', 'failed', 'inactive')),
    code_url TEXT NOT NULL,
    docker_image_url TEXT,
    build_job_name TEXT,
    build_pod_name TEXT,
    build_log TEXT,
    error_message TEXT,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    last_retry_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(agent_id, version)
);

-- Add foreign key for active_submission_id
ALTER TABLE agents
    ADD CONSTRAINT fk_active_submission
    FOREIGN KEY (active_submission_id)
    REFERENCES submissions(id)
    ON DELETE SET NULL;

-- Matches table with submission tracking
CREATE TABLE matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id VARCHAR(50) NOT NULL REFERENCES environments(id),
    agent1_id UUID NOT NULL REFERENCES agents(id),
    agent2_id UUID NOT NULL REFERENCES agents(id),
    agent1_submission_id UUID REFERENCES submissions(id),
    agent2_submission_id UUID REFERENCES submissions(id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    winner_id UUID REFERENCES agents(id),
    agent1_score DOUBLE PRECISION,
    agent2_score DOUBLE PRECISION,
    agent1_elo_change INTEGER,
    agent2_elo_change INTEGER,
    is_public BOOLEAN NOT NULL DEFAULT TRUE,
    replay_url TEXT,
    replay_html_url TEXT,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_agents_user_id ON agents(user_id);
CREATE INDEX idx_agents_environment_id ON agents(environment_id);
CREATE INDEX idx_agents_elo ON agents(elo DESC);
CREATE INDEX idx_submissions_agent_id ON submissions(agent_id);
CREATE INDEX idx_submissions_status ON submissions(status);
CREATE INDEX idx_submissions_created_at ON submissions(created_at DESC);
CREATE INDEX idx_matches_agent1_id ON matches(agent1_id);
CREATE INDEX idx_matches_agent2_id ON matches(agent2_id);
CREATE INDEX idx_matches_status ON matches(status);
CREATE INDEX idx_matches_created_at ON matches(created_at DESC);
CREATE INDEX idx_matches_agent1_submission ON matches(agent1_submission_id);
CREATE INDEX idx_matches_agent2_submission ON matches(agent2_submission_id);
CREATE INDEX idx_matches_is_public ON matches(is_public);

-- Triggers
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_environments_updated_at BEFORE UPDATE ON environments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_submissions_updated_at BEFORE UPDATE ON submissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Agent match statistics table (for rate limiting)
CREATE TABLE agent_match_stats (
    agent_id UUID PRIMARY KEY REFERENCES agents(id) ON DELETE CASCADE,
    last_match_at TIMESTAMP,
    matches_today INT DEFAULT 0,
    daily_reset_at TIMESTAMP DEFAULT (CURRENT_DATE + INTERVAL '1 day'),
    total_matches INT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Matchmaking queue table
CREATE TABLE matchmaking_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    environment_id VARCHAR(50) NOT NULL REFERENCES environments(id),
    elo_rating INTEGER NOT NULL,
    priority INTEGER DEFAULT 5,
    queued_at TIMESTAMP NOT NULL DEFAULT NOW(),
    status VARCHAR(50) DEFAULT 'waiting',
    matched_at TIMESTAMP,
    UNIQUE(agent_id, environment_id)
);

CREATE INDEX idx_matchmaking_queue_status ON matchmaking_queue(status);
CREATE INDEX idx_matchmaking_queue_environment ON matchmaking_queue(environment_id);
CREATE INDEX idx_matchmaking_queue_queued_at ON matchmaking_queue(queued_at);

CREATE INDEX idx_agent_match_stats_last_match ON agent_match_stats(last_match_at);
CREATE INDEX idx_agent_match_stats_daily_reset ON agent_match_stats(daily_reset_at);

-- Trigger for agent_match_stats updated_at
CREATE TRIGGER update_agent_match_stats_updated_at BEFORE UPDATE ON agent_match_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default environment
INSERT INTO environments (id, name, description, version, config) 
VALUES (
    'pong', 
    'Pong', 
    'Classic Pong game environment', 
    'v1.0.0',
    '{"state_space": {"type": "continuous"}, "action_space": {"type": "discrete", "n": 3}, "max_steps": 1000}'::jsonb
);

-- Success message
SELECT 'Database reset complete! All tables recreated and ready for use.' AS status;
