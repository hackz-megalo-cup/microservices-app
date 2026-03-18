#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TLS_DIR="$REPO_ROOT/deploy/tls"

echo "=== Local TLS Setup for game-server ==="

# Check mkcert
if ! command -v mkcert &>/dev/null; then
  echo "ERROR: mkcert is not installed."
  echo "  Install via: brew install mkcert  /  nix-env -i mkcert"
  exit 1
fi

# Install CA if not already
echo "==> Installing mkcert CA (may ask for password)..."
mkcert -install

# Generate certs
mkdir -p "$TLS_DIR"

if [ -f "$TLS_DIR/tls.crt" ] && [ -f "$TLS_DIR/tls.key" ]; then
  echo "==> Certs already exist at $TLS_DIR, skipping generation."
  echo "    Delete them and re-run to regenerate."
else
  echo "==> Generating localhost certs..."
  mkcert -key-file "$TLS_DIR/tls.key" -cert-file "$TLS_DIR/tls.crt" localhost 127.0.0.1 ::1
fi

# Also copy to /tmp for local (non-docker) dev
cp "$TLS_DIR/tls.crt" /tmp/tls.crt
cp "$TLS_DIR/tls.key" /tmp/tls.key

echo ""
echo "=== Done! ==="
echo "  Docker:  deploy/tls/ is mounted into game-server container"
echo "  Local:   /tmp/tls.{crt,key} ready for WS_CERT_PATH=/tmp"
echo ""
echo "  Run: docker compose up --build -d"
echo "  Or:  LOCAL_DEV=true WS_CERT_PATH=/tmp go run . (from game-server/)"
