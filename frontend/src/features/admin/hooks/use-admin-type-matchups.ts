import { createClient } from "@connectrpc/connect";
import { useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { MasterdataService } from "../../../gen/masterdata/v1/masterdata_pb";
import { listTypeMatchups } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";

export function useAdminTypeMatchups() {
  const transport = useTransport();
  const queryClient = useQueryClient();
  const query = useQuery(listTypeMatchups, {});
  const client = useMemo(() => createClient(MasterdataService, transport), [transport]);

  const createMutation = useMutation({
    mutationFn: async (vars: Parameters<typeof client.createTypeMatchup>[0]) =>
      client.createTypeMatchup(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  const updateMutation = useMutation({
    mutationFn: async (vars: Parameters<typeof client.updateTypeMatchup>[0]) =>
      client.updateTypeMatchup(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  const deleteMutation = useMutation({
    mutationFn: async (vars: Parameters<typeof client.deleteTypeMatchup>[0]) =>
      client.deleteTypeMatchup(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  return {
    matchups: query.data?.matchups ?? [],
    isLoading: query.isPending,
    error: query.error,
    refetch: query.refetch,
    createMutation,
    updateMutation,
    deleteMutation,
  };
}
