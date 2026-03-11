#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
SECRETS_FILE="${REPO_ROOT}/deploy/k8s/secrets.nix"
COMPOSE_FILE="${REPO_ROOT}/docker-compose.yml"
K8S_NAMESPACE="${K8S_NAMESPACE:-database}"
K8S_INSTANCE_NAME="${K8S_INSTANCE_NAME:-postgresql}"
K8S_DB_OWNER="${K8S_DB_OWNER:-devuser}"
K8S_WAIT_TIMEOUT_SECONDS="${K8S_WAIT_TIMEOUT_SECONDS:-300}"
COMPOSE_DB_SERVICE="${COMPOSE_DB_SERVICE:-postgres}"
COMPOSE_DB_OWNER="${COMPOSE_DB_OWNER:-devuser}"
COMPOSE_DB_HOST="${COMPOSE_DB_HOST:-postgres}"
COMPOSE_DB_PORT="${COMPOSE_DB_PORT:-5432}"
COMPOSE_DB_NAME="${COMPOSE_DB_NAME:-postgres}"
COMPOSE_DB_USER="${COMPOSE_DB_USER:-devuser}"
COMPOSE_DB_PASSWORD="${COMPOSE_DB_PASSWORD:-devpass}"
LOG_FILE="${LOG_FILE:-/tmp/manage-service-db.log}"

if [[ -n "$LOG_FILE" ]]; then
  mkdir -p "$(dirname "$LOG_FILE")"
  exec > >(tee -a "$LOG_FILE") 2>&1
fi

usage() {
  cat <<'EOF'
Usage:
  manage-service-db.sh sync-k8s
  manage-service-db.sh sync-compose
  manage-service-db.sh drop-k8s <db-name>
  manage-service-db.sh drop-compose <db-name>
EOF
  exit 1
}

log() {
  echo "[$(basename "$0")] $*"
}

trap 'log "failed at line ${LINENO}: ${BASH_COMMAND}"' ERR

validate_db_name() {
  local db_name="$1"
  [[ "$db_name" =~ ^[a-z][a-z0-9_]*$ ]]
}

extract_k8s_databases() {
  if [[ ! -f "$SECRETS_FILE" ]]; then
    log "missing ${SECRETS_FILE}"
    return 1
  fi

  awk -F'/' '/DATABASE_URL = "/ { print $NF }' "$SECRETS_FILE" \
    | sed -E 's/"[;[:space:]]*$//' \
    | sed -E 's/\?.*$//' \
    | awk 'NF' \
    | sort -u
}

extract_compose_databases() {
  if [[ ! -f "$COMPOSE_FILE" ]]; then
    log "missing ${COMPOSE_FILE}"
    return 1
  fi

  awk -F'/' '/DATABASE_URL:/ { print $NF }' "$COMPOSE_FILE" \
    | sed -E 's/"[[:space:]]*$//' \
    | sed -E "s/'[[:space:]]*$//" \
    | sed -E 's/\?.*$//' \
    | awk 'NF' \
    | sort -u
}

wait_for_k8s_postgres() {
  local pod=""
  local try
  local max_tries
  max_tries=$((K8S_WAIT_TIMEOUT_SECONDS / 2))
  if (( max_tries < 1 )); then
    max_tries=1
  fi

  for try in $(seq 1 "$max_tries"); do
    pod="$(kubectl get pods -n "$K8S_NAMESPACE" \
      -l "app.kubernetes.io/instance=${K8S_INSTANCE_NAME},app.kubernetes.io/component=primary" \
      -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"

    if [[ -z "$pod" ]]; then
      pod="$(kubectl get pod -n "$K8S_NAMESPACE" "${K8S_INSTANCE_NAME}-0" \
        -o jsonpath='{.metadata.name}' 2>/dev/null || true)"
    fi

    if [[ -n "$pod" ]]; then
      if kubectl exec -n "$K8S_NAMESPACE" "$pod" -- sh -lc \
        '
          password_file="${POSTGRES_POSTGRES_PASSWORD_FILE:-${POSTGRES_PASSWORD_FILE:-}}"
          if [[ -n "$password_file" && -f "$password_file" ]]; then
            export PGPASSWORD="$(cat "$password_file")"
          else
            export PGPASSWORD="${POSTGRES_PASSWORD:-}"
          fi
          pg_isready -U postgres -d postgres >/dev/null 2>&1
        '; then
        printf '%s\n' "$pod"
        return 0
      fi
    fi

    if (( try == 1 || try % 10 == 0 )); then
      log "waiting for k8s postgres in namespace ${K8S_NAMESPACE} (${try}/${max_tries})"
    fi

    sleep 2
  done

  log "postgres primary pod in namespace ${K8S_NAMESPACE} is not ready after ${K8S_WAIT_TIMEOUT_SECONDS}s"
  return 1
}

psql_create_sql() {
  local db_name="$1"
  printf "CREATE DATABASE %s;" "$db_name"
}

psql_exists_sql() {
  local db_name="$1"
  printf "SELECT 1 FROM pg_database WHERE datname = '%s';" "$db_name"
}

