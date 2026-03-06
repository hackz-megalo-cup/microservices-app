# Go スタイルガイド

> 本ドキュメントは [Google Go Style Guide](https://google.github.io/styleguide/go/)、
> [Style Decisions](https://google.github.io/styleguide/go/decisions)、
> [Best Practices](https://google.github.io/styleguide/go/best-practices) に基づき、
> microservice-app プロジェクト向けにカスタマイズした日本語版ガイドである。
> 原文ライセンス: [CC BY 3.0](https://creativecommons.org/licenses/by/3.0/) — Copyright Google LLC

---

## 1. 命名規則

### 1.1 基本方針

Go の命名は **MixedCaps**（大文字区切り）を使う。スネークケースは使用しない。

| 対象 | 規則 | 例 |
|------|------|----|
| エクスポートされる名前 | PascalCase | `NewService`, `RetryBudget` |
| 非エクスポート名 | camelCase | `callExternal`, `retryCount` |
| パッケージ名 | 小文字、1 単語推奨 | `greeter`, `platform` |
| インターフェース | 動詞 + `-er` が慣例 | `Reader`, `Handler` |
| レシーバ名 | 1〜2 文字、一貫性を保つ | `s`, `b`（型名の頭文字） |

### 1.2 頭字語

- 全大文字または全小文字で一貫させる: `HTTP`, `ID`, `URL`
- 混在させない: ~~`HttpClient`~~ → `HTTPClient`

### 1.3 パッケージ名

- 短く、小文字、アンダースコアなし
- パッケージ名の繰り返しを避ける: ~~`greeter.GreeterService`~~ → `greeter.Service`

---

## 2. エラーハンドリング

### 2.1 基本方針

- エラーは **必ずチェック** する（`errcheck` で検証済み）
- エラーを無視する場合は明示的に `_ =` を使い、理由をコメントする

### 2.2 エラーの比較

- `errors.Is()` / `errors.As()` を使用する（`errorlint` で検証済み）
- `==` でエラーを直接比較しない

```go
// BAD
if err == sql.ErrNoRows { ... }

// GOOD
if errors.Is(err, sql.ErrNoRows) { ... }
```

### 2.3 エラーのラッピング

- `fmt.Errorf("...: %w", err)` でコンテキストを追加する
- エラーチェーンを壊す `%v` は使わない

```go
// BAD
return fmt.Errorf("db query failed: %v", err)

// GOOD
return fmt.Errorf("db query failed: %w", err)
```

### 2.4 エラー文字列

- 小文字で始める、末尾に句読点をつけない（`revive: error-strings` で検証済み）

```go
// BAD
return errors.New("Failed to connect.")

// GOOD
return errors.New("failed to connect")
```

### 2.5 エラー型の命名

- カスタムエラー型は `...Error` で終わる（`errname` で検証済み）

---

## 3. インポート

### 3.1 グループ分け

`goimports` によって自動的に以下の順序でグループ化される:

```go
import (
    // 標準ライブラリ
    "context"
    "fmt"

    // 外部ライブラリ
    "connectrpc.com/connect"
    "go.opentelemetry.io/otel"

    // プロジェクト内パッケージ
    "github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)
```

### 3.2 ブランクインポート

- サイドエフェクト用のブランクインポート `_ "pkg"` は `main` パッケージに集約する
- `internal` パッケージでのブランクインポートは避ける

### 3.3 ドットインポート禁止

```go
// BAD
import . "testing"

// GOOD
import "testing"
```

---

## 4. 構造体と初期化

### 4.1 フィールド名付きリテラル

構造体リテラルではフィールド名を常に指定する。

```go
// BAD
srv := &http.Server{"", handler, 30 * time.Second}

// GOOD
srv := &http.Server{
    Handler:      handler,
    ReadTimeout:  30 * time.Second,
}
```

### 4.2 nil スライス

空スライスを返すときは `nil` を優先する。

```go
// BAD — 空スライスを明示的に構築
return []string{}

// GOOD — nil スライスを返す（JSON では null）
return nil
```

---

## 5. 並行処理

### 5.1 goroutine のライフタイム管理

**goroutine がいつ終了するか常に明確にする。**

```go
// BAD — 終了しない goroutine（リーク）
go func() {
    for range ticker.C {
        b.mu.Lock()
        b.used = 0
        b.mu.Unlock()
    }
}()

// GOOD — context によるキャンセル
go func() {
    for {
        select {
        case <-ctx.Done():
            ticker.Stop()
            return
        case <-ticker.C:
            b.mu.Lock()
            b.used = 0
            b.mu.Unlock()
        }
    }
}()
```

### 5.2 fire-and-forget goroutine

- 可能な限り `sync.WaitGroup` や `errgroup` で待機する
- やむを得ない場合はコメントでリスクを明記する

```go
// BAD
go func() {
    _ = insertToDB(context.Background(), data)
}()

// GOOD
g.Go(func() error {
    return insertToDB(ctx, data)
})
```

### 5.3 context の使用

- すべての関数の第一引数に `context.Context` を渡す
- `context.Background()` は `main` または最上位のみで使用する

---

## 6. テスト

### 6.1 テーブル駆動テスト

テストケースが複数ある場合はテーブル駆動テストを使用する。

```go
tests := []struct {
    name string
    input string
    want  string
}{
    {name: "empty", input: "", want: "Hello, World!"},
    {name: "with name", input: "Go", want: "Hello, Go!"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := greet(tt.input)
        if got != tt.want {
            t.Errorf("greet(%q) = %q, want %q", tt.input, got, tt.want)
        }
    })
}
```

### 6.2 テストヘルパー

テストヘルパー関数は先頭で `t.Helper()` を呼ぶ。

```go
func setupTestDB(t *testing.T) *pgxpool.Pool {
    t.Helper()
    // ...
}
```

---

## 7. 関数設計

### 7.1 関数の長さ

- 1 関数は画面 1 つ分（概ね 60 行以内）に収める
- 長くなる場合はヘルパー関数に分割する
- `run()` 関数の DB 初期化などは独立した関数に切り出す

### 7.2 戻り値

- 名前付き戻り値は **ドキュメント目的** でのみ使用
- naked return（裸の `return`）は避ける

```go
// BAD
func find(id int) (result string, err error) {
    // ...
    return // naked return
}

// GOOD
func find(id int) (string, error) {
    // ...
    return result, nil
}
```

---

## 8. golangci-lint 推奨追加設定

現在の `.golangci.yml` に対して、以下の追加を推奨する:

```yaml
linters:
  enable:
    - gocritic       # 広範なスタイルチェック
    - nakedret       # naked return 検出
    - gocognit       # 関数複雑度チェック
    - thelper        # テストヘルパーの t.Helper() 検出

  settings:
    revive:
      rules:
        - name: dot-imports       # ドットインポート禁止
        - name: context-as-argument  # context.Context を第一引数に
    gocognit:
      min-complexity: 30  # 段階的に厳格化
```

---

## 9. 自動生成コードの除外

`gen/` 配下のコードは protobuf / connect-go から自動生成されるため、lint の対象外とする（設定済み）。

---

*本ガイドは Google Go Style Guide (CC BY 3.0, Copyright Google LLC) を翻案したものです。*
