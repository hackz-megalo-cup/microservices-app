import { useQuery } from "@connectrpc/connect-query";
import { getActivePokemon } from "../../../gen/lobby/v1/lobby-LobbyService_connectquery";
import { listPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { useAuthContext } from "../../../lib/auth";
import { adaptPokemonToUi } from "../api/pokemon";
import type { Pokemon } from "../types";

export function useActivePokemon(): {
  activePokemon: Pokemon | null;
  activePokemonId: string;
  isLoading: boolean;
  error: Error | null;
} {
  const { user } = useAuthContext();

  const activePokemonQuery = useQuery(getActivePokemon, { userId: user?.id ?? "" });
  const pokemonListQuery = useQuery(listPokemon, {});

  const activePokemonId = activePokemonQuery.data?.pokemonId ?? "";
  const pokemonList = pokemonListQuery.data?.pokemon ?? [];

  const pokemonIndex = pokemonList.findIndex((p) => p.id === activePokemonId);
  const activePokemon: Pokemon | null =
    pokemonIndex >= 0 ? adaptPokemonToUi(pokemonList[pokemonIndex], pokemonIndex) : null;

  const error =
    activePokemonQuery.error instanceof Error
      ? activePokemonQuery.error
      : pokemonListQuery.error instanceof Error
        ? pokemonListQuery.error
        : null;

  return {
    activePokemon,
    activePokemonId,
    isLoading: activePokemonQuery.isPending || pokemonListQuery.isPending,
    error,
  };
}
