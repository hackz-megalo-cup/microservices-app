#!/bin/bash
set -e
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE auth_db;
    CREATE DATABASE gateway_db;
    CREATE DATABASE item_db;
    CREATE DATABASE masterdata_db;
    CREATE DATABASE raid_lobby_db;
    CREATE DATABASE lobby_db;
    CREATE DATABASE capture_db;
EOSQL

# CQRS projector: event_log table in gateway_db
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "gateway_db" <<-EOSQL
    CREATE TABLE IF NOT EXISTS event_log (
        event_id TEXT PRIMARY KEY,
        event_type TEXT NOT NULL,
        source TEXT NOT NULL,
        data JSONB,
        version INTEGER NOT NULL DEFAULT 1,
        created_at TIMESTAMPTZ NOT NULL,
        processed_at TIMESTAMPTZ DEFAULT NOW()
    );
    CREATE INDEX IF NOT EXISTS idx_event_log_type ON event_log(event_type);
    CREATE INDEX IF NOT EXISTS idx_event_log_created ON event_log(created_at);
EOSQL
