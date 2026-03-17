CREATE TABLE IF NOT EXISTS raid_lobby (
    id              TEXT PRIMARY KEY,
    boss_pokemon_id TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'waiting',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS raid_participant (
    id        TEXT PRIMARY KEY,
    lobby_id  TEXT NOT NULL REFERENCES raid_lobby(id),
    user_id   TEXT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_raid_participant_lobby ON raid_participant (lobby_id);
