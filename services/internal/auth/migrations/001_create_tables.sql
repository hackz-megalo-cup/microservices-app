-- Create users table
CREATE TABLE IF NOT EXISTS users (
  id uuid PRIMARY KEY,
  email varchar UNIQUE NOT NULL,
  password_hash text NOT NULL,
  role varchar DEFAULT 'user',
  created_at timestamptz NOT NULL,
  last_login_at timestamptz,
  updated_at timestamptz NOT NULL
);

CREATE INDEX idx_users_email ON users(email);

-- Create user_pokemon table (pokedex)
CREATE TABLE IF NOT EXISTS user_pokemon (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  pokemon_id varchar NOT NULL,
  caught_at timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, pokemon_id)
);

CREATE INDEX idx_user_pokemon_user_id ON user_pokemon(user_id);
