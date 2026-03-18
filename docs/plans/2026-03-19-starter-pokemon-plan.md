# Starter Pokémon Selection Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** ゲスト登録直後に御三家（Go/Python/Whitespace）から1匹選ばせ、全アイテム付与し、E2Eテストで全フローを検証する。

**Architecture:** フロントエンド主導。auth proto に `ChooseStarter` RPC を1つ追加し、フロントから `ChooseStarter` + `SetActivePokemon` + `GrantItem×7` を呼ぶ。Home で `caughtCount === 0` なら `/starter-select` にリダイレクト。

**Tech Stack:** Go (connect-rpc), React 19, protobuf (buf), testcontainers (E2E)

---

### Task 1: Add `ChooseStarter` RPC to auth proto

**Files:**
- Modify: `proto/auth/v1/auth.proto`

**Step 1: Add RPC and messages to proto**

末尾に追加:

```protobuf
  // スターターポケモン選択
  rpc ChooseStarter(ChooseStarterRequest) returns (ChooseStarterResponse) {}
```

`service AuthService` ブロック内に上記を追加。ファイル末尾にメッセージ定義:

```protobuf
// スターターポケモン選択リクエスト
message ChooseStarterRequest {
  string user_id = 1;
  string pokemon_id = 2;
}

// スターターポケモン選択レスポンス
message ChooseStarterResponse {}
```

**Step 2: buf lint + generate**

```bash
buf lint && buf generate
```

**Step 3: Format generated Go code**

```bash
find services/gen -name '*.go' -exec gofmt -w {} +
```

**Step 4: Stage generated files**

```bash
git add proto/auth/v1/auth.proto services/gen/ frontend/src/gen/
```

---

### Task 2: Implement `ChooseStarter` handler in auth service

**Files:**
- Modify: `services/internal/auth/service.go`

**Step 1: Add `ChooseStarter` method**

`service.go` の末尾（`isUniqueViolation` 関数の前）に追加:

```go
// starterPokemonIDs is the set of allowed starter Pokémon.
var starterPokemonIDs = map[string]bool{
	"00000000-0000-0000-0000-000000000001": true, // Go
	"00000000-0000-0000-0000-000000000002": true, // Python
	"00000000-0000-0000-0000-000000000009": true, // Whitespace
}

// ChooseStarter registers a starter Pokémon for a new user.
func (s *Service) ChooseStarter(ctx context.Context, req *connect.Request[authv1.ChooseStarterRequest]) (*connect.Response[authv1.ChooseStarterResponse], error) {
	userID := req.Msg.GetUserId()
	pokemonID := req.Msg.GetPokemonId()

	if userID == "" || pokemonID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id and pokemon_id are required"))
	}
	if !starterPokemonIDs[pokemonID] {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid starter pokemon_id: %s", pokemonID))
	}

	if err := s.RegisterPokemon(ctx, userID, pokemonID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register starter pokemon: %w", err))
	}

	return connect.NewResponse(&authv1.ChooseStarterResponse{}), nil
}
```

**Step 2: Build check**

```bash
cd services && go build ./cmd/auth
```

**Step 3: Commit**

```bash
git add services/internal/auth/service.go
git commit -m "feat(auth): add ChooseStarter RPC for starter Pokémon selection"
```

---

### Task 3: Create `use-starter-select.ts` hook

**Files:**
- Create: `frontend/src/features/auth/hooks/use-starter-select.ts`

**Ref docs:**
- `frontend/src/features/showcase/hooks/use-active-pokemon.ts` — connect-query mutation pattern
- `frontend/src/gen/auth/v1/auth-AuthService_connectquery.ts` — generated `chooseStarter`
- `frontend/src/gen/item/v1/item-ItemService_connectquery.ts` — generated `grantItem`
- `frontend/src/gen/lobby/v1/lobby-LobbyService_connectquery.ts` — generated `setActivePokemon`

**Step 1: Write the hook**

