# CI Optimization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Optimize CI to reduce runner time ~75-80% using Blacksmith runners, Cachix/sticky disk Nix caching, path filtering, and main-branch scoping.

**Architecture:** Single workflow file rewrite. Add concurrency control, a `changes` gate job with `dorny/paths-filter`, conditional job execution, Blacksmith runners, and Nix caching via Cachix + sticky disk. Lint/test jobs only on PRs; nix-build + render-manifests only on main.

**Tech Stack:** GitHub Actions, Blacksmith runners, Cachix, dorny/paths-filter, useblacksmith/stickydisk

---

### Task 1: Add concurrency control and path filter gate job

**Files:**
- Modify: `.github/workflows/ci.yml:1-9`

**Step 1: Add concurrency block and changes job**

Replace the top of `ci.yml` (lines 1-9) and add the `changes` job as the first job. Keep existing trigger structure but add concurrency.

```yaml
name: CI

on:
  workflow_dispatch:
  pull_request:
  push:
    branches:
      - main

concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.sha }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' }}

jobs:
  changes:
    runs-on: blacksmith-2vcpu-ubuntu-2404
    permissions:
      contents: read
      pull-requests: read
    outputs:
      proto: ${{ steps.filter.outputs.proto }}
      go: ${{ steps.filter.outputs.go }}
      frontend: ${{ steps.filter.outputs.frontend }}
      node: ${{ steps.filter.outputs.node }}
      nix: ${{ steps.filter.outputs.nix }}
    steps:
      - uses: actions/checkout@v4
      - uses: dorny/paths-filter@v3
        id: filter
        with:
          filters: |
            proto:
              - 'proto/**'
            go:
              - 'services/**'
              - 'proto/**'
            frontend:
              - 'frontend/**'
            node:
              - 'node-services/**'
            nix:
              - 'flake.nix'
              - 'flake.lock'
              - '**/*.nix'
```

**Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add concurrency control and path filter gate job"
```

---

### Task 2: Migrate lint/test jobs to Blacksmith with conditional execution

**Files:**
- Modify: `.github/workflows/ci.yml` (contract, go-lint, go-test, frontend-lint, frontend-build, node-lint, node-test jobs)

**Step 1: Update all lint/test jobs**

Each job gets:
- `needs: [changes]`
- `if` condition based on path filter output + NOT main push (lint/test only on PR/dispatch)
- `runs-on: blacksmith-4vcpu-ubuntu-2404`

```yaml
  contract:
    needs: [changes]
    if: >-
      needs.changes.outputs.proto == 'true'
      && github.event_name != 'push'
    runs-on: blacksmith-4vcpu-ubuntu-2404
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: bufbuild/buf-action@v1
      - name: Buf lint
        run: buf lint
      - name: Buf breaking
        run: |
          if git cat-file -e main:proto 2>/dev/null; then
            buf breaking --against '.git#branch=main'
          else
            echo "Skipping buf breaking: proto baseline does not exist on main yet"
          fi

  go-lint:
    needs: [changes]
    if: >-
      needs.changes.outputs.go == 'true'
      && github.event_name != 'push'
    runs-on: blacksmith-4vcpu-ubuntu-2404
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: services/go.mod
      - uses: golangci/golangci-lint-action@v8
        with:
          version: v2.10
          working-directory: services

  go-test:
    needs: [changes]
    if: >-
      needs.changes.outputs.go == 'true'
      && github.event_name != 'push'
    runs-on: blacksmith-4vcpu-ubuntu-2404
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: services/go.mod
      - name: Go test
        run: go test ./...
        working-directory: services

  frontend-lint:
    needs: [changes]
    if: >-
      needs.changes.outputs.frontend == 'true'
      && github.event_name != 'push'
    runs-on: blacksmith-4vcpu-ubuntu-2404
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "22"
          cache: npm
          cache-dependency-path: frontend/package-lock.json
      - run: npm ci
        working-directory: frontend
      - name: Biome check
        run: npx biome check src/
        working-directory: frontend

  frontend-build:
    needs: [changes]
    if: >-
      needs.changes.outputs.frontend == 'true'
      && github.event_name != 'push'
    runs-on: blacksmith-4vcpu-ubuntu-2404
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "22"
          cache: npm
          cache-dependency-path: frontend/package-lock.json
      - name: Frontend build
        run: |
          npm ci
          npm run build
        working-directory: frontend

  node-lint:
    needs: [changes]
    if: >-
      needs.changes.outputs.node == 'true'
      && github.event_name != 'push'
    runs-on: blacksmith-4vcpu-ubuntu-2404
    strategy:
      matrix:
        service: [auth-service, custom-lang-service]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "22"
          cache: npm
          cache-dependency-path: node-services/${{ matrix.service }}/package-lock.json
      - run: npm ci
        working-directory: node-services/${{ matrix.service }}
      - name: Biome check
        run: npx biome check .
        working-directory: node-services/${{ matrix.service }}

  node-test:
    needs: [changes]
    if: >-
      needs.changes.outputs.node == 'true'
      && github.event_name != 'push'
    runs-on: blacksmith-4vcpu-ubuntu-2404
    strategy:
      matrix:
        service: [auth-service, custom-lang-service]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "22"
          cache: npm
          cache-dependency-path: node-services/${{ matrix.service }}/package-lock.json
      - run: npm ci
        working-directory: node-services/${{ matrix.service }}
      - name: Vitest
        run: npm test
        working-directory: node-services/${{ matrix.service }}
```

**Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: migrate lint/test jobs to Blacksmith with path filtering"
```

---

