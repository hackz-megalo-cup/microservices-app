-- item_master からエフェクト関連カラムを削除し、item_effect テーブルに分離する

ALTER TABLE item_master
    DROP COLUMN IF EXISTS effect_type,
    DROP COLUMN IF EXISTS target_type,
    DROP COLUMN IF EXISTS capture_rate_bonus;

CREATE TABLE item_effect (
    id                 UUID        PRIMARY KEY,
    item_id            UUID        NOT NULL REFERENCES item_master(id) ON DELETE CASCADE,
    effect_type        TEXT        NOT NULL,
    target_type        TEXT,
    capture_rate_bonus REAL        NOT NULL DEFAULT 0,
    flavor_text        TEXT,
    priority           INT         NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
