import { useQuery } from "@connectrpc/connect-query";
import { listPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { adaptPokemonToUi } from "../api/pokemon";

export function useCollectionPokemon() {
  const query = useQuery(listPokemon, {});

  const pokemon = query.data?.pokemon.map((entry, index) => adaptPokemonToUi(entry, index)) ?? [];
  const error =
    query.error instanceof Error
      ? query.error
      : query.error
        ? new Error("マスターデータ取得失敗")
        : null;

  return {
    pokemon,
    isLoading: query.isPending,
    error,
    refetch: query.refetch,
  };
}