psql_grant_sql() {
  local db_name="$1"
  local owner="$2"
  printf "GRANT ALL PRIVILEGES ON DATABASE %s TO %s;" "$db_name" "$owner"
}

psql_schema_grant_sql() {
  local owner="$1"
  printf "GRANT ALL ON SCHEMA public TO %s;" "$owner"
}

psql_terminate_connections_sql() {
  local db_name="$1"
  cat <<EOF
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = '${db_name}' AND pid <> pg_backend_pid();
EOF
}

psql_drop_sql() {
  local db_name="$1"
  printf "DROP DATABASE IF EXISTS %s;" "$db_name"
}

run_k8s_psql_cmd() {
  local pod="$1"
  local sql="$2"
  local escaped_sql
  escaped_sql="$(printf '%q' "$sql")"

  kubectl exec -n "$K8S_NAMESPACE" "$pod" -- bash -lc \
    '
      password_file="${POSTGRES_POSTGRES_PASSWORD_FILE:-${POSTGRES_PASSWORD_FILE:-}}"
      if [[ -n "$password_file" && -f "$password_file" ]]; then
        export PGPASSWORD="$(cat "$password_file")"
      else
        export PGPASSWORD="${POSTGRES_PASSWORD:-}"
      fi
      psql -v ON_ERROR_STOP=1 -U postgres -d postgres -c '"${escaped_sql}"'
    '
}

run_k8s_psql_cmd_on_db() {
  local pod="$1"
  local db_name="$2"
  local sql="$3"
  local escaped_sql escaped_db
  escaped_sql="$(printf '%q' "$sql")"
  escaped_db="$(printf '%q' "$db_name")"

  kubectl exec -n "$K8S_NAMESPACE" "$pod" -- bash -lc \
    '
      password_file="${POSTGRES_POSTGRES_PASSWORD_FILE:-${POSTGRES_PASSWORD_FILE:-}}"
      if [[ -n "$password_file" && -f "$password_file" ]]; then
        export PGPASSWORD="$(cat "$password_file")"
      else
        export PGPASSWORD="${POSTGRES_PASSWORD:-}"
      fi
      psql -v ON_ERROR_STOP=1 -U postgres -d '"${escaped_db}"' -c '"${escaped_sql}"'
    '
}

run_k8s_psql_query() {
  local pod="$1"
  local sql="$2"
  local escaped_sql
  escaped_sql="$(printf '%q' "$sql")"

  kubectl exec -n "$K8S_NAMESPACE" "$pod" -- bash -lc \
    '
      password_file="${POSTGRES_POSTGRES_PASSWORD_FILE:-${POSTGRES_PASSWORD_FILE:-}}"
      if [[ -n "$password_file" && -f "$password_file" ]]; then
        export PGPASSWORD="$(cat "$password_file")"
      else
        export PGPASSWORD="${POSTGRES_PASSWORD:-}"
      fi
      psql -tA -U postgres -d postgres -c '"${escaped_sql}"'
    '
}

run_compose_psql_cmd_via_docker() {
  local sql="$1"

  docker compose exec -T "$COMPOSE_DB_SERVICE" env PGPASSWORD="$COMPOSE_DB_PASSWORD" \
    psql -v ON_ERROR_STOP=1 -U "$COMPOSE_DB_USER" -d "$COMPOSE_DB_NAME" -c "$sql"
}

run_compose_psql_query_via_docker() {
  local sql="$1"

  docker compose exec -T "$COMPOSE_DB_SERVICE" env PGPASSWORD="$COMPOSE_DB_PASSWORD" \
    psql -tA -U "$COMPOSE_DB_USER" -d "$COMPOSE_DB_NAME" -c "$sql"
}

run_compose_psql_cmd_on_db_via_docker() {
  local db_name="$1"
  local sql="$2"

  docker compose exec -T "$COMPOSE_DB_SERVICE" env PGPASSWORD="$COMPOSE_DB_PASSWORD" \
    psql -v ON_ERROR_STOP=1 -U "$COMPOSE_DB_USER" -d "$db_name" -c "$sql"
}

run_compose_psql_cmd_direct() {
  local sql="$1"

  PGPASSWORD="$COMPOSE_DB_PASSWORD" psql \
    -v ON_ERROR_STOP=1 \
    -h "$COMPOSE_DB_HOST" \
    -p "$COMPOSE_DB_PORT" \
    -U "$COMPOSE_DB_USER" \
    -d "$COMPOSE_DB_NAME" \
    -c "$sql"
}

run_compose_psql_query_direct() {
  local sql="$1"

  PGPASSWORD="$COMPOSE_DB_PASSWORD" psql \
    -tA \
    -h "$COMPOSE_DB_HOST" \
    -p "$COMPOSE_DB_PORT" \
    -U "$COMPOSE_DB_USER" \
    -d "$COMPOSE_DB_NAME" \
    -c "$sql"
}

