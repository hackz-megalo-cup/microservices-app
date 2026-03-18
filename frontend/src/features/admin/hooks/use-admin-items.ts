import { createClient } from "@connectrpc/connect";
import { createConnectQueryKey, useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { MasterdataService } from "../../../gen/masterdata/v1/masterdata_pb";
import {
  getItem,
  listItems,
} from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";

const masterdataQueryKey = createConnectQueryKey({
  schema: MasterdataService,
  cardinality: undefined,
});

interface UseAdminItemsOptions {
  id?: string;
  mode?: "create" | "edit";
}

export function useAdminItems(options?: UseAdminItemsOptions) {
  const transport = useTransport();
  const queryClient = useQueryClient();

  const query = useQuery(listItems, {});
  const detailQuery = useQuery(
    getItem,
    { id: options?.id ?? "" },
    { enabled: options?.mode === "edit" && Boolean(options?.id) },
  );

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
    onSuccess: () => queryClient.invalidateQueries({ queryKey: masterdataQueryKey }),
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
    onSuccess: () => queryClient.invalidateQueries({ queryKey: masterdataQueryKey }),
  });

  const deleteMutation = useMutation({
    mutationFn: async (vars: { id: string }) =>
      client.deleteItem(
        { id: vars.id },
        { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: masterdataQueryKey }),
  });

  return {
    items: query.data?.items ?? [],
    isLoading: query.isPending,
    error: query.error,
    refetch: query.refetch,
    itemDetail: detailQuery.data?.item,
    isDetailLoading: detailQuery.isPending,
    detailError: detailQuery.error,
    createMutation,
    updateMutation,
    deleteMutation,
  };
}
