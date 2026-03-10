# API Test Commands

## 1. Auth

```bash
# Register
curl -s -X POST http://localhost:30081/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}'

# Login
curl -s -X POST http://localhost:30081/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}'
```

## 2. Greeter (CQRS success flow)

```bash
TOKEN="<jwt token from login>"

# v1（認証必要）
curl -s -X POST http://localhost:30081/greeter.v1.GreeterService/Greet \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Alice"}'

# v2（認証なしでも可、レスポンスに traceId 付き、externalBodyLength が int64）
curl -s -X POST http://localhost:30081/greeter.v2.GreeterService/Greet \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","locale":"ja"}'
```

## 3. Gateway InvokeCustom

```bash
# Success
curl -s -X POST http://localhost:30081/gateway.v1.GatewayService/InvokeCustom \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Bob"}'

# Triggers saga compensation (unauthorized)
curl -s -X POST http://localhost:30081/gateway.v1.GatewayService/InvokeCustom \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"unauthorized"}'

# Triggers saga compensation (unavailable)
curl -s -X POST http://localhost:30081/gateway.v1.GatewayService/InvokeCustom \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"unavailable"}'
```

## 4. 分散トレーシングの確認 (Tempo)

Greeter → Caller → 外部 HTTP や Gateway → Custom-Lang のリクエストが、単一のトレースとして繋がっていることを確認する。

```bash
# 直近のトレースを検索
kubectl exec -n observability tempo-0 -- \
  wget -qO- 'http://localhost:3200/api/search?limit=5'

# 特定サービスのトレースを検索
kubectl exec -n observability tempo-0 -- \
  wget -qO- 'http://localhost:3200/api/search?tags=service.name%3Dgreeter-service&limit=5'

# トレース詳細を取得（traceID は上記の結果から取得）
kubectl exec -n observability tempo-0 -- \
  wget -qO- 'http://localhost:3200/api/traces/<traceID>'

# parent-child 関係の検証（サービス名・スパン名・親子関係を一覧表示）
kubectl exec -n observability tempo-0 -- \
  wget -qO- 'http://localhost:3200/api/traces/<traceID>' | \
  jq '[.batches[] | {service: (.resource.attributes[] | select(.key=="service.name") | .value.stringValue), spans: [.scopeSpans[].spans[] | {name: .name, spanId: .spanId, parentSpanId: .parentSpanId, kind: .kind}]}]'
```

正しく分散トレーシングが動いている場合、1 つのトレースに複数サービスの batch が含まれる:

- **Greeter → Caller**: greeter-service と caller-service のスパンが同一 traceID で parent-child 関係になる
- **Gateway → Custom-Lang**: gateway-service と custom-lang-service のスパンが同一 traceID で parent-child 関係になる

link だけで繋がっている場合は `otelconnect.WithTrustRemote()` または `otelhttp.NewTransport` の設定が漏れている。

## 5. CQRS Projection の確認 (Greeter)

Greeter サービスは event_store に書き込んだイベントを、バックグラウンドの Projection が 1 秒間隔でポーリングし、greetings テーブル（read model）に実体化する。

```bash
# 1. Greet リクエストを送信（event_store にイベントが書き込まれる）
curl -s -X POST http://localhost:30081/greeter.v1.GreeterService/Greet \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"ProjectionTest"}'

# 2. 数秒待ってから greetings テーブルを確認（Projection が反映済み）
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d greeter_db \
  -c "SELECT id, name, message, status, created_at FROM greetings ORDER BY created_at;"

# 3. event_store の stream_id と greetings の id が一致することを確認
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d greeter_db \
  -c "SELECT e.stream_id, e.event_type, g.name, g.status
       FROM event_store e
       LEFT JOIN greetings g ON g.id::text = e.stream_id
       ORDER BY e.created_at;"
```

正しく動いている場合:
- greetings テーブルに Greet リクエストで送った name が入っている（status = `created`）
- event_store の stream_id と greetings の id が 1:1 で対応している
- greeting.compensated イベントが発生した行は status が `compensated` に更新される

## 6. DB Verification

```bash
# List all databases
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d postgres -c "\l"

# Check tables in each DB
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d greeter_db -c "\dt"

kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d gateway_db -c "\dt"

kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d caller_db -c "\dt"

kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d auth_db -c "\dt"

kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d lang_db -c "\dt"

# Check migration status
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d greeter_db -c "SELECT * FROM schema_migrations;"

# Check event_store (CQRS)
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d greeter_db \
  -c "SELECT event_id, stream_id, event_type, version, created_at FROM event_store ORDER BY created_at;"

# Check outbox_events
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d greeter_db \
  -c "SELECT id, event_type, topic, published, created_at FROM outbox_events ORDER BY created_at;"

# Check greetings data (CQRS read model — Projection により event_store から実体化される)
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d greeter_db \
  -c "SELECT * FROM greetings ORDER BY created_at;"

# Check invocations data (Projection の read model — 現時点では未実装のため空)
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d gateway_db \
  -c "SELECT * FROM invocations ORDER BY created_at;"

# Check event_store (gateway — Saga イベント)
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d gateway_db \
  -c "SELECT event_id, stream_id, event_type, version, created_at FROM event_store ORDER BY created_at;"

# Check outbox_events (gateway)
kubectl exec -n database postgresql-0 -c postgresql -- \
  env PGPASSWORD=devpass psql -U devuser -d gateway_db \
  -c "SELECT id, event_type, topic, published, created_at FROM outbox_events ORDER BY created_at;"
```
