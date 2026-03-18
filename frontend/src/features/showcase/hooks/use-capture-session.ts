import { createClient } from "@connectrpc/connect";
import { useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation } from "@tanstack/react-query";
import { useMemo } from "react";
import { getCaptureSession } from "../../../gen/capture/v1/capture-CaptureService_connectquery";
import { CaptureService } from "../../../gen/capture/v1/capture_pb";

export function useCaptureSession(sessionId: string) {
  const transport = useTransport();
  const client = useMemo(() => createClient(CaptureService, transport), [transport]);

  const sessionQuery = useQuery(getCaptureSession, { sessionId });

  const useItemMutation = useMutation({
    mutationFn: async ({ itemId }: { itemId: string }) => {
      return client.useItem(
        { sessionId, itemId },
        { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
      );
    },
  });

  const throwBallMutation = useMutation({
    mutationFn: async () => {
      return client.throwBall(
        { sessionId },
        { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
      );
    },
  });

  const endSessionMutation = useMutation({
    mutationFn: async () => {
      return client.endSession({ sessionId });
    },
  });

  const error =
    sessionQuery.error instanceof Error
      ? sessionQuery.error
      : sessionQuery.error
        ? new Error("セッション取得失敗")
        : null;

  return {
    session: sessionQuery.data,
    isLoading: sessionQuery.isPending,
    error,
    refetch: () => void sessionQuery.refetch(),
    useItemMutation,
    throwBallMutation,
    endSessionMutation,
  };
}
