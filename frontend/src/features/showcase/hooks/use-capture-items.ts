import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useCallback, useMemo } from "react";
import type { UserItem } from "../../../gen/item/v1/item_pb";
import {
  getUserItems,
  useItem as useItemMutation,
} from "../../../gen/item/v1/item-ItemService_connectquery";
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
  const listItemsQuery = useQuery(listItems, {});
  const getUserItemsQuery = useQuery(getUserItems, { userId });
  const useItemMut = useMutation(useItemMutation);

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

  const handleUseItem = useCallback(
    (itemId: string, _bonus: number) => {
      useItemMut.mutate(
        { userId, itemId, quantity: 1 },
        {
          onSuccess: () => {
            void getUserItemsQuery.refetch();
          },
        },
      );
    },
    [userId, useItemMut, getUserItemsQuery],
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
