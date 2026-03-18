import { createClient } from "@connectrpc/connect";
import { useCallback, useEffect, useRef, useState } from "react";
import { RaidLobbyService } from "../../../gen/raid_lobby/v1/raid_lobby_pb";
import { transport } from "../../../lib/transport";
import type { Participant } from "../types";
import { parseBattleStarted, parseParticipants } from "../utils/parse-lobby-payload";

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

      for await (const event of client.streamLobby(
        { lobbyId },
        { signal: abortController.signal },
      )) {
        console.log("[StreamLobby]", event.eventType, event.payload);

        switch (event.eventType) {
          case "participant_joined":
          case "participant_left": {
            const parsed = parseParticipants(event.payload);
            if (parsed) {
              setParticipants(parsed.participants);
            }
            break;
          }
          case "battle_started": {
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
  }, [lobbyId]);

  useEffect(() => {
    subscribe();
    return () => {
      // アンマウント時にストリームを切断
      abortRef.current?.abort();
    };
  }, [subscribe]);

  return { participants, isConnected, error, battleSessionId, reconnect: subscribe };
}
