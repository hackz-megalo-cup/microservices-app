#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

usage() {
  echo "Usage: delete-service <service-name>"
  echo ""
  echo "  Removes all files and wiring created by new-service."
  echo ""
  echo "Examples:"
  echo "  delete-service order"
  echo "  delete-service my-service"
  exit 1
}

if [[ $# -lt 1 ]]; then
  usage
fi

SERVICE_NAME="$1"

# Convert kebab-case to PascalCase
to_pascal() {
  echo "$1" | awk -F'-' '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2)}1' OFS=''
}

SERVICE_NAME_PASCAL="$(to_pascal "$SERVICE_NAME")"
SERVICE_NAME_SNAKE="$(echo "$SERVICE_NAME" | tr '-' '_')"

# --- Remove files ---

remove_dirs() {
  local dirs=(
    # Go service
    "${REPO_ROOT}/services/cmd/${SERVICE_NAME}"
    "${REPO_ROOT}/services/internal/${SERVICE_NAME_SNAKE}"
    "${REPO_ROOT}/services/gen/go/${SERVICE_NAME}"
    "${REPO_ROOT}/proto/${SERVICE_NAME}"
    # Custom (Node.js) service
    "${REPO_ROOT}/node-services/${SERVICE_NAME}"
    # Shared
    "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}"
    # Frontend generated
    "${REPO_ROOT}/frontend/src/gen/${SERVICE_NAME}"
  )

  for dir in "${dirs[@]}"; do
    if [[ -d "$dir" ]]; then
      rm -rf "$dir"
      echo "  Removed $dir"
    fi
  done

  # Single files
  local files=(
    "${REPO_ROOT}/deploy/k8s/${SERVICE_NAME}.nix"
    "${REPO_ROOT}/services/${SERVICE_NAME}"
  )

  for file in "${files[@]}"; do
    if [[ -f "$file" ]]; then
      rm -f "$file"
      echo "  Removed $file"
    fi
  done
}

# --- Remove from docker-compose.yml ---

remove_from_docker_compose() {
  local compose_file="${REPO_ROOT}/docker-compose.yml"
  local tmp_file
  tmp_file=$(mktemp)

  # Remove the service block: starts with "  <service-name>:" and continues
  # until the next top-level key (no indent) or next service (2-space indent).
  awk -v svc="${SERVICE_NAME}" '
    BEGIN { skip=0 }
    {
      # Match "  <service-name>:" at start of line (exactly 2 spaces)
      if ($0 ~ "^  " svc ":") { skip=1; next }
      if (skip) {
        # Stop skipping at root-level key or next service definition
        if (/^[^ ]/ || /^  [a-zA-Z]/) { skip=0 }
        else { next }
      }
      print
    }
  ' "$compose_file" > "$tmp_file"

  # Clean up double blank lines left behind
  awk 'NR==1{print; blank=0; next} /^$/{blank++; if(blank<=1) print; next} {blank=0; print}' "$tmp_file" > "${tmp_file}.clean"
  mv "${tmp_file}.clean" "$compose_file"
  rm -f "$tmp_file"
  echo "  Updated docker-compose.yml"
}

# --- Remove from init-db.sh ---

remove_from_init_db() {
  local db_name="${SERVICE_NAME_SNAKE}_db"
  local init_file="${REPO_ROOT}/scripts/init-db.sh"
  local tmp_file
  tmp_file=$(mktemp)

  grep -v "CREATE DATABASE ${db_name};" "$init_file" > "$tmp_file"
  mv "$tmp_file" "$init_file"
  chmod +x "$init_file"
  echo "  Updated scripts/init-db.sh"
}

# --- Remove from topics.go ---

remove_from_topics() {
  local topics_file="${REPO_ROOT}/services/internal/platform/topics.go"

  if [[ ! -f "$topics_file" ]]; then
    return
  fi

  local pascal="${SERVICE_NAME_PASCAL}"
  local tmp_file
  tmp_file=$(mktemp)

  # Remove all lines containing Topic<Pascal>Created, Topic<Pascal>Failed,
  # Topic<Pascal>Compensated, Topic<Pascal>CreatedDLQ (constants + map entries)
  grep -v "Topic${pascal}Created\|Topic${pascal}Failed\|Topic${pascal}Compensated\|Topic${pascal}CreatedDLQ" \
    "$topics_file" > "$tmp_file"

  # Clean up leftover double blank lines
  awk 'NR==1{print; blank=0; next} /^$/{blank++; if(blank<=1) print; next} {blank=0; print}' "$tmp_file" > "${tmp_file}.clean"
  mv "${tmp_file}.clean" "$topics_file"
  rm -f "$tmp_file"
  echo "  Updated services/internal/platform/topics.go"
}

# --- Remove from secrets.nix ---

remove_from_secrets() {
  local secrets_file="${REPO_ROOT}/deploy/k8s/secrets.nix"

  if [[ ! -f "$secrets_file" ]]; then
    return
  fi

  local secret_name="${SERVICE_NAME}-secrets"
  local tmp_file
  tmp_file=$(mktemp)

  # Remove the block: "      <name>-secrets = { ... };" (6-space indent)
  awk -v sn="$secret_name" '
    BEGIN { skip=0 }
    $0 ~ "^      " sn " = \\{" { skip=1; next }
    skip && /^      };$/ { skip=0; next }
    skip { next }
    { print }
  ' "$secrets_file" > "$tmp_file"

  # Clean up double blank lines
  awk 'NR==1{print; blank=0; next} /^$/{blank++; if(blank<=1) print; next} {blank=0; print}' "$tmp_file" > "${tmp_file}.clean"
  mv "${tmp_file}.clean" "$secrets_file"
  rm -f "$tmp_file"
  echo "  Updated deploy/k8s/secrets.nix"
}

# --- Remove from local.nix ---

remove_from_local_nix() {
  local local_nix="${REPO_ROOT}/deploy/nixidy/env/local.nix"

  if [[ ! -f "$local_nix" ]]; then
    return
  fi

  local tmp_file
  tmp_file=$(mktemp)

  grep -v "../../k8s/${SERVICE_NAME}.nix" "$local_nix" > "$tmp_file"
  mv "$tmp_file" "$local_nix"
  echo "  Updated deploy/nixidy/env/local.nix"
}

# --- Stop and remove Docker container/image ---

stop_docker() {
  local container="microservices-app-${SERVICE_NAME}-1"

  if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
    echo "  Stopping container ${container}..."
    docker rm -f "$container" 2>/dev/null || true
    echo "  Removed container ${container}"
  fi

  local image="microservices-app-${SERVICE_NAME}"
  if docker images --format '{{.Repository}}' | grep -q "^${image}$"; then
    docker rmi "$image" 2>/dev/null || true
    echo "  Removed image ${image}"
  fi
}

# --- Main ---

echo "==> Deleting service: ${SERVICE_NAME}"

echo "==> Stopping Docker container..."
stop_docker

echo "==> Removing files..."
remove_dirs

echo "==> Removing wiring..."
remove_from_docker_compose
remove_from_init_db
remove_from_topics
remove_from_secrets
remove_from_local_nix

echo "==> Staging nix changes..."
(cd "${REPO_ROOT}" && \
  git rm -f --cached "deploy/k8s/${SERVICE_NAME}.nix" 2>/dev/null || true
  git add "deploy/nixidy/env/local.nix" "deploy/k8s/secrets.nix" 2>/dev/null || true
)

echo ""
echo "Deleted service '${SERVICE_NAME}'."
echo ""
echo "Removed:"
echo "  services/cmd/${SERVICE_NAME}/"
echo "  services/internal/${SERVICE_NAME_SNAKE}/"
echo "  services/gen/go/${SERVICE_NAME}/"
echo "  frontend/src/gen/${SERVICE_NAME}/"
echo "  proto/${SERVICE_NAME}/"
echo "  deploy/docker/${SERVICE_NAME}/"
echo "  deploy/k8s/${SERVICE_NAME}.nix"
echo "  node-services/${SERVICE_NAME}/ (if custom)"
echo ""
echo "Un-wired:"
echo "  docker-compose.yml  (service entry removed)"
echo "  scripts/init-db.sh  (database removed)"
echo "  services/internal/platform/topics.go  (topics removed)"
echo "  deploy/k8s/secrets.nix  (secrets removed)"
echo "  deploy/nixidy/env/local.nix  (nix import removed)"
