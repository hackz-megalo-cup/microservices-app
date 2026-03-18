import { createClient } from "@connectrpc/connect";
import { useTransport } from "@connectrpc/connect-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { LobbyService } from "../../../gen/lobby/v1/lobby_pb";
import { useAuthContext } from "../../../lib/auth";

export function useSetActivePokemon() {
  const transport = useTransport();
  const { user } = useAuthContext();
  const client = useMemo(() => createClient(LobbyService, transport), [transport]);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (pokemonId: string) => {
      if (!user?.id) {
        throw new Error("User not authenticated");
      }
      return client.setActivePokemon({ userId: user.id, pokemonId });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries();
    },
  });
}
