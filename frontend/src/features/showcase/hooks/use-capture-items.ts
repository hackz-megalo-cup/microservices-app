import { createClient } from "@connectrpc/connect";
import { useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import type { UserItem } from "../../../gen/item/v1/item_pb";
import { ItemService } from "../../../gen/item/v1/item_pb";
import { getUserItems } from "../../../gen/item/v1/item-ItemService_connectquery";
import type { Item } from "../../../gen/masterdata/v1/masterdata_pb";
import { listItems } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";

export interface CaptureItemsState {
  availableItems: Array<Item & { quantity: number }>;
  isLoading: boolean;
  error: Error | null;
  handleUseItem: (itemId: string, bonus: number) => void;
  isPending: boolean;
  refetch: () => void;
}

export function useCaptureItems(userId: string): CaptureItemsState {
  const transport = useTransport();
  const client = useMemo(() => createClient(ItemService, transport), [transport]);
  const invokeUseItemRpc = useMemo(() => client.useItem.bind(client), [client]);
  const invokeItemMutation = useCallback(
    (itemId: string, quantity: number) => {
      return invokeUseItemRpc(
        { userId, itemId, quantity },
        {
          headers: new Headers({
            "idempotency-key": crypto.randomUUID(),
          }),
        },
      );
    },
    [invokeUseItemRpc, userId],
  );
  const listItemsQuery = useQuery(listItems, {});
  const getUserItemsQuery = useQuery(getUserItems, { userId });

  const masterItems = useMemo<Item[]>(
    () => listItemsQuery.data?.items ?? [],
    [listItemsQuery.data?.items],
  );

  const userInventory = useMemo<UserItem[]>(
    () => getUserItemsQuery.data?.items ?? [],
    [getUserItemsQuery.data?.items],
  );

  const availableItems = useMemo(
    () =>
      masterItems
        .filter((item) => userInventory.some((inv) => inv.itemId === item.id && inv.quantity > 0))
        .map((item) => {
          const invEntry = userInventory.find((inv) => inv.itemId === item.id);
          return {
            ...item,
            quantity: invEntry?.quantity ?? 0,
          };
        }),
    [masterItems, userInventory],
  );

  const useItemMut = useMutation({
    mutationFn: async ({ itemId, quantity }: { itemId: string; quantity: number }) => {
      return invokeItemMutation(itemId, quantity);
    },
  });

  const handleUseItem = useCallback(
    (itemId: string, _bonus: number) => {
      useItemMut.mutate(
        { itemId, quantity: 1 },
        {
          onSuccess: () => {
            void getUserItemsQuery.refetch();
          },
        },
      );
    },
    [useItemMut, getUserItemsQuery],
  );

  const listItemsError =
    listItemsQuery.error instanceof Error
      ? listItemsQuery.error
      : listItemsQuery.error
        ? new Error("アイテム一覧取得失敗")
        : null;

  const getUserItemsError =
    getUserItemsQuery.error instanceof Error
      ? getUserItemsQuery.error
      : getUserItemsQuery.error
        ? new Error("ユーザーアイテム取得失敗")
        : null;

  const error = listItemsError ?? getUserItemsError ?? null;

  return {
    availableItems,
    isLoading: listItemsQuery.isPending || getUserItemsQuery.isPending,
    error,
    handleUseItem,
    isPending: useItemMut.isPending,
    refetch: () => {
      void listItemsQuery.refetch();
      void getUserItemsQuery.refetch();
    },
  };
}
