# Microservices App - Claude Code Instructions

## プロジェクト構造

- **Go サービス**: `services/` （go.mod はここ）
- **Node.js サービス**: `node-services/`
- **フロントエンド**: `frontend/`
- **Proto 定義**: `proto/<service>/<version>/<service>.proto`
- **生成コード**: `services/gen/` (Go), `frontend/src/gen/` (TypeScript)
- **K8s マニフェスト**: `deploy/manifests/`

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
# ✅ コード変更後は必ずこれ
docker compose up --build -d

# ❌ 古いイメージが使われる
docker compose up -d
```

### Proto 変更後は buf generate を実行
```bash
buf lint && buf generate
find services/gen -name '*.go' -exec gofmt -w {} +
git add services/gen/ frontend/src/gen/
```

## ローカル開発環境

- docker-compose: Traefik が `localhost:30081` で全サービスをルーティング
- API テスト: `/api-test` スキルを使う
- スモークテスト: `/smoke-test` スキルを使う

## コミット規約

`<type>(<scope>): <description>` 形式を使う。
- type: feat, fix, docs, refactor, style, chore, test
- scope: サービス名やコンポーネント名

## スタイルガイド

- Go: `docs/go-style-guide.md`
- TypeScript: `docs/typescript-style-guide.md`
- React: `docs/bulletproof-react.md`
- レビュー: `REVIEW.md`
