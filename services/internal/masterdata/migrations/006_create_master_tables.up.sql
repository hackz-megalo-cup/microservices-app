CREATE TABLE pokemon (
    id                  UUID PRIMARY KEY,
    name                TEXT NOT NULL,
    type                TEXT NOT NULL,
    hp                  INT  NOT NULL,
    attack              INT  NOT NULL,
    speed               INT  NOT NULL,
    special_move_name   TEXT NOT NULL,
    special_move_damage INT  NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE type_matchup (
    attacking_type TEXT NOT NULL,
    defending_type TEXT NOT NULL,
    effectiveness  REAL NOT NULL,
    PRIMARY KEY (attacking_type, defending_type)
);

CREATE TABLE item_master (
    id                 UUID PRIMARY KEY,
    name               TEXT NOT NULL,
    effect_type        TEXT NOT NULL,
    target_type        TEXT,
    capture_rate_bonus REAL NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
