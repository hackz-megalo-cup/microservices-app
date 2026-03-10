#!/usr/bin/env bash
# Sync Go vendor directory with go.mod
# Usage: bash scripts/sync-vendor.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== Syncing Go vendor ==="
(cd "$REPO_ROOT/services" && go mod tidy && go mod vendor)
echo "=== Done ==="

if git -C "$REPO_ROOT" diff --quiet services/vendor/; then
  echo "vendor/ is already in sync."
else
  echo "vendor/ updated. Staging changes..."
  git -C "$REPO_ROOT" add services/vendor/ services/go.mod services/go.sum
  echo "Ready to commit."
fi
