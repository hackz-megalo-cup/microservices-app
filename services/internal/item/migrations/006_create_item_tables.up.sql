CREATE TABLE user_item (
    user_id  UUID NOT NULL,
    item_id  UUID NOT NULL,
    quantity INT  NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, item_id)
);

CREATE TABLE item_history (
    id         UUID PRIMARY KEY,  -- UUIDv7
    user_id    UUID NOT NULL,
    item_id    UUID NOT NULL,
    change     INT  NOT NULL,
    reason     TEXT NOT NULL,     -- login_bonus, capture_used
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);