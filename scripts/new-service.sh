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

case "$LANG" in
  go)
    echo "==> Creating Go service: ${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/services/cmd/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/services/internal/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/deploy/k8s"

    apply_template "${TEMPLATES_DIR}/go/main.go.tmpl" "${REPO_ROOT}/services/cmd/${SERVICE_NAME}/main.go"
    apply_template "${TEMPLATES_DIR}/go/Dockerfile.dev.tmpl" "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}/Dockerfile.dev"
    apply_template "${TEMPLATES_DIR}/go/k8s.nix.tmpl" "${REPO_ROOT}/deploy/k8s/${SERVICE_NAME}.nix"

    echo "==> Creating proto definition"
    mkdir -p "${REPO_ROOT}/proto/${SERVICE_NAME}/v1"
    apply_template "${TEMPLATES_DIR}/proto.tmpl" "${REPO_ROOT}/proto/${SERVICE_NAME}/v1/${SERVICE_NAME}.proto"

    echo "==> Running buf generate..."
    (cd "${REPO_ROOT}" && buf generate)

    echo ""
    echo "Created Go service '${SERVICE_NAME}' (port ${PORT})."
    echo "Files:"
    echo "  services/cmd/${SERVICE_NAME}/main.go"
    echo "  services/internal/${SERVICE_NAME}/  (add your service implementation)"
    echo "  deploy/docker/${SERVICE_NAME}/Dockerfile.dev"
    echo "  deploy/k8s/${SERVICE_NAME}.nix"
    echo "  proto/${SERVICE_NAME}/v1/${SERVICE_NAME}.proto"
    echo ""
    echo "Next steps:"
    echo "  1. Implement services/internal/${SERVICE_NAME}/service.go"
    echo "  2. Add the nixidy import to deploy/nixidy/env/local.nix"
    echo "  3. Add to Tiltfile"
    ;;

  custom)
    echo "==> Creating custom (Node.js) service: ${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/node-services/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}"
    mkdir -p "${REPO_ROOT}/deploy/k8s"

    apply_template "${TEMPLATES_DIR}/custom/server.js.tmpl" "${REPO_ROOT}/node-services/${SERVICE_NAME}/server.js"
    apply_template "${TEMPLATES_DIR}/custom/Dockerfile.tmpl" "${REPO_ROOT}/deploy/docker/${SERVICE_NAME}/Dockerfile"
    apply_template "${TEMPLATES_DIR}/custom/k8s.nix.tmpl" "${REPO_ROOT}/deploy/k8s/${SERVICE_NAME}.nix"

    # Init package.json
    (cd "${REPO_ROOT}/node-services/${SERVICE_NAME}" && npm init -y --silent)

    echo ""
    echo "Created custom service '${SERVICE_NAME}' (port ${PORT})."
    echo "Files:"
    echo "  node-services/${SERVICE_NAME}/server.js"
    echo "  deploy/docker/${SERVICE_NAME}/Dockerfile"
    echo "  node-services/${SERVICE_NAME}/package.json"
    echo "  deploy/k8s/${SERVICE_NAME}.nix"
    echo ""
    echo "Next steps:"
    echo "  1. Add the nixidy import to deploy/nixidy/env/local.nix"
    echo "  2. Add to Tiltfile"
    ;;

  *)
    echo "Error: Unknown language '${LANG}'. Use 'go' or 'custom'."
    usage
    ;;
esac
