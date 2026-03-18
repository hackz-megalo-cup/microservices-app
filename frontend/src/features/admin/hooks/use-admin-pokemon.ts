import { createClient } from "@connectrpc/connect";
import { useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { MasterdataService } from "../../../gen/masterdata/v1/masterdata_pb";
import { listPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";

export function useAdminPokemon() {
  const transport = useTransport();
  const queryClient = useQueryClient();
  const query = useQuery(listPokemon, {});
  const client = useMemo(() => createClient(MasterdataService, transport), [transport]);

  const createMutation = useMutation({
    mutationFn: async (vars: Parameters<typeof client.createPokemon>[0]) =>
      client.createPokemon(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  const updateMutation = useMutation({
    mutationFn: async (vars: Parameters<typeof client.updatePokemon>[0]) =>
      client.updatePokemon(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  const deleteMutation = useMutation({
    mutationFn: async (vars: Parameters<typeof client.deletePokemon>[0]) =>
      client.deletePokemon(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  return {
    pokemon: query.data?.pokemon ?? [],
    isLoading: query.isPending,
    error: query.error,
    refetch: query.refetch,
    createMutation,
    updateMutation,
    deleteMutation,
  };
}
