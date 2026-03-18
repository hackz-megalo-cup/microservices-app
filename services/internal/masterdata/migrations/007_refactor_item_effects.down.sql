DROP TABLE IF EXISTS item_effect;

ALTER TABLE item_master
    ADD COLUMN effect_type        TEXT NOT NULL DEFAULT '',
    ADD COLUMN target_type        TEXT,
    ADD COLUMN capture_rate_bonus REAL NOT NULL DEFAULT 0;
