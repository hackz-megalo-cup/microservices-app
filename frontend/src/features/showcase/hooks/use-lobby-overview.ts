import { useQuery } from "@connectrpc/connect-query";
import { getLobbyOverview } from "../../../gen/lobby/v1/lobby-LobbyService_connectquery";

export function useLobbyOverview(userId: string) {
  const query = useQuery(getLobbyOverview, { userId }, { enabled: !!userId });

  const items = query.data?.items ?? [];
  const pokedex = query.data?.pokedex ?? [];
  const raids = query.data?.raids ?? [];

  const caughtCount = pokedex.filter((entry) => entry.caught).length;
  const totalPokemonCount = pokedex.length;
  const activeRaidCount = raids.length;

  const error =
    query.error instanceof Error
      ? query.error
      : query.error
        ? new Error("ロビー概要取得失敗")
        : null;

  return {
    items,
    pokedex,
    caughtCount,
    totalPokemonCount,
    activeRaidCount,
    isLoading: query.isPending,
    error,
    refetch: query.refetch,
  };
}
