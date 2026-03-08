# CI Optimization Design: Blacksmith + Cachix + Path Filtering

## Problem

Current CI runs 11 jobs on every push/PR regardless of what changed, with no caching for Nix builds, no concurrency control, and full duplication between PR and main-branch runs. This wastes ~14-15 runner-minutes per run.

## Measurements

| Job | Avg Duration | Notes |
|-----|-------------|-------|
| contract | ~5s | buf lint/breaking |
| go-lint | ~40s | golangci-lint |
| go-test | ~30s | |
| frontend-lint | ~12s | npm ci + biome |
| frontend-build | ~8s | npm ci + build |
| node-lint x2 | ~12s each | matrix: auth-service, custom-lang-service |
| node-test x2 | ~15s each | matrix: auth-service, custom-lang-service |
| nix-build | ~3-4min | **bottleneck**, no cache |
| render-manifests | main only | depends on nix-build |

All recent failures caused by "Push images to ghcr.io" step (GITHUB_TOKEN permissions).

## Design

### 1. Concurrency control

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.sha }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' }}
```

- PRs: new push cancels in-progress run for same PR
- main: no cancellation (each sha is unique group), so image push is never interrupted

### 2. Path-based conditional execution with `dorny/paths-filter@v3`

A lightweight `changes` job (~5s) detects which areas changed, downstream jobs skip if irrelevant.

Filters:
- `proto`: `proto/**`
- `go`: `services/**`, `proto/**`
- `frontend`: `frontend/**`
- `node`: `node-services/**`
- `nix`: `flake.nix`, `flake.lock`, `deploy/**/*.nix`, `services/**`

### 3. Main-branch scoping

On `push` to `main`, only run:
- `changes` (path detection)
- `nix-build` (build + push images) - when go or nix files changed
- `render-manifests` (deploy manifests) - after nix-build

Lint/test jobs only run on `pull_request` and `workflow_dispatch`, since they were already validated on the PR.

### 4. Blacksmith runners

Replace `runs-on: ubuntu-latest` with `runs-on: blacksmith-4vcpu-ubuntu-2404`.

Benefits:
- Half the per-minute cost ($0.004 vs $0.008)
- ~2x faster execution
- 3,000 free minutes/month on 2vCPU (1,500 on 4vCPU)
- Automatic cache interception for actions/cache and actions/setup-*

Requires: Install Blacksmith GitHub App on the repo.

### 5. Nix caching: Cachix + Sticky Disk

**Cachix** (binary cache):
- Add `cachix/cachix-action@v15` after `install-nix-action`
- Pushes built derivations so future runs download instead of rebuild
- Requires: `CACHIX_AUTH_TOKEN` secret

**Sticky Disk** (Nix store persistence):
- Use `useblacksmith/stickydisk@v1` to mount `/nix` as persistent volume
- Nix store survives across CI runs (~3s mount vs full rebuild)
- Combined with Cachix for belt-and-suspenders caching

### 6. Bump install-nix-action

Bump `cachix/install-nix-action` from v27 to v31. Pass `github_access_token` to avoid API rate limits.

## Expected Impact

| Metric | Before | After |
|--------|--------|-------|
| Runner minutes/run (PR, all paths) | ~15min | ~5-6min |
| Runner minutes/run (PR, single area) | ~15min | ~1-2min |
| Runner minutes/run (main merge) | ~15min | ~2-3min (nix-build only) |
| Nix build time | ~3-4min | ~10-30s (cached) |
| Cost per minute | $0.008 | $0.004 |
| Estimated overall cost reduction | baseline | ~75-80% |

## Required Setup (Manual)

1. Install Blacksmith GitHub App on the repo
2. Create Cachix cache (e.g., `hackz-megalo-cup`)
3. Add `CACHIX_AUTH_TOKEN` to repo secrets
