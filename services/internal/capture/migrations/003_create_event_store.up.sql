CREATE TABLE IF NOT EXISTS event_store (
    id BIGSERIAL,
    stream_id TEXT NOT NULL,
    stream_type TEXT NOT NULL,
    version INTEGER NOT NULL,
    event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    data JSONB NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (stream_id, version)
);

CREATE INDEX IF NOT EXISTS idx_event_store_type ON event_store (event_type);
CREATE INDEX IF NOT EXISTS idx_event_store_created ON event_store (created_at);