```typescript
import { createClient } from "@connectrpc/connect";
import { useTransport } from "@connectrpc/connect-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { AuthService } from "../../../gen/auth/v1/auth_pb";
import { ItemService } from "../../../gen/item/v1/item_pb";
import { LobbyService } from "../../../gen/lobby/v1/lobby_pb";

const STARTER_ITEM_IDS = [
  "018f4e1a-0001-7000-8000-000000000001", // どりーさん
  "018f4e1a-0002-7000-8000-000000000002", // ざつくん
  "018f4e1a-0003-7000-8000-000000000003", // レッドブル
  "018f4e1a-0004-7000-8000-000000000004", // モンスター
  "018f4e1a-0005-7000-8000-000000000005", // こんにゃく
  "018f4e1a-0006-7000-8000-000000000006", // クッション
  "018f4e1a-0007-7000-8000-000000000007", // ひよこ
];

export function useStarterSelect(userId: string) {
  const transport = useTransport();
  const queryClient = useQueryClient();

  const authClient = useMemo(() => createClient(AuthService, transport), [transport]);
  const lobbyClient = useMemo(() => createClient(LobbyService, transport), [transport]);
  const itemClient = useMemo(() => createClient(ItemService, transport), [transport]);

  const mutation = useMutation({
    mutationFn: async (pokemonId: string) => {
      // 1. Register starter Pokémon
      await authClient.chooseStarter({ userId, pokemonId });

      // 2. Set as active Pokémon
      await lobbyClient.setActivePokemon(
        { userId, pokemonId },
        { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
      );

      // 3. Grant all starter items
      await Promise.all(
        STARTER_ITEM_IDS.map((itemId) =>
          itemClient.grantItem(
            { userId, itemId, quantity: 1, reason: "starter_bonus" },
            { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
          ),
        ),
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries();
    },
  });

  return {
    selectStarter: mutation.mutateAsync,
    isPending: mutation.isPending,
    error: mutation.error instanceof Error ? mutation.error : null,
  };
}
```

**Step 2: Lint check**

```bash
cd frontend && npx biome check src/features/auth/hooks/use-starter-select.ts
```

---

### Task 4: Create `starter-select.tsx` component

**Files:**
- Create: `frontend/src/features/auth/components/starter-select.tsx`

**Ref docs:**
- `frontend/src/features/showcase/api/pokemon.ts` — `getPokemonImageUrl`
- `frontend/src/styles/global.css` — existing styles (showcase-screen, etc.)

**Step 1: Write the component**

