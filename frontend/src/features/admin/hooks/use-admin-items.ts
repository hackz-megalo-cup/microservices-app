import { createClient } from "@connectrpc/connect";
import { useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { MasterdataService } from "../../../gen/masterdata/v1/masterdata_pb";
import { listItems } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";

export function useAdminItems() {
  const transport = useTransport();
  const queryClient = useQueryClient();

  const query = useQuery(listItems, {});

  const client = useMemo(() => createClient(MasterdataService, transport), [transport]);

  const createMutation = useMutation({
    mutationFn: async (vars: {
      name: string;
      effects: {
        effectType: string;
        targetType: string;
        captureRateBonus: number;
        flavorText: string;
      }[];
    }) =>
      client.createItem(
        { name: vars.name, effects: vars.effects },
        { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
      ),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  const updateMutation = useMutation({
    mutationFn: async (vars: {
      id: string;
      name: string;
      effects: {
        effectType: string;
        targetType: string;
        captureRateBonus: number;
        flavorText: string;
      }[];
    }) =>
      client.updateItem(
        { id: vars.id, name: vars.name, effects: vars.effects },
        { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
      ),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  const deleteMutation = useMutation({
    mutationFn: async (vars: { id: string }) =>
      client.deleteItem(
        { id: vars.id },
        { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
      ),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  return {
    items: query.data?.items ?? [],
    isLoading: query.isPending,
    error: query.error,
    refetch: query.refetch,
    createMutation,
    updateMutation,
    deleteMutation,
  };
}
