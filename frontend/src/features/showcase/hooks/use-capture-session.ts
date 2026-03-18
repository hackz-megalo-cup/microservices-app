import { createClient } from "@connectrpc/connect";
import { useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation } from "@tanstack/react-query";
import { useMemo } from "react";
import { CaptureService } from "../../../gen/capture/v1/capture_pb";
import { getCaptureSession } from "../../../gen/capture/v1/capture-CaptureService_connectquery";

export function useCaptureSession(sessionId: string) {
  const transport = useTransport();
  const client = useMemo(() => createClient(CaptureService, transport), [transport]);
  const invokeCaptureUseItem = useMemo(() => client.useItem.bind(client), [client]);

  const sessionQuery = useQuery(getCaptureSession, { sessionId });

  // All mutations must be declared at top level
  const itemMutation = useMutation({
    mutationFn: async ({ itemId }: { itemId: string }) => {
      const headers = new Headers({ "idempotency-key": crypto.randomUUID() });
      return invokeCaptureUseItem({ sessionId, itemId }, { headers });
    },
  });

  const ballMutation = useMutation({
    mutationFn: async () => {
      const headers = new Headers({ "idempotency-key": crypto.randomUUID() });
      return client.throwBall({ sessionId }, { headers });
    },
  });

  const sessionEndMutation = useMutation({
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
    itemMutation,
    ballMutation,
    sessionEndMutation,
  };
}
