# Getting Started

## 前提条件

- **devenv** が動作していること (`devenv shell` でシェルに入る)
- **Docker** が起動していること

devenv に入ると `buf`, `go`, `grpcurl` 等の必要なツールが全て揃う。

## 新しいサービスを作る

```bash
new-service go <service-name> <port>
```

例: `new-service go order 8084`

ソースコード、proto、Dockerfile、docker-compose エントリ、DB、Kafka トピックが全て自動生成される。

## 編集する 4 ファイル

### 1. `proto/<service>/v1/<service>.proto`

gRPC の API 定義。編集後 `buf generate` でコードを再生成する。

```protobuf
service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse) {}
}
```

### 2. `services/internal/<service>/events.go`

イベント型とペイロードを定義する。イベントは「起きた事実」を表す。

```go
const EventOrderCreated = "order.created"

type OrderCreatedData struct {
    ItemID string `json:"item_id"`
    Amount int    `json:"amount"`
}
```

### 3. `services/internal/<service>/aggregate.go`

集約の状態と、イベント適用ロジック。

- `ApplyEvent` -- 保存済みイベントから状態を復元する
- コマンドメソッド -- `Raise()` でイベントを発行し、状態を更新する

```go
func (a *OrderAggregate) ApplyEvent(eventType string, data json.RawMessage) {
    switch eventType {
    case EventOrderCreated:
        var d OrderCreatedData
        json.Unmarshal(data, &d)
        a.ItemID = d.ItemID
        a.Status = "created"
    }
}
```

### 4. `services/internal/<service>/service.go`

ビジネスロジックの本体。gRPC ハンドラを実装する。
集約を作り、コマンドを呼び、`SaveAggregate` で永続化する:

```go
agg := NewOrderAggregate(uuid.NewString())
agg.Create(itemID, amount)
platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, OrderTopicMapper)
```

## 起動方法

```bash
docker compose up
```

全サービスが起動する。個別起動は `docker compose up <service-name>`。
フロントエンドは http://localhost:30081 でアクセスできる。

## Event Sourcing 30秒解説

1. **イベントは事実** -- 「注文が作成された」等、起きたことをそのまま記録する
2. **Aggregate はイベントを再生して状態を復元する** -- DB に現在の状態は保存しない。イベント列が真実
3. **`SaveAggregate` が残りを全部やる** -- イベント保存、楽観的ロック、Outbox 発行を 1 トランザクションで処理

開発者がやることは「イベントを定義し、Aggregate に Apply を書き、コマンドで Raise する」だけ。

## 触らないファイル

以下はインフラ層のボイラープレート。スクリプトが自動生成するので編集不要:

- `services/cmd/<service>/main.go` -- サーバー起動、DB接続、gRPC登録
- `services/internal/<service>/embed.go` -- マイグレーション埋め込み
- `services/internal/platform/` -- EventStore, Outbox, CircuitBreaker 等の共通基盤
- `services/internal/<service>/migrations/` -- DDL マイグレーション
