import { createClient } from "@connectrpc/connect";
import { useTransport } from "@connectrpc/connect-query";
import { useMutation } from "@tanstack/react-query";
import { useMemo } from "react";
import { RaidLobbyService } from "../../../gen/raid_lobby/v1/raid_lobby_pb";

interface JoinLobbyVariables {
  lobbyId: string;
}

interface StartBattleVariables {
  lobbyId: string;
}

export function useLobbyActions() {
  const transport = useTransport();
  const client = useMemo(() => createClient(RaidLobbyService, transport), [transport]);

  const joinMutation = useMutation({
    mutationFn: async ({ lobbyId }: JoinLobbyVariables) => {
      return client.joinRaid(
        { lobbyId },
        {
          headers: new Headers({
            "idempotency-key": crypto.randomUUID(),
          }),
        },
      );
    },
  });

  const startMutation = useMutation({
    mutationFn: async ({ lobbyId }: StartBattleVariables) => {
      return client.startBattle(
        { lobbyId },
        {
          headers: new Headers({
            "idempotency-key": crypto.randomUUID(),
          }),
        },
      );
    },
  });

  return {
    joinMutation,
    startMutation,
  };
}
