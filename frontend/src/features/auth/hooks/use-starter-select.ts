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
