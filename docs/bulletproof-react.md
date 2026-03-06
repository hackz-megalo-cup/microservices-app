# Frontend アーキテクチャ: bulletproof-react パターン

## ディレクトリ構成

```
src/
├── app/                    # アプリケーションエントリーポイント
│   ├── main.tsx            # ブートストラップ (MSW初期化, ReactDOM.render)
│   ├── App.tsx             # ルートコンポーネント
│   └── providers/
│       └── app-provider.tsx # TransportProvider + QueryClientProvider
├── features/               # 機能別モジュール
│   ├── auth/
│   │   ├── api/            # API 呼び出しロジック (fetch)
│   │   ├── components/     # UI コンポーネント
│   │   ├── hooks/          # カスタムフック (状態管理 + API ラップ)
│   │   └── types/          # 型定義
│   ├── greeter/
│   │   ├── components/     # connect-query useQuery パターン
│   │   └── types/
│   └── gateway/
│       ├── components/     # connect-query useMutation パターン
│       └── types/
├── gen/                    # 自動生成コード (buf generate)
├── interceptors/           # connect-go インターセプター (認証トークン付与)
├── lib/                    # 共有ユーティリティ
│   ├── transport.ts        # Connect transport 設定
│   └── query-client.ts     # React Query クライアント設定
├── testing/                # テスト用ユーティリティ
│   ├── browser.ts          # MSW ワーカー
│   ├── handlers.ts         # MSW ハンドラー
│   └── test-utils.tsx      # テスト用プロバイダーラッパー
└── types/                  # 共有型定義
```

## Feature-First 設計原則

- 各 feature は独立したモジュールとして設計
- feature 間の直接 import は禁止（共有が必要な場合は `lib/` または `types/` に置く）
- 各 feature は `api/`, `components/`, `hooks/`, `types/` のサブディレクトリを持つ

## 各サブディレクトリの役割

### api/

- 外部 API との通信ロジック
- auth feature: REST API (fetch) のパターン
- greeter/gateway: connect-query が API 層の役割を担う

### components/

- UI コンポーネント
- `hooks/` と `api/` にのみ依存し、直接 fetch や transport を触らない

### hooks/

- カスタムフック
- auth: `useAuth()` で状態管理 + API 呼び出しをラップ
- greeter/gateway: connect-query の `useQuery` / `useMutation` がそのまま hooks パターン

### types/

- TypeScript 型定義
- API レスポンス型、コンポーネントプロパティ型

## connect-query との統合パターン

### Query パターン (greeter)

`useQuery(greet, { name }, { enabled: false })` で手動トリガー

### Mutation パターン (gateway)

`useMutation` + `createClient` で idempotency-key 付きリクエスト

## 新しい feature の追加手順

1. `src/features/<feature-name>/` ディレクトリ作成
2. `types/index.ts` に型定義
3. `api/` に API 呼び出しロジック（REST なら fetch、gRPC なら connect-query）
4. `hooks/` にカスタムフック
5. `components/` に UI コンポーネント
6. `App.tsx` にコンポーネントを追加
