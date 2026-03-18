import { useQuery } from "@connectrpc/connect-query";
import { getPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { adaptPokemonToUi } from "../api/pokemon";

function parsePokemonIndex(id: string): number {
  // UUID v7 (from backend) has `-` characters, numeric IDs (from MSW mock) don't
  // Backend returns UUID, so we can't reliably extract a Pokemon number from it.
  // As a workaround, return 0. When backend adds an `index` field to Pokemon,
  // this function can be updated to use that field instead.
  if (id.includes("-")) {
    return 0;
  }

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
