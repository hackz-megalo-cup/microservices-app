# microservice-app

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/hackz-megalo-cup/microservices-app)
[![Mintlify Docs](https://img.shields.io/badge/docs-Mintlify-0ea5e9?logo=mintlify&logoColor=white)](https://mintlify.com/hackz-megalo-cup/microservices-app)

マイクロサービスアーキテクチャで構成されたアプリケーションリポジトリ。

## 技術スタック

| カテゴリ | 技術 |
|---------|------|
| Go サービス | connect-go / gRPC (greeter, caller, gateway) |
| Node.js サービス | Express (auth-service, custom-lang-service) |
| フロントエンド | React 19 + TypeScript + Vite + connect-query + React Query |
| データベース | PostgreSQL 17 |
| リバースプロキシ | Traefik v3 |
| ローカル開発 | Docker Compose / Tilt (Kubernetes) |
| スキーマ管理 | Protocol Buffers (buf) |
| Lint / Format | golangci-lint (Go) / Biome (TS) / treefmt (Nix) |
| 開発環境 | Nix / devenv |

## ディレクトリ構成

```
.
├── services/           # Go マイクロサービス (greeter, caller, gateway)
│   ├── cmd/            # エントリーポイント
│   ├── internal/       # サービス実装
│   └── gen/            # buf generate で生成されたコード
├── node-services/      # Node.js マイクロサービス (auth-service, custom-lang-service)
├── frontend/           # React フロントエンド
│   ├── src/app/        # ルートコンポーネント・プロバイダ
│   ├── src/features/   # 機能モジュール (auth, greeter, gateway)
│   ├── src/gen/        # buf generate で生成された TypeScript コード
│   ├── src/interceptors/ # connect-rpc インターセプタ
│   ├── src/lib/        # 共通ユーティリティ
│   └── src/testing/    # テストユーティリティ・モック
├── proto/              # Protocol Buffers 定義
├── deploy/             # デプロイ設定 (Docker, Traefik, k8s/nixidy)
├── scripts/            # ユーティリティスクリプト
├── templates/          # 新規サービス雛形テンプレート
├── docs/               # スタイルガイド
├── docker-compose.yml  # ローカル開発用
├── Tiltfile            # Tilt (Kubernetes ローカル開発) 設定
├── buf.yaml            # buf 設定
└── devenv.nix          # 開発環境定義
```

## セットアップ手順

### 1. Nix & direnv インストール

```zsh
curl -fsSL https://install.determinate.systems/nix | sh -s -- install
nix profile install nixpkgs#direnv nixpkgs#nix-direnv
eval "$(direnv hook zsh)"  # bash の場合は bash
```

### 2. clone & ディレクトリ移動

```zsh
git clone https://github.com/hackz-megalo-cup/microservices-infra
cd microservice-app
direnv allow
```

`direnv allow` を実行すると、devenv が自動で開発に必要なツール (Go, Node.js, buf, kubectl, tilt, etc.) をすべてインストールする。

### 3. 環境変数の設定

```zsh
cp .env.example .env
cp frontend/.env.example frontend/.env
```

デフォルト値のまま Docker Compose / Tilt で動作する。

## 開発手順

アプリの起動方法は 3 つある。

### 1. Kind + Tilt で k8s 起動 (重量版)

監視基盤 (Prometheus, Grafana, Loki, Tempo) をフルで動かすので重たいが、Observability を体験できるのでリソースに余裕があるならおすすめ。

```zsh
# インフラリポジトリを clone して bootstrap
git clone https://github.com/hackz-megalo-cup/microservices-infra
cd microservice-infra
direnv allow
full-bootstrap   # Docker が起動している状態で実行
```

```zsh
# アプリリポジトリに戻って Tilt 起動
cd microservice-app
tilt up
```

> **Tips**: ターミナルを占有したくない場合はバックグラウンドで起動できる:
>
> ```zsh
> tilt up > /dev/null 2>&1 &
> ```

http://localhost:10350/ で Tilt ダッシュボードからサービスの起動状況が確認できる。

| URL | サービス |
|-----|----------|
| http://localhost:10350 | Tilt ダッシュボード |
| http://localhost:30081 | Traefik (API ゲートウェイ) |
| http://localhost:30300 | Grafana (admin/admin) |
| http://localhost:30090 | Prometheus |
| http://localhost:31235 | Hubble UI (ネットワーク可視化) |

### 2. Kind + Tilt で k8s 起動 (軽量版)

k8s で動かしたいがメモリに余裕がない場合はこちら。Istio・ArgoCD を無効化し、Worker ノードが少ない構成。

```zsh
cd microservice-infra
bootstrap        # full-bootstrap ではなく bootstrap を使う
```

```zsh
cd microservice-app
tilt up
```

> **Tips**: ターミナルを占有したくない場合はバックグラウンドで起動できる:
>
> ```zsh
> tilt up > /dev/null 2>&1 &
> ```

### 3. Docker Compose で起動

監視基盤がいらない、スペック的に k8s が厳しい場合は Docker Compose でも起動できる。

```zsh
docker compose up
```

| URL | サービス |
|-----|----------|
| http://localhost:30081 | Traefik (API ゲートウェイ) |
| http://localhost:5432 | PostgreSQL |

## フロントエンド開発

### 概要

フロントエンドは React 19 + TypeScript + Vite で構成されている。バックエンドとの通信は connect-rpc (connect-query + TanStack Query) を使い、Protocol Buffers で定義された型安全な API 呼び出しを行う。

### 開発サーバーの起動

```zsh
cd frontend
npm install
npm run dev
```

http://localhost:5173 で開発サーバーが起動する。ホットリロード対応。

> バックエンドが必要な場合は、先に Docker Compose か Tilt でバックエンドを起動しておくこと。

### モックモードで起動

バックエンドなしでフロントエンドだけ開発したい場合は、MSW (Mock Service Worker) を使ったモックモードが利用できる。

```zsh
# frontend/.env を編集
VITE_USE_MOCK=true
```

### ディレクトリ構成

```
frontend/src/
├── app/            # App コンポーネント、プロバイダ設定
├── features/       # 機能ごとのモジュール
│   ├── auth/       # 認証 (JWT ログイン/サインアップ)
│   ├── greeter/    # Greeter サービス UI
│   └── gateway/    # Gateway サービス UI
├── gen/            # buf generate で自動生成 (編集禁止)
├── interceptors/   # connect-rpc インターセプタ (認証ヘッダ付与等)
├── lib/            # 共通ユーティリティ (transport, query-client)
├── testing/        # テストユーティリティ
└── types/          # TypeScript 型定義
```

### 主なコマンド

| コマンド | 説明 |
|---------|------|
| `npm run dev` | 開発サーバー起動 (http://localhost:5173) |
| `npm run build` | TypeScript 型チェック + Vite ビルド |
| `npm run lint` | Biome でリントチェック |
| `npm run lint:fix` | Biome でリント自動修正 |
| `npm run format` | Biome でフォーマット |
| `npm run format:check` | フォーマットチェック (CI 用) |

### API コード生成

バックエンドの proto 定義を変更した場合、TypeScript のクライアントコードを再生成する。

```zsh
buf generate
```

生成先は `frontend/src/gen/` 配下。このディレクトリは Biome の lint/format 対象外に設定されている。

## Protocol Buffers (buf)

### proto 定義の編集

```
proto/
├── greeter/v1/greeter.proto
├── caller/v1/caller.proto
└── gateway/v1/gateway.proto
```

### コード生成

```zsh
buf generate
```

以下が自動生成される:

| 生成先 | 内容 |
|--------|------|
| `services/gen/go/` | Go の protobuf + connect-go スタブ |
| `frontend/src/gen/` | TypeScript の protobuf + connect-query ヘルパー |

### lint & breaking change チェック

```zsh
buf-check
```

`buf lint` と `buf breaking --against main` を実行する。CI でも自動実行される。

## 新しいマイクロサービスの追加

### Go サービス

```bash
new-service go <service-name> [port]
```

生成されるファイル:

- `services/cmd/<service-name>/main.go` -- エントリーポイント
- `services/internal/<service-name>/` -- サービス実装ディレクトリ
- `deploy/docker/<service-name>/Dockerfile.dev` -- 開発用 Dockerfile
- `deploy/k8s/<service-name>.nix` -- nixidy モジュール
- `proto/<service-name>/v1/<service-name>.proto` -- proto 定義

追加手順:

1. `services/internal/<service-name>/service.go` にサービス実装を追加
2. `buf generate` で Go / TypeScript コードを再生成
3. `docker-compose.yml` にサービスを追記 — 既存の Go サービス (greeter 等) を参考に以下を追加:

    ```yaml
    <service-name>:
      build:
        context: .
        dockerfile: deploy/docker/<service-name>/Dockerfile.dev
      environment:
        PORT: "<port>"
        OTEL_EXPORTER_OTLP_ENDPOINT: ""
        OTEL_SERVICE_NAME: "<service-name>-service"
      labels:
        - "traefik.enable=true"
        - "traefik.http.routers.<service-name>.rule=PathPrefix(`/<service-name>.v1.<ServiceName>Service`)"
        - "traefik.http.routers.<service-name>.entrypoints=web"
        - "traefik.http.routers.<service-name>.priority=100"
        - "traefik.http.routers.<service-name>.middlewares=cors@file,rate-limit@file"
        - "traefik.http.services.<service-name>.loadbalancer.server.port=<port>"
        - "traefik.http.services.<service-name>.loadbalancer.server.scheme=h2c"
      networks:
        - app
    ```

4. `deploy/nixidy/env/local.nix` の `imports` に追加:

    ```nix
    ../../k8s/<service-name>.nix
    ```

5. `Tiltfile` に以下を追加:

    - `gen-manifests` の `deps` に `'deploy/k8s/<service-name>.nix'` を追加
    - `manifests` に `manifests += find_yaml('deploy/manifests/<service-name>-service')` を追加
    - `go_service('<service-name>', 'cmd/<service-name>')` を追加
    - `if manifests:` ブロック内に以下を追加:

    ```python
    <service-name>_deps = cluster_bootstrap_deps + ['gen-manifests', 'buf-generate']
    # if not use_nix: ブロック内に追加
    <service-name>_deps += ['<service-name>-compile']
    # k8s_resource を追加
    k8s_resource('<service-name>-service', port_forwards=<port>, resource_deps=<service-name>_deps)
    ```

6. **新規ファイルを `git add` する** — Nix は git tree に含まれるファイルしか参照できないため、`gen-manifests` を実行する前に新規ファイルを `git add` しておく必要がある

### Node.js (カスタム) サービス

```bash
new-service custom <service-name> [port]
```

生成されるファイル:

- `node-services/<service-name>/server.js` -- サーバー実装
- `node-services/<service-name>/package.json` -- パッケージ定義
- `deploy/docker/<service-name>/Dockerfile` -- Dockerfile
- `deploy/k8s/<service-name>.nix` -- nixidy モジュール

## 開発コマンド一覧

`direnv allow` で devenv シェルに入ると、以下のコマンドが使える。

### コード品質

| コマンド | 説明 |
|---------|------|
| `fmt` | 全言語 (Go + TypeScript + Nix) をフォーマットして `git add -u` |
| `lint` | 全言語をリント (golangci-lint + Biome) |
| `buf-check` | proto の lint + breaking change チェック |

### テスト

```bash
# Go テスト
cd services && go test ./...

# Node.js テスト (サービスごと)
cd node-services/auth-service && npm test
cd node-services/custom-lang-service && npm test

# フロントエンドビルド (型チェック含む)
cd frontend && npm run build

# スモークテスト (サービス起動中に実行)
test-smoke
```

### Kubernetes / マニフェスト

| コマンド | 説明 |
|---------|------|
| `gen-manifests` | nixidy モジュールから `deploy/manifests/` を再生成 |
| `load-microservice-images` | Nix でコンテナイメージをビルドして Kind にロード |
| `watch-manifests` | nixidy モジュールの変更を監視して自動で `kubectl apply` |
| `fix-chart-hash` | nixidy の空 `chartHash` をビルドエラーから自動修正 |

### デバッグ

| コマンド | 説明 |
|---------|------|
| `debug-k8s` | 全 namespace の Pod 状態 + 最近のイベントを表示 |
| `debug-grpc` | greeter / gateway の gRPC エンドポイントを `grpcurl` で確認 |
| `nix-check` | Nix 式の評価チェック (マニフェスト生成が通るか確認) |

### サービス追加

| コマンド | 説明 |
|---------|------|
| `new-service go <name> [port]` | Go サービスの雛形を生成 |
| `new-service custom <name> [port]` | Node.js サービスの雛形を生成 |

## Pre-commit フック

devenv により、コミット時に以下が自動実行される:

- **treefmt** -- Nix / Go / TypeScript のフォーマット
- **golangci-lint** -- Go のリント
- **goimports** -- Go の import 整理
- **biome** -- TypeScript / TSX のリント
- **go test** -- Go のユニットテスト

フックが失敗した場合はコミットがブロックされるので、`fmt` と `lint` で修正してから再コミットする。

## CI (GitHub Actions)

PR と main への push で以下が自動実行される:

| ジョブ | 内容 |
|--------|------|
| `contract` | buf lint + breaking change チェック |
| `go-lint` | golangci-lint |
| `go-test` | `go test ./...` |
| `frontend-lint` | Biome check |
| `frontend-build` | TypeScript 型チェック + Vite ビルド |
| `node-lint` | Node.js サービスの Biome check |
| `node-test` | Node.js サービスの Vitest |
| `nix-build` | Nix でバイナリ + コンテナイメージをビルド |
| `render-manifests` | nixidy マニフェスト再生成 (main push 時のみ) |

## スタイルガイド

- Go: [Google Go Style Guide](https://google.github.io/styleguide/go/) -- 詳細は `docs/go-style-guide.md`
- TypeScript: [Google TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html) -- 詳細は `docs/typescript-style-guide.md`
- React: [Bulletproof React](https://github.com/alan2207/bulletproof-react) -- 詳細は `docs/bulletproof-react.md`

## 関連リポジトリ

- [microservice-infra](https://github.com/hackz-megalo-cup/microservices-infra) -- 監視スタック、ArgoCD、Traefik 設定、Kind クラスタ設定
