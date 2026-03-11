# microservice-app

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/hackz-megalo-cup/microservices-app)
[![Mintlify Docs](https://img.shields.io/badge/docs-Mintlify-0ea5e9?logo=mintlify&logoColor=white)](https://mintlify.com/hackz-megalo-cup/microservices-app)

マイクロサービスアーキテクチャで構成されたアプリケーションリポジトリ。

## 技術スタック

| カテゴリ | 技術 |
|---------|------|
| Go サービス | connect-go / gRPC (greeter, caller, gateway, projector) |
| Node.js サービス | Express (auth-service, custom-lang-service) |
| フロントエンド | React 19 + TypeScript + Vite + connect-query + React Query |
| データベース | PostgreSQL 17 |
| イベントストリーミング | Redpanda (Kafka 互換) |
| リバースプロキシ | Traefik v3 |
| ローカル開発 | Docker Compose / Tilt (Kubernetes) |
| スキーマ管理 | Protocol Buffers (buf) |
| Lint / Format | golangci-lint (Go) / Biome (TS) / treefmt (Nix) |
| 開発環境 | Nix / devenv |

## ディレクトリ構成

```
.
├── services/           # Go マイクロサービス (greeter, caller, gateway, projector)
│   ├── cmd/            # エントリーポイント
│   ├── internal/       # サービス実装
│   └── gen/            # buf generate で生成されたコード
├── node-services/      # Node.js マイクロサービス (auth-service, custom-lang-service, shared)
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
├── tilt-services.json  # Tilt サービス登録 (new-service が自動管理)
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
cd microservices-infra
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
| http://localhost:5173 | フロントエンド (Vite dev server) |
| http://localhost:5432 | PostgreSQL |
| http://localhost:8888 | Redpanda Console (Kafka UI) |
| http://localhost:19092 | Redpanda Kafka (外部アクセス) |

## フロントエンド開発

### 概要

フロントエンドは React 19 + TypeScript + Vite で構成されている。バックエンドとの通信は connect-rpc (connect-query + TanStack Query) を使い、Protocol Buffers で定義された型安全な API 呼び出しを行う。

### 開発の流れ

`tilt up` するだけでフロントエンドもバックエンドもまとめて起動される。個別に `npm install` や `npm run dev` を実行する必要はない。

```zsh
tilt up
```

- フロントエンド: http://localhost:30081 （Traefik 経由）
- Tilt ダッシュボード: http://localhost:10350

`frontend/` 配下のコードを変更すると、Tilt が自動でイメージを再ビルド・再デプロイする。

### Docker Compose で開発する場合

```zsh
docker compose up --build
```

フロントエンドは Traefik 経由で http://localhost:30081 にアクセスできる。

Docker Compose の場合、フロントエンドは本番と同じく nginx で静的ファイルを配信する構成になっている。コード変更後はイメージの再ビルドが必要:

```zsh
docker compose up --build frontend -d
```

> **Tips**: ホットリロードで開発したい場合は、バックエンドだけ Docker Compose で起動して、フロントエンドは Vite dev server をローカルで動かす方法もある:
>
> ```zsh
> # バックエンドだけ起動
> docker compose up --build -d --scale frontend=0
>
> # フロントエンドはローカルで起動
> cd frontend && npm install && npm run dev
> ```
>
> http://localhost:5173 でホットリロード対応の開発サーバーが起動する。

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
| `npm run build` | TypeScript 型チェック + Vite ビルド |
| `npm run lint` | Biome でリントチェック |
| `npm run lint:fix` | Biome でリント自動修正 |
| `npm run format` | Biome でフォーマット |
| `npm run format:check` | フォーマットチェック (CI 用) |

### API コード生成

proto 定義を変更すると、Tilt が自動で `buf generate` を実行して TypeScript のクライアントコードを再生成する。手動で実行する場合は:

```zsh
buf generate
```

生成先は `frontend/src/gen/` 配下。このディレクトリは Biome の lint/format 対象外に設定されている。

## Protocol Buffers (buf)

### proto 定義の編集

```
proto/
├── greeter/v1/greeter.proto
├── greeter/v2/greeter.proto
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

## Getting Started

### 前提条件

- **devenv** が動作していること (`devenv shell` でシェルに入る)
- **Docker** が起動していること

devenv に入ると `buf`, `go`, `grpcurl`, `jq`, `tilt` 等の必要なツールが全て揃う。

### 新しいサービスを作る

```bash
new-service go <service-name> [port]
```

例: `new-service go todo`（port 省略時のデフォルトは 8080）

ソースコード、proto、Dockerfile、docker-compose エントリ、DB、Kafka トピック、Tilt 設定が全て自動生成される。
Tiltfile の編集は不要 -- `tilt-services.json` にエントリが自動追加され、`tilt up` が新サービスを自動検出する。

### データの流れ

リクエストがどう処理されるかの全体像:

```
クライアント
  │
  ▼
proto (gRPC API 定義)
  │  buf generate でコード生成
  ▼
service.go (gRPC ハンドラ)
  │  aggregate を作り、コマンドを呼ぶ
  ▼
aggregate.go (コマンド → Raise でイベント発行)
  │
  ▼
platform.SaveAggregate ── 1 トランザクションで処理 ──┐
  │                                                  │
  ├─→ event_store (イベント保存)                     │
  └─→ outbox_events (Kafka 発行キュー)               │
                │                                    │
                ▼                                    │
             Kafka (非同期イベント配信) ◄─────────────┘

--- イベント再生（LoadAggregate）---

event_store
  │  保存済みイベントを順番に読み出す
  ▼
aggregate.ApplyEvent (イベントごとに状態を復元)
  │
  ▼
aggregate の現在の状態が復元される
```

### 編集する 4 ファイル

#### 1. `proto/<service>/v1/<service>.proto`

gRPC の API 定義。編集後 `buf generate` でコードを再生成する。

```protobuf
service TodoService {
  rpc CreateTodo(CreateTodoRequest) returns (CreateTodoResponse) {}
  rpc CompleteTodo(CompleteTodoRequest) returns (CompleteTodoResponse) {}
}
```

#### 2. `services/internal/<service>/events.go`

イベント型とペイロードを定義する。イベントは「起きた事実」を表す。

> **注意:** テンプレートが生成する `Failed` / `Compensated` イベントは `main.go` の補償ハンドラが参照している。**削除すると `main.go` がコンパイルエラーになる**ので残すこと。自分のドメインイベントはこれらに追加する形で定義する。

```go
const (
    EventTodoCreated   = "todo.created"
    EventTodoCompleted = "todo.completed"                // ← 追加
    EventTodoFailed    = "todo.failed"                    // 削除禁止
    EventTodoCompensated = "todo.compensated"             // 削除禁止
)

type TodoCreatedData struct {
    Title string `json:"title"`
}

type TodoCompletedData struct{}
```

#### 3. `services/internal/<service>/aggregate.go`

集約の状態と、イベント適用ロジック。

- `ApplyEvent` -- 保存済みイベントから状態を復元する
- コマンドメソッド -- `Raise()` でイベントを発行し、状態を更新する
- `Fail` / `Compensate` -- main.go が参照するので削除しない

```go
type TodoAggregate struct {
    platform.AggregateBase
    Title  string                     // ← ドメインのフィールド
    Status string
}

func (a *TodoAggregate) ApplyEvent(eventType string, data json.RawMessage) {
    switch eventType {
    case EventTodoCreated:
        var d TodoCreatedData
        json.Unmarshal(data, &d)
        a.Title = d.Title
        a.Status = "created"
    case EventTodoCompleted:          // ← 追加イベント
        a.Status = "completed"
    }
}

func (a *TodoAggregate) Create(title string) {
    a.Raise(EventTodoCreated, TodoCreatedData{Title: title})
    a.Title = title
    a.Status = "created"
}

func (a *TodoAggregate) Complete() {  // ← 追加コマンド
    a.Raise(EventTodoCompleted, TodoCompletedData{})
    a.Status = "completed"
}
```

#### 4. `services/internal/<service>/service.go`

ビジネスロジックの本体。gRPC ハンドラを実装する。

**新規作成パターン** -- 集約を作り、コマンドを呼び、`SaveAggregate` で永続化する。`AggregateID()` で採番された ID を取得できる:

```go
func (s *Service) CreateTodo(ctx context.Context, req *connect.Request[pb.CreateTodoRequest]) (*connect.Response[pb.CreateTodoResponse], error) {
    title := req.Msg.GetTitle()

    agg := NewTodoAggregate(uuid.NewString())
    agg.Create(title)
    platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, TodoTopicMapper)

    return connect.NewResponse(&pb.CreateTodoResponse{
        Id: agg.AggregateID(),
    }), nil
}
```

**既存更新パターン** -- `LoadAggregate` でイベントを再生して状態を復元し、コマンドを呼ぶ:

```go
func (s *Service) CompleteTodo(ctx context.Context, req *connect.Request[pb.CompleteTodoRequest]) (*connect.Response[pb.CompleteTodoResponse], error) {
    id := req.Msg.GetId()

    agg := NewTodoAggregate(id)
    platform.LoadAggregate(ctx, s.eventStore, agg)
    agg.Complete()
    platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, TodoTopicMapper)

    return connect.NewResponse(&pb.CompleteTodoResponse{}), nil
}
```

### 起動方法

#### Tilt（推奨）

```bash
tilt up
```

全サービスが起動し、コード変更時に自動でリビルド・リデプロイされる。
`new-service` で追加したサービスも Tiltfile を編集せずに自動検出される。

- フロントエンド: http://localhost:30081
- Tilt UI: http://localhost:10350
- 各サービスは `tilt-services.json` で定義されたポートでフォワードされる

#### docker compose

```bash
docker compose up
```

全サービスが起動する。個別起動は `docker compose up <service-name>`。
フロントエンドは http://localhost:30081 でアクセスできる。

#### ビルドとデプロイ時の注意（docker compose の場合）

コード変更後、Docker イメージを再構築する必要がある:

```bash
# 1. サービスをビルド（Go コマンドは services/ ディレクトリで実行）
cd services && go build ./cmd/<service-name>

# 2. Docker イメージを再構築（古いキャッシュを使わないため必須）
docker compose build <service-name>

# 3. サービスを起動
docker compose up <service-name> -d
```

`docker compose up` だけでは古いイメージが使用される場合がある。
Tilt を使っている場合はこの手順は不要（自動でビルド・デプロイされる）。

### curl でテストする

#### Traefik 経由（Tilt / docker compose 共通）

Tilt でも docker compose でも Traefik が `localhost:30081` でリクエストを受ける。
認証が必要なエンドポイントは JWT トークンを取得してから叩く:

```bash
# ユーザー登録（初回のみ）
curl -s -X POST http://localhost:30081/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}'

# トークン取得
TOKEN=$(curl -s -X POST http://localhost:30081/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}' | jq -r '.token')

# CreateTodo
TODO_ID=$(curl -s -X POST http://localhost:30081/todo.v1.TodoService/CreateTodo \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title": "buy milk"}' | jq -r '.id')
# => xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# CompleteTodo
curl -s -X POST http://localhost:30081/todo.v1.TodoService/CompleteTodo \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"id\":\"${TODO_ID}\"}"
# => {}
```

#### Tilt のポートフォワード経由（サービス直接）

Tilt がサービスごとにポートフォワードするので、直接アクセスもできる:

```bash
# ヘルスチェック（ポート番号は tilt-services.json で確認）
curl -sf http://localhost:8080/healthz
# => ok

# CreateTodo
curl -s -X POST http://localhost:8080/todo.v1.TodoService/CreateTodo \
  -H 'Content-Type: application/json' \
  -d '{"title": "buy milk"}'
# => {"id":"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"}
```

#### docker compose の場合（Docker ネットワーク経由）

サービスに直接アクセスする場合は `docker run` 経由で curl を実行する:

```bash
docker run --rm --network microservices-app_app curlimages/curl:latest \
  -s -X POST http://todo:8080/todo.v1.TodoService/CreateTodo \
  -H 'Content-Type: application/json' \
  -d '{"title": "buy milk"}'
# => {"id":"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"}
```

URL のパターンは `http://<service>:<port>/<proto package>.<Service>/<RPC>` になる。

### DB を確認する

各サービスは `<service>_db` という専用データベースを持つ。

#### Tilt (k8s) の場合

PostgreSQL は `database` namespace の `postgresql-0` Pod にいる。コンテナが 2 つあるので `-c postgresql` を指定し、`PGPASSWORD` を環境変数で渡す:

```bash
# イベントストアの中身
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d <service>_db \
  -c 'SELECT stream_id, event_type, version, data FROM event_store ORDER BY created_at;'

# Outbox（Kafka への発行状況）
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d <service>_db \
  -c 'SELECT id, topic, published, created_at FROM outbox_events ORDER BY created_at;'
```

#### docker compose の場合

```bash
# イベントストアの中身
docker compose exec postgres psql -U devuser -d <service>_db \
  -c 'SELECT stream_id, event_type, version, data, created_at FROM event_store ORDER BY created_at;'

# Outbox（Kafka への発行状況）
docker compose exec postgres psql -U devuser -d <service>_db \
  -c 'SELECT id, topic, published, created_at FROM outbox_events ORDER BY created_at;'
```

### Event Sourcing 30秒解説

1. **イベントは事実** -- 「Todo が作成された」「Todo が完了した」等、起きたことをそのまま記録する
2. **Aggregate はイベントを再生して状態を復元する** -- DB に現在の状態は保存しない。イベント列が真実
3. **`SaveAggregate` が残りを全部やる** -- イベント保存、楽観的ロック、Outbox 発行を 1 トランザクションで処理

開発者がやることは「イベントを定義し、Aggregate に Apply を書き、コマンドで Raise する」だけ。

### 他サービスへの HTTP 呼び出しを追加する場合

サービスが他のサービスへ HTTP リクエストを送る場合、分散トレーシングを正しく動作させるために以下の 2 点が必要:

#### 1. HTTP Client を `otelhttp.NewTransport` でラップする

```go
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

client := &http.Client{
    Timeout:   3 * time.Second,
    Transport: otelhttp.NewTransport(http.DefaultTransport),
}
```

これにより、送信 HTTP リクエストに W3C `traceparent` ヘッダが自動付与される。素の `http.Client` を使うとトレースが途切れる。

#### 2. `otelconnect.WithTrustRemote()` を確認する

テンプレートが生成する `main.go` には設定済みだが、Connect RPC ハンドラの `otelconnect.NewInterceptor()` に `WithTrustRemote()` が付いていることを確認する:

```go
otelInterceptor, err := otelconnect.NewInterceptor(otelconnect.WithTrustRemote())
```

これがないと、受信側が incoming trace context を無視して新しいトレースを開始してしまう。

### 触らないファイル

以下はインフラ層のボイラープレート。スクリプトが自動生成するので編集不要:

- `services/cmd/<service>/main.go` -- サーバー起動、DB接続、gRPC登録、補償ハンドラ
- `services/internal/<service>/embed.go` -- マイグレーション埋め込み
- `services/internal/platform/` -- EventStore, Outbox, CircuitBreaker 等の共通基盤
- `services/internal/<service>/migrations/` -- DDL マイグレーション
- `tilt-services.json` -- Tilt のサービス登録（`new-service` / `delete-service` が自動管理）
- `Tiltfile` -- Tilt の設定（`tilt-services.json` を読んで動的にサービスを登録するので直接編集不要）

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
| `new-service go <name> [port]` | Go サービスの雛形を生成（Tilt 自動登録込み） |
| `new-service custom <name> [port]` | Node.js サービスの雛形を生成（Tilt 自動登録込み） |
| `delete-service <name>` | サービスの全ファイルと配線を削除（Tilt 登録解除込み） |

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
| `contract` | buf lint + breaking change チェック (PR のみ) |
| `go-check` | golangci-lint + `go test ./...` |
| `frontend-check` | Biome check + TypeScript 型チェック + Vite ビルド |
| `node-check` | Node.js サービスの Biome check + Vitest |
| `nix-build` | Nix でバイナリ + コンテナイメージをビルド、main push 時は ghcr.io へ push |
| `render-manifests` | nixidy マニフェスト再生成 (main push 時のみ) |

## スタイルガイド

- Go: [Google Go Style Guide](https://google.github.io/styleguide/go/) -- 詳細は `docs/go-style-guide.md`
- TypeScript: [Google TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html) -- 詳細は `docs/typescript-style-guide.md`
- React: [Bulletproof React](https://github.com/alan2207/bulletproof-react) -- 詳細は `docs/bulletproof-react.md`

## 関連リポジトリ

- [microservice-infra](https://github.com/hackz-megalo-cup/microservices-infra) -- 監視スタック、ArgoCD、Traefik 設定、Kind クラスタ設定
