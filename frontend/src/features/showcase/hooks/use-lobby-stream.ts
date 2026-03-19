import { createClient } from "@connectrpc/connect";
import { useTransport } from "@connectrpc/connect-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { RaidLobbyService } from "../../../gen/raid_lobby/v1/raid_lobby_pb";
import type { Participant } from "../../../lib/parse-lobby-payload";
import {
  parseBattleStarted,
  parseParticipantEvent,
  parseParticipants,
} from "../../../lib/parse-lobby-payload";

interface UseLobbyStreamResult {
  participants: Participant[];
  isConnected: boolean;
  error: Error | null;
  battleSessionId: string | null;
  reconnect: () => void;
}

/**
 * StreamLobby RPC を購読してロビーイベントをリアルタイムで受信する
 *
 * @param lobbyId - ロビーID
 * @returns ロビー状態とストリーム接続状態
 */
export function useLobbyStream(lobbyId: string): UseLobbyStreamResult {
  const transport = useTransport();
  const [participants, setParticipants] = useState<Participant[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [battleSessionId, setBattleSessionId] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const subscribe = useCallback(async () => {
    if (!lobbyId) {
      return;
    }

    // 既存の接続があれば切断
    abortRef.current?.abort();
    const abortController = new AbortController();
    abortRef.current = abortController;

    const client = createClient(RaidLobbyService, transport);

    try {
      setIsConnected(true);
      setError(null);
      setParticipants([]);
      setBattleSessionId(null);

      for await (const event of client.streamLobby(
        { lobbyId },
        { signal: abortController.signal, timeoutMs: 0 },
      )) {
        switch (event.eventType) {
          case "raid.participant_snapshot":
          case "raid.user_joined": {
            const participantEvent = parseParticipantEvent(event.payload);
            if (participantEvent) {
              setParticipants((prev) => {
                if (
                  prev.some((participant) => participant.id === participantEvent.participant.id)
                ) {
                  return prev;
                }
                return [...prev, participantEvent.participant];
              });
              break;
            }

            const parsed = parseParticipants(event.payload);
            if (parsed) {
              setParticipants(parsed.participants);
            }
            break;
          }
          case "raid.battle_started": {
            const parsed = parseBattleStarted(event.payload);
            if (parsed) {
              setBattleSessionId(parsed.battleSessionId);
            }
            break;
          }
        }
      }
    } catch (err) {
      if (abortController.signal.aborted) {
        // 意図的な切断 - エラーとして扱わない
        return;
      }
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setIsConnected(false);
    }
  }, [lobbyId, transport]);

  useEffect(() => {
    subscribe();
    return () => {
      // アンマウント時にストリームを切断
      abortRef.current?.abort();
    };
  }, [subscribe]);

  return { participants, isConnected, error, battleSessionId, reconnect: subscribe };
}
