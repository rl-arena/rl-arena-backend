-- 007_add_environment_to_submissions.sql
-- Agent 제출 시 어떤 환경용인지 추적

ALTER TABLE submissions
ADD COLUMN environment_id VARCHAR(50) REFERENCES environments(id);

-- 기존 데이터는 pong 환경으로 설정 (환경 ID는 실제 DB에서 확인 필요)
-- UPDATE submissions SET environment_id = (SELECT id FROM environments WHERE name = 'pong' LIMIT 1);

CREATE INDEX idx_submissions_environment ON submissions(environment_id);

COMMENT ON COLUMN submissions.environment_id IS 'Which environment this agent is built for';
