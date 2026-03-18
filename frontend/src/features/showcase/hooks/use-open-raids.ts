import { useQuery } from "@connectrpc/connect-query";
import { listPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { listOpenRaids } from "../../../gen/raid_lobby/v1/raid_lobby-RaidLobbyService_connectquery";
import { getPokemonImageUrl } from "../api/pokemon";
import type { Raid } from "../types";

function formatElapsed(createdAtMs: number | null): string {
  if (createdAtMs === null) {
    return "-";
  }

  const elapsedMs = Date.now() - createdAtMs;
  if (elapsedMs <= 0) {
    return "just now";
  }

  const elapsedMin = Math.floor(elapsedMs / 60000);
  const elapsedSec = Math.floor((elapsedMs % 60000) / 1000);
  return `${elapsedMin}:${String(elapsedSec).padStart(2, "0")}`;
}

function toEpochMs(
  createdAt:
    | {
        seconds: bigint;
        nanos: number;
      }
    | undefined,
): number | null {
  if (!createdAt) {
    return null;
  }

  const seconds = Number(createdAt.seconds);
  if (!Number.isFinite(seconds)) {
    return null;
  }

  return seconds * 1000 + Math.floor(createdAt.nanos / 1_000_000);
}

export function useOpenRaids() {
  const openRaidsQuery = useQuery(listOpenRaids, { statusFilter: "waiting" });
  const pokemonQuery = useQuery(listPokemon, {});

  const pokemonMap = new Map(pokemonQuery.data?.pokemon.map((pokemon) => [pokemon.id, pokemon]));

  const raids: Raid[] =
    openRaidsQuery.data?.raids.map((entry) => {
      const boss = pokemonMap.get(entry.bossPokemonId);
      return {
        id: entry.id,
        name: boss?.name ?? `Boss ${entry.bossPokemonId}`,
        type: boss?.type || "Unknown",
        players: `${entry.currentParticipants}/${entry.maxParticipants}`,
        timer: formatElapsed(toEpochMs(entry.createdAt)),
        image: boss
          ? getPokemonImageUrl({ name: boss.name })
          : "/images/collection-placeholder.png",
      };
    }) ?? [];

  const error =
    openRaidsQuery.error instanceof Error
      ? openRaidsQuery.error
      : pokemonQuery.error instanceof Error
        ? pokemonQuery.error
        : openRaidsQuery.error || pokemonQuery.error
          ? new Error("レイド一覧取得失敗")
          : null;

  return {
    raids,
    isLoading: openRaidsQuery.isPending || pokemonQuery.isPending,
    error,
    refetch: openRaidsQuery.refetch,
  };
}
