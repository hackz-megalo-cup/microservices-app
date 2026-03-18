import { useQuery } from "@connectrpc/connect-query";
import { getLobbyOverview } from "../../../gen/lobby/v1/lobby-LobbyService_connectquery";
import { listPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { useAuthContext } from "../../../lib/auth";
import { adaptPokemonToUi } from "../api/pokemon";

export function useCollectionPokemon() {
  const { user } = useAuthContext();
  const userId = user?.id ?? "";

  const masterQuery = useQuery(listPokemon, {});
  const overviewQuery = useQuery(getLobbyOverview, { userId }, { enabled: !!userId });

  const caughtSet = new Set(
    (overviewQuery.data?.pokedex ?? []).filter((e) => e.caught).map((e) => e.pokemonId),
  );

  const pokemon =
    masterQuery.data?.pokemon.map((entry, index) =>
      adaptPokemonToUi(entry, index, caughtSet.has(entry.id)),
    ) ?? [];

  const error =
    masterQuery.error instanceof Error
      ? masterQuery.error
      : masterQuery.error
        ? new Error("マスターデータ取得失敗")
        : null;

  return {
    pokemon,
    isLoading: masterQuery.isPending || overviewQuery.isPending,
    error,
    refetch: masterQuery.refetch,
  };
}
