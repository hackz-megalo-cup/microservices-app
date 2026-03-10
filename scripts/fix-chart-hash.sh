#!/usr/bin/env bash
set -euo pipefail

# nixhelmに無いチャートのchartHashを自動で埋めるスクリプト
# Usage: fix-chart-hash
#
# nix build を実行し、hash mismatch エラーから正しいハッシュを抽出して
# 対応する .nix ファイルの chartHash を自動で書き換える。
# エラーが無くなるまで繰り返す。

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

MAX_ITERATIONS=10
ITERATION=0

while [ "$ITERATION" -lt "$MAX_ITERATIONS" ]; do
  ITERATION=$((ITERATION + 1))
  echo "=== Attempt ${ITERATION}/${MAX_ITERATIONS} ==="

  BUILD_OUTPUT=$(nix build "${REPO_ROOT}#legacyPackages.$(nix eval --raw --impure --expr 'builtins.currentSystem').nixidyEnvs.local.environmentPackage" 2>&1) && {
    echo "Build succeeded!"
    exit 0
  }

  # hash mismatch を検出
  GOT_HASH=$(echo "$BUILD_OUTPUT" | grep "got:" | head -1 | awk '{print $2}')

  if [ -z "$GOT_HASH" ]; then
    echo "No hash mismatch found. Build error:"
    echo "$BUILD_OUTPUT" | tail -20
    exit 1
  fi

  echo "Found correct hash: ${GOT_HASH}"

  # chartHash = "" または chartHash = "sha256-AAAA..." のパターンを置換
  # 全 .nix ファイルを走査して空のchartHashを見つける
  FIXED=false
  while IFS= read -r -d '' file; do
    if grep -q 'chartHash = ""' "$file" || grep -q 'chartHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="' "$file"; then
      if sed --version >/dev/null 2>&1; then
        sed -i "s|chartHash = \".*\"|chartHash = \"${GOT_HASH}\"|" "$file"
      else
        sed -i '' "s|chartHash = \".*\"|chartHash = \"${GOT_HASH}\"|" "$file"
      fi
      echo "Updated: ${file}"
      FIXED=true
      break
    fi
  done < <(find "${REPO_ROOT}/nixidy" -name '*.nix' -print0)

  if [ "$FIXED" = false ]; then
    echo "Could not find a file with empty chartHash to update."
    echo "Build error:"
    echo "$BUILD_OUTPUT" | tail -20
    exit 1
  fi
done

echo "Exceeded max iterations (${MAX_ITERATIONS}). Something is wrong."
exit 1
