# Getting Started

## 前提条件

- **devenv** が動作していること (`devenv shell` でシェルに入る)
- **Docker** が起動していること

devenv に入ると `buf`, `go`, `grpcurl`, `jq`, `tilt` 等の必要なツールが全て揃う。

## 新しいサービスを作る

```bash
new-service go <service-name> [port]
```

例: `new-service go todo`（port 省略時のデフォルトは 8080）

ソースコード、proto、Dockerfile、docker-compose エントリ、DB、Kafka トピック、Tilt 設定が全て自動生成される。
Tiltfile の編集は不要 -- `tilt-services.json` にエントリが自動追加され、`tilt up` が新サービスを自動検出する。

## データの流れ

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

## 編集する 4 ファイル

### 1. `proto/<service>/v1/<service>.proto`

gRPC の API 定義。編集後 `buf generate` でコードを再生成する。

```protobuf
service TodoService {
  rpc CreateTodo(CreateTodoRequest) returns (CreateTodoResponse) {}
  rpc CompleteTodo(CompleteTodoRequest) returns (CompleteTodoResponse) {}
}
```

### 2. `services/internal/<service>/events.go`

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

### 3. `services/internal/<service>/aggregate.go`

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

### 4. `services/internal/<service>/service.go`

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

## 起動方法

### Tilt（推奨）

```bash
tilt up
```

全サービスが起動し、コード変更時に自動でリビルド・リデプロイされる。
`new-service` で追加したサービスも Tiltfile を編集せずに自動検出される。

- フロントエンド: http://localhost:30081
- Tilt UI: http://localhost:10350
- 各サービスは `tilt-services.json` で定義されたポートでフォワードされる

### docker compose

```bash
docker compose up
```

全サービスが起動する。個別起動は `docker compose up <service-name>`。
フロントエンドは http://localhost:30081 でアクセスできる。

### ビルドとデプロイ時の注意（docker compose の場合）

コード変更後、Docker イメージを再構築する必要があります:

```bash
# 1. サービスをビルド（Go コマンドは services/ ディレクトリで実行）
cd services && go build ./cmd/<service-name>

# 2. Docker イメージを再構築（古いキャッシュを使わないため必須）
docker compose build <service-name>

# 3. サービスを起動
docker compose up <service-name> -d
```

`docker compose up` だけでは古いイメージが使用される場合があります。
Tilt を使っている場合はこの手順は不要（自動でビルド・デプロイされる）。

## curl でテストする

### Traefik 経由（Tilt / docker compose 共通）

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

### Tilt のポートフォワード経由（サービス直接）

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

### docker compose の場合（Docker ネットワーク経由）

サービスに直接アクセスする場合は `docker run` 経由で curl を実行する:

```bash
docker run --rm --network microservices-app_app curlimages/curl:latest \
  -s -X POST http://todo:8080/todo.v1.TodoService/CreateTodo \
  -H 'Content-Type: application/json' \
  -d '{"title": "buy milk"}'
# => {"id":"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"}
```

URL のパターンは `http://<service>:<port>/<proto package>.<Service>/<RPC>` になる。

## DB を確認する

各サービスは `<service>_db` という専用データベースを持つ。

### Tilt (k8s) の場合

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

### docker compose の場合

```bash
# イベントストアの中身
docker compose exec postgres psql -U devuser -d <service>_db \
  -c 'SELECT stream_id, event_type, version, data, created_at FROM event_store ORDER BY created_at;'

# Outbox（Kafka への発行状況）
docker compose exec postgres psql -U devuser -d <service>_db \
  -c 'SELECT id, topic, published, created_at FROM outbox_events ORDER BY created_at;'
```

## Event Sourcing 30秒解説

1. **イベントは事実** -- 「Todo が作成された」「Todo が完了した」等、起きたことをそのまま記録する
2. **Aggregate はイベントを再生して状態を復元する** -- DB に現在の状態は保存しない。イベント列が真実
3. **`SaveAggregate` が残りを全部やる** -- イベント保存、楽観的ロック、Outbox 発行を 1 トランザクションで処理

開発者がやることは「イベントを定義し、Aggregate に Apply を書き、コマンドで Raise する」だけ。

## 他サービスへの HTTP 呼び出しを追加する場合

サービスが他のサービスへ HTTP リクエストを送る場合、分散トレーシングを正しく動作させるために以下の 2 点が必要:

### 1. HTTP Client を `otelhttp.NewTransport` でラップする

```go
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

client := &http.Client{
    Timeout:   3 * time.Second,
    Transport: otelhttp.NewTransport(http.DefaultTransport),
}
```

これにより、送信 HTTP リクエストに W3C `traceparent` ヘッダが自動付与される。素の `http.Client` を使うとトレースが途切れる。

### 2. `otelconnect.WithTrustRemote()` を確認する

テンプレートが生成する `main.go` には設定済みだが、Connect RPC ハンドラの `otelconnect.NewInterceptor()` に `WithTrustRemote()` が付いていることを確認する:

```go
otelInterceptor, err := otelconnect.NewInterceptor(otelconnect.WithTrustRemote())
```

これがないと、受信側が incoming trace context を無視して新しいトレースを開始してしまう。

## 触らないファイル

以下はインフラ層のボイラープレート。スクリプトが自動生成するので編集不要:

- `services/cmd/<service>/main.go` -- サーバー起動、DB接続、gRPC登録、補償ハンドラ
- `services/internal/<service>/embed.go` -- マイグレーション埋め込み
- `services/internal/platform/` -- EventStore, Outbox, CircuitBreaker 等の共通基盤
- `services/internal/<service>/migrations/` -- DDL マイグレーション
- `tilt-services.json` -- Tilt のサービス登録（`new-service` / `delete-service` が自動管理）
- `Tiltfile` -- Tilt の設定（`tilt-services.json` を読んで動的にサービスを登録するので直接編集不要）
