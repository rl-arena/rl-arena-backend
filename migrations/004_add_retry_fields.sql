-- 004_add_retry_fields.sql
-- Add retry tracking fields to submissions table

-- Add retry count and timestamp
ALTER TABLE submissions
ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0 NOT NULL,
ADD COLUMN IF NOT EXISTS last_retry_at TIMESTAMP;

-- Create index for efficient querying
CREATE INDEX IF NOT EXISTS idx_submissions_retry_count ON submissions(retry_count);

-- Add comment
COMMENT ON COLUMN submissions.retry_count IS 'Number of times this submission build has been retried';
COMMENT ON COLUMN submissions.last_retry_at IS 'Timestamp of the last retry attempt';
