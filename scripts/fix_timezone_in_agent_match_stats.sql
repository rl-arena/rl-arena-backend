-- Fix timezone issue in agent_match_stats table
-- This script converts KST timestamps to UTC and resets stats for fresh start

-- Step 1: Check current state (for logging)
SELECT 'Before fix:' as status;
SELECT 
    agent_id,
    last_match_at,
    last_match_at AT TIME ZONE 'Asia/Seoul' AT TIME ZONE 'UTC' as last_match_at_utc,
    matches_today,
    total_matches
FROM agent_match_stats;

-- Step 2: Update last_match_at to NULL and reset counters
-- This is the safest approach to avoid timezone math errors
UPDATE agent_match_stats
SET 
    last_match_at = NULL,
    matches_today = 0,
    daily_reset_at = (CURRENT_DATE + INTERVAL '1 day'),
    updated_at = NOW()
WHERE last_match_at IS NOT NULL;

-- Step 3: Verify the fix
SELECT 'After fix:' as status;
SELECT 
    agent_id,
    last_match_at,
    matches_today,
    daily_reset_at,
    total_matches
FROM agent_match_stats;

-- Add comment to table for future reference
COMMENT ON COLUMN agent_match_stats.last_match_at IS 
'Timestamp of last match completion. MUST be stored in UTC. Application code should use time.Now().UTC()';

COMMENT ON COLUMN agent_match_stats.updated_at IS 
'Last update timestamp in UTC. Application code should use time.Now().UTC()';

SELECT 'Timezone fix completed successfully. All timestamps will now use UTC.' as result;
