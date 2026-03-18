import { createClient } from "@connectrpc/connect";
import { useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { LobbyService } from "../../../gen/lobby/v1/lobby_pb";
import { getActivePokemon } from "../../../gen/lobby/v1/lobby-LobbyService_connectquery";
import { getPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { adaptPokemonToUi } from "../api/pokemon";

export function useActivePokemon(userId: string) {
  const transport = useTransport();
  const queryClient = useQueryClient();
  const client = useMemo(() => createClient(LobbyService, transport), [transport]);

  const activePokemonQuery = useQuery(getActivePokemon, { userId }, { enabled: !!userId });
  const pokemonId = activePokemonQuery.data?.pokemonId ?? "";

  const pokemonDetailQuery = useQuery(getPokemon, { id: pokemonId }, { enabled: !!pokemonId });

  const activePokemon = pokemonDetailQuery.data?.pokemon
    ? adaptPokemonToUi(pokemonDetailQuery.data.pokemon, 0)
    : null;

  const setActivePokemonMutation = useMutation({
    mutationFn: async (newPokemonId: string) => {
      return client.setActivePokemon(
        { userId, pokemonId: newPokemonId },
        {
          headers: new Headers({
            "idempotency-key": crypto.randomUUID(),
          }),
        },
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries();
    },
  });

  const error =
    activePokemonQuery.error instanceof Error
      ? activePokemonQuery.error
      : activePokemonQuery.error
        ? new Error("アクティブポケモン取得失敗")
        : null;

  return {
    activePokemonId: pokemonId,
    activePokemon,
    isLoading: activePokemonQuery.isPending || (!!pokemonId && pokemonDetailQuery.isPending),
    error,
    setActivePokemon: setActivePokemonMutation.mutate,
    isSettingPokemon: setActivePokemonMutation.isPending,
    setError:
      setActivePokemonMutation.error instanceof Error ? setActivePokemonMutation.error : null,
  };
}
