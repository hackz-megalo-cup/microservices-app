# コードレビューチェックリスト

本ドキュメントは `docs/` 配下のスタイルガイドに基づき、PRレビュー時に確認すべき項目をまとめたものである。

### 参照ドキュメント

- [Go スタイルガイド](docs/go-style-guide.md)
- [TypeScript スタイルガイド](docs/typescript-style-guide.md)
- [Frontend アーキテクチャ: bulletproof-react パターン](docs/bulletproof-react.md)

---

## 共通

- [ ] 自動生成コード（`gen/`, `src/gen/`）に対して手動変更を加えていないか
- [ ] CI（lint / format / test）が通っているか
- [ ] 変更の意図がPRの説明やコミットメッセージから読み取れるか

---

## Go

> 詳細: [Go スタイルガイド](docs/go-style-guide.md)

### 命名規則

- [ ] エクスポート名は PascalCase、非エクスポート名は camelCase になっているか
- [ ] パッケージ名は小文字1単語で、パッケージ名の繰り返しを避けているか（`greeter.Service` ○ / `greeter.GreeterService` ×）
- [ ] 頭字語は全大文字または全小文字で統一されているか（`HTTPClient` ○ / `HttpClient` ×）
- [ ] レシーバ名は1〜2文字で一貫しているか

### エラーハンドリング

- [ ] エラーを必ずチェックしているか（無視する場合は `_ =` + コメント）
- [ ] エラー比較に `errors.Is()` / `errors.As()` を使っているか（`==` で直接比較していないか）
- [ ] エラーのラッピングに `%w` を使っているか（`%v` で chain を壊していないか）
- [ ] エラー文字列は小文字始まり・末尾句読点なしか
- [ ] カスタムエラー型は `...Error` で終わっているか

### インポート

- [ ] インポートが「標準ライブラリ / 外部ライブラリ / プロジェクト内」の順にグループ化されているか
- [ ] ブランクインポート `_ "pkg"` が `main` パッケージに集約されているか
- [ ] ドットインポート（`import . "pkg"`）を使っていないか

### 構造体と初期化

- [ ] 構造体リテラルでフィールド名を指定しているか
- [ ] 空スライスの返却に `nil` を使っているか（`[]string{}` ではなく）

### 並行処理

- [ ] goroutine のライフタイムが明確か（context によるキャンセルや WaitGroup での待機）
- [ ] fire-and-forget goroutine を避けているか（やむを得ない場合はコメントあり）
- [ ] 関数の第一引数に `context.Context` を渡しているか
- [ ] `context.Background()` の使用が `main` または最上位に限定されているか

### 関数設計

- [ ] 1関数が概ね60行以内に収まっているか
- [ ] naked return（裸の `return`）を使っていないか
- [ ] 名前付き戻り値がドキュメント目的以外で使われていないか

### Lint

- [ ] `golangci-lint` の警告がないか

---

## TypeScript / React

> 詳細: [TypeScript スタイルガイド](docs/typescript-style-guide.md)

### 命名規則

- [ ] ファイル名はケバブケース（`my-component.tsx`）か
- [ ] 変数・関数は camelCase、型・インターフェースは PascalCase か
- [ ] モジュールレベル定数は CONSTANT_CASE か

### 型システム

- [ ] `any` を使っていないか（`unknown` + 型ガードで代替）
- [ ] 推論可能な型注釈を冗長に書いていないか（`const name: string = "hello"` ×）
- [ ] `as Type` より型注釈 `: Type` を優先しているか
- [ ] 非 null アサーション `!` を避け、明示的なチェックをしているか
- [ ] `object` 型の代わりに具体的な interface / type を定義しているか

### インポート

- [ ] 型のみのインポートに `import type` を使っているか
- [ ] default export を使っていないか（設定ファイル除く）
- [ ] `import *` を避け、名前付きインポートを使っているか

### 変数と制御構造

- [ ] `var` を使わず `const` / `let` のみか
- [ ] 単一行の `if` / `for` でもブレースを付けているか
- [ ] 等値比較に `===` / `!==` を使っているか（`==` ×）
- [ ] `&&` チェーンではなくオプショナルチェーン `?.` を使っているか

### 関数

- [ ] トップレベル関数は `function` 宣言、コールバックはアロー関数か
- [ ] React コンポーネントは `function` 宣言か
- [ ] `arguments` オブジェクトではなくレストパラメータ `...args` を使っているか

### フォーマット

- [ ] インデントはスペース2つか
- [ ] セミコロン・トレイリングカンマが付いているか

### Lint

- [ ] `biome` の警告がないか

---

## フロントエンドアーキテクチャ（bulletproof-react）

> 詳細: [Frontend アーキテクチャ: bulletproof-react パターン](docs/bulletproof-react.md)

### ディレクトリ構成

- [ ] 新しい機能は `src/features/<feature-name>/` に配置されているか
- [ ] feature 内が `api/`, `components/`, `hooks/`, `types/` で構成されているか
- [ ] feature 間の直接 import がないか（共有は `lib/` または `types/` を経由）

### 責務の分離

- [ ] コンポーネントが直接 fetch や transport を呼んでいないか（hooks / api 経由）
- [ ] API 呼び出しロジックが `api/` または connect-query に集約されているか
- [ ] 状態管理 + API ラップがカスタムフック（`hooks/`）にまとめられているか

### connect-query 統合

- [ ] Query パターン: `useQuery` を正しく使っているか
- [ ] Mutation パターン: `useMutation` + idempotency-key を正しく使っているか

---

## レビューの心構え

- 自動生成コード（`gen/`）の変更は基本的にスキップしてよい
- lint / format で検出できるものは CI に任せ、レビューではロジックや設計に集中する
- 「なぜこの実装にしたか」が不明な場合は、指摘ではなく質問として聞く
