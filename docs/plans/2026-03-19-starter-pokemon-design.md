# Starter Pokémon Selection + E2E Test Design

**Date**: 2026-03-19
**Status**: Approved

## Overview

ゲスト登録直後にポケモンを持っていないユーザーに対し、御三家（Go / Python / Whitespace）から1匹を選ばせる画面を追加する。選択時にアイテム全7種を付与する。加えて、ゲスト登録→スターター選択→レイド→バトル→捕獲までのE2Eテストを追加する。

## Approach

**フロントエンド完結型**。バックエンドに新規RPCは追加せず、既存の `RegisterPokemon` + `SetActivePokemon` + `GrantItem` を組み合わせる。

## Starter Pokémon

| ID | Name | Type |
|---|---|---|
| `00000000-0000-0000-0000-000000000001` | Go | procedural |
| `00000000-0000-0000-0000-000000000002` | Python | dynamic |
| `00000000-0000-0000-0000-000000000009` | Whitespace | functional |

## Items (全7種 × 各1個)

| ID | Name |
|---|---|
| `018f4e1a-0001-7000-8000-000000000001` | どりーさん |
| `018f4e1a-0002-7000-8000-000000000002` | ざつくん |
| `018f4e1a-0003-7000-8000-000000000003` | レッドブル |
| `018f4e1a-0004-7000-8000-000000000004` | モンスター |
| `018f4e1a-0005-7000-8000-000000000005` | こんにゃく |
| `018f4e1a-0006-7000-8000-000000000006` | クッション |
| `018f4e1a-0007-7000-8000-000000000007` | ひよこ |

## User Flow

```
ゲスト登録 → ログイン → Home
  → caughtCount === 0 → redirect /starter-select
  → 御三家から1匹選択 → 確認
  → API calls:
    1. RegisterPokemon(userId, pokemonId)   ← auth service (直接呼出)
    2. SetActivePokemon(userId, pokemonId)  ← lobby service
    3. GrantItem(userId, itemId, 1) × 7     ← item service
  → navigate("/")
```

## File Changes

### New Files

| File | Description |
|---|---|
| `frontend/src/features/auth/components/starter-select.tsx` | 御三家選択画面UI |
| `frontend/src/features/auth/hooks/use-starter-select.ts` | 選択時のAPI呼び出しロジック |
| `services/internal/e2e/full_flow_test.go` | ゲスト→スターター→レイド→捕獲 E2Eテスト |

### Modified Files

| File | Change |
|---|---|
| `frontend/src/app/App.tsx` | `/starter-select` ルート追加 |
| `frontend/src/features/showcase/components/home.tsx` | `caughtCount === 0` でリダイレクト |

### No Backend Changes

既存RPCをそのまま使用:
- `auth.RegisterPokemon` — フロントから直接呼出（Traefik経由）
- `lobby.SetActivePokemon` — 既存RPC
- `item.GrantItem` — 既存RPC

## Frontend Component: StarterSelect

- 御三家3匹をカード表示（画像・名前・タイプ・ステータス）
- 1匹をタップで選択状態に → 「この子にする！」ボタンで確定
- 確定後ローディング表示 → 全API呼び出し完了後 `/` へ遷移
- エラー時はリトライ可能（リロードで `/starter-select` に戻る）

## Home Page Redirect Logic

```typescript
// home.tsx 内
const { caughtCount, isLoading } = useLobbyOverview(userId);

useEffect(() => {
  if (!isLoading && caughtCount === 0) {
    navigate("/starter-select", { replace: true });
  }
}, [isLoading, caughtCount, navigate]);
```

## E2E Test Flow

```
1. ゲスト登録 (RegisterUser)
2. ログイン (LoginUser) → JWT取得
3. スターターポケモン選択:
   - RegisterPokemon(userId, starterPokemonId)
   - SetActivePokemon(userId, starterPokemonId)
   - GrantItem × 7
4. GetUserPokemon → スターター1匹確認
5. GetUserItems → 7アイテム確認
6. GetActivePokemon → スターター確認
7. レイド作成 (CreateRaid)
8. レイド参加 (JoinRaid)
9. バトル開始 (StartBattle)
10. バトル終了 (HandleBattleFinished)
11. キャプチャセッション作成 (rate=1.0)
12. ボール投げ (ThrowBall) → success
13. セッション終了 (EndSession)
14. ポケモン登録 (RegisterPokemon for captured)
15. GetUserPokemon → 2匹確認（スターター + 捕獲）
16. イベントストア整合性検証
```

テスト実行: `cd services && go test -tags=integration -v -timeout 120s ./internal/e2e/...`
