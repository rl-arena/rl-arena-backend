-- Add index for daily submission quota checking
-- Enables efficient counting of submissions per user per day

-- Add index on agent_id and created_at for quota checks
CREATE INDEX IF NOT EXISTS idx_submissions_agent_created 
ON submissions(agent_id, created_at DESC);

-- Add index on created_at for daily cleanup and analytics
CREATE INDEX IF NOT EXISTS idx_submissions_created_date 
ON submissions(DATE(created_at));

-- Add comment for documentation
COMMENT ON INDEX idx_submissions_agent_created IS 'Optimizes daily submission quota queries by agent and date';