run_compose_psql_cmd_direct_on_db() {
  local db_name="$1"
  local sql="$2"

  PGPASSWORD="$COMPOSE_DB_PASSWORD" psql \
    -v ON_ERROR_STOP=1 \
    -h "$COMPOSE_DB_HOST" \
    -p "$COMPOSE_DB_PORT" \
    -U "$COMPOSE_DB_USER" \
    -d "$db_name" \
    -c "$sql"
}

run_compose_psql_cmd() {
  local sql="$1"

  if command -v docker >/dev/null 2>&1; then
    run_compose_psql_cmd_via_docker "$sql"
    return
  fi

  if command -v psql >/dev/null 2>&1; then
    run_compose_psql_cmd_direct "$sql"
    return
  fi

  log "docker or psql is required to manage compose databases"
  return 1
}

run_compose_psql_cmd_on_db() {
  local db_name="$1"
  local sql="$2"

  if command -v docker >/dev/null 2>&1; then
    run_compose_psql_cmd_on_db_via_docker "$db_name" "$sql"
    return
  fi

  if command -v psql >/dev/null 2>&1; then
    run_compose_psql_cmd_direct_on_db "$db_name" "$sql"
    return
  fi

  log "docker or psql is required to manage compose databases"
  return 1
}

run_compose_psql_query() {
  local sql="$1"

  if command -v docker >/dev/null 2>&1; then
    run_compose_psql_query_via_docker "$sql"
    return
  fi

  if command -v psql >/dev/null 2>&1; then
    run_compose_psql_query_direct "$sql"
    return
  fi

  log "docker or psql is required to manage compose databases"
  return 1
}

drop_k8s_database() {
  local pod="$1"
  local db_name="$2"

  run_k8s_psql_cmd "$pod" "$(psql_terminate_connections_sql "$db_name")"
  run_k8s_psql_cmd "$pod" "$(psql_drop_sql "$db_name")"
}

drop_compose_database() {
  local db_name="$1"

  run_compose_psql_cmd "$(psql_terminate_connections_sql "$db_name")"
  run_compose_psql_cmd "$(psql_drop_sql "$db_name")"
}

sync_k8s() {
  local pod
  local db_name
  pod="$(wait_for_k8s_postgres)"

  while IFS= read -r db_name; do
    [[ -z "$db_name" ]] && continue
    if ! validate_db_name "$db_name"; then
      log "invalid database name in ${SECRETS_FILE}: ${db_name}"
      return 1
    fi
    if [[ "$(run_k8s_psql_query "$pod" "$(psql_exists_sql "$db_name")")" != "1" ]]; then
      log "creating k8s database ${db_name}"
      run_k8s_psql_cmd "$pod" "$(psql_create_sql "$db_name")"
    fi
    log "granting k8s database ${db_name}"
    run_k8s_psql_cmd "$pod" "$(psql_grant_sql "$db_name" "$K8S_DB_OWNER")"
    log "granting k8s schema public on ${db_name}"
    run_k8s_psql_cmd_on_db "$pod" "$db_name" "$(psql_schema_grant_sql "$K8S_DB_OWNER")"
  done < <(extract_k8s_databases)
}

sync_compose() {
  local db_name

  while IFS= read -r db_name; do
    [[ -z "$db_name" ]] && continue
    if ! validate_db_name "$db_name"; then
      log "invalid database name in ${COMPOSE_FILE}: ${db_name}"
      return 1
    fi
    if [[ "$(run_compose_psql_query "$(psql_exists_sql "$db_name")")" != "1" ]]; then
      log "creating compose database ${db_name}"
      run_compose_psql_cmd "$(psql_create_sql "$db_name")"
    fi
    log "granting compose database ${db_name}"
    run_compose_psql_cmd "$(psql_grant_sql "$db_name" "$COMPOSE_DB_OWNER")"
    log "granting compose schema public on ${db_name}"
    run_compose_psql_cmd_on_db "$db_name" "$(psql_schema_grant_sql "$COMPOSE_DB_OWNER")"
  done < <(extract_compose_databases)
}

drop_k8s() {
  local db_name="$1"
  local pod

  if ! validate_db_name "$db_name"; then
    log "invalid database name: ${db_name}"
    return 1
  fi

  pod="$(wait_for_k8s_postgres)"
  log "dropping k8s database ${db_name}"
  drop_k8s_database "$pod" "$db_name"
}

drop_compose() {
  local db_name="$1"

  if ! validate_db_name "$db_name"; then
    log "invalid database name: ${db_name}"
    return 1
  fi

  log "dropping compose database ${db_name}"
  drop_compose_database "$db_name"
}

main() {
  local cmd="${1:-}"

  case "$cmd" in
    sync-k8s)
      sync_k8s
      ;;
    sync-compose)
      sync_compose
      ;;
    drop-k8s)
      [[ $# -eq 2 ]] || usage
      drop_k8s "$2"
      ;;
    drop-compose)
      [[ $# -eq 2 ]] || usage
      drop_compose "$2"
      ;;
    *)
      usage
      ;;
  esac
}

main "$@"
