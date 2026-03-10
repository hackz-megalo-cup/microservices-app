# TypeScript スタイルガイド

> 本ドキュメントは [Google TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html) に基づき、
> microservice-app プロジェクト向けにカスタマイズした日本語版ガイドである。
> 原文ライセンス: [CC BY 3.0](https://creativecommons.org/licenses/by/3.0/) — Copyright Google LLC

---

## 1. ソースファイルの基本

### 1.1 ファイル名

- **ケバブケース**（`my-component.tsx`）を使用する
- テストファイルは `.test.ts` / `.test.tsx` とする

### 1.2 エンコーディング

- UTF-8 のみ。BOM なし

### 1.3 インポート

```typescript
// 1) Node / 外部ライブラリ
import { useState } from "react";

// 2) プロジェクト内モジュール
import { transport } from "@/lib/transport";
```

- **`import type`** を型のみのインポートに必ず使用する（biome: `useImportType: "error"`）
- **default export 禁止**（biome: `noDefaultExport: "error"`）
  - 例外: `vite.config.ts` など設定ファイル
- **dot import (`import *`) は避ける** — 名前付きインポートを使用

---

## 2. 型システム

### 2.1 `any` の禁止

- `any` は使用しない（biome: `noExplicitAny: "error"` 推奨）
- 型が不明な場合は `unknown` を使い、型ガードで絞り込む

```typescript
// BAD
function parse(input: any): any { ... }

// GOOD
function parse(input: unknown): Result {
  if (typeof input === "string") { ... }
}
```

### 2.2 型推論の活用

- 推論可能な型注釈は書かない（biome: `noInferrableTypes: "error"`）

```typescript
// BAD
const name: string = "hello";

// GOOD
const name = "hello";
```

### 2.3 型アサーション

- `as Type` より型注釈 `: Type` を優先する
- 非 null アサーション `!` は避け、明示的なチェックを行う

```typescript
// BAD
const el = document.getElementById("root")!;

// GOOD
const el = document.getElementById("root");
if (!el) throw new Error("root element not found");
```

### 2.4 `object` 型の禁止

- `object` は曖昧すぎるので、具体的な interface / type を定義する

```typescript
// BAD
const [response, setResponse] = useState<object | null>(null);

// GOOD
interface AuthResponse {
  token?: string;
  error?: string;
}
const [response, setResponse] = useState<AuthResponse | null>(null);
```

### 2.5 interface vs type

- オブジェクトの形状には `interface` を優先する
- ユニオン型やマップ型には `type` を使用する

---

## 3. 変数と宣言

### 3.1 `var` 禁止

- `const` / `let` のみ使用する（biome: `noVar: "error"`）
- 変更しない変数は `const` を使う

### 3.2 命名規則

| 対象 | 規則 | 例 |
|------|------|----|
| 変数・関数 | camelCase | `userName`, `fetchData()` |
| 型・インターフェース・クラス | PascalCase | `UserProfile`, `AuthService` |
| enum メンバ | CONSTANT_CASE or PascalCase | `HTTP_OK`, `NotFound` |
| 定数（モジュールレベル） | CONSTANT_CASE | `MAX_RETRY_COUNT` |
| ファイル名 | kebab-case | `auth-panel.tsx` |

### 3.3 略語

- 2文字略語は全大文字: `IO`, `DB`
- 3文字以上はパスカルケース: `Http`, `Xml`

---

## 4. 制御構造

### 4.1 ブレース必須

- 単一行の `if` / `for` / `while` でもブレースを省略しない（biome: `useBlockStatements: "error"` 推奨）

```typescript
// BAD
if (err) return;

// GOOD
if (err) {
  return;
}
```

### 4.2 等値比較

- 常に `===` / `!==` を使用する（biome: `noDoubleEquals: "error"`）

### 4.3 オプショナルチェーン

- `&&` チェーンよりオプショナルチェーン `?.` を使う（biome: `useOptionalChain: "error"`）

---

## 5. 関数

### 5.1 アロー関数 vs function 宣言

- **トップレベルの名前付き関数**: `function` 宣言を使用
- **コールバック・ネストされた関数**: アロー関数を使用
- React コンポーネント: `function` 宣言を使用

```typescript
// トップレベル — function 宣言
export function GreeterDemo() {
  // ネスト — アロー関数
  const handleClick = () => { ... };
}
```

### 5.2 引数

- `arguments` オブジェクトは使わず、レストパラメータ `...args` を使用する（biome: `noArguments: "error"` 推奨）
- アロー関数の引数は常に括弧で囲む（biome: `arrowParentheses: "always"`）

---

## 6. フォーマット

| 項目 | 設定 |
|------|------|
| インデント | スペース 2 つ |
| 行幅 | 100 文字 |
| クォート | ダブルクォート |
| セミコロン | 常に付与 |
| トレイリングカンマ | 常に付与 |

---

## 7. biome.json 推奨追加ルール

現在の設定に対して、以下の追加を推奨する:

```jsonc
{
  "linter": {
    "rules": {
      "style": {
        "useBlockStatements": "error", // ブレース必須
        "noNonNullAssertion": "warn",  // 非 null アサーション警告
        "noParameterAssign": "error",  // パラメータ再代入禁止
        "useConst": "error",           // const 優先
        "noNamespace": "error",        // namespace 禁止
        "noExplicitAny": "error"       // any 禁止（warn → error に昇格）
      }
    }
  }
}
```

---

## 8. 自動生成コードの除外

`src/gen/` 配下のコードは protobuf から自動生成されるため、lint / format の対象外とする（設定済み）。

---

*本ガイドは Google TypeScript Style Guide (CC BY 3.0, Copyright Google LLC) を翻案したものです。*