```tsx
import { useState } from "react";
import { useNavigate } from "react-router";
import { getPokemonImageUrl } from "../../showcase/api/pokemon";
import { useAuthContext } from "../hooks/use-auth-context";
import { useStarterSelect } from "../hooks/use-starter-select";
import "../../../styles/global.css";

const STARTERS = [
  {
    id: "00000000-0000-0000-0000-000000000001",
    name: "Go",
    type: "procedural",
    hp: 85,
    attack: 70,
    speed: 90,
    move: "ゴルーチン乱舞",
  },
  {
    id: "00000000-0000-0000-0000-000000000002",
    name: "Python",
    type: "dynamic",
    hp: 90,
    attack: 65,
    speed: 60,
    move: "インデント地獄",
  },
  {
    id: "00000000-0000-0000-0000-000000000009",
    name: "Whitespace",
    type: "functional",
    hp: 50,
    attack: 50,
    speed: 100,
    move: "虚空の一撃",
  },
] as const;

export function StarterSelect() {
  const { user } = useAuthContext();
  const userId = user?.id ?? "";
  const navigate = useNavigate();
  const { selectStarter, isPending, error } = useStarterSelect(userId);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const handleConfirm = async () => {
    if (!selectedId) return;
    try {
      await selectStarter(selectedId);
      navigate("/", { replace: true });
    } catch {
      // error is exposed via hook
    }
  };

  return (
    <div className="showcase-screen items-center justify-center gap-6 px-6 py-8">
      <div className="flex flex-col items-center gap-2">
        <span className="text-4xl">🎉</span>
        <h1 className="text-2xl font-bold text-text-primary m-0">パートナーを選ぼう！</h1>
        <p className="text-sm text-text-secondary m-0 text-center">
          最初のポケモンを1匹選んでね
        </p>
      </div>

      <div className="flex flex-col gap-3 w-full max-w-sm">
        {STARTERS.map((pokemon) => (
          <button
            key={pokemon.id}
            type="button"
            onClick={() => setSelectedId(pokemon.id)}
            disabled={isPending}
            className={`flex items-center gap-4 p-4 rounded-2xl border-2 cursor-pointer transition-all ${
              selectedId === pokemon.id
                ? "border-accent bg-accent/10"
                : "border-transparent bg-bg-card hover:bg-bg-hover"
            } disabled:opacity-50`}
          >
            <img
              src={getPokemonImageUrl({ name: pokemon.name })}
              alt={pokemon.name}
              className="w-16 h-16 rounded-full object-cover"
            />
            <div className="flex flex-col items-start gap-1 flex-1">
              <span className="text-lg font-bold text-text-primary">{pokemon.name}</span>
              <span className="text-xs text-text-secondary">{pokemon.type}</span>
              <div className="flex gap-3 text-xs text-text-secondary">
                <span>HP {pokemon.hp}</span>
                <span>ATK {pokemon.attack}</span>
                <span>SPD {pokemon.speed}</span>
              </div>
              <span className="text-xs text-accent">⚡ {pokemon.move}</span>
            </div>
            {selectedId === pokemon.id && (
              <span className="text-2xl">✓</span>
            )}
          </button>
        ))}
      </div>

      {error && <p className="text-red-400 text-sm m-0">{error.message}</p>}

      <button
        type="button"
        onClick={() => void handleConfirm()}
        disabled={!selectedId || isPending}
        className="w-full max-w-sm h-14 bg-accent text-bg-primary font-bold text-lg rounded-3xl cursor-pointer hover:opacity-90 transition-opacity disabled:opacity-50"
      >
        {isPending ? "準備中..." : "この子にする！"}
      </button>
    </div>
  );
}
```

**Step 2: Lint check**

```bash
cd frontend && npx biome check src/features/auth/components/starter-select.tsx
```

---

### Task 5: Update routing and home page redirect

**Files:**
- Modify: `frontend/src/app/App.tsx`
- Modify: `frontend/src/features/showcase/components/home.tsx`

**Step 1: Add route in `App.tsx`**

Import を追加:

```typescript
import { StarterSelect } from "../features/auth/components/starter-select";
```

`<Route path="/login" .../>` の直後に追加:

```tsx
<Route
  path="/starter-select"
  element={
    <RequireAuth>
      <StarterSelect />
    </RequireAuth>
  }
/>
```

**Step 2: Add redirect in `home.tsx`**

Import に `useNavigate` を追加（`react-router` から）:

```typescript
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
```

`Home` 関数内、既存の hooks の直後（`const [showPokemonSelector, ...]` の前）に追加:

```typescript
const navigate = useNavigate();

useEffect(() => {
  if (!overviewLoading && caughtCount === 0) {
    navigate("/starter-select", { replace: true });
  }
}, [overviewLoading, caughtCount, navigate]);
```

**Step 3: Lint check**

```bash
cd frontend && npx biome check src/app/App.tsx src/features/showcase/components/home.tsx
```

**Step 4: Build check**

```bash
cd frontend && npm run build
```

