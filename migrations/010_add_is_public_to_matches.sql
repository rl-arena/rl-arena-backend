-- Add is_public column to matches table for public/private leaderboard split
-- Public matches: Counted in real-time leaderboard (default)
-- Private matches: Only revealed after competition end

ALTER TABLE matches 
ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT TRUE;

-- Add index for efficient filtering
CREATE INDEX IF NOT EXISTS idx_matches_is_public ON matches(is_public);

-- Add composite index for leaderboard queries (environment + public status)
CREATE INDEX IF NOT EXISTS idx_matches_env_public ON matches(environment_id, is_public) WHERE finished_at IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN matches.is_public IS 'Whether match counts towards public leaderboard. Private matches revealed after competition end.';
