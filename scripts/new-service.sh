#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

usage() {
  echo "Usage: new-service <lang> <service-name> [port]"
  echo ""
  echo "  lang:         go | custom"
  echo "  service-name: kebab-case name (e.g. my-service)"
  echo "  port:         optional port number (default: 8080)"
  echo ""
  echo "Examples:"
  echo "  new-service go my-service"
  echo "  new-service custom my-lang-service 3001"
  exit 1
}

if [[ $# -lt 2 ]]; then
  usage
fi

LANG="$1"
SERVICE_NAME="$2"
PORT="${3:-8080}"

# Convert kebab-case to PascalCase
to_pascal() {
  echo "$1" | sed -E 's/(^|-)([a-z])/\U\2/g'
}

SERVICE_NAME_PASCAL="$(to_pascal "$SERVICE_NAME")"
# Convert kebab-case to snake_case for Go package names
SERVICE_NAME_SNAKE="$(echo "$SERVICE_NAME" | tr '-' '_')"

TEMPLATES_DIR="${REPO_ROOT}/templates"

apply_template() {
  local src="$1"
  local dst="$2"
  sed -e "s/__SERVICE_NAME__/${SERVICE_NAME}/g" \
      -e "s/__SERVICE_NAME_PASCAL__/${SERVICE_NAME_PASCAL}/g" \
      -e "s/__SERVICE_NAME_SNAKE__/${SERVICE_NAME_SNAKE}/g" \
      -e "s/__PORT__/${PORT}/g" \
      "$src" > "$dst"
}

# --- Auto-wiring helpers ---

add_to_docker_compose() {
  local tmpl="$1"
  local compose_file="${REPO_ROOT}/docker-compose.yml"
  local rendered
  rendered=$(sed -e "s/__SERVICE_NAME__/${SERVICE_NAME}/g" \
                 -e "s/__SERVICE_NAME_PASCAL__/${SERVICE_NAME_PASCAL}/g" \
                 -e "s/__SERVICE_NAME_SNAKE__/${SERVICE_NAME_SNAKE}/g" \
                 -e "s/__PORT__/${PORT}/g" \
                 "$tmpl")
  # Insert before the 'networks:' line at root level (not indented)
  local tmp_file
  tmp_file=$(mktemp)
  awk -v entry="$rendered" '/^networks:/ { print entry; print ""; } { print }' "$compose_file" > "$tmp_file"
  mv "$tmp_file" "$compose_file"
  echo "  Updated docker-compose.yml"
}

add_to_init_db() {
  local db_name="${SERVICE_NAME_SNAKE}_db"
  local init_file="${REPO_ROOT}/scripts/init-db.sh"
  # Insert CREATE DATABASE before the first EOSQL
  local tmp_file
  tmp_file=$(mktemp)
  awk -v db="    CREATE DATABASE ${db_name};" 'NR==1{found=0} /EOSQL/ && !found { print db; found=1 } { print }' "$init_file" > "$tmp_file"
  mv "$tmp_file" "$init_file"
  chmod +x "$init_file"
  echo "  Updated scripts/init-db.sh"
}

add_to_topics() {
  local topics_file="${REPO_ROOT}/services/internal/platform/topics.go"

  local created_const="Topic${SERVICE_NAME_PASCAL}Created"
  local created_value="${SERVICE_NAME_SNAKE}.created"
  local failed_const="Topic${SERVICE_NAME_PASCAL}Failed"
  local failed_value="${SERVICE_NAME_SNAKE}.failed"
  local compensated_const="Topic${SERVICE_NAME_PASCAL}Compensated"
  local compensated_value="${SERVICE_NAME_SNAKE}.compensated"
  local dlq_const="Topic${SERVICE_NAME_PASCAL}CreatedDLQ"
  local dlq_value="${SERVICE_NAME_SNAKE}.created.dlq"

  # Add topic constants before the "Dead Letter Queue" comment
  local tmp_file
  tmp_file=$(mktemp)
  awk -v t1="\t${created_const}   = \"${created_value}\"" \
      -v t2="\t${failed_const}   = \"${failed_value}\"" \
      -v t3="\t${compensated_const}   = \"${compensated_value}\"" \
      '/\/\/ Dead Letter Queue topics\./ { print t1; print t2; print t3; print ""; }
       { print }' "$topics_file" > "$tmp_file"
  mv "$tmp_file" "$topics_file"

  # Add DLQ constant before the const block closing paren (skip import block's paren)
  tmp_file=$(mktemp)
  awk -v dlq="\t${dlq_const}   = \"${dlq_value}\"" \
      'BEGIN { in_dlq=0 }
       /\/\/ Dead Letter Queue topics\./ { in_dlq=1 }
       /^\)/ && in_dlq { print dlq; in_dlq=0 }
       { print }' "$topics_file" > "$tmp_file"
  mv "$tmp_file" "$topics_file"

  # Add to DLQTopic mapping (before the map literal's closing brace)
  tmp_file=$(mktemp)
  awk -v mapping="\t\t${created_const}:   ${dlq_const}," \
      'BEGIN { in_dlq_func=0 }
       /func DLQTopic/ { in_dlq_func=1 }
       /^\t\}/ && in_dlq_func { print mapping; in_dlq_func=0 }
       { print }' "$topics_file" > "$tmp_file"
  mv "$tmp_file" "$topics_file"

  # Add to DefaultTopics (before the map literal's closing brace)
  tmp_file=$(mktemp)
  awk -v main="\t\t${created_const}:      3," \
      -v failed="\t\t${failed_const}:         1," \
      -v comp="\t\t${compensated_const}:   1," \
      -v dlq="\t\t${dlq_const}:   1," \
      'BEGIN { in_defaults=0 }
       /func DefaultTopics/ { in_defaults=1 }
       /^\t\}/ && in_defaults { print main; print failed; print comp; print dlq; in_defaults=0 }
       { print }' "$topics_file" > "$tmp_file"
  mv "$tmp_file" "$topics_file"

  echo "  Updated services/internal/platform/topics.go"
}

