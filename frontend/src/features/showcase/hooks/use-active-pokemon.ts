import { useQuery } from "@connectrpc/connect-query";
import { getActivePokemon } from "../../../gen/lobby/v1/lobby-LobbyService_connectquery";
import { listPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { useAuthContext } from "../../../lib/auth";
import { adaptPokemonToUi } from "../api/pokemon";
import type { Pokemon } from "../types";

export function useActivePokemon() {
  const { user } = useAuthContext();
  const userId = user?.id ?? "";

  const activePokemonQuery = useQuery(getActivePokemon, { userId }, { enabled: !!userId });
  const pokemonListQuery = useQuery(listPokemon, {});

  const activeId = activePokemonQuery.data?.pokemonId ?? null;

  const activePokemon: Pokemon | null = (() => {
    if (!activeId || !pokemonListQuery.data) {
      return null;
    }
    const index = pokemonListQuery.data.pokemon.findIndex((p) => p.id === activeId);
    if (index < 0) {
      return null;
    }
    return adaptPokemonToUi(pokemonListQuery.data.pokemon[index], index);
  })();

  const error =
    activePokemonQuery.error instanceof Error
      ? activePokemonQuery.error
      : pokemonListQuery.error instanceof Error
        ? pokemonListQuery.error
        : activePokemonQuery.error || pokemonListQuery.error
          ? new Error("アクティブポケモン取得失敗")
          : null;

  return {
    activePokemon,
    activeId,
    isLoading: activePokemonQuery.isPending || pokemonListQuery.isPending,
    error,
  };
}
