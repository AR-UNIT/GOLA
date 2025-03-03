CREATE TABLE IF NOT EXISTS user_events (
                                           id SERIAL PRIMARY KEY,
                                           event_type VARCHAR(50) NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    target_id VARCHAR(100) NOT NULL,
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
    );
