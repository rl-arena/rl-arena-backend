-- 006_add_priority_queue.sql
-- 빌드 우선순위 큐 시스템

ALTER TABLE submissions
ADD COLUMN priority INTEGER DEFAULT 5,
ADD COLUMN queued_at TIMESTAMP;

CREATE INDEX idx_submissions_priority ON submissions(priority DESC, queued_at ASC) WHERE status = 'pending';

COMMENT ON COLUMN submissions.priority IS 'Priority level: 1 (highest) to 10 (lowest), default 5';
COMMENT ON COLUMN submissions.queued_at IS 'Timestamp when submission entered the build queue';
