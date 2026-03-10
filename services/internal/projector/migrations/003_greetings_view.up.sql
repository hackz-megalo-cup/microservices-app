CREATE TABLE IF NOT EXISTS greetings_view (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    message TEXT,
    external_status INTEGER,
    status TEXT NOT NULL DEFAULT 'created',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_greetings_view_status ON greetings_view(status);
CREATE INDEX IF NOT EXISTS idx_greetings_view_created ON greetings_view(created_at);
