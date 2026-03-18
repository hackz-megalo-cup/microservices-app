import { useQuery } from "@connectrpc/connect-query";
import { getPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { adaptPokemonToUi } from "../api/pokemon";

function parsePokemonIndex(id: string): number {
  const numericId = Number.parseInt(id, 10);

  if (!Number.isFinite(numericId) || numericId <= 0) {
    return 0;
  }

  return numericId - 1;
}

export function usePokemonDetail(id: string) {
  const query = useQuery(getPokemon, { id });
  const pokemon = query.data?.pokemon
    ? adaptPokemonToUi(query.data.pokemon, parsePokemonIndex(id))
    : null;
  const error =
    query.error instanceof Error
      ? query.error
      : query.error
        ? new Error("ポケモン詳細取得失敗")
        : null;

  return {
    pokemon,
    isLoading: query.isPending,
    error,
    refetch: query.refetch,
  };
}
