import { createClient } from "@connectrpc/connect";
import { useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { listPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { RaidLobbyService } from "../../../gen/raid_lobby/v1/raid_lobby_pb";
import { listOpenRaids } from "../../../gen/raid_lobby/v1/raid_lobby-RaidLobbyService_connectquery";
import type { AdminRaid } from "../types";

function toEpochMs(createdAt: { seconds: bigint; nanos: number } | undefined): number | null {
  if (!createdAt) {
    return null;
  }
  const seconds = Number(createdAt.seconds);
  return Number.isFinite(seconds) ? seconds * 1000 + Math.floor(createdAt.nanos / 1_000_000) : null;
}

export function useAdminRaids() {
  const transport = useTransport();
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilter] = useState("");

  const raidsQuery = useQuery(listOpenRaids, { statusFilter });
  const pokemonQuery = useQuery(listPokemon, {});

  const pokemonMap = new Map(pokemonQuery.data?.pokemon.map((p) => [p.id, p]));

  const raids: AdminRaid[] =
    raidsQuery.data?.raids.map((entry) => {
      const boss = pokemonMap.get(entry.bossPokemonId);
      return {
        id: entry.id,
        bossName: boss?.name ?? `Boss ${entry.bossPokemonId}`,
        currentParticipants: entry.currentParticipants,
        maxParticipants: entry.maxParticipants,
        status: entry.status,
        createdAtMs: toEpochMs(entry.createdAt),
      };
    }) ?? [];

  const error =
    raidsQuery.error instanceof Error
      ? raidsQuery.error
      : pokemonQuery.error instanceof Error
        ? pokemonQuery.error
        : raidsQuery.error || pokemonQuery.error
          ? new Error("レイド一覧取得失敗")
          : null;

  const client = useMemo(() => createClient(RaidLobbyService, transport), [transport]);

  const createMutation = useMutation({
    mutationFn: async (vars: { bossPokemonId: string }) =>
      client.createRaid(vars, { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) }),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  const startMutation = useMutation({
    mutationFn: async (vars: { lobbyId: string }) =>
      client.startBattle(vars, {
        headers: new Headers({ "idempotency-key": crypto.randomUUID() }),
      }),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  return {
    raids,
    isLoading: raidsQuery.isPending || pokemonQuery.isPending,
    error,
    statusFilter,
    setStatusFilter,
    createMutation,
    startMutation,
  };
}
