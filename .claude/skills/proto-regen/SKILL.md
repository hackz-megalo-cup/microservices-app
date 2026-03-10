---
name: proto-regen
description: Regenerate protobuf code after proto file changes. Use when editing .proto files, adding new RPC methods, or updating message types.
user-invocable: true
allowed-tools: Bash(buf *), Bash(gofmt *), Bash(git add *)
---

# Proto Regeneration Skill

`.proto` ファイルの変更後に必要な一連の作業を自動化する。

## 実行手順

### Step 1: buf lint で proto の検証
```bash
cd /Users/thirdlf03/src/github.com/hackz-megalo-cup/microservices-app && buf lint
```
エラーがあれば修正を提案してから先に進む。

### Step 2: buf breaking で破壊的変更チェック
```bash
buf breaking --against '.git#branch=main'
```
破壊的変更がある場合はユーザーに警告する（CI でも検出されるため）。

### Step 3: コード生成
```bash
buf generate
```

### Step 4: Go コードのフォーマット
```bash
find services/gen -name '*.go' -exec gofmt -w {} +
```

### Step 5: 生成ファイルのステージング
```bash
git add services/gen/ frontend/src/gen/
```

## 注意事項

- `buf generate` は必ずリポジトリルートから実行する
- 生成された `*.pb.go`, `*.connect.go`, `*_pb.ts` ファイルは git にコミットする（gitignore されていない）
- Go のビルドは `services/` ディレクトリから行う（`go.mod` がそこにあるため）
- proto ファイルは `proto/<service>/<version>/<service>.proto` の構造

## 生成されるファイル

| 言語 | 出力先 | ファイル |
|------|--------|---------|
| Go | `services/gen/go/<service>/<version>/` | `*.pb.go` |
| Go (connect) | `services/gen/go/<service>/<version>/<service><version>connect/` | `*.connect.go` |
| TypeScript | `frontend/src/gen/<service>/<version>/` | `*_pb.ts` |
