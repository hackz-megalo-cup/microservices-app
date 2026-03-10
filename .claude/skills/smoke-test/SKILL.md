---
name: smoke-test
description: Run comprehensive smoke tests against local docker-compose or k8s environment. Use after deploying or restarting services to verify everything works.
disable-model-invocation: true
allowed-tools: Bash(curl *), Bash(jq *), Bash(docker *), Bash(kubectl *)
---

# Smoke Test Skill

ローカル環境の全サービスに対してスモークテストを実行する。

## 環境判定

docker-compose か k8s かを自動判定する:

```bash
# docker-compose チェック
docker compose ps --format json 2>/dev/null | jq -e 'length > 0' >/dev/null 2>&1
```

## docker-compose 環境でのテスト

ベース URL: `http://localhost:30081`

### Step 1: ヘルスチェック

全サービスの状態を確認:
```bash
docker compose ps
```

Traefik 経由で各サービスのヘルスをチェック:
```bash
curl -fsS http://localhost:30081/ >/dev/null && echo "frontend: OK"
```

### Step 2: Auth テスト

```bash
# Register（既存なら 409 でも OK）
curl -s -X POST http://localhost:30081/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"smoke@test.com","password":"pass123"}'

# Login
TOKEN=$(curl -s -X POST http://localhost:30081/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"smoke@test.com","password":"pass123"}' | jq -r '.token')

echo "Token acquired: ${TOKEN:0:20}..."
```

### Step 3: Greeter テスト

```bash
# v1 (認証必要)
curl -s -X POST http://localhost:30081/greeter.v1.GreeterService/Greet \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"SmokeTest"}'

# v2 (認証不要)
curl -s -X POST http://localhost:30081/greeter.v2.GreeterService/Greet \
  -H "Content-Type: application/json" \
  -d '{"name":"SmokeTest"}'
```

### Step 4: Gateway Saga テスト

```bash
# 正常系
curl -s -X POST http://localhost:30081/gateway.v1.GatewayService/InvokeCustom \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Bob"}'

# compensation (unauthorized)
curl -s -X POST http://localhost:30081/gateway.v1.GatewayService/InvokeCustom \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"unauthorized"}'

# compensation (unavailable)
curl -s -X POST http://localhost:30081/gateway.v1.GatewayService/InvokeCustom \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"unavailable"}'
```

### Step 5: DB 検証（オプション）

```bash
# イベントストアにイベントが記録されているか
docker compose exec -T postgres psql -U devuser -d greeter_db \
  -c "SELECT count(*) FROM event_store;"
```

## k8s 環境でのテスト

k8s 環境の場合は `scripts/smoke-test.sh` を実行する:
```bash
bash scripts/smoke-test.sh
```

## 結果の報告

各ステップの結果を以下の形式で報告する:
- OK: 正常に動作
- FAIL: エラー内容と推定原因
- SKIP: 環境が対応していない

全テスト通過なら「Smoke test passed」と報告。失敗があれば修正方法を提案する。
