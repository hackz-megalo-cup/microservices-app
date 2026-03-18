# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 絶対に守るルール

### Go ビルドは services/ ディレクトリから
```bash
# ✅ 正しい
cd services && go build ./cmd/<service-name>
cd services && go test ./...

# ❌ 間違い（go.mod がルートにないので失敗する）
go build ./services/cmd/<service-name>
```

### Docker は必ず --build 付きで起動
```bash
docker compose up --build -d
```

### Proto 変更後は buf generate を実行
```bash
buf lint && buf generate
find services/gen -name '*.go' -exec gofmt -w {} +
git add services/gen/ frontend/src/gen/
```

## よく使うコマンド

### ビルド
```bash
cd services && go build ./cmd/greeter          # 単一 Go サービス
cd frontend && npm run build                    # フロントエンド (tsc + vite)
docker compose build <service-name>            # Docker イメージ再構築
nix build .#greeter .#caller .#gateway         # Nix ビルド (CI用)
```

### テスト
```bash
cd services && go test ./...                    # Go 全テスト
cd services && go test -race -v ./internal/greeter  # 単一パッケージ (race検出付き)
cd node-services && npm test --workspaces --if-present  # Node.js 全テスト
```

### Lint / Format
```bash
cd services && golangci-lint run ./...          # Go lint
cd frontend && npx biome check src/             # Frontend lint
cd node-services && npm run lint --workspaces --if-present  # Node lint
fmt                                             # devenv: 全言語フォーマット
lint                                            # devenv: 全言語 lint
```

### Proto
```bash
buf lint                                        # Proto lint
buf breaking --against '.git#branch=main'       # 破壊的変更チェック
buf generate                                    # コード生成 (Go + TypeScript)
```

### ローカル開発
```bash
docker compose up --build -d                    # Docker Compose 起動
tilt up                                         # K8s + Tilt 起動
cd frontend && VITE_USE_MOCK=true npm run dev   # フロントエンド単体 (MSW モック)
test-smoke                                      # devenv: スモークテスト
```

### サービス管理
```bash
new-service go <name> [port]                    # Go サービス雛形生成
new-service custom <name> [port]                # Node.js サービス雛形生成
delete-service <name>                           # サービス削除
```

## アーキテクチャ

### サービス構成
```
Frontend (React 19 + Vite)
    │ connect-rpc + JWT
Traefik (localhost:30081) ─── CORS / Auth / Rate-limit / Retry
    ├── auth-service     (Node/Express :8090)  JWT認証, JWKS
    ├── greeter          (Go/connect   :8080)  挨拶 + 外部API呼出
    ├── caller           (Go/connect   :8081)  外部エンドポイント呼出
    ├── gateway          (Go/connect   :8082)  サービスオーケストレーション
    ├── projector        (Go)                  Kafka → イベント投影
    └── frontend         (nginx :80)           SPA配信
```

### 通信パターン
- **Frontend → Services**: connect-rpc (HTTP/2 h2c) via Traefik
- **Service → Service**: connect-go クライアント + `otelhttp` トレーシング
- **非同期イベント**: Watermill + Kafka (Redpanda)
- **認証フロー**: Traefik forward-auth → auth-service `/verify` → `X-User-Id` ヘッダ注入

### イベントソーシング (全 Go サービス共通)
各サービスは 4 ファイル構成:
- `events.go` — ドメインイベント定義
- `aggregate.go` — 状態マシン (`ApplyEvent` でイベント再生)
- `service.go` — gRPC ハンドラ (Aggregate 操作 → EventStore + Outbox に保存)
- `migrations/` — SQL マイグレーション (`//go:embed`)

標準テーブル: `event_store`, `outbox_events`, `idempotency_keys`, `snapshots`

### 耐障害パターン (platform パッケージ)
サービス間呼出に Circuit Breaker → Bulkhead → Retry (指数バックオフ) を積層。
失敗時は Saga 補償トランザクション (Failed/Compensated イベント)。

### プロジェクト構造
- **Go サービス**: `services/` (go.mod はここ)
  - `cmd/<service>/main.go` — エントリポイント
  - `internal/<service>/` — ビジネスロジック
  - `internal/platform/` — 共有基盤 (EventStore, Outbox, CB, Retry等)
  - `gen/go/` — protobuf 生成コード (編集禁止)
- **Node.js サービス**: `node-services/{auth-service,shared}/`
- **フロントエンド**: `frontend/` (bulletproof-react 構成: `src/features/`, `src/lib/`, `src/gen/`)
- **Proto 定義**: `proto/<service>/<version>/<service>.proto`
- **Docker / K8s**: `deploy/{docker,k8s,manifests,traefik,nixidy}/`
- **スクリプト**: `scripts/` (new-service, delete-service, gen-manifests 等)

### Traefik ルーティング
PathPrefix でサービスを振り分け。ミドルウェア: cors, auth, rate-limit, retry。
- gRPC: `/greeter.v1.GreeterService`, `/gateway.v1.GatewayService` 等
- REST: `/auth/register`, `/auth/login`, `/auth/.well-known/jwks.json`
- Frontend: `/` (priority=1, catch-all)

### データベース
サービスごとに独立 DB (`greeter_db`, `caller_db`, `gateway_db`, `lang_db`, `auth_db`)。
マイグレーションは `golang-migrate` (Go) / `node-pg-migrate` (Node)。

## コミット規約

`<type>(<scope>): <description>` 形式。
- type: feat, fix, docs, refactor, style, chore, test
- scope: サービス名やコンポーネント名

## スタイルガイド

- Go: `docs/go-style-guide.md`
- TypeScript: `docs/typescript-style-guide.md`
- React: `docs/bulletproof-react.md`
- レビュー: `REVIEW.md`

## CI パイプライン

PR → `contract` (buf lint/breaking) + `go-check` + `frontend-check` + `node-check`
main push → 上記 + `nix-build` (イメージ push to ghcr.io) + `render-manifests` (K8s YAML 自動生成)

## API テスト / スモークテスト

- `/api-test` スキル: JWT取得 → gRPC エンドポイントテスト
- `/smoke-test` スキル: ヘルスチェック + RPC 疎通確認
