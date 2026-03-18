CREATE TABLE IF NOT EXISTS user_active_pokemon (
    user_id    UUID PRIMARY KEY,
    pokemon_id TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
