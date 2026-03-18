import { createClient } from "@connectrpc/connect";
import { createConnectQueryKey, useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { MasterdataService } from "../../../gen/masterdata/v1/masterdata_pb";
import { listTypeMatchups } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";

const masterdataQueryKey = createConnectQueryKey({
  schema: MasterdataService,
  cardinality: undefined,
});

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
    onSuccess: () => queryClient.invalidateQueries({ queryKey: masterdataQueryKey }),
  });

  const updateMutation = useMutation({
    mutationFn: async (vars: Parameters<typeof client.updateTypeMatchup>[0]) =>
      client.updateTypeMatchup(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: masterdataQueryKey }),
  });

  const deleteMutation = useMutation({
    mutationFn: async (vars: Parameters<typeof client.deleteTypeMatchup>[0]) =>
      client.deleteTypeMatchup(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: masterdataQueryKey }),
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
