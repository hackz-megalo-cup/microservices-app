#!/bin/bash
set -e
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE auth_db;
    CREATE DATABASE lang_db;
    CREATE DATABASE greeter_db;
    CREATE DATABASE caller_db;
    CREATE DATABASE gateway_db;
EOSQL
