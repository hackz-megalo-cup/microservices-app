---
name: dev-up
description: Start development environment with docker-compose, ensuring images are rebuilt. Use when starting up or restarting services after code changes.
argument-hint: "[service-name...]"
disable-model-invocation: true
allowed-tools: Bash(docker *)
---

# Dev Up Skill

docker-compose 環境を正しく起動する。**コード変更後に `docker compose up` だけでは古いイメージが使われる問題**を防ぐ。

## 実行手順

### 引数なし（全サービス）
```bash
cd /Users/thirdlf03/src/github.com/hackz-megalo-cup/microservices-app && docker compose up --build -d
```

### 特定サービスのみ
```bash
docker compose up --build -d $ARGUMENTS
```

### 起動後のヘルスチェック
```bash
# Traefik が起動するまで待機
for i in $(seq 1 30); do
  curl -fsS http://localhost:30081/ >/dev/null 2>&1 && break
  sleep 1
done

# 各サービスのヘルスチェック
docker compose ps --format json | jq -r '.[] | "\(.Name): \(.State) (\(.Health))"'
```

## よくある問題と対処

| 症状 | 原因 | 対処 |
|------|------|------|
| API が古い挙動のまま | イメージが再ビルドされていない | `docker compose build <service>` してから `up` |
| postgres 接続エラー | DB が起動前にサービスが接続 | `depends_on` + `service_healthy` を確認 |
| ポート衝突 | 前回の compose が残っている | `docker compose down` してから再起動 |

## クリーンリスタート

完全にやり直したい場合:
```bash
docker compose down -v  # ボリュームも削除
docker compose up --build -d
```

## 重要なポイント

- `--build` フラグは**必ず**つける。これを忘れるとキャッシュされた古いバイナリが使われる
- Traefik は `:30081` でリッスン（全 gRPC/REST をルーティング）
- PostgreSQL は `healthcheck` 付きなので `service_healthy` で依存を待てる
