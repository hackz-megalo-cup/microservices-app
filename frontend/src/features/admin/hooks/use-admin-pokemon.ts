import { createClient } from "@connectrpc/connect";
import { createConnectQueryKey, useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { MasterdataService } from "../../../gen/masterdata/v1/masterdata_pb";
import {
  getPokemon,
  listPokemon,
} from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";

const masterdataQueryKey = createConnectQueryKey({
  schema: MasterdataService,
  cardinality: undefined,
});

interface UseAdminPokemonOptions {
  id?: string;
  mode?: "create" | "edit";
}

export function useAdminPokemon(options?: UseAdminPokemonOptions) {
  const transport = useTransport();
  const queryClient = useQueryClient();
  const query = useQuery(listPokemon, {});
  const detailQuery = useQuery(
    getPokemon,
    { id: options?.id ?? "" },
    { enabled: options?.mode === "edit" && Boolean(options?.id) },
  );
  const client = useMemo(() => createClient(MasterdataService, transport), [transport]);

  const createMutation = useMutation({
    mutationFn: async (vars: Parameters<typeof client.createPokemon>[0]) =>
      client.createPokemon(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: masterdataQueryKey }),
  });

  const updateMutation = useMutation({
    mutationFn: async (vars: Parameters<typeof client.updatePokemon>[0]) =>
      client.updatePokemon(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: masterdataQueryKey }),
  });

  const deleteMutation = useMutation({
    mutationFn: async (vars: Parameters<typeof client.deletePokemon>[0]) =>
      client.deletePokemon(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: masterdataQueryKey }),
  });

  return {
    pokemon: query.data?.pokemon ?? [],
    isLoading: query.isPending,
    error: query.error,
    refetch: query.refetch,
    pokemonDetail: detailQuery.data?.pokemon,
    isDetailLoading: detailQuery.isPending,
    detailError: detailQuery.error,
    createMutation,
    updateMutation,
    deleteMutation,
  };
}
