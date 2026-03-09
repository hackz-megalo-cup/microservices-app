CREATE TABLE outbox_events (
    id            UUID PRIMARY KEY,
    event_type    TEXT NOT NULL,
    topic         TEXT NOT NULL,
    payload       JSONB NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published     BOOLEAN NOT NULL DEFAULT FALSE,
    published_at  TIMESTAMPTZ
);
CREATE INDEX idx_outbox_unpublished ON outbox_events (created_at) WHERE published = FALSE;
