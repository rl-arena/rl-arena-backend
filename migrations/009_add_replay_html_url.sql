-- Add replay_html_url column to matches table
-- This allows storing both JSON (data) and HTML (visualization) replay formats
-- Ensures users see the same visualization in competition as during training

ALTER TABLE matches
ADD COLUMN IF NOT EXISTS replay_html_url TEXT;

-- Create index for efficient queries
CREATE INDEX IF NOT EXISTS idx_matches_replay_html_url 
ON matches(replay_html_url) 
WHERE replay_html_url IS NOT NULL;

COMMENT ON COLUMN matches.replay_html_url IS 'URL to HTML replay file (Kaggle-style visualization using rl-arena-env renderer)';
