CREATE TABLE IF NOT EXISTS outbox_events (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    topic TEXT NOT NULL,
    payload JSONB NOT NULL,
    published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    published_at TIMESTAMPTZ
);
