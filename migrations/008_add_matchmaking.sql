-- 008_add_matchmaking.sql
-- 자동 매칭 시스템을 위한 큐 및 기록 테이블

-- 매칭 큐 테이블
CREATE TABLE IF NOT EXISTS matchmaking_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    environment_id UUID NOT NULL REFERENCES environments(id),
    elo_rating INTEGER NOT NULL,
    priority INTEGER DEFAULT 5,
    queued_at TIMESTAMP NOT NULL DEFAULT NOW(),
    status VARCHAR(50) DEFAULT 'waiting',
    matched_at TIMESTAMP,
    
    UNIQUE(agent_id, environment_id)
);

CREATE INDEX idx_matchmaking_queue_status ON matchmaking_queue(status);
CREATE INDEX idx_matchmaking_queue_elo ON matchmaking_queue(environment_id, elo_rating) WHERE status = 'waiting';
CREATE INDEX idx_matchmaking_queue_priority ON matchmaking_queue(environment_id, priority DESC, queued_at ASC) WHERE status = 'waiting';

COMMENT ON TABLE matchmaking_queue IS 'Queue for automatic matchmaking';
COMMENT ON COLUMN matchmaking_queue.status IS 'waiting, matched, expired';
COMMENT ON COLUMN matchmaking_queue.priority IS '1 (highest) to 10 (lowest)';

-- 매칭 기록 테이블
CREATE TABLE IF NOT EXISTS matchmaking_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent1_id UUID NOT NULL REFERENCES agents(id),
    agent2_id UUID NOT NULL REFERENCES agents(id),
    environment_id UUID NOT NULL REFERENCES environments(id),
    match_id UUID REFERENCES matches(id),
    elo_difference INTEGER,
    matched_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_matchmaking_history_agent1 ON matchmaking_history(agent1_id);
CREATE INDEX idx_matchmaking_history_agent2 ON matchmaking_history(agent2_id);
CREATE INDEX idx_matchmaking_history_match ON matchmaking_history(match_id);

COMMENT ON TABLE matchmaking_history IS 'History of automatic matchmaking';
