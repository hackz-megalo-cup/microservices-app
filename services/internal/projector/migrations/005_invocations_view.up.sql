CREATE TABLE IF NOT EXISTS invocations_view (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    message TEXT,
    status TEXT NOT NULL DEFAULT 'completed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_invocations_view_status ON invocations_view(status);
CREATE INDEX IF NOT EXISTS idx_invocations_view_created ON invocations_view(created_at);