**Step 5: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add starter Pokémon selection screen"
```

---

### Task 6: Update E2E test with starter + item flow

**Files:**
- Modify: `services/internal/e2e/full_flow_test.go`

**Ref docs:**
- `services/internal/item/service.go` — GrantItem handler
- `services/internal/item/embed.go` — MigrationsFS
- `services/internal/lobby/service.go` — SetActivePokemon handler (requires authClient)
- `services/internal/lobby/embed.go` — MigrationsFS

**Step 1: Rewrite E2E test with full flow**

Replace the entire file with the updated version that adds:
- Item service setup (new DB + migrations)
- Lobby service setup (new DB + migrations)
- Step 3: ChooseStarter (calls auth service directly)
- Step 4: SetActivePokemon (lobby service — skip authClient ownership check by passing nil)
- Step 5: GrantItem × 7 (item service)
- Step 6: Verify items via DB query
- Step 7-13: Existing raid + capture flow
- Step 14: Verify 2 Pokémon owned

The key additions to imports:

```go
import (
    // existing imports...
    "github.com/hackz-megalo-cup/microservices-app/services/internal/item"
    "github.com/hackz-megalo-cup/microservices-app/services/internal/lobby"
    itemv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/item/v1"
    lobbyv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/lobby/v1"
)
```

Add 2 more databases: `item_e2e`, `lobby_e2e`.

New steps after login:

```go
// Step 3: ChooseStarter
starterPokemonID := "00000000-0000-0000-0000-000000000001" // Go
chooseResp, err := authSvc.ChooseStarter(ctx, connect.NewRequest(&authv1.ChooseStarterRequest{
    UserId:    userID,
    PokemonId: starterPokemonID,
}))
// verify no error

// Step 4: SetActivePokemon
// lobby service with nil authClient skips ownership check
setResp, err := lobbySvc.SetActivePokemon(ctx, connect.NewRequest(&lobbyv1.SetActivePokemonRequest{
    UserId:    userID,
    PokemonId: starterPokemonID,
}))
// verify success

// Step 5: GrantItem × 7
starterItemIDs := []string{
    "018f4e1a-0001-7000-8000-000000000001",
    "018f4e1a-0002-7000-8000-000000000002",
    "018f4e1a-0003-7000-8000-000000000003",
    "018f4e1a-0004-7000-8000-000000000004",
    "018f4e1a-0005-7000-8000-000000000005",
    "018f4e1a-0006-7000-8000-000000000006",
    "018f4e1a-0007-7000-8000-000000000007",
}
for _, itemID := range starterItemIDs {
    _, err := itemSvc.GrantItem(ctx, connect.NewRequest(&itemv1.GrantItemRequest{
        UserId:   userID,
        ItemId:   itemID,
        Quantity: 1,
        Reason:   "starter_bonus",
    }))
    // verify no error
}

// Step 6: Verify starter ownership
pokemonResp, err := authSvc.GetUserPokemon(ctx, connect.NewRequest(&authv1.GetUserPokemonRequest{
    UserId: userID,
}))
// verify 1 pokemon, pokemonIDs contains starterPokemonID
```

Final verification (step 15) — check 2 Pokémon:

```go
// Verify final state: 2 Pokémon (starter + captured)
finalPokemonResp, err := authSvc.GetUserPokemon(ctx, ...)
// verify len == 2
// verify contains starterPokemonID and bossPokemonID
```

**Step 2: Build check**

```bash
cd services && go vet -tags=integration ./internal/e2e/...
```

**Step 3: Commit**

```bash
git add services/internal/e2e/
git commit -m "test(e2e): add full flow test with starter selection and capture"
```

---

### Task Summary

| # | Task | Type | Files |
|---|---|---|---|
| 1 | Add ChooseStarter RPC to proto | proto | `proto/auth/v1/auth.proto`, gen files |
| 2 | Implement ChooseStarter handler | backend | `services/internal/auth/service.go` |
| 3 | Create use-starter-select hook | frontend | `frontend/src/features/auth/hooks/use-starter-select.ts` |
| 4 | Create StarterSelect component | frontend | `frontend/src/features/auth/components/starter-select.tsx` |
| 5 | Update routing + home redirect | frontend | `App.tsx`, `home.tsx` |
| 6 | Update E2E test | test | `services/internal/e2e/full_flow_test.go` |