case "$LANG" in
  go)
    echo "==> Creating Go service: ${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/services/cmd/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/services/internal/${SERVICE_NAME_SNAKE}"
    mkdir -p "${REPO_ROOT}/services/internal/${SERVICE_NAME_SNAKE}/migrations"
    mkdir -p "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/deploy/k8s"

    apply_template "${TEMPLATES_DIR}/go/main.go.tmpl" "${REPO_ROOT}/services/cmd/${SERVICE_NAME}/main.go"
    apply_template "${TEMPLATES_DIR}/go/embed.go.tmpl" "${REPO_ROOT}/services/internal/${SERVICE_NAME_SNAKE}/embed.go"
    apply_template "${TEMPLATES_DIR}/go/events.go.tmpl" "${REPO_ROOT}/services/internal/${SERVICE_NAME_SNAKE}/events.go"
    apply_template "${TEMPLATES_DIR}/go/aggregate.go.tmpl" "${REPO_ROOT}/services/internal/${SERVICE_NAME_SNAKE}/aggregate.go"
    apply_template "${TEMPLATES_DIR}/go/service.go.tmpl" "${REPO_ROOT}/services/internal/${SERVICE_NAME_SNAKE}/service.go"
    apply_template "${TEMPLATES_DIR}/go/Dockerfile.tmpl" "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}/Dockerfile"
    apply_template "${TEMPLATES_DIR}/go/Dockerfile.dev.tmpl" "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}/Dockerfile.dev"
    apply_template "${TEMPLATES_DIR}/go/k8s.nix.tmpl" "${REPO_ROOT}/deploy/k8s/${SERVICE_NAME}.nix"

    # Generate boilerplate migrations
    if [[ -d "${TEMPLATES_DIR}/go/migrations" ]]; then
      for tmpl in "${TEMPLATES_DIR}/go/migrations/"*.tmpl; do
        local_name="$(basename "$tmpl" .tmpl)"
        apply_template "$tmpl" "${REPO_ROOT}/services/internal/${SERVICE_NAME_SNAKE}/migrations/${local_name}"
      done
      echo "  Generated migrations"
    fi

    echo "==> Creating proto definition"
    mkdir -p "${REPO_ROOT}/proto/${SERVICE_NAME}/v1"
    apply_template "${TEMPLATES_DIR}/proto.tmpl" "${REPO_ROOT}/proto/${SERVICE_NAME}/v1/${SERVICE_NAME}.proto"

    echo "==> Running buf generate..."
    (cd "${REPO_ROOT}" && buf generate)

    echo "==> Auto-wiring integrations..."
    add_to_docker_compose "${TEMPLATES_DIR}/docker-compose-entry.go.yml.tmpl"
    add_to_init_db
    add_to_topics

    echo ""
    echo "Created Go service '${SERVICE_NAME}' (port ${PORT})."
    echo "Files:"
    echo "  services/cmd/${SERVICE_NAME}/main.go          (DON'T TOUCH - infrastructure wiring)"
    echo "  services/internal/${SERVICE_NAME_SNAKE}/embed.go      (DON'T TOUCH)"
    echo "  services/internal/${SERVICE_NAME_SNAKE}/events.go     (EDIT - define your events)"
    echo "  services/internal/${SERVICE_NAME_SNAKE}/aggregate.go  (EDIT - define state + Apply)"
    echo "  services/internal/${SERVICE_NAME_SNAKE}/service.go    (EDIT - implement business logic)"
    echo "  services/internal/${SERVICE_NAME_SNAKE}/migrations/"
    echo "  deploy/docker/${SERVICE_NAME}/Dockerfile.dev"
    echo "  deploy/k8s/${SERVICE_NAME}.nix"
    echo "  proto/${SERVICE_NAME}/v1/${SERVICE_NAME}.proto"
    echo ""
    echo "Auto-wired:"
    echo "  docker-compose.yml  (service entry added)"
    echo "  scripts/init-db.sh  (database added)"
    echo "  services/internal/platform/topics.go  (topics added)"
    echo ""
    echo "Next steps:"
    echo "  1. Edit proto/${SERVICE_NAME}/v1/${SERVICE_NAME}.proto (define your API)"
    echo "  2. Run: buf generate"
    echo "  3. Edit services/internal/${SERVICE_NAME_SNAKE}/events.go (define your events)"
    echo "  4. Edit services/internal/${SERVICE_NAME_SNAKE}/aggregate.go (define state + Apply)"
    echo "  5. Edit services/internal/${SERVICE_NAME_SNAKE}/service.go (implement business logic)"
    echo "  6. Run: docker compose up ${SERVICE_NAME}"
    ;;

  custom)
    echo "==> Creating custom (Node.js) service: ${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/node-services/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/deploy/k8s"

    apply_template "${TEMPLATES_DIR}/custom/server.js.tmpl" "${REPO_ROOT}/node-services/${SERVICE_NAME}/server.js"
    apply_template "${TEMPLATES_DIR}/custom/Dockerfile.tmpl" "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}/Dockerfile"
    apply_template "${TEMPLATES_DIR}/custom/k8s.nix.tmpl" "${REPO_ROOT}/deploy/k8s/${SERVICE_NAME}.nix"

    # Init package.json with shared dependency
    (cd "${REPO_ROOT}/node-services/${SERVICE_NAME}" && npm init -y --silent)
    # Add @microservices/shared workspace dependency
    local pkg="${REPO_ROOT}/node-services/${SERVICE_NAME}/package.json"
    if command -v node &>/dev/null; then
      node -e "
        const fs = require('fs');
        const pkg = JSON.parse(fs.readFileSync('$pkg', 'utf8'));
        pkg.type = 'module';
        pkg.dependencies = pkg.dependencies || {};
        pkg.dependencies['@microservices/shared'] = 'workspace:*';
        fs.writeFileSync('$pkg', JSON.stringify(pkg, null, 2) + '\n');
      "
    fi

    echo "==> Auto-wiring integrations..."
    add_to_docker_compose "${TEMPLATES_DIR}/docker-compose-entry.custom.yml.tmpl"
    add_to_init_db

    echo ""
    echo "Created custom service '${SERVICE_NAME}' (port ${PORT})."
    echo "Files:"
    echo "  node-services/${SERVICE_NAME}/server.js"
    echo "  deploy/docker/${SERVICE_NAME}/Dockerfile"
    echo "  node-services/${SERVICE_NAME}/package.json"
    echo "  deploy/k8s/${SERVICE_NAME}.nix"
    echo ""
    echo "Auto-wired:"
    echo "  docker-compose.yml  (service entry added)"
    echo "  scripts/init-db.sh  (database added)"
    echo ""
    echo "Next steps:"
    echo "  1. Add the nixidy import to deploy/nixidy/env/local.nix"
    ;;

  *)
    echo "Error: Unknown language '${LANG}'. Use 'go' or 'custom'."
    usage
    ;;
esac
