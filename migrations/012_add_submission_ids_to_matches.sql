-- Add submission IDs to matches table to track which submission version was used
ALTER TABLE matches
ADD COLUMN agent1_submission_id UUID REFERENCES submissions(id),
ADD COLUMN agent2_submission_id UUID REFERENCES submissions(id);

-- Add index for faster queries
CREATE INDEX idx_matches_agent1_submission ON matches(agent1_submission_id);
CREATE INDEX idx_matches_agent2_submission ON matches(agent2_submission_id);

-- Update existing matches to set submission IDs based on active submissions at the time
-- (This is best-effort - if the active submission has changed, this will set the current active one)
UPDATE matches m
SET agent1_submission_id = a.active_submission_id
FROM agents a
WHERE m.agent1_id = a.id AND m.agent1_submission_id IS NULL;

UPDATE matches m
SET agent2_submission_id = a.active_submission_id
FROM agents a
WHERE m.agent2_id = a.id AND m.agent2_submission_id IS NULL;
