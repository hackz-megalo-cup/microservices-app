---
name: api-test
description: Register, login, get JWT token, and test API endpoints in one step. Use when testing gRPC/REST APIs locally via docker-compose or k8s.
argument-hint: "[endpoint] [json-body]"
allowed-tools: Bash(curl *), Bash(jq *)
---

# API Test Skill

ローカル環境（docker-compose or k8s）で動いているマイクロサービスの API を一発でテストする。
トークン取得 → API 呼び出しまでを自動化する。

## 環境の自動判定

まず以下の順でベース URL を判定する:

1. docker-compose 環境: `http://localhost:30081`（Traefik 経由）
2. k8s (port-forward) 環境: 各サービスの個別ポート

判定方法:
```bash
curl -fsS http://localhost:30081/ >/dev/null 2>&1
```
成功なら docker-compose、失敗なら k8s 環境と判断。

## トークン取得フロー

### Step 1: ユーザー登録（初回のみ、既に存在すればスキップ）
```bash
curl -s -X POST http://localhost:30081/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}'
```

### Step 2: ログインしてトークン取得
```bash
TOKEN=$(curl -s -X POST http://localhost:30081/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}' | jq -r '.token')
```

### Step 3: API 呼び出し

トークンが必要なエンドポイント:
```bash
curl -s -X POST http://localhost:30081/<endpoint> \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '<json-body>'
```

## 利用可能なエンドポイント一覧

| サービス | エンドポイント | 認証 | 用途 |
|---------|-------------|------|------|
| Greeter v1 | `greeter.v1.GreeterService/Greet` | 必要 | 挨拶 + イベントソーシング |
| Greeter v2 | `greeter.v2.GreeterService/Greet` | 不要 | 公開 API |
| Gateway | `gateway.v1.GatewayService/InvokeCustom` | 必要 | Saga パターンテスト |
| Auth | `/auth/register`, `/auth/login` | 不要 | 認証 |

## 引数の解釈

- `$ARGUMENTS` が空の場合: 全エンドポイントをヘルスチェック＋基本テスト
- エンドポイントが指定された場合: そのエンドポイントのみテスト
- `saga` と指定された場合: Gateway の Saga パターン（正常系 + compensation）を全パターンテスト

## Saga パターンのテスト

```bash
# 正常系
curl -s -X POST ... -d '{"name":"Bob"}'
# compensation (unauthorized)
curl -s -X POST ... -d '{"name":"unauthorized"}'
# compensation (unavailable)
curl -s -X POST ... -d '{"name":"unavailable"}'
```

## 出力

結果は `jq` でフォーマットして見やすく表示する。エラーがあれば原因を分析して報告する。
