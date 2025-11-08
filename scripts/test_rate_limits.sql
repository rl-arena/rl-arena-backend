-- Test script for match rate limiting

-- View current match stats for all agents
SELECT 
    a.name as agent_name,
    ams.matches_today,
    ams.last_match_at,
    ams.daily_reset_at,
    ams.total_matches,
    EXTRACT(EPOCH FROM (NOW() - ams.last_match_at))/60 as minutes_since_last_match
FROM agents a
LEFT JOIN agent_match_stats ams ON a.id = ams.agent_id
ORDER BY a.name;

-- Check which agents can match (based on rate limits)
-- Cooldown: 5 minutes, Daily limit: 100
SELECT 
    mq.agent_id,
    a.name,
    mq.status as queue_status,
    ams.matches_today,
    ams.last_match_at,
    CASE 
        WHEN ams.last_match_at IS NULL THEN 'No cooldown'
        WHEN ams.last_match_at < NOW() - INTERVAL '5 minutes' THEN 'Ready (cooldown passed)'
        ELSE 'Cooling down (' || ROUND(EXTRACT(EPOCH FROM (NOW() - ams.last_match_at))/60, 1) || ' min ago)'
    END as cooldown_status,
    CASE
        WHEN ams.daily_reset_at <= NOW() THEN 'Ready (new day)'
        WHEN ams.matches_today < 100 THEN 'Ready (' || ams.matches_today || '/100 matches)'
        ELSE 'Daily limit reached'
    END as daily_limit_status
FROM matchmaking_queue mq
JOIN agents a ON mq.agent_id = a.id
LEFT JOIN agent_match_stats ams ON mq.agent_id = ams.agent_id
WHERE mq.status = 'waiting'
ORDER BY a.name;

-- Reset daily stats for testing (TESTING ONLY)
-- UPDATE agent_match_stats SET matches_today = 0, daily_reset_at = (CURRENT_DATE + INTERVAL '1 day');

-- Manually set a high match count to test daily limit (TESTING ONLY)
-- UPDATE agent_match_stats SET matches_today = 98 WHERE agent_id = '<AGENT_ID>';

-- Manually set recent match time to test cooldown (TESTING ONLY)  
-- UPDATE agent_match_stats SET last_match_at = NOW() - INTERVAL '2 minutes' WHERE agent_id = '<AGENT_ID>';
