#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
SYSTEM="$(nix eval --raw --impure --expr 'builtins.currentSystem')"

echo "==> Building nixidy manifests..."
nix build "${REPO_ROOT}#legacyPackages.${SYSTEM}.nixidyEnvs.local.environmentPackage" -o "${REPO_ROOT}/manifests-result"

echo "==> Copying to deploy/manifests/..."
rm -rf "${REPO_ROOT}/deploy/manifests"
cp -rL "${REPO_ROOT}/manifests-result" "${REPO_ROOT}/deploy/manifests"
chmod -R u+w "${REPO_ROOT}/deploy/manifests"

# ArgoCD は argocd-bootstrap で手動デプロイするため、Application から除外
rm -f "${REPO_ROOT}/deploy/manifests/apps/Application-argocd.yaml"

echo "==> Done. deploy/manifests/ updated."
echo ""
git -C "${REPO_ROOT}" diff --stat -- deploy/manifests/