### Task 3: Rewrite nix-build job with Blacksmith + Cachix + Sticky Disk

**Files:**
- Modify: `.github/workflows/ci.yml` (nix-build job)

**Step 1: Rewrite nix-build with caching**

The nix-build job runs on both PR and main, but with different behavior:
- PR: build only (verify it compiles), with caching
- main push: build + push images

```yaml
  nix-build:
    needs: [changes]
    if: >-
      needs.changes.outputs.go == 'true'
      || needs.changes.outputs.nix == 'true'
    runs-on: blacksmith-4vcpu-ubuntu-2404
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4

      - name: Create Nix directories
        run: |
          sudo mkdir -p /nix/store
          sudo chmod -R 777 /nix

      - name: Mount Nix store (sticky disk)
        uses: useblacksmith/stickydisk@v1
        with:
          key: ${{ github.repository }}-nix-store
          path: /nix

      - uses: cachix/install-nix-action@v31
        with:
          github_access_token: ${{ secrets.GITHUB_TOKEN }}
          extra_nix_config: |
            accept-flake-config = true

      - uses: cachix/cachix-action@v15
        with:
          name: hackz-megalo-cup
          authToken: ${{ secrets.CACHIX_AUTH_TOKEN }}

      - name: Build service binaries
        run: nix build .#caller .#greeter .#gateway

      - name: Build service images
        run: nix build .#caller-image .#greeter-image .#gateway-image

      - name: Push images to ghcr.io
        if: github.ref == 'refs/heads/main' && github.event_name == 'push'
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          echo "$GITHUB_TOKEN" | skopeo login ghcr.io -u ${{ github.actor }} --password-stdin
          for svc in caller greeter gateway; do
            nix build .#${svc}-image
            ./result | skopeo copy docker-archive:/dev/stdin \
              docker://ghcr.io/hackz-megalo-cup/${svc}:${{ github.sha }}
            skopeo copy \
              docker://ghcr.io/hackz-megalo-cup/${svc}:${{ github.sha }} \
              docker://ghcr.io/hackz-megalo-cup/${svc}:latest
          done
```

**Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add Cachix + sticky disk caching to nix-build job"
```

---

### Task 4: Update render-manifests job with Blacksmith + Cachix

**Files:**
- Modify: `.github/workflows/ci.yml` (render-manifests job)

**Step 1: Update render-manifests**

```yaml
  render-manifests:
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    needs: [nix-build]
    runs-on: blacksmith-4vcpu-ubuntu-2404
    permissions:
      contents: write
      packages: read
    steps:
      - uses: actions/checkout@v4

      - name: Create Nix directories
        run: |
          sudo mkdir -p /nix/store
          sudo chmod -R 777 /nix

      - name: Mount Nix store (sticky disk)
        uses: useblacksmith/stickydisk@v1
        with:
          key: ${{ github.repository }}-nix-store
          path: /nix

      - uses: cachix/install-nix-action@v31
        with:
          github_access_token: ${{ secrets.GITHUB_TOKEN }}
          extra_nix_config: |
            accept-flake-config = true

      - uses: cachix/cachix-action@v15
        with:
          name: hackz-megalo-cup
          authToken: ${{ secrets.CACHIX_AUTH_TOKEN }}
          skipPush: true

      - name: Render nixidy manifests
        run: bash scripts/gen-manifests.sh

      - uses: stefanzweifel/git-auto-commit-action@v5
        with:
          commit_message: "chore: render nixidy manifests [skip ci]"
          file_pattern: "deploy/manifests/**"
```

**Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: update render-manifests with Blacksmith + Cachix"
```

---

### Task 5: Validate final ci.yml with actionlint

**Step 1: Install and run actionlint**

```bash
go install github.com/rhysd/actionlint/cmd/actionlint@latest
actionlint .github/workflows/ci.yml
```

Expected: No errors.

**Step 2: Verify YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"
```

Expected: No errors.

**Step 3: Final commit if any fixes needed**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: fix actionlint warnings"
```

---

### Task 6: Manual setup (user action required)

These steps cannot be automated and must be done by the user:

**Step 1: Install Blacksmith GitHub App**

Go to https://github.com/apps/blacksmith and install it on the `hackz-megalo-cup` organization (or just this repo).

**Step 2: Create Cachix cache**

Go to https://app.cachix.org/cache and create a cache named `hackz-megalo-cup`.

**Step 3: Add CACHIX_AUTH_TOKEN secret**

Generate an auth token from the Cachix dashboard (Settings > Auth Tokens). Add it as a repository secret named `CACHIX_AUTH_TOKEN` at:
`https://github.com/hackz-megalo-cup/microservices-app/settings/secrets/actions`

**Step 4: Trigger a test run**

Push the branch and open a PR to verify:
- `changes` job runs in ~5s
- Only relevant jobs trigger based on changed files
- `nix-build` uses cached Nix store from sticky disk
- On main merge, only `nix-build` + `render-manifests` run

---

## Summary of Final ci.yml Structure

```
on: workflow_dispatch / pull_request / push:main

concurrency: cancel-in-progress for PRs only

jobs:
  changes          → path detection gate (~5s, 2vcpu)
  contract         → PR only, when proto/** changed
  go-lint          → PR only, when services/** or proto/** changed
  go-test          → PR only, when services/** or proto/** changed
  frontend-lint    → PR only, when frontend/** changed
  frontend-build   → PR only, when frontend/** changed
  node-lint        → PR only, when node-services/** changed
  node-test        → PR only, when node-services/** changed
  nix-build        → PR + main, when go or nix changed (cached)
  render-manifests → main only, after nix-build
```
