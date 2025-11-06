-- Add Pong environment to the environments table

INSERT INTO environments (id, name, description, version, config)
VALUES
    (
        'pong',
        'Pong',
        'Classic Pong game with paddles and ball',
        '1.0.0',
        '{"fieldWidth": 800, "fieldHeight": 400, "paddleHeight": 80, "paddleWidth": 12, "ballRadius": 8, "maxScore": 11, "timeLimit": 300000}'::jsonb
    )
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    version = EXCLUDED.version,
    config = EXCLUDED.config,
    updated_at = NOW();