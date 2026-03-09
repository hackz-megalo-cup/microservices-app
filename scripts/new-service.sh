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
  local topic_const="Topic${SERVICE_NAME_PASCAL}Created"
  local topic_value="${SERVICE_NAME_SNAKE}.created"
  local topic_dlq_const="Topic${SERVICE_NAME_PASCAL}CreatedDLQ"
  local topic_dlq_value="${SERVICE_NAME_SNAKE}.created.dlq"

  # Add topic constants before the DLQ section
  local tmp_file
  tmp_file=$(mktemp)
  awk -v tc="\t${topic_const}   = \"${topic_value}\"" \
      '/\/\/ Dead Letter Queue topics\./ { print tc; print ""; }
       { print }' "$topics_file" > "$tmp_file"
  mv "$tmp_file" "$topics_file"

  # Add DLQ constant before the closing paren
  tmp_file=$(mktemp)
  awk -v dlq="\t${topic_dlq_const}   = \"${topic_dlq_value}\"" \
      '/^)/ && !done { print dlq; done=1 }
       { print }' "$topics_file" > "$tmp_file"
  mv "$tmp_file" "$topics_file"

  # Add to DLQTopic mapping
  tmp_file=$(mktemp)
  awk -v mapping="\t\t${topic_const}:   ${topic_dlq_const}," \
      '/return m\[source\]/ { print mapping; }
       { print }' "$topics_file" > "$tmp_file"
  mv "$tmp_file" "$topics_file"

  # Add to DefaultTopics
  tmp_file=$(mktemp)
  awk -v main="\t\t${topic_const}:      3," \
      -v dlq="\t\t${topic_dlq_const}:   1," \
      '/}$/ && /return map/ { next }
       /TopicUserRegisteredDLQ:/ { print; print dlq; next }
       /TopicUserRegistered:.*3,/ && !/DLQ/ { print; print main; next }
       { print }' "$topics_file" > "$tmp_file"
  mv "$tmp_file" "$topics_file"

  echo "  Updated services/internal/platform/topics.go"
}

case "$LANG" in
  go)
    echo "==> Creating Go service: ${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/services/cmd/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/services/internal/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/services/internal/${SERVICE_NAME}/migrations"
    mkdir -p "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/deploy/k8s"

    apply_template "${TEMPLATES_DIR}/go/main.go.tmpl" "${REPO_ROOT}/services/cmd/${SERVICE_NAME}/main.go"
    apply_template "${TEMPLATES_DIR}/go/Dockerfile.dev.tmpl" "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}/Dockerfile.dev"
    apply_template "${TEMPLATES_DIR}/go/k8s.nix.tmpl" "${REPO_ROOT}/deploy/k8s/${SERVICE_NAME}.nix"

    # Generate boilerplate migrations
    if [[ -d "${TEMPLATES_DIR}/go/migrations" ]]; then
      for tmpl in "${TEMPLATES_DIR}/go/migrations/"*.tmpl; do
        local_name="$(basename "$tmpl" .tmpl)"
        apply_template "$tmpl" "${REPO_ROOT}/services/internal/${SERVICE_NAME}/migrations/${local_name}"
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
    echo "  services/cmd/${SERVICE_NAME}/main.go"
    echo "  services/internal/${SERVICE_NAME}/  (add your service implementation)"
    echo "  services/internal/${SERVICE_NAME}/migrations/  (idempotency + outbox)"
    echo "  deploy/docker/${SERVICE_NAME}/Dockerfile.dev"
    echo "  deploy/k8s/${SERVICE_NAME}.nix"
    echo "  proto/${SERVICE_NAME}/v1/${SERVICE_NAME}.proto"
    echo ""
    echo "Auto-wired:"
    echo "  docker-compose.yml  (service entry added)"
    echo "  scripts/init-db.sh  (database added)"
    echo "  services/internal/platform/topics.go  (topic added)"
    echo ""
    echo "Next steps:"
    echo "  1. Implement services/internal/${SERVICE_NAME}/service.go"
    echo "  2. Add the nixidy import to deploy/nixidy/env/local.nix"
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
