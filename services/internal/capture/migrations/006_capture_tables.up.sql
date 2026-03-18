CREATE TABLE IF NOT EXISTS capture_session (
    id                UUID PRIMARY KEY,
    battle_session_id UUID NOT NULL,
    user_id           UUID NOT NULL,
    pokemon_id        UUID NOT NULL,
    base_rate         REAL NOT NULL,
    current_rate      REAL NOT NULL,
    result            TEXT NOT NULL DEFAULT 'pending',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS capture_action (
    id          UUID PRIMARY KEY,
    session_id  UUID NOT NULL REFERENCES capture_session(id),
    action_type TEXT NOT NULL,
    item_id     UUID,
    rate_change REAL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_capture_action_session ON capture_action (session_id);
CREATE INDEX IF NOT EXISTS idx_capture_session_user ON capture_session (user_id);
CREATE INDEX IF NOT EXISTS idx_capture_session_battle ON capture_session (battle_session_id);
